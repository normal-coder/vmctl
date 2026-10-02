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

// WrapNotFound attaches the ErrNotFound sentinel to an already
// localized detail sentence, keeping errors.Is matching intact.
func WrapNotFound(detail string) error {
	return fmt.Errorf("%w: %s", ErrNotFound, detail)
}

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

// CloneOptions controls VM cloning.
type CloneOptions struct {
	// Name is the new VM's display name (required).
	Name string
	// Full makes an independent copy. Default (with Full=false) is a
	// linked clone, which is fast but keeps a base snapshot on the source.
	Full bool
	// Snapshot selects the base snapshot (linked/full clones).
	Snapshot string
	// Path overrides the destination: a .vmx file or a bundle directory.
	// Default: alongside the source, as <name>.vmwarevm/<name>.vmx.
	Path string
}

// CreateOptions provisions a new VM.
type CreateOptions struct {
	// Name is the new VM's display name (required).
	Name string
	// From is the source VM ref to clone from. Required by backends
	// that cannot create a bare VM (vmrun).
	From     string
	Full     bool
	Snapshot string
	Path     string
	// MemoryMB and CPUs optionally override the configuration after
	// cloning (0 = keep source values).
	MemoryMB int
	CPUs     int
}

// SetOptions describes configuration changes; nil fields are left
// untouched.
type SetOptions struct {
	Name     *string
	MemoryMB *int
	CPUs     *int
}

// Empty reports whether no change was requested.
func (o SetOptions) Empty() bool {
	return o.Name == nil && o.MemoryMB == nil && o.CPUs == nil
}

// ExecOptions describes a command to run inside the guest.
type ExecOptions struct {
	// Script is the shell command text, executed via /bin/sh -c in
	// the guest. Support for quoting, pipes etc. follows /bin/sh.
	Script string
	// User and Password authenticate against the guest OS
	// (vmrun's -gu/-gp). Empty values let the backend decide.
	User     string
	Password string
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

	// Lifecycle operations.
	// Clone copies the VM matching ref and returns the new vmx path.
	Clone(ctx context.Context, ref string, opts CloneOptions) (string, error)
	// Create provisions a new VM and returns the new vmx path.
	Create(ctx context.Context, opts CreateOptions) (string, error)
	// Set applies configuration changes (powered-off VMs only).
	Set(ctx context.Context, ref string, opts SetOptions) error
	// Delete permanently removes the VM and its files.
	Delete(ctx context.Context, ref string) error

	// Snapshot operations.
	// Snapshots lists the snapshots of the VM matching ref. With
	// tree=true the backend reports hierarchy via Snapshot.Depth.
	Snapshots(ctx context.Context, ref string, tree bool) ([]model.Snapshot, error)
	// SnapshotCreate creates a snapshot with the given name.
	SnapshotCreate(ctx context.Context, ref, name string) error
	// SnapshotDelete removes one snapshot; deleteChildren also
	// removes its descendants.
	SnapshotDelete(ctx context.Context, ref, name string, deleteChildren bool) error
	// SnapshotRevert resets the VM to a snapshot.
	SnapshotRevert(ctx context.Context, ref, name string) error

	// Guest operations.
	// Exec runs Script inside the guest and returns the captured
	// combined output and the guest program's exit code. A non-nil
	// error means the execution layer itself failed (auth, Tools,
	// unreachable guest) — distinct from a non-zero guest exit code.
	Exec(ctx context.Context, ref string, opts ExecOptions) (output string, exitCode int, err error)
	// GuestIP returns the guest's IP address. wait=true blocks until
	// the guest reports one (backend-dependent).
	GuestIP(ctx context.Context, ref string, wait bool) (string, error)
}

// Options carries backend construction settings. vmrun reads only
// VMRunPath; vsphere reads Endpoint/User/Password/Insecure. Fields a
// backend does not use are ignored.
type Options struct {
	// VMRunPath overrides auto-detection of the vmrun binary ("" = detect).
	VMRunPath string
	// VMCliPath overrides auto-detection of the vmcli binary ("" =
	// detect); used by the vmrun backend for snapshot uid references.
	VMCliPath string
	// Endpoint is the vCenter/ESXi URL, e.g. https://vcenter.example.com.
	Endpoint string
	// User and Password authenticate against the vSphere API.
	User     string
	Password string
	// Insecure skips TLS certificate verification.
	Insecure bool
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
