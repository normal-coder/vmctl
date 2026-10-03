package vsphere

import (
	"context"

	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"gitee.com/normalcoder/vmctl/internal/model"
)

// Snapshots implements driver.Driver. With tree=true the snapshot
// hierarchy is reported via Snapshot.Depth (root = 0, pre-order);
// flat listings report every entry at depth 0.
func (d *Driver) Snapshots(ctx context.Context, ref string, tree bool) ([]model.Snapshot, error) {
	vm, err := d.resolve(ctx, ref)
	if err != nil {
		return nil, err
	}
	c, err := d.connect(ctx)
	if err != nil {
		return nil, err
	}
	var m mo.VirtualMachine
	if err := c.RetrieveOne(ctx, vm.Reference(), []string{"snapshot"}, &m); err != nil {
		return nil, err
	}

	out := []model.Snapshot{}
	if m.Snapshot == nil {
		return out, nil
	}
	var walk func(nodes []types.VirtualMachineSnapshotTree, depth int)
	walk = func(nodes []types.VirtualMachineSnapshotTree, depth int) {
		for i := range nodes {
			d := 0
			if tree {
				d = depth
			}
			// The MoRef value ("snapshot-42" in vcsim) — the same key
			// govmomi's FindSnapshot resolves, so it can be pasted
			// straight into delete/revert. Empty when unset.
			uid := nodes[i].Snapshot.Value
			out = append(out, model.Snapshot{Name: nodes[i].Name, Depth: d, UID: uid})
			walk(nodes[i].ChildSnapshotList, depth+1)
		}
	}
	walk(m.Snapshot.RootSnapshotList, 0)
	return out, nil
}

// SnapshotCreate implements driver.Driver: a quiesced-less snapshot
// taken without memory (matching vmrun's defaults).
func (d *Driver) SnapshotCreate(ctx context.Context, ref, name string) error {
	vm, err := d.resolve(ctx, ref)
	if err != nil {
		return err
	}
	t, err := vm.CreateSnapshot(ctx, name, "", false, false)
	if err != nil {
		return err
	}
	return waitTask(ctx, t)
}

// SnapshotDelete implements driver.Driver. Like the vmrun backend, a
// missing snapshot surfaces the backend's own error rather than
// ErrNotFound. Children are re-parented unless deleteChildren is set.
func (d *Driver) SnapshotDelete(ctx context.Context, ref, name string, deleteChildren bool) error {
	vm, err := d.resolve(ctx, ref)
	if err != nil {
		return err
	}
	t, err := vm.RemoveSnapshot(ctx, name, deleteChildren, nil)
	if err != nil {
		return err
	}
	return waitTask(ctx, t)
}

// SnapshotRevert implements driver.Driver. The VM must be powered
// off, and suppressPowerOn keeps vSphere from auto-starting the VM
// afterwards — the same "revert does not boot" semantics as vmrun.
func (d *Driver) SnapshotRevert(ctx context.Context, ref, name string) error {
	vm, err := d.ensurePoweredOff(ctx, ref)
	if err != nil {
		return err
	}
	t, err := vm.RevertToSnapshot(ctx, name, true)
	if err != nil {
		return err
	}
	return waitTask(ctx, t)
}
