package vsphere

import (
	"testing"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/i18n"
)

func TestSnapshotsEmpty(t *testing.T) {
	d, ctx := newSim(t)
	vm := firstVM(t, d, ctx)

	snaps, err := d.Snapshots(ctx, vm.Name, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) != 0 {
		t.Errorf("want no snapshots, got %v", snaps)
	}
}

func TestSnapshotsTreeAndFlat(t *testing.T) {
	d, ctx := newSim(t)
	vm := firstVM(t, d, ctx)
	for _, name := range []string{"a", "b"} {
		if err := d.SnapshotCreate(ctx, vm.Name, name); err != nil {
			t.Fatal(err)
		}
	}

	tree, err := d.Snapshots(ctx, vm.Name, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(tree) != 2 || tree[0].Name != "a" || tree[0].Depth != 0 || tree[1].Name != "b" || tree[1].Depth != 1 {
		t.Errorf("tree = %+v, want a@0 then b@1", tree)
	}

	flat, err := d.Snapshots(ctx, vm.Name, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range flat {
		if s.Depth != 0 {
			t.Errorf("flat listing depth = %d, want 0 (%s)", s.Depth, s.Name)
		}
	}
}

func TestSnapshotRevertRequiresOff(t *testing.T) {
	d, ctx := newSim(t)
	vm := firstVM(t, d, ctx)
	if err := d.SnapshotCreate(ctx, vm.Name, "s1"); err != nil {
		t.Fatal(err)
	}
	if err := d.Start(ctx, vm.Name, driver.StartOptions{}); err != nil {
		t.Fatal(err)
	}
	err := d.SnapshotRevert(ctx, vm.Name, "s1")
	if err == nil {
		t.Fatal("revert of a running VM must error")
	}
	if err.Error() != i18n.T("err.vsphere.mustBeOff") {
		t.Errorf("want mustBeOff, got %v", err)
	}
}

func TestSnapshotRevert(t *testing.T) {
	d, ctx := newSim(t)
	vm := firstVM(t, d, ctx)
	if err := d.SnapshotCreate(ctx, vm.Name, "s1"); err != nil {
		t.Fatal(err)
	}
	if err := d.SnapshotRevert(ctx, vm.Name, "s1"); err != nil {
		t.Fatalf("revert: %v", err)
	}
}

func TestSnapshotDeleteChildren(t *testing.T) {
	d, ctx := newSim(t)
	vm := firstVM(t, d, ctx)
	for _, name := range []string{"parent", "child"} {
		if err := d.SnapshotCreate(ctx, vm.Name, name); err != nil {
			t.Fatal(err)
		}
	}

	if err := d.SnapshotDelete(ctx, vm.Name, "parent", true); err != nil {
		t.Fatalf("delete with children: %v", err)
	}
	snaps, err := d.Snapshots(ctx, vm.Name, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) != 0 {
		t.Errorf("want no snapshots after deleting parent+children, got %+v", snaps)
	}
}

func TestSnapshotDeleteReparents(t *testing.T) {
	d, ctx := newSim(t)
	vm := firstVM(t, d, ctx)
	for _, name := range []string{"parent", "child"} {
		if err := d.SnapshotCreate(ctx, vm.Name, name); err != nil {
			t.Fatal(err)
		}
	}

	if err := d.SnapshotDelete(ctx, vm.Name, "parent", false); err != nil {
		t.Fatalf("delete without children: %v", err)
	}
	snaps, err := d.Snapshots(ctx, vm.Name, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) != 1 || snaps[0].Name != "child" {
		t.Errorf("want the child re-parented at depth 0, got %+v", snaps)
	}
}

func TestSnapshotDeleteMissing(t *testing.T) {
	d, ctx := newSim(t)
	vm := firstVM(t, d, ctx)
	if err := d.SnapshotDelete(ctx, vm.Name, "no-such-snapshot", false); err == nil {
		t.Error("deleting a missing snapshot must error")
	}
}
