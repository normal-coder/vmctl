package vmrun

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gitee.com/normalcoder/vmctl/internal/i18n"
	"gitee.com/normalcoder/vmctl/internal/model"
)

// Snapshots implements driver.Driver.
func (d *Driver) Snapshots(ctx context.Context, ref string, tree bool) ([]model.Snapshot, error) {
	path, err := d.resolve(ref)
	if err != nil {
		return nil, err
	}
	args := []string{"listSnapshots", path}
	if tree {
		args = append(args, "showTree")
	}
	out, err := d.runner.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	return parseSnapshotList(out), nil
}

// parseSnapshotList decodes `vmrun listSnapshots` output:
//
//	Total snapshots: 2
//	snap1
//	    snap2          (showTree indents children with tabs)
//
// A "Total snapshots: 0" header yields an empty list.
func parseSnapshotList(out string) []model.Snapshot {
	var snaps []model.Snapshot
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" || strings.HasPrefix(strings.TrimSpace(line), "Total snapshots:") {
			continue
		}
		depth := 0
		for strings.HasPrefix(line, "\t") {
			depth++
			line = line[1:]
		}
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		snaps = append(snaps, model.Snapshot{Name: name, Depth: depth})
	}
	return snaps
}

// SnapshotCreate implements driver.Driver.
func (d *Driver) SnapshotCreate(ctx context.Context, ref, name string) error {
	if err := validateName(name); err != nil {
		return fmt.Errorf("invalid snapshot name: %w", err)
	}
	path, err := d.resolve(ref)
	if err != nil {
		return err
	}
	_, err = d.runner.run(ctx, "snapshot", path, name)
	return err
}

// SnapshotDelete implements driver.Driver.
//
// A "<name>#<uid>" reference selects the snapshot by uid via vmcli —
// the only way to tell same-named snapshots apart. Plain names are
// pre-scanned for duplicates and rejected with candidates when more
// than one snapshot matches.
func (d *Driver) SnapshotDelete(ctx context.Context, ref, name string, deleteChildren bool) error {
	base, uid, hasUID := splitSnapshotRef(name)
	if base == "" {
		return errors.New(i18n.T("err.snapshot.needName"))
	}
	path, err := d.resolve(ref)
	if err != nil {
		return err
	}

	if hasUID {
		args := []string{"Snapshot", "Delete"}
		if deleteChildren {
			args = append(args, "-d")
		}
		args = append(args, strconv.Itoa(uid))
		_, err := d.vmcliExec(ctx, path, args...)
		return err
	}

	if err := d.checkDuplicateNames(ctx, path, base); err != nil {
		return err
	}
	args := []string{"deleteSnapshot", path, base}
	if deleteChildren {
		args = append(args, "andDeleteChildren")
	}
	_, err = d.runner.run(ctx, args...)
	return err
}

// SnapshotRevert implements driver.Driver. vmrun itself rejects the
// operation while the VM is powered on; its message is passed through.
// A "<name>#<uid>" reference reverts by uid via vmcli instead.
func (d *Driver) SnapshotRevert(ctx context.Context, ref, name string) error {
	base, uid, hasUID := splitSnapshotRef(name)
	if base == "" {
		return errors.New(i18n.T("err.snapshot.needName"))
	}
	path, err := d.resolve(ref)
	if err != nil {
		return err
	}

	if hasUID {
		_, err := d.vmcliExec(ctx, path, "Snapshot", "Revert", strconv.Itoa(uid))
		return err
	}

	if err := d.checkDuplicateNames(ctx, path, base); err != nil {
		return err
	}
	_, err = d.runner.run(ctx, "revertToSnapshot", path, base)
	return err
}

// checkDuplicateNames rejects a by-name snapshot operation when more
// than one snapshot carries that name. Listing failures are ignored:
// vmrun then reports whatever the real problem is.
func (d *Driver) checkDuplicateNames(ctx context.Context, path, name string) error {
	out, err := d.runner.run(ctx, "listSnapshots", path)
	if err != nil {
		return nil
	}
	matches := 0
	for _, s := range parseSnapshotList(out) {
		if s.Name == name {
			matches++
		}
	}
	if matches < 2 {
		return nil
	}
	return d.errVMCliAmbiguous(ctx, path, name)
}
