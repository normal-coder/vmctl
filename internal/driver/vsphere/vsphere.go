// Package vsphere implements driver.Driver on top of govmomi,
// talking to vCenter or ESXi over the SOAP API.
package vsphere

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/i18n"
	"gitee.com/normalcoder/vmctl/internal/model"
)

func init() {
	driver.Register("vsphere", func(opts driver.Options) (driver.Driver, error) {
		return New(opts)
	})
}

// Driver controls VMs on a vCenter/ESXi endpoint.
type Driver struct {
	endpoint string
	user     string
	password string
	insecure bool

	// Lazy session: established on first use so New stays offline
	// (profile validation must not require a live endpoint).
	once    sync.Once
	session *govmomi.Client
	connErr error

	// guest overrides the guest-operations implementation (tests
	// inject a fake; vcsim only runs guest programs in container VMs).
	guest guestOps
}

// New builds a vsphere driver. Only the options are validated here;
// the connection is established lazily on the first API call.
func New(opts driver.Options) (*Driver, error) {
	if opts.Endpoint == "" {
		return nil, errors.New(i18n.T("err.vsphere.endpoint"))
	}
	if opts.User == "" {
		return nil, errors.New(i18n.T("err.vsphere.user"))
	}
	return &Driver{
		endpoint: opts.Endpoint,
		user:     opts.User,
		password: opts.Password,
		insecure: opts.Insecure,
	}, nil
}

var _ driver.Driver = (*Driver)(nil)

// Name implements driver.Driver.
func (d *Driver) Name() string { return "vsphere" }

// connect establishes the session on first use. A failed attempt is
// sticky — vmctl is a short-lived CLI, so reconnecting is pointless.
func (d *Driver) connect(ctx context.Context) (*govmomi.Client, error) {
	d.once.Do(func() {
		u, err := url.Parse(d.endpoint)
		if err != nil {
			d.connErr = fmt.Errorf(i18n.T("err.vsphere.connect"), d.endpoint, err)
			return
		}
		if u.Path == "" || u.Path == "/" {
			u.Path = "/sdk" // standard vimService endpoint
		}
		u.User = url.UserPassword(d.user, d.password)
		c, err := govmomi.NewClient(ctx, u, d.insecure)
		if err != nil {
			// Never leak the password: both the endpoint we echo and
			// the underlying error may carry the full URL with
			// userinfo.
			d.connErr = fmt.Errorf(i18n.T("err.vsphere.connect"),
				redact(u.Redacted(), d.password), redact(err.Error(), d.password))
			return
		}
		d.session = c
	})
	return d.session, d.connErr
}

func redact(s, secret string) string {
	if secret == "" {
		return s
	}
	return strings.ReplaceAll(s, secret, "***")
}

// waitTask blocks until a vSphere task completes. Every mutation must
// pass through here: returning before the task finishes would report
// success for an operation still in flight.
func waitTask(ctx context.Context, t *object.Task) error {
	if t == nil {
		return errors.New(i18n.T("err.vsphere.nilTask"))
	}
	return t.Wait(ctx)
}

// powerState maps the vSphere enum onto the backend-agnostic model.
func powerState(s types.VirtualMachinePowerState) model.State {
	switch s {
	case types.VirtualMachinePowerStatePoweredOn:
		return model.StateOn
	case types.VirtualMachinePowerStatePoweredOff:
		return model.StateOff
	case types.VirtualMachinePowerStateSuspended:
		return model.StateSuspended
	default:
		return model.StateUnknown
	}
}
