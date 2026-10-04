package vsphere

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vmware/govmomi/vim25/types"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/i18n"
	"gitee.com/normalcoder/vmctl/internal/model"
)

// fakeGuest records what Exec orchestrated and plays back canned
// results. vcsim runs guest programs only inside container VMs, so
// this fake stands in for the real guestOps API.
type fakeGuest struct {
	program string
	args    string
	read    string // path passed to readFile
	deleted string // path passed to deleteFile
	calls   []string

	pid  int64
	code int
	out  string

	startErr, waitErr, readErr, deleteErr error
	block                                 bool // waitProcess blocks until ctx ends
}

func (f *fakeGuest) startProgram(ctx context.Context, program, args string) (int64, error) {
	f.calls = append(f.calls, "start")
	f.program, f.args = program, args
	return f.pid, f.startErr
}

func (f *fakeGuest) waitProcess(ctx context.Context, pid int64) (int, error) {
	f.calls = append(f.calls, "wait")
	if f.block {
		<-ctx.Done()
		return 0, ctx.Err()
	}
	return f.code, f.waitErr
}

func (f *fakeGuest) readFile(ctx context.Context, path string) (string, error) {
	f.calls = append(f.calls, "read")
	f.read = path
	return f.out, f.readErr
}

func (f *fakeGuest) deleteFile(ctx context.Context, path string) error {
	f.calls = append(f.calls, "delete")
	f.deleted = path
	return f.deleteErr
}

// execSim starts a simulator with a running first VM and the fake
// guest ops attached.
func execSim(t *testing.T, fake *fakeGuest) (*Driver, context.Context, model.VM) {
	t.Helper()
	d, ctx := newSim(t)
	vm := firstVM(t, d, ctx)
	if err := d.Start(ctx, vm.Name, driver.StartOptions{}); err != nil {
		t.Fatal(err)
	}
	d.guest = fake
	return d, ctx, vm
}

func TestExecEmptyScript(t *testing.T) {
	d, ctx := newSim(t)
	vm := firstVM(t, d, ctx)
	if _, _, err := d.Exec(ctx, vm.Name, driver.ExecOptions{Script: "   "}); err == nil {
		t.Fatal("empty script must error")
	} else if err.Error() != i18n.T("err.exec.empty") {
		t.Errorf("want err.exec.empty, got %v", err)
	}
}

func TestExecRequiresRunning(t *testing.T) {
	d, ctx := newSim(t)
	vm := firstVM(t, d, ctx) // off by default
	fake := &fakeGuest{}
	d.guest = fake
	if _, _, err := d.Exec(ctx, vm.Name, driver.ExecOptions{Script: "true"}); err == nil {
		t.Fatal("exec on a powered-off VM must error")
	}
	if len(fake.calls) != 0 {
		t.Errorf("guest ops must not be touched, got %v", fake.calls)
	}
}

