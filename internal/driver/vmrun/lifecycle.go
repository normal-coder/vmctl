package vmrun

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/i18n"
)

// validateName rejects names that cannot be represented safely in
// vmx files or bundle paths.
func validateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New(i18n.T("err.name.empty"))
	}
	if strings.ContainsAny(name, `/\\"`) || strings.ContainsAny(name, "\n\r\t") {
		return fmt.Errorf(i18n.T("err.name.invalid"), name)
	}
	return nil
}

// ensureUniqueName errors if another registered VM already uses name.
// Pass selfPath for renames to exempt the VM being changed.
func (d *Driver) ensureUniqueName(name, selfPath string) error {
	entries, err := d.inventory()
	if err != nil {
		return err
	}
	for _, e := range entries {
		if selfPath != "" && absPath(e.Path) == absPath(selfPath) {
			continue
		}
		if strings.EqualFold(e.Name, name) {
			return fmt.Errorf(i18n.T("err.name.exists"), e.Name)
		}
	}
	return nil
}

// ensurePoweredOff rejects the action while the VM is running.
func (d *Driver) ensurePoweredOff(ctx context.Context, path, action string) error {
	running, err := d.listRunning(ctx)
	if err != nil {
		return err
	}
	if running[absPath(path)] {
		return fmt.Errorf(i18n.T("err.vmrun.runningFor"), action)
	}
	return nil
}

// ensureRunning rejects the action unless the VM is powered on.
func (d *Driver) ensureRunning(ctx context.Context, path, action string) error {
	running, err := d.listRunning(ctx)
	if err != nil {
		return err
	}
	if !running[absPath(path)] {
		return fmt.Errorf(i18n.T("err.vmrun.notRunningFor"), action)
	}
	return nil
}

// cloneDestPath computes the destination vmx path for a clone.
func cloneDestPath(src string, opts driver.CloneOptions) string {
	if opts.Path != "" {
		p := expandPath(opts.Path)
		if isVMXFile(p) {
			return p
		}
		return filepath.Join(p, opts.Name+".vmx")
	}
	// Default: alongside the source, as <name>.vmwarevm/<name>.vmx.
	parent := filepath.Dir(src)
	if isVMBundleDir(parent) {
		parent = filepath.Dir(parent)
	}
	return filepath.Join(parent, opts.Name+".vmwarevm", opts.Name+".vmx")
}

// checkDestAvailable fails early if the destination already exists.
func checkDestAvailable(dest string) error {
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf(i18n.T("err.dest.exists"), dest)
	}
	dir := filepath.Dir(dest)
	if fi, err := os.Stat(dir); err == nil {
		if !fi.IsDir() {
			return fmt.Errorf(i18n.T("err.dest.notDir"), dir)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		if len(entries) > 0 {
			return fmt.Errorf(i18n.T("err.dest.notEmpty"), dir)
		}
	}
	return nil
}

// Clone implements driver.Driver. The source must be powered off.
func (d *Driver) Clone(ctx context.Context, ref string, opts driver.CloneOptions) (string, error) {
	if err := validateName(opts.Name); err != nil {
		return "", err
	}
	src, err := d.resolve(ref)
	if err != nil {
		return "", err
	}
	if err := d.ensureUniqueName(opts.Name, ""); err != nil {
		return "", err
	}
	if err := d.ensurePoweredOff(ctx, src, "clone"); err != nil {
		return "", err
	}

	dest := cloneDestPath(src, opts)
	if err := checkDestAvailable(dest); err != nil {
		return "", err
	}
	bundleDir := filepath.Dir(dest)
	created := false
	if _, err := os.Stat(bundleDir); os.IsNotExist(err) {
		if err := os.MkdirAll(bundleDir, 0o755); err != nil {
			return "", err
		}
		created = true
	}

	mode := "linked"
	if opts.Full {
		mode = "full"
	}
	args := []string{"clone", src, dest, mode}
	if opts.Snapshot != "" {
		args = append(args, "-snapshot="+opts.Snapshot)
	}
	args = append(args, "-cloneName="+opts.Name)

	if _, err := d.runner.run(ctx, args...); err != nil {
		if created {
			os.Remove(bundleDir) // only if vmrun left nothing behind
		}
		return "", err
	}

	// vmrun blanks uuid.bios in clones; give the new VM a stable
	// identity right away instead of waiting for first boot, then
	// make it visible to `vmctl list`.
	uuid := fillCloneUUID(dest)
	if err := registerVM(d.invPath, dest, opts.Name, uuid); err != nil {
		return dest, fmt.Errorf(i18n.T("err.clone.regFail"), dest, err)
	}
	return dest, nil
}

