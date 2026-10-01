// Package driver defines the unified backend interface for VMCTL.
//
// Every backend (local vmrun, remote vsphere, ...) implements Driver so
// the CLI layer stays backend-agnostic. Ref strings are resolved by the
// driver itself: name, UUID or path are all acceptable where the
// backend supports them.
package driver

import (
	"context"
	"errors"
	"fmt"

	"gitee.com/normalcoder/vmctl/internal/model"
)

// Sentinel errors. Drivers wrap them with %w so callers can match with errors.Is.
var (
	ErrNotFound     = errors.New("virtual machine not found")
	ErrNotSupported = errors.New("operation not supported by this backend")
)

// StartOptions controls how a VM is powered on.
type StartOptions struct {
	// NoGUI starts the VM without opening a window (headless).
	NoGUI bool
}

// StopOptions controls how a VM is powered off.
type StopOptions struct {
	// Hard bypasses guest shutdown and cuts power directly.
	Hard bool
}

// SuspendOptions controls how a VM is suspended to disk.
type SuspendOptions struct {
	Hard bool
}

// ResetOptions controls how a VM is reset.
type ResetOptions struct {
	Hard bool
}

// Driver is the unified backend interface.
type Driver interface {
	// Name returns the backend identifier ("vmrun", "vsphere", ...).
	Name() string

	// List returns all virtual machines known to the backend.
	List(ctx context.Context) ([]model.VM, error)

	// Info returns detailed information for the VM matching ref.
	Info(ctx context.Context, ref string) (*model.VMInfo, error)

	// Power operations. ref accepts a name, UUID or path.
	Start(ctx context.Context, ref string, opts StartOptions) error
	Stop(ctx context.Context, ref string, opts StopOptions) error
	Suspend(ctx context.Context, ref string, opts SuspendOptions) error
	// Resume continues a suspended VM or unpauses a paused one.
	// Resuming a running VM is a no-op.
	Resume(ctx context.Context, ref string) error
	Pause(ctx context.Context, ref string) error
	Reset(ctx context.Context, ref string, opts ResetOptions) error
}

// Options carries backend construction settings.
type Options struct {
	// VMRunPath overrides auto-detection of the vmrun binary ("" = detect).
	VMRunPath string
}

// Factory builds a Driver from options.
type Factory func(opts Options) (Driver, error)

var registry = map[string]Factory{}

// Register makes a backend available under name. Intended to be called
// from a backend package's init.
func Register(name string, f Factory) {
	registry[name] = f
}

// Open constructs the named backend.
func Open(name string, opts Options) (Driver, error) {
	f, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown backend %q", name)
	}
	return f(opts)
}

// Available lists registered backend names.
func Available() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	return names
}