func TestExecWrapsAndReports(t *testing.T) {
	fake := &fakeGuest{pid: 42, code: 7, out: "hello\n"}
	d, ctx, vm := execSim(t, fake)

	out, code, err := d.Exec(ctx, vm.Name, driver.ExecOptions{Script: "echo hello"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "hello\n" || code != 7 {
		t.Errorf("got (%q, %d), want (%q, 7)", out, code, "hello\n")
	}
	if fake.program != "/bin/sh" {
		t.Errorf("program = %q, want /bin/sh", fake.program)
	}
	if !strings.Contains(fake.args, "-c ") || !strings.Contains(fake.args, "echo hello") {
		t.Errorf("args = %q, want -c with the script", fake.args)
	}
	if !strings.Contains(fake.args, "/tmp/.vmctl-exec-") || !strings.Contains(fake.args, "2>&1") {
		t.Errorf("args = %q, want output redirect to a temp file", fake.args)
	}
	// The temp file read back must be the same one the redirect wrote,
	// and cleanup must target it too.
	if fake.read == "" || fake.deleted != fake.read {
		t.Errorf("cleanup path %q != read path %q", fake.deleted, fake.read)
	}
}

func TestExecStartFailure(t *testing.T) {
	fake := &fakeGuest{startErr: errors.New("boom")}
	d, ctx, vm := execSim(t, fake)
	if _, _, err := d.Exec(ctx, vm.Name, driver.ExecOptions{Script: "true"}); err == nil || err.Error() != "boom" {
		t.Errorf("want start error, got %v", err)
	}
	// No cleanup without a created file.
	for _, c := range fake.calls {
		if c == "delete" {
			t.Error("delete must not run when start failed")
		}
	}
}

func TestExecWaitFailure(t *testing.T) {
	fake := &fakeGuest{pid: 1, waitErr: errors.New("poll failed")}
	d, ctx, vm := execSim(t, fake)
	if _, _, err := d.Exec(ctx, vm.Name, driver.ExecOptions{Script: "true"}); err == nil || err.Error() != "poll failed" {
		t.Errorf("want wait error, got %v", err)
	}
	if fake.deleted == "" {
		t.Error("temp file must be cleaned up after a wait failure")
	}
}

func TestExecReadFailure(t *testing.T) {
	fake := &fakeGuest{pid: 1, code: 0, readErr: errors.New("no transfer")}
	d, ctx, vm := execSim(t, fake)
	out, _, err := d.Exec(ctx, vm.Name, driver.ExecOptions{Script: "true"})
	// The localized wrapper has %w expanded by fmt.Errorf; match on
	// the message prefix without it.
	want := strings.TrimSuffix(i18n.T("err.exec.fetchOutput"), "%w")
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("want wrapped read error, got %v", err)
	}
	if out != "" {
		t.Errorf("output = %q, want empty", out)
	}
}

func TestExecContextCancel(t *testing.T) {
	fake := &fakeGuest{pid: 1, block: true}
	d, ctx, vm := execSim(t, fake)

	sctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	if _, _, err := d.Exec(sctx, vm.Name, driver.ExecOptions{Script: "sleep 999"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("want context deadline error, got %v", err)
	}
}

func TestGuestIPEmpty(t *testing.T) {
	d, ctx := newSim(t)
	vm := firstVM(t, d, ctx)
	_, err := d.GuestIP(ctx, vm.Name, false)
	if err == nil || err.Error() != i18n.T("err.vsphere.noIP") {
		t.Errorf("want noIP error, got %v", err)
	}
}

// setGuestIP injects an address through ExtraConfig, the same trick
// govmomi's simulator tests use (SET.* keys are applied on reconfigure).
func setGuestIP(t *testing.T, d *Driver, ctx context.Context, ref, ip string) {
	t.Helper()
	vm, err := d.resolve(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	spec := types.VirtualMachineConfigSpec{
		ExtraConfig: []types.BaseOptionValue{
			&types.OptionValue{Key: "SET.guest.ipAddress", Value: ip},
		},
	}
	if err := d.reconfigure(ctx, vm, spec); err != nil {
		t.Fatal(err)
	}
}

func TestGuestIPRead(t *testing.T) {
	d, ctx := newSim(t)
	vm := firstVM(t, d, ctx)
	setGuestIP(t, d, ctx, vm.Name, "10.0.0.42")

	ip, err := d.GuestIP(ctx, vm.Name, false)
	if err != nil {
		t.Fatal(err)
	}
	if ip != "10.0.0.42" {
		t.Errorf("IP = %q, want 10.0.0.42", ip)
	}
}

func TestGuestIPWait(t *testing.T) {
	d, ctx := newSim(t)
	vm := firstVM(t, d, ctx)
	setGuestIP(t, d, ctx, vm.Name, "10.0.0.7")

	ip, err := d.GuestIP(ctx, vm.Name, true)
	if err != nil {
		t.Fatal(err)
	}
	if ip != "10.0.0.7" {
		t.Errorf("IP = %q, want 10.0.0.7", ip)
	}
}
