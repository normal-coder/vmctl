// Package vmrun implements driver.Driver on top of VMware's vmrun CLI
// (local Fusion/Workstation).
package vmrun

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/model"
)

func init() {
	driver.Register("vmrun", func(opts driver.Options) (driver.Driver, error) {
		return New(opts)
	})
}

// Driver controls local VMs through the vmrun binary.
type Driver struct {
	runner  *runner
	invPath string // inventory file, overridable in tests
}

// New builds a vmrun driver, locating the vmrun binary.
func New(opts driver.Options) (*Driver, error) {
	bin, hostType, err := locate(opts.VMRunPath)
	if err != nil {
		return nil, err
	}
	return &Driver{
		runner:  &runner{bin: bin, hostType: hostType, execFn: defaultExec},
		invPath: defaultInventoryPath(),
	}, nil
}

var _ driver.Driver = (*Driver)(nil)

// Name implements driver.Driver.
func (d *Driver) Name() string { return "vmrun" }

// inventory reads registered VMs from the Fusion inventory file.
func (d *Driver) inventory() ([]InventoryEntry, error) {
	return readInventory(d.invPath)
}

// listRunning returns the set of currently running (incl. paused) vmx paths.
func (d *Driver) listRunning(ctx context.Context) (map[string]bool, error) {
	out, err := d.runner.run(ctx, "list")
	if err != nil {
		return nil, err
	}
	running := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.EqualFold(filepath.Ext(line), ".vmx") {
			continue // skips the "Total running VMs: N" header
		}
		running[absPath(line)] = true
	}
	return running, nil
}

// List implements driver.Driver.
func (d *Driver) List(ctx context.Context) ([]model.VM, error) {
	running, err := d.listRunning(ctx)
	if err != nil {
		return nil, err
	}
	entries, err := d.inventory()
	if err != nil {
		return nil, err
	}

	type row struct {
		path, name, uuid string
	}
	rows := []row{}
	seen := map[string]bool{}
	add := func(path, name, uuid string) {
		p := absPath(path)
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		rows = append(rows, row{path: p, name: name, uuid: uuid})
	}
	for _, e := range entries {
		add(e.Path, e.Name, e.UUID)
	}
	// VMs running but missing from the inventory still belong in the list.
	for p := range running {
		add(p, "", "")
	}

	vms := make([]model.VM, 0, len(rows))
	for _, r := range rows {
		vm := model.VM{Path: r.path, ID: r.uuid, Name: r.name, State: model.StateOff}

		if cfg, err := readVMX(r.path); err == nil {
			if vm.Name == "" {
				vm.Name = cfg.DisplayName
			}
			if vm.ID == "" {
				vm.ID = cfg.UUID
			}
			vm.CPUs = cfg.NumCPUs
			vm.MemoryMB = cfg.MemSizeMB
			vm.GuestOS = cfg.GuestOS
		}
		if vm.Name == "" {
			vm.Name = vmNameFromPath(r.path)
		}
		vm.State = d.stateOf(r.path, running)
		vms = append(vms, vm)
	}
	return vms, nil
}

// stateOf derives power state: running set wins, then suspend-file
// detection, else powered off. vmrun has no query for power state.
func (d *Driver) stateOf(path string, running map[string]bool) model.State {
	if running[absPath(path)] {
		return model.StateOn
	}
	if suspendFileExists(path) {
		return model.StateSuspended
	}
	if _, err := os.Stat(path); err != nil {
		return model.StateUnknown // vmx missing entirely
	}
	return model.StateOff
}

func suspendFileExists(vmxPath string) bool {
	vmss := strings.TrimSuffix(vmxPath, ".vmx") + ".vmss"
	if _, err := os.Stat(vmss); err == nil {
		return true
	}
	// Some VMs use the bundle name for the suspend file.
	if dir := filepath.Dir(vmxPath); strings.EqualFold(filepath.Ext(dir), ".vmwarevm") {
		vmss = strings.TrimSuffix(dir, ".vmwarevm") + ".vmss"
		if _, err := os.Stat(vmss); err == nil {
			return true
		}
	}
	return false
}

