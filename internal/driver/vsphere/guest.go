package vsphere

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os/user"
	"strings"
	"time"

	"github.com/vmware/govmomi/guest"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/i18n"
)

// guestOps is the seam between Exec's orchestration and the vSphere
// guest operations API. vcsim only runs guest programs inside
// container VMs, so tests inject a fake through Driver.guest.
type guestOps interface {
	startProgram(ctx context.Context, program, args string) (int64, error)
	waitProcess(ctx context.Context, pid int64) (int, error)
	readFile(ctx context.Context, path string) (string, error)
	deleteFile(ctx context.Context, path string) error
}

// Exec implements driver.Driver.
//
// Like the vmrun backend, the command runs as `sh -c '<script>' with
// its output redirected to a temp file inside the guest; there is no
// exit-code file because GuestProcessInfo reports ExitCode directly.
// The wrapper keeps syntax errors inside the inner shell: they land
// in the captured output with a non-zero exit code.
func (d *Driver) Exec(ctx context.Context, ref string, opts driver.ExecOptions) (string, int, error) {
	if strings.TrimSpace(opts.Script) == "" {
		return "", 0, errors.New(i18n.T("err.exec.empty"))
	}
	vm, err := d.resolve(ctx, ref)
	if err != nil {
		return "", 0, err
	}
	state, err := d.powerStateOf(ctx, vm)
	if err != nil {
		return "", 0, err
	}
	if state != types.VirtualMachinePowerStatePoweredOn {
		return "", 0, errors.New(i18n.T("err.vsphere.notRunning"))
	}

	ops := d.guest
	if ops == nil {
		ops, err = d.newGuestOps(ctx, vm, opts)
		if err != nil {
			return "", 0, err
		}
	}

	tag, err := randomTag()
	if err != nil {
		return "", 0, err
	}
	outGuest := "/tmp/.vmctl-exec-" + tag
	args := fmt.Sprintf("-c %s >%s 2>&1", shQuote(opts.Script), shQuote(outGuest))

	pid, err := ops.startProgram(ctx, "/bin/sh", args)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = ops.deleteFile(ctx, outGuest) }()

	code, err := ops.waitProcess(ctx, pid)
	if err != nil {
		return "", code, err
	}
	out, err := ops.readFile(ctx, outGuest)
	if err != nil {
		return "", code, fmt.Errorf(i18n.T("err.exec.fetchOutput"), err)
	}
	return out, code, nil
}

// GuestIP implements driver.Driver. wait=false reports the current
// guest.ipAddress (an empty value is an error); wait=true blocks
// until VMware Tools publishes one.
func (d *Driver) GuestIP(ctx context.Context, ref string, wait bool) (string, error) {
	vm, err := d.resolve(ctx, ref)
	if err != nil {
		return "", err
	}
	if wait {
		ip, err := vm.WaitForIP(ctx, true)
		if err != nil {
			return "", err
		}
		if ip == "" {
			return "", errors.New(i18n.T("err.vsphere.noIP"))
		}
		return ip, nil
	}
	c, err := d.connect(ctx)
	if err != nil {
		return "", err
	}
	var m mo.VirtualMachine
	if err := c.RetrieveOne(ctx, vm.Reference(), []string{"guest.ipAddress"}, &m); err != nil {
		return "", err
	}
	if m.Guest == nil || m.Guest.IpAddress == "" {
		return "", errors.New(i18n.T("err.vsphere.noIP"))
	}
	return m.Guest.IpAddress, nil
}

// realGuestOps talks to the vSphere guest operations API, creating
// managers per call so each one picks up a fresh session.
type realGuestOps struct {
	d     *Driver
	vmRef types.ManagedObjectReference
	auth  types.BaseGuestAuthentication
}

// newGuestOps builds the API-backed guestOps. An empty user falls
// back to the current host user (mirroring vmrun's behaviour of
// reusing the local account when -gu is absent).
func (d *Driver) newGuestOps(ctx context.Context, vm *object.VirtualMachine, opts driver.ExecOptions) (guestOps, error) {
	if _, err := d.connect(ctx); err != nil {
		return nil, err
	}
	userName := opts.User
	if userName == "" {
		if u, err := user.Current(); err == nil {
			userName = u.Username
		}
	}
	return &realGuestOps{
		d:     d,
		vmRef: vm.Reference(),
		auth:  &types.NamePasswordAuthentication{Username: userName, Password: opts.Password},
	}, nil
}

func (g *realGuestOps) startProgram(ctx context.Context, program, args string) (int64, error) {
	pm, err := g.processManager(ctx)
	if err != nil {
		return 0, err
	}
	return pm.StartProgram(ctx, g.auth, &types.GuestProgramSpec{
		ProgramPath: program,
		Arguments:   args,
	})
}

// guestPollInterval is the pause between process state checks.
const guestPollInterval = 500 * time.Millisecond

// waitProcess polls until the process ends. EndTime is available for
// up to five minutes after completion (vSphere API contract), which
// comfortably bounds normal commands.
func (g *realGuestOps) waitProcess(ctx context.Context, pid int64) (int, error) {
	pm, err := g.processManager(ctx)
	if err != nil {
		return 0, err
	}
	for {
		procs, err := pm.ListProcesses(ctx, g.auth, []int64{pid})
		if err != nil {
			return 0, err
		}
		for _, p := range procs {
			if p.Pid == pid && p.EndTime != nil {
				return int(p.ExitCode), nil
			}
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(guestPollInterval):
		}
	}
}

func (g *realGuestOps) readFile(ctx context.Context, path string) (string, error) {
	fm, err := g.fileManager(ctx)
	if err != nil {
		return "", err
	}
	info, err := fm.InitiateFileTransferFromGuest(ctx, g.auth, path)
	if err != nil {
		return "", err
	}
	u, err := fm.TransferURL(ctx, info.Url)
	if err != nil {
		return "", err
	}
	c, err := g.d.connect(ctx)
	if err != nil {
		return "", err
	}
	p := soap.DefaultDownload
	p.Close = true // disable Keep-Alive, matching govmomi's guest client
	rc, _, err := c.Client.Download(ctx, u, &p)
	if err != nil {
		return "", err
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (g *realGuestOps) deleteFile(ctx context.Context, path string) error {
	fm, err := g.fileManager(ctx)
	if err != nil {
		return err
	}
	return fm.DeleteFile(ctx, g.auth, path)
}

func (g *realGuestOps) processManager(ctx context.Context) (*guest.ProcessManager, error) {
	c, err := g.d.connect(ctx)
	if err != nil {
		return nil, err
	}
	return guest.NewOperationsManager(c.Client, g.vmRef).ProcessManager(ctx)
}

func (g *realGuestOps) fileManager(ctx context.Context) (*guest.FileManager, error) {
	c, err := g.d.connect(ctx)
	if err != nil {
		return nil, err
	}
	return guest.NewOperationsManager(c.Client, g.vmRef).FileManager(ctx)
}

// shQuote wraps s in single quotes for /bin/sh.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// randomTag returns a hex token for guest temp file names.
func randomTag() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}
