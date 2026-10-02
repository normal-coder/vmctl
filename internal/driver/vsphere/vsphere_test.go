package vsphere

import (
	"context"
	"errors"
	"strings"
	"testing"

	sim "github.com/vmware/govmomi/simulator"
	"github.com/vmware/govmomi/vim25/types"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/model"
)

// newSim starts a vcsim server and returns a connected driver plus
// its context. Autostart is off so every test begins from a known
// powered-off state.
func newSim(t *testing.T) (*Driver, context.Context) {
	t.Helper()
	m := sim.VPX()
	m.Autostart = false
	if err := m.Create(); err != nil {
		t.Fatal(err)
	}
	s := m.Service.NewServer()
	t.Cleanup(s.Close)

	d, err := New(driver.Options{
		Endpoint: s.URL.String(),
		User:     "vmctl-test",
		Password: "s3cret",
		Insecure: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return d, context.Background()
}

// firstVM picks any VM from the simulated inventory for tests that
// do not care which one they operate on.
func firstVM(t *testing.T, d *Driver, ctx context.Context) model.VM {
	t.Helper()
	vms, err := d.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(vms) == 0 {
		t.Fatal("simulator has no VMs")
	}
	return vms[0]
}

func TestNewValidation(t *testing.T) {
	if _, err := New(driver.Options{User: "u"}); err == nil {
		t.Error("missing endpoint must error")
	}
	if _, err := New(driver.Options{Endpoint: "https://vc"}); err == nil {
		t.Error("missing user must error")
	}
	if _, err := New(driver.Options{Endpoint: "https://vc", User: "u"}); err != nil {
		t.Errorf("valid options must build offline: %v", err)
	}
}

func TestListAndInfo(t *testing.T) {
	d, ctx := newSim(t)
	vms, err := d.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(vms) == 0 {
		t.Fatal("want VMs, got none")
	}
	for _, vm := range vms {
		if !strings.HasPrefix(vm.ID, "vm-") {
			t.Errorf("ID = %q, want vm-… (ManagedObjectReference)", vm.ID)
		}
		if vm.Name == "" {
			t.Error("empty name")
		}
		if vm.State != model.StateOff {
			t.Errorf("%s state = %q, want off (Autostart disabled)", vm.Name, vm.State)
		}
	}

	vm := vms[0]
	info, err := d.Info(ctx, vm.Name)
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != vm.Name {
		t.Errorf("Info name = %q, want %q", info.Name, vm.Name)
	}
	if info.State != model.StateOff {
		t.Errorf("Info state = %q, want off", info.State)
	}
	if info.GuestIP != "" {
		t.Errorf("powered-off VM must not report GuestIP, got %q", info.GuestIP)
	}
}

func TestResolveByEveryStrategy(t *testing.T) {
	d, ctx := newSim(t)
	vm := firstVM(t, d, ctx)

	// UUID prefix from the inventory record.
	var uuid string
	inv, err := d.listInventory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, iv := range inv {
		if iv.Name == vm.Name {
			uuid = iv.UUID
		}
	}
	if len(uuid) < 8 {
		t.Fatalf("uuid too short: %q", uuid)
	}
	// Strip dashes for a bare-hex prefix query.
	bare := strings.ReplaceAll(uuid, "-", "")

	cases := []struct {
		name string
		ref  string
	}{
		{"exact name", vm.Name},
		{"case-insensitive name", strings.ToUpper(vm.Name)},
		{"uuid prefix", bare[:8]},
		{"dashed uuid prefix", uuid[:8]},
		{"substring", vm.Name[1:]}, // drop the first char: still unique among DC0_H0_VM*
		{"moref", vm.ID},
		{"typed moref", "VirtualMachine:" + vm.ID},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := d.resolve(ctx, c.ref)
			if err != nil {
				t.Fatalf("resolve(%q): %v", c.ref, err)
			}
			if got.Reference() == (types.ManagedObjectReference{}) {
				t.Errorf("resolve(%q): empty ref", c.ref)
			}
		})
	}

	t.Run("inventory path", func(t *testing.T) {
		if _, err := d.resolve(ctx, "DC0/vm/"+vm.Name); err != nil {
			t.Errorf("inventory path resolve: %v", err)
		}
	})
	t.Run("not found", func(t *testing.T) {
		_, err := d.resolve(ctx, "no-such-vm-anywhere")
		if !errors.Is(err, driver.ErrNotFound) {
			t.Errorf("want ErrNotFound, got %v", err)
		}
	})
	t.Run("empty ref", func(t *testing.T) {
		_, err := d.resolve(ctx, "  ")
		if !errors.Is(err, driver.ErrNotFound) {
			t.Errorf("want ErrNotFound, got %v", err)
		}
	})
}

