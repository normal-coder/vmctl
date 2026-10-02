package vsphere

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/i18n"
)

// ensurePoweredOff is the shared precondition for destructive
// operations (set, delete, revert). vcsim does not enforce power
// state, so the check here keeps simulated and real behavior aligned.
func (d *Driver) ensurePoweredOff(ctx context.Context, ref string) (*object.VirtualMachine, error) {
	vm, err := d.resolve(ctx, ref)
	if err != nil {
		return nil, err
	}
	state, err := d.powerStateOf(ctx, vm)
	if err != nil {
		return nil, err
	}
	if state != types.VirtualMachinePowerStatePoweredOff {
		return nil, errors.New(i18n.T("err.vsphere.mustBeOff"))
	}
	return vm, nil
}

// Clone implements driver.Driver, returning the new VM's inventory
// path — the vSphere counterpart of the vmrun backend's vmx path
// (the interface contract says "path"; each backend reports the path
// its users actually navigate by).
//
// Unlike vmrun, vSphere clones do not require the source to be
// powered off: full copies read disk backing files as-is, and linked
// clones are anchored on a snapshot.
func (d *Driver) Clone(ctx context.Context, ref string, opts driver.CloneOptions) (string, error) {
	src, err := d.resolve(ctx, ref)
	if err != nil {
		return "", err
	}
	if opts.Name == "" {
		return "", errors.New(i18n.T("err.clone.name"))
	}
	_, path, err := d.cloneVM(ctx, src, opts)
	return path, err
}

// cloneVM is the shared Clone/Create implementation. The source must
// already be resolved; opts.Name is required. It returns the new VM
// together with its inventory path.
func (d *Driver) cloneVM(ctx context.Context, src *object.VirtualMachine, opts driver.CloneOptions) (*object.VirtualMachine, string, error) {
	c, err := d.connect(ctx)
	if err != nil {
		return nil, "", err
	}

	// Destination folder: opts.Path (inventory path) or the source's parent.
	var srcMO mo.ManagedEntity
	if err := src.Properties(ctx, src.Reference(), []string{"parent", "name"}, &srcMO); err != nil {
		return nil, "", err
	}
	folder := object.NewFolder(c.Client, *srcMO.Parent)
	if opts.Path != "" {
		found, err := object.NewSearchIndex(c.Client).FindByInventoryPath(ctx, opts.Path)
		if err != nil || found == nil {
			return nil, "", fmt.Errorf(i18n.T("err.vsphere.cloneFolder"), opts.Path)
		}
		f, ok := found.(*object.Folder)
		if !ok {
			return nil, "", fmt.Errorf(i18n.T("err.vsphere.cloneNotFolder"), opts.Path)
		}
		folder = f
	}

	spec := types.VirtualMachineCloneSpec{}
	if opts.Full {
		spec.Location.DiskMoveType = string(types.VirtualMachineRelocateDiskMoveOptionsMoveAllDiskBackingsAndAllowSharing)
	} else {
		// Linked clone: anchor on a snapshot (explicit or current)
		// so the child disks are deltas off the base.
		snapRef, err := d.baseSnapshot(ctx, src, opts)
		if err != nil {
			return nil, "", err
		}
		spec.Snapshot = snapRef
		spec.Location.DiskMoveType = string(types.VirtualMachineRelocateDiskMoveOptionsCreateNewChildDiskBacking)
	}

	t, err := src.Clone(ctx, folder, opts.Name, spec)
	if err != nil {
		return nil, "", err
	}
	info, err := t.WaitForResult(ctx)
	if err != nil {
		return nil, "", err
	}
	moRef, ok := info.Result.(types.ManagedObjectReference)
	if !ok {
		return nil, "", errors.New(i18n.T("err.vsphere.cloneResult"))
	}
	newVM := object.NewVirtualMachine(c.Client, moRef)
	path, err := d.inventoryPath(ctx, newVM)
	if err != nil {
		return nil, "", err
	}
	return newVM, path, nil
}

