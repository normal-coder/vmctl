package vsphere

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/i18n"
)

// invVM is the minimal inventory record resolve matches against.
type invVM struct {
	Ref  types.ManagedObjectReference
	Name string
	UUID string
}

// resolve maps a user-supplied reference to a VM object. Accepts, in
// order: a ManagedObjectReference (vm-42 / VirtualMachine:vm-42), an
// exact name (case-insensitive), a config.uuid prefix (>= 8 hex
// chars), a unique substring of a name, and an inventory path.
func (d *Driver) resolve(ctx context.Context, ref string) (*object.VirtualMachine, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, fmt.Errorf("%w: %s", driver.ErrNotFound, i18n.T("err.resolve.empty"))
	}

	c, err := d.connect(ctx)
	if err != nil {
		return nil, err
	}

	// 1. ManagedObjectReference. Only "vm-…" values are tried so
	// plain names fall through to matching below.
	if moRef, ok := parseMoRef(ref); ok {
		vm := object.NewVirtualMachine(c.Client, moRef)
		var check mo.VirtualMachine
		if err := c.RetrieveOne(ctx, moRef, []string{"name"}, &check); err == nil {
			return vm, nil
		}
		// Stale/foreign MoRef: keep matching by other criteria.
	}

	// 2-4. name / UUID prefix / substring against the inventory.
	vms, err := d.listInventory(ctx)
	if err != nil {
		return nil, err
	}
	if m, err := matchVM(ref, vms); err != nil {
		return nil, err
	} else if m != nil {
		return object.NewVirtualMachine(c.Client, m.Ref), nil
	}

	// 5. inventory path, e.g. DC0/vm/my-vm.
	if found, err := object.NewSearchIndex(c.Client).FindByInventoryPath(ctx, ref); err == nil && found != nil {
		if vm, ok := found.(*object.VirtualMachine); ok {
			return vm, nil
		}
	}

	names := make([]string, 0, len(vms))
	for _, v := range vms {
		if v.Name != "" {
			names = append(names, v.Name)
		}
	}
	return nil, fmt.Errorf("%w: %q (known: %s)", driver.ErrNotFound, ref, joinNames(names))
}

// listInventory fetches name and UUID for every VM in the inventory.
func (d *Driver) listInventory(ctx context.Context) ([]invVM, error) {
	c, err := d.connect(ctx)
	if err != nil {
		return nil, err
	}
	// The view must be created server-side; view.NewContainerView is
	// only a client-side constructor for an existing view.
	cv, err := view.NewManager(c.Client).CreateContainerView(
		ctx, c.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cv.Destroy(ctx) }()

	var mos []mo.VirtualMachine
	if err := cv.Retrieve(ctx, []string{"VirtualMachine"},
		[]string{"name", "config.uuid"}, &mos); err != nil {
		return nil, err
	}
	vms := make([]invVM, 0, len(mos))
	for _, m := range mos {
		uuid := ""
		if m.Config != nil {
			uuid = m.Config.Uuid
		}
		vms = append(vms, invVM{Ref: m.Self, Name: m.Name, UUID: uuid})
	}
	return vms, nil
}

// matchVM picks the VM for ref from inventory records: exact name
// first, then a UUID prefix, then a unique name substring. It returns
// (nil, nil) when nothing matches and the caller should try other
// strategies (inventory path) before failing.
func matchVM(ref string, vms []invVM) (*invVM, error) {
	// exact name (case-insensitive)
	var hits []invVM
	for _, v := range vms {
		if strings.EqualFold(v.Name, ref) {
			hits = append(hits, v)
		}
	}
	switch len(hits) {
	case 1:
		return &hits[0], nil
	case 0:
		// keep matching
	default:
		return nil, fmt.Errorf(i18n.T("err.resolve.ambiguous"), ref, len(hits))
	}

	// UUID prefix (bare hex, dashed or braced, >= 8 hex chars)
	if refHex := compactHex(ref); len(refHex) >= 8 {
		var uuidHits []invVM
		for _, v := range vms {
			if strings.HasPrefix(compactHex(v.UUID), refHex) {
				uuidHits = append(uuidHits, v)
			}
		}
		if len(uuidHits) == 1 {
			return &uuidHits[0], nil
		}
		if len(uuidHits) > 1 {
			return nil, fmt.Errorf(i18n.T("err.resolve.ambiguous"), ref, len(uuidHits))
		}
	}

	// unique substring of a name
	var partial []invVM
	lower := strings.ToLower(ref)
	for _, v := range vms {
		if strings.Contains(strings.ToLower(v.Name), lower) {
			partial = append(partial, v)
		}
	}
	switch len(partial) {
	case 1:
		return &partial[0], nil
	case 0:
		return nil, nil // caller tries remaining strategies
	default:
		names := make([]string, 0, len(partial))
		for _, v := range partial {
			names = append(names, v.Name)
		}
		return nil, fmt.Errorf(i18n.T("err.resolve.ambiguousNames"), ref, joinNames(names))
	}
}

// parseMoRef accepts "vm-42" and "VirtualMachine:vm-42".
func parseMoRef(ref string) (types.ManagedObjectReference, bool) {
	if typ, val, ok := strings.Cut(ref, ":"); ok {
		if typ == "VirtualMachine" && strings.HasPrefix(val, "vm-") {
			return types.ManagedObjectReference{Type: typ, Value: val}, true
		}
		return types.ManagedObjectReference{}, false
	}
	if strings.HasPrefix(ref, "vm-") {
		return types.ManagedObjectReference{Type: "VirtualMachine", Value: ref}, true
	}
	return types.ManagedObjectReference{}, false
}

// compactHex strips separators from s and lowercases it. Returns "" if
// any character is not a hex digit (so plain names never hit the UUID
// matching path). Mirrors the vmrun driver's helper.
func compactHex(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f',
			r == '-', r == ' ', r == '{', r == '}':
			if r != '-' && r != ' ' && r != '{' && r != '}' {
				b.WriteRune(r)
			}
		default:
			return ""
		}
	}
	return b.String()
}

// joinNames renders a sorted, comma-joined list for error messages.
func joinNames(names []string) string {
	if len(names) == 0 {
		return "none"
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
