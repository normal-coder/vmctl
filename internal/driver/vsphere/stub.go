package vsphere

import (
	"context"

	"gitee.com/normalcoder/vmctl/internal/driver"
)

// Guest operations land in the next commit; until then they report
// ErrNotSupported so the CLI degrades gracefully.
// TODO(M4): replace with the real guest.go implementation.

func (d *Driver) Exec(ctx context.Context, ref string, opts driver.ExecOptions) (string, int, error) {
	return "", 0, driver.ErrNotSupported
}

func (d *Driver) GuestIP(ctx context.Context, ref string, wait bool) (string, error) {
	return "", driver.ErrNotSupported
}
