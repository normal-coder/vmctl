package vsphere

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/vmware/govmomi/vim25/types"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/i18n"
)

// Start implements driver.Driver.
func (d *Driver) Start(ctx context.Context, ref string, opts driver.StartOptions) error {
	vm, err := d.resolve(ctx, ref)
	if err != nil {
		return err
	}
	t, err := vm.PowerOn(ctx)
	if err != nil {
		return err
	}
	return waitTask(ctx, t)
}

// Stop implements driver.Driver. The VM must be running: --hard cuts
// power directly, soft shutdown asks the guest (via Tools) and waits
// up to softStopTimeout — long enough for a clean guest shutdown,
// bounded so missing Tools cannot hang the command forever.
func (d *Driver) Stop(ctx context.Context, ref string, opts driver.StopOptions) error {
	vm, err := d.resolve(ctx, ref)
	if err != nil {
		return err
	}
	state, err := d.powerStateOf(ctx, vm)
	if err != nil {
		return err
	}
	if state != types.VirtualMachinePowerStatePoweredOn {
		return errors.New(i18n.T("err.vsphere.notRunning"))
	}

	if opts.Hard {
		t, err := vm.PowerOff(ctx)
		if err != nil {
			return err
		}
		return waitTask(ctx, t)
	}

	if err := vm.ShutdownGuest(ctx); err != nil {
		return err
	}
	waitCtx, cancel := context.WithTimeout(ctx, softStopTimeout)
	defer cancel()
	if err := vm.WaitForPowerState(waitCtx, types.VirtualMachinePowerStatePoweredOff); err != nil {
		if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
			return errors.New(i18n.T("err.vsphere.stopTimeout"))
		}
		return err
	}
	return nil
}

// softStopTimeout bounds the wait for a guest-initiated shutdown.
const softStopTimeout = 60 * time.Second

// Suspend implements driver.Driver (the Hard flag is a no-op: vSphere
// suspend is always a suspend-to-disk operation).
func (d *Driver) Suspend(ctx context.Context, ref string, opts driver.SuspendOptions) error {
	vm, err := d.resolve(ctx, ref)
	if err != nil {
		return err
	}
	t, err := vm.Suspend(ctx)
	if err != nil {
		return err
	}
	return waitTask(ctx, t)
}

// Resume implements driver.Driver: powers a suspended VM back on and
// is a no-op for a running one.
func (d *Driver) Resume(ctx context.Context, ref string) error {
	vm, err := d.resolve(ctx, ref)
	if err != nil {
		return err
	}
	state, err := d.powerStateOf(ctx, vm)
	if err != nil {
		return err
	}
	switch state {
	case types.VirtualMachinePowerStatePoweredOff:
		return errors.New(i18n.T("err.vsphere.offForResume"))
	case types.VirtualMachinePowerStatePoweredOn:
		return nil
	default: // suspended
		t, err := vm.PowerOn(ctx)
		if err != nil {
			return err
		}
		return waitTask(ctx, t)
	}
}

// Pause implements driver.Driver. vSphere has no pause concept; the
// sentinel lets the CLI render a friendly explanation.
func (d *Driver) Pause(ctx context.Context, ref string) error {
	return fmt.Errorf("%w: %s", driver.ErrNotSupported, i18n.T("err.vsphere.noPause"))
}

// Reset implements driver.Driver.
func (d *Driver) Reset(ctx context.Context, ref string, opts driver.ResetOptions) error {
	vm, err := d.resolve(ctx, ref)
	if err != nil {
		return err
	}
	t, err := vm.Reset(ctx)
	if err != nil {
		return err
	}
	return waitTask(ctx, t)
}