// TestMatchVM exercises the pure matching logic, including the
// ambiguous cases that are awkward to set up in vcsim.
func TestMatchVM(t *testing.T) {
	vms := []invVM{
		{Ref: types.ManagedObjectReference{Type: "VirtualMachine", Value: "vm-1"}, Name: "alpha", UUID: "42100000-1111-2222-3333-444455556666"},
		{Ref: types.ManagedObjectReference{Type: "VirtualMachine", Value: "vm-2"}, Name: "beta", UUID: "4210aaaa-bbbb-cccc-dddd-eeeeffff0000"},
		{Ref: types.ManagedObjectReference{Type: "VirtualMachine", Value: "vm-3"}, Name: "gamma", UUID: "4210aaaa-bbbb-cccc-dddd-eeeeffff1111"},
		{Ref: types.ManagedObjectReference{Type: "VirtualMachine", Value: "vm-4"}, Name: "alpha-2", UUID: ""},
	}

	cases := []struct {
		ref     string
		wantVM  string // "" = expect miss
		wantErr bool   // ambiguous reference
	}{
		{ref: "beta", wantVM: "beta"},
		{ref: "BETA", wantVM: "beta"},
		{ref: "42100000", wantVM: "alpha"}, // unique uuid prefix
		{ref: "4210aaaa", wantErr: true},   // shared uuid prefix (vm-2/vm-3)
		{ref: "gam", wantVM: "gamma"},      // unique substring
		{ref: "alpha", wantVM: "alpha"},    // exact wins over substring ambiguity
		{ref: "nomatch", wantVM: ""},       // miss: caller falls through
		{ref: "a", wantErr: true},          // matches alpha + alpha-2
	}
	for _, c := range cases {
		t.Run(c.ref, func(t *testing.T) {
			got, err := matchVM(c.ref, vms)
			if c.wantErr {
				if err == nil {
					t.Fatal("want ambiguous error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c.wantVM == "" {
				if got != nil {
					t.Fatalf("want miss, got %+v", got)
				}
				return
			}
			if got == nil || got.Name != c.wantVM {
				t.Errorf("got %+v, want %q", got, c.wantVM)
			}
		})
	}
}

func TestPowerChain(t *testing.T) {
	d, ctx := newSim(t)
	vm := firstVM(t, d, ctx)
	name := vm.Name

	state := func() model.State {
		t.Helper()
		info, err := d.Info(ctx, name)
		if err != nil {
			t.Fatal(err)
		}
		return info.State
	}
	assertState := func(want model.State) {
		t.Helper()
		if got := state(); got != want {
			t.Fatalf("state = %q, want %q", got, want)
		}
	}

	assertState(model.StateOff)

	if err := d.Start(ctx, name, driver.StartOptions{}); err != nil {
		t.Fatalf("start: %v", err)
	}
	assertState(model.StateOn)

	if err := d.Suspend(ctx, name, driver.SuspendOptions{}); err != nil {
		t.Fatalf("suspend: %v", err)
	}
	assertState(model.StateSuspended)

	if err := d.Resume(ctx, name); err != nil {
		t.Fatalf("resume: %v", err)
	}
	assertState(model.StateOn)

	// Reset while running: back to on.
	if err := d.Reset(ctx, name, driver.ResetOptions{}); err != nil {
		t.Fatalf("reset: %v", err)
	}
	assertState(model.StateOn)

	if err := d.Stop(ctx, name, driver.StopOptions{Hard: true}); err != nil {
		t.Fatalf("stop hard: %v", err)
	}
	assertState(model.StateOff)
}

func TestStopRequiresRunning(t *testing.T) {
	d, ctx := newSim(t)
	vm := firstVM(t, d, ctx)

	err := d.Stop(ctx, vm.Name, driver.StopOptions{Hard: true})
	if err == nil {
		t.Fatal("stopping a powered-off VM must error")
	}
	if !strings.Contains(err.Error(), "未处于运行") {
		t.Errorf("want notRunning message, got: %v", err)
	}
}

func TestResumeWhenOff(t *testing.T) {
	d, ctx := newSim(t)
	vm := firstVM(t, d, ctx)

	err := d.Resume(ctx, vm.Name)
	if err == nil {
		t.Fatal("resuming a powered-off VM must error")
	}
	if !strings.Contains(err.Error(), "start") {
		t.Errorf("error should point at start: %v", err)
	}
}

func TestPauseNotSupported(t *testing.T) {
	d, ctx := newSim(t)
	err := d.Pause(ctx, "anything")
	if !errors.Is(err, driver.ErrNotSupported) {
		t.Errorf("want ErrNotSupported, got %v", err)
	}
}

func TestConnectFailureRedactsPassword(t *testing.T) {
	d, err := New(driver.Options{
		Endpoint: "http://127.0.0.1:1", // nothing listens here
		User:     "u",
		Password: "top-secret-value",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.List(context.Background())
	if err == nil {
		t.Fatal("unreachable endpoint must error")
	}
	if strings.Contains(err.Error(), "top-secret-value") {
		t.Errorf("password leaked in error: %v", err)
	}
}

func TestStopSoft(t *testing.T) {
	d, ctx := newSim(t)
	vm := firstVM(t, d, ctx)

	if err := d.Start(ctx, vm.Name, driver.StartOptions{}); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := d.Stop(ctx, vm.Name, driver.StopOptions{}); err != nil {
		t.Fatalf("soft stop: %v", err)
	}
	info, err := d.Info(ctx, vm.Name)
	if err != nil {
		t.Fatal(err)
	}
	if info.State != model.StateOff {
		t.Errorf("state = %q, want off", info.State)
	}
}