// Create implements driver.Driver. The vmrun backend provisions by
// cloning a source VM (Fusion's vmrun has no createVM command).
func (d *Driver) Create(ctx context.Context, opts driver.CreateOptions) (string, error) {
	if err := validateName(opts.Name); err != nil {
		return "", err
	}
	if opts.From == "" {
		return "", fmt.Errorf("%w: %s", driver.ErrNotSupported, i18n.T("err.clone.noFrom"))
	}
	dest, err := d.Clone(ctx, opts.From, driver.CloneOptions{
		Name:     opts.Name,
		Full:     opts.Full,
		Snapshot: opts.Snapshot,
		Path:     opts.Path,
	})
	if err != nil {
		return "", err
	}
	if opts.MemoryMB > 0 || opts.CPUs > 0 {
		set := driver.SetOptions{}
		if opts.MemoryMB > 0 {
			set.MemoryMB = &opts.MemoryMB
		}
		if opts.CPUs > 0 {
			set.CPUs = &opts.CPUs
		}
		if err := d.Set(ctx, dest, set); err != nil {
			return dest, fmt.Errorf(i18n.T("err.clone.configFail"), dest, err)
		}
	}
	return dest, nil
}

// Set implements driver.Driver: edits the vmx of a powered-off VM.
func (d *Driver) Set(ctx context.Context, ref string, opts driver.SetOptions) error {
	if opts.Empty() {
		return errors.New(i18n.T("err.set.flags"))
	}
	path, err := d.resolve(ref)
	if err != nil {
		return err
	}
	if err := d.ensurePoweredOff(ctx, path, "set"); err != nil {
		return err
	}

	cfg, err := readVMX(path)
	if err != nil {
		return err
	}

	keys := map[string]string{}
	rename := false
	if opts.Name != nil {
		if err := validateName(*opts.Name); err != nil {
			return err
		}
		if *opts.Name != cfg.DisplayName {
			if err := d.ensureUniqueName(*opts.Name, path); err != nil {
				return err
			}
			rename = true
		}
		keys["displayName"] = *opts.Name
	}
	if opts.MemoryMB != nil {
		if *opts.MemoryMB <= 0 {
			return errors.New(i18n.T("err.set.memoryPositive"))
		}
		keys["memsize"] = strconv.Itoa(*opts.MemoryMB)
	}
	if opts.CPUs != nil {
		if *opts.CPUs <= 0 {
			return errors.New(i18n.T("err.set.cpusPositive"))
		}
		keys["numvcpus"] = strconv.Itoa(*opts.CPUs)
	}

	if err := setVMXKeys(path, keys); err != nil {
		return err
	}
	if rename {
		if err := renameInInventory(d.invPath, path, *opts.Name); err != nil {
			return fmt.Errorf(i18n.T("err.set.renameFail"), path, err)
		}
	}
	return nil
}

// Delete implements driver.Driver: removes the VM and its files
// permanently (vmrun deleteVM deletes the bundle from disk).
func (d *Driver) Delete(ctx context.Context, ref string) error {
	path, err := d.resolve(ref)
	if err != nil {
		return err
	}
	if err := d.ensurePoweredOff(ctx, path, "delete"); err != nil {
		return err
	}
	if _, err := d.runner.run(ctx, "deleteVM", path); err != nil {
		return err
	}
	// Belt and braces: drop the inventory entry in case vmrun left it.
	if err := unregisterVM(d.invPath, path); err != nil {
		return fmt.Errorf(i18n.T("err.delete.cleanupFail"), err)
	}
	return nil
}
