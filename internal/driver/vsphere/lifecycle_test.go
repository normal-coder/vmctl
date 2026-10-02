package vsphere

import (
	"errors"
	"testing"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/i18n"
)

func TestCreateRequiresFrom(t *testing.T) {
	d, ctx := newSim(t)
	_, err := d.Create(ctx, driver.CreateOptions{Name: "bare-vm"})
	if !errors.Is(err, driver.ErrNotSupported) {
		t.Errorf("want ErrNotSupported, got %v", err)
	}
}

func TestCloneFull(t *testing.T) {
	d, ctx := newSim(t)
	src := firstVM(t, d, ctx)

	path, err := d.Clone(ctx, src.Name, driver.CloneOptions{
		Name: "full-clone",
		Full: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if path == "" {
		t.Fatal("clone must return the new VM's inventory path")
	}

	info, err := d.Info(ctx, "full-clone")
	if err != nil {
		t.Fatalf("cloned VM not resolvable: %v", err)
	}
	if info.Name != "full-clone" {
		t.Errorf("Name = %q, want full-clone", info.Name)
	}
}

func TestCloneNeedsName(t *testing.T) {
	d, ctx := newSim(t)
	src := firstVM(t, d, ctx)
	if _, err := d.Clone(ctx, src.Name, driver.CloneOptions{}); err == nil {
		t.Error("clone without a name must error")
	}
}

func TestCloneLinkedWithoutSnapshot(t *testing.T) {
	d, ctx := newSim(t)
	src := firstVM(t, d, ctx)
	_, err := d.Clone(ctx, src.Name, driver.CloneOptions{Name: "linked-clone"})
	if err == nil {
		t.Fatal("linked clone without a base snapshot must error")
	}
	if errors.Is(err, driver.ErrNotFound) {
		t.Errorf("want a snapshot error, got %v", err)
	}
}

func TestCloneLinkedWithSnapshot(t *testing.T) {
	d, ctx := newSim(t)
	src := firstVM(t, d, ctx)
	if err := d.SnapshotCreate(ctx, src.Name, "base"); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Clone(ctx, src.Name, driver.CloneOptions{Name: "linked-clone"}); err != nil {
		t.Fatalf("linked clone: %v", err)
	}
	if _, err := d.Info(ctx, "linked-clone"); err != nil {
		t.Errorf("cloned VM not resolvable: %v", err)
	}
}

func TestCreateFromClonesAndResizes(t *testing.T) {
	d, ctx := newSim(t)
	src := firstVM(t, d, ctx)

	path, err := d.Create(ctx, driver.CreateOptions{
		Name:     "resized",
		From:     src.Name,
		Full:     true,
		MemoryMB: 4096,
		CPUs:     4,
	})
	if err != nil {
		t.Fatal(err)
	}
	if path == "" {
		t.Fatal("create must return the new VM's path")
	}

	info, err := d.Info(ctx, "resized")
	if err != nil {
		t.Fatal(err)
	}
	if info.MemoryMB != 4096 {
		t.Errorf("MemoryMB = %d, want 4096", info.MemoryMB)
	}
	if info.CPUs != 4 {
		t.Errorf("CPUs = %d, want 4", info.CPUs)
	}
}

func TestSetPrecondition(t *testing.T) {
	d, ctx := newSim(t)
	vm := firstVM(t, d, ctx)
	if err := d.Start(ctx, vm.Name, driver.StartOptions{}); err != nil {
		t.Fatal(err)
	}

	name := "renamed"
	err := d.Set(ctx, vm.Name, driver.SetOptions{Name: &name})
	if err == nil {
		t.Fatal("Set on a running VM must error")
	}
	if err.Error() != i18n.T("err.vsphere.mustBeOff") {
		t.Errorf("want mustBeOff, got %v", err)
	}
}

func TestSetApplies(t *testing.T) {
	d, ctx := newSim(t)
	vm := firstVM(t, d, ctx)

	name := "renamed"
	mem := 2048
	cpus := 4
	if err := d.Set(ctx, vm.Name, driver.SetOptions{Name: &name, MemoryMB: &mem, CPUs: &cpus}); err != nil {
		t.Fatal(err)
	}

	info, err := d.Info(ctx, "renamed")
	if err != nil {
		t.Fatal(err)
	}
	if info.MemoryMB != 2048 || info.CPUs != 4 {
		t.Errorf("got mem=%d cpus=%d, want 2048/4", info.MemoryMB, info.CPUs)
	}
}

func TestSetNeedsFlags(t *testing.T) {
	d, ctx := newSim(t)
	vm := firstVM(t, d, ctx)
	if err := d.Set(ctx, vm.Name, driver.SetOptions{}); err == nil {
		t.Error("Set with no flags must error")
	}
}

func TestDeletePreconditionAndRemoval(t *testing.T) {
	d, ctx := newSim(t)
	vm := firstVM(t, d, ctx)
	if err := d.Start(ctx, vm.Name, driver.StartOptions{}); err != nil {
		t.Fatal(err)
	}

	if err := d.Delete(ctx, vm.Name); err == nil {
		t.Fatal("Delete of a running VM must error")
	}
	if err := d.Stop(ctx, vm.Name, driver.StopOptions{Hard: true}); err != nil {
		t.Fatal(err)
	}
	if err := d.Delete(ctx, vm.Name); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := d.Info(ctx, vm.Name); !errors.Is(err, driver.ErrNotFound) {
		t.Errorf("deleted VM still resolves: %v", err)
	}
}