func vmNameFromPath(vmxPath string) string {
	base := filepath.Base(vmxPath)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// Info implements driver.Driver.
func (d *Driver) Info(ctx context.Context, ref string) (*model.VMInfo, error) {
	path, err := d.resolve(ref)
	if err != nil {
		return nil, err
	}

	vm := model.VM{Path: path, Name: vmNameFromPath(path), State: model.StateUnknown}
	if cfg, err := readVMX(path); err == nil {
		if cfg.DisplayName != "" {
			vm.Name = cfg.DisplayName
		}
		vm.ID = cfg.UUID
		vm.CPUs = cfg.NumCPUs
		vm.MemoryMB = cfg.MemSizeMB
		vm.GuestOS = cfg.GuestOS
	}

	running, err := d.listRunning(ctx)
	if err != nil {
		return nil, err
	}
	vm.State = d.stateOf(path, running)

	info := &model.VMInfo{VM: vm}

	// Tools state: fails while powered off — report empty, not an error.
	if out, err := d.runner.run(ctx, "checkToolsState", path); err == nil {
		info.ToolsState = firstLine(out)
	}
	if vm.State == model.StateOn {
		if out, err := d.runner.run(ctx, "getGuestIPAddress", path); err == nil {
			info.GuestIP = parseGuestIP(out)
		}
	}
	return info, nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

func parseGuestIP(out string) string {
	for _, field := range strings.Fields(out) {
		field = strings.Trim(field, ",;")
		if ip, err := netip.ParseAddr(field); err == nil {
			return ip.String()
		}
		// vmrun prints "Guest IP: 192.168.x.x" — try the token after it.
		if strings.EqualFold(field, "IP:") || strings.EqualFold(field, "Guest") {
			continue
		}
	}
	// Handle the "Guest IP: x.x.x.x" single-line form explicitly.
	if _, after, ok := strings.Cut(out, ":"); ok {
		for _, field := range strings.Fields(after) {
			if ip, err := netip.ParseAddr(strings.Trim(field, ",;")); err == nil {
				return ip.String()
			}
		}
	}
	return ""
}

// Start implements driver.Driver.
//
// vmrun start blocks until the guest reports ready (Tools running),
// which can take minutes — or never finish — on guests without
// VMware Tools. vmrun is therefore launched in the background and
// Start returns as soon as the VM shows up as powered on, matching
// prlctl's semantics. The detached vmrun keeps waiting for guest
// readiness on its own and exits once the guest settles.
func (d *Driver) Start(ctx context.Context, ref string, opts driver.StartOptions) error {
	path, err := d.resolve(ref)
	if err != nil {
		return err
	}
	mode := "gui"
	if opts.NoGUI {
		mode = "nogui"
	}
	go func() {
		bg, cancel := context.WithTimeout(context.Background(), startBackgroundTimeout)
		defer cancel()
		_, _ = d.runner.run(bg, "start", path, mode)
	}()
	return d.waitPoweredOn(ctx, path)
}

// startBackgroundTimeout bounds the detached vmrun start process.
const startBackgroundTimeout = 10 * time.Minute

// powerOnTimeout is how long Start waits for the VM to appear in
// `vmrun list` before giving up.
const powerOnTimeout = 30 * time.Second

// waitPoweredOn polls `vmrun list` until the VM is running.
func (d *Driver) waitPoweredOn(ctx context.Context, path string) error {
	deadline := time.Now().Add(powerOnTimeout)
	for {
		running, err := d.listRunning(ctx)
		if err != nil {
			return err
		}
		if running[absPath(path)] {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for %q to power on", path)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}

// Stop implements driver.Driver.
func (d *Driver) Stop(ctx context.Context, ref string, opts driver.StopOptions) error {
	path, err := d.resolve(ref)
	if err != nil {
		return err
	}
	mode := "soft"
	if opts.Hard {
		mode = "hard"
	}
	_, err = d.runner.run(ctx, "stop", path, mode)
	return err
}

// Suspend implements driver.Driver.
func (d *Driver) Suspend(ctx context.Context, ref string, opts driver.SuspendOptions) error {
	path, err := d.resolve(ref)
	if err != nil {
		return err
	}
	mode := "soft"
	if opts.Hard {
		mode = "hard"
	}
	_, err = d.runner.run(ctx, "suspend", path, mode)
	return err
}

// Resume implements driver.Driver: starts a suspended VM, unpauses a
// paused one, and no-ops on a running VM. vmrun cannot distinguish
// paused from running, so unpause errors on a running VM are ignored.
func (d *Driver) Resume(ctx context.Context, ref string) error {
	path, err := d.resolve(ref)
	if err != nil {
		return err
	}
	running, err := d.listRunning(ctx)
	if err != nil {
		return err
	}
	switch d.stateOf(path, running) {
	case model.StateSuspended:
		_, err = d.runner.run(ctx, "start", path)
		return err
	case model.StateOff:
		return fmt.Errorf("%q is powered off, use start", ref)
	default: // on or unknown — attempt unpause
		if _, uerr := d.runner.run(ctx, "unpause", path); uerr != nil {
			if running[absPath(path)] {
				return nil // running and not paused: nothing to do
			}
			return uerr
		}
		return nil
	}
}

// Pause implements driver.Driver.
func (d *Driver) Pause(ctx context.Context, ref string) error {
	path, err := d.resolve(ref)
	if err != nil {
		return err
	}
	_, err = d.runner.run(ctx, "pause", path)
	return err
}

// Reset implements driver.Driver.
func (d *Driver) Reset(ctx context.Context, ref string, opts driver.ResetOptions) error {
	path, err := d.resolve(ref)
	if err != nil {
		return err
	}
	mode := "soft"
	if opts.Hard {
		mode = "hard"
	}
	_, err = d.runner.run(ctx, "reset", path, mode)
	return err
}