// baseSnapshot picks the snapshot a linked clone builds on: the
// requested one, or the current snapshot when none was given.
func (d *Driver) baseSnapshot(ctx context.Context, src *object.VirtualMachine, opts driver.CloneOptions) (*types.ManagedObjectReference, error) {
	if opts.Snapshot != "" {
		return src.FindSnapshot(ctx, opts.Snapshot)
	}
	c, err := d.connect(ctx)
	if err != nil {
		return nil, err
	}
	var m mo.VirtualMachine
	if err := c.RetrieveOne(ctx, src.Reference(), []string{"snapshot", "name"}, &m); err != nil {
		return nil, err
	}
	if m.Snapshot == nil || m.Snapshot.CurrentSnapshot == nil {
		return nil, fmt.Errorf(i18n.T("err.vsphere.linkedNoSnapshot"), m.Name)
	}
	return m.Snapshot.CurrentSnapshot, nil
}

// inventoryPath walks the parent chain up to the root folder and
// joins the names with "/", e.g. "DC0/vm/my-vm" — the same syntax
// FindByInventoryPath accepts.
func (d *Driver) inventoryPath(ctx context.Context, vm *object.VirtualMachine) (string, error) {
	c, err := d.connect(ctx)
	if err != nil {
		return "", err
	}
	var parts []string
	ref := vm.Reference()
	for {
		if ref == c.ServiceContent.RootFolder {
			break
		}
		var e mo.ManagedEntity
		if err := c.RetrieveOne(ctx, ref, []string{"name", "parent"}, &e); err != nil {
			return "", err
		}
		parts = append([]string{e.Name}, parts...)
		if e.Parent == nil {
			break
		}
		ref = *e.Parent
	}
	return strings.Join(parts, "/"), nil
}

// Create implements driver.Driver. vSphere cannot provision a bare
// VM from nothing (that is a job for templates or content libraries),
// so a missing --from reports ErrNotSupported; with --from it clones
// and then optionally resizes memory and CPUs.
func (d *Driver) Create(ctx context.Context, opts driver.CreateOptions) (string, error) {
	if opts.From == "" {
		return "", fmt.Errorf("%w: %s", driver.ErrNotSupported, i18n.T("err.vsphere.createNoFrom"))
	}
	src, err := d.resolve(ctx, opts.From)
	if err != nil {
		return "", err
	}
	if opts.Name == "" {
		return "", errors.New(i18n.T("err.clone.name"))
	}
	newVM, path, err := d.cloneVM(ctx, src, driver.CloneOptions{
		Name:     opts.Name,
		Full:     opts.Full,
		Snapshot: opts.Snapshot,
		Path:     opts.Path,
	})
	if err != nil {
		return "", err
	}

	if opts.MemoryMB > 0 || opts.CPUs > 0 {
		spec := types.VirtualMachineConfigSpec{}
		if opts.MemoryMB > 0 {
			spec.MemoryMB = int64(opts.MemoryMB)
		}
		if opts.CPUs > 0 {
			spec.NumCPUs = int32(opts.CPUs)
		}
		if err := d.reconfigure(ctx, newVM, spec); err != nil {
			return "", err
		}
	}
	return path, nil
}

// Set implements driver.Driver (powered-off VMs only, like vmrun).
func (d *Driver) Set(ctx context.Context, ref string, opts driver.SetOptions) error {
	if opts.Empty() {
		return errors.New(i18n.T("err.set.flags"))
	}
	vm, err := d.ensurePoweredOff(ctx, ref)
	if err != nil {
		return err
	}
	spec := types.VirtualMachineConfigSpec{}
	if opts.Name != nil {
		spec.Name = *opts.Name
	}
	if opts.MemoryMB != nil {
		spec.MemoryMB = int64(*opts.MemoryMB)
	}
	if opts.CPUs != nil {
		spec.NumCPUs = int32(*opts.CPUs)
	}
	return d.reconfigure(ctx, vm, spec)
}

func (d *Driver) reconfigure(ctx context.Context, vm *object.VirtualMachine, spec types.VirtualMachineConfigSpec) error {
	t, err := vm.Reconfigure(ctx, spec)
	if err != nil {
		return err
	}
	return waitTask(ctx, t)
}

// Delete implements driver.Driver: the VM must be powered off, then
// it is destroyed (removed from inventory and disk).
func (d *Driver) Delete(ctx context.Context, ref string) error {
	vm, err := d.ensurePoweredOff(ctx, ref)
	if err != nil {
		return err
	}
	t, err := vm.Destroy(ctx)
	if err != nil {
		return err
	}
	return waitTask(ctx, t)
}
