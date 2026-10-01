package vmrun

import (
	"context"
	"fmt"
	"strings"

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
func (d *Driver) SnapshotDelete(ctx context.Context, ref, name string, deleteChildren bool) error {
	if name == "" {
		return fmt.Errorf("snapshot name is required")
	}
	path, err := d.resolve(ref)
	if err != nil {
		return err
	}
	args := []string{"deleteSnapshot", path, name}
	if deleteChildren {
		args = append(args, "andDeleteChildren")
	}
	_, err = d.runner.run(ctx, args...)
	return err
}

// SnapshotRevert implements driver.Driver. vmrun itself rejects the
// operation while the VM is powered on; its message is passed through.
func (d *Driver) SnapshotRevert(ctx context.Context, ref, name string) error {
	if name == "" {
		return fmt.Errorf("snapshot name is required")
	}
	path, err := d.resolve(ref)
	if err != nil {
		return err
	}
	_, err = d.runner.run(ctx, "revertToSnapshot", path, name)
	return err
}
