package vsphere

import (
	"context"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/model"
)

// The lifecycle, snapshot and guest methods land in the following
// commits; until then they report ErrNotSupported so the CLI degrades
// gracefully instead of failing to compile the interface.
// TODO(M4): replace with real implementations in lifecycle.go,
// snapshot.go and guest.go.

func (d *Driver) Clone(ctx context.Context, ref string, opts driver.CloneOptions) (string, error) {
	return "", driver.ErrNotSupported
}

func (d *Driver) Create(ctx context.Context, opts driver.CreateOptions) (string, error) {
	return "", driver.ErrNotSupported
}

func (d *Driver) Set(ctx context.Context, ref string, opts driver.SetOptions) error {
	return driver.ErrNotSupported
}

func (d *Driver) Delete(ctx context.Context, ref string) error {
	return driver.ErrNotSupported
}

func (d *Driver) Snapshots(ctx context.Context, ref string, tree bool) ([]model.Snapshot, error) {
	return nil, driver.ErrNotSupported
}

func (d *Driver) SnapshotCreate(ctx context.Context, ref, name string) error {
	return driver.ErrNotSupported
}

func (d *Driver) SnapshotDelete(ctx context.Context, ref, name string, deleteChildren bool) error {
	return driver.ErrNotSupported
}

func (d *Driver) SnapshotRevert(ctx context.Context, ref, name string) error {
	return driver.ErrNotSupported
}

func (d *Driver) Exec(ctx context.Context, ref string, opts driver.ExecOptions) (string, int, error) {
	return "", 0, driver.ErrNotSupported
}

func (d *Driver) GuestIP(ctx context.Context, ref string, wait bool) (string, error) {
	return "", driver.ErrNotSupported
}
