package vsphere

import (
	"context"
	"errors"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"gitee.com/normalcoder/vmctl/internal/model"
)

// listProps are the properties both List and Info start from.
var listProps = []string{
	"name",
	"runtime.powerState",
	"config.uuid",
	"config.hardware.numCPU",
	"config.hardware.memoryMB",
	"guest.guestFullName",
	"config.files.vmPathName",
	"config.template",
}

// List implements driver.Driver: every VM in the inventory, visible
// templates included. ID is the ManagedObjectReference value (vm-42).
func (d *Driver) List(ctx context.Context) ([]model.VM, error) {
	c, err := d.connect(ctx)
	if err != nil {
		return nil, err
	}
	cv, err := view.NewManager(c.Client).CreateContainerView(
		ctx, c.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cv.Destroy(ctx) }()

	var mos []mo.VirtualMachine
	if err := cv.Retrieve(ctx, []string{"VirtualMachine"}, listProps, &mos); err != nil {
		return nil, err
	}

	vms := make([]model.VM, 0, len(mos))
	for _, m := range mos {
		vms = append(vms, vmFromMO(m))
	}
	return vms, nil
}

// Info implements driver.Driver with per-VM guest details on top of
// the List fields.
func (d *Driver) Info(ctx context.Context, ref string) (*model.VMInfo, error) {
	vm, err := d.resolve(ctx, ref)
	if err != nil {
		return nil, err
	}
	c, err := d.connect(ctx)
	if err != nil {
		return nil, err
	}

	var m mo.VirtualMachine
	props := append(append([]string{}, listProps...),
		"guest.toolsRunningStatus", "guest.ipAddress")
	if err := c.RetrieveOne(ctx, vm.Reference(), props, &m); err != nil {
		return nil, err
	}

	info := &model.VMInfo{VM: vmFromMO(m)}
	if m.Guest != nil {
		info.ToolsState = m.Guest.ToolsRunningStatus
		// Guest IP is only meaningful while the VM is running.
		if info.State == model.StateOn {
			info.GuestIP = m.Guest.IpAddress
		}
	}
	return info, nil
}

// vmFromMO projects a property-retrieved MO onto the model type.
func vmFromMO(m mo.VirtualMachine) model.VM {
	out := model.VM{
		ID:    m.Self.Value, // e.g. vm-42 (see model.VM.ID)
		Name:  m.Name,
		State: powerState(m.Runtime.PowerState),
	}
	if m.Config != nil {
		out.Path = m.Config.Files.VmPathName
		if m.Config.Template {
			out.State = model.StateOff // templates are never powered on
		}
		out.CPUs = int(m.Config.Hardware.NumCPU)
		out.MemoryMB = int(m.Config.Hardware.MemoryMB)
	}
	if m.Guest != nil {
		out.GuestOS = m.Guest.GuestFullName
	}
	return out
}

// powerStateOf reads the current power state of one VM.
func (d *Driver) powerStateOf(ctx context.Context, vm *object.VirtualMachine) (types.VirtualMachinePowerState, error) {
	c, err := d.connect(ctx)
	if err != nil {
		return "", err
	}
	var m mo.VirtualMachine
	if err := c.RetrieveOne(ctx, vm.Reference(), []string{"runtime.powerState"}, &m); err != nil {
		return "", err
	}
	if m.Runtime.PowerState == "" {
		return "", errors.New("runtime properties unavailable")
	}
	return m.Runtime.PowerState, nil
}
