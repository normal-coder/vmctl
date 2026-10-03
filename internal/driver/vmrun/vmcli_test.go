package vmrun

import (
	"context"
	"os"
	"strings"
	"testing"
)

// --- splitSnapshotRef -------------------------------------------------

func TestSplitSnapshotRef(t *testing.T) {
	cases := []struct {
		in   string
		name string
		uid  int
		ok   bool
	}{
		{"snap#3", "snap", 3, true},
		{"a#b#4", "a#b", 4, true},
		{"v1.2#12", "v1.2", 12, true},
		{"snap#03", "snap", 3, true},
		{"snap", "snap", 0, false},
		{"", "", 0, false},
		{"snap#0", "snap#0", 0, false}, // uid 0 (helper) is not a selector
		{"snap#x", "snap#x", 0, false}, // non-digit suffix stays in the name
		{"snap#", "snap#", 0, false},   // empty suffix
		{"#3", "#3", 0, false},         // '#' at the start: whole name
		{"clone#", "clone#", 0, false}, // trailing '#'
		{"100%", "100%", 0, false},     // no '#'
	}
	for _, c := range cases {
		name, uid, ok := splitSnapshotRef(c.in)
		if name != c.name || uid != c.uid || ok != c.ok {
			t.Errorf("splitSnapshotRef(%q) = (%q, %d, %v), want (%q, %d, %v)",
				c.in, name, uid, ok, c.name, c.uid, c.ok)
		}
	}
}

// --- parseVMCliSnapshots ----------------------------------------------

const vmcliQueryFixture = `currentUID: 5
helperUID: 0
snapshots:
  - displayName: Clone
    parentUID: 0
    uid: 1
  - displayName: Clone
    parentUID: 1
    uid: 2
  - displayName: base
    parentUID: 2
    uid: 3
  - displayName: Clone
    parentUID: 3
    uid: 5
`

func TestParseVMCliSnapshots(t *testing.T) {
	snaps := parseVMCliSnapshots(vmcliQueryFixture)
	if len(snaps) != 4 {
		t.Fatalf("want 4 snapshots, got %d: %+v", len(snaps), snaps)
	}
	want := []vmcliSnapshot{
		{Name: "Clone", ParentUID: 0, UID: 1},
		{Name: "Clone", ParentUID: 1, UID: 2},
		{Name: "base", ParentUID: 2, UID: 3},
		{Name: "Clone", ParentUID: 3, UID: 5},
	}
	for i, w := range want {
		if snaps[i] != w {
			t.Errorf("snap[%d] = %+v, want %+v", i, snaps[i], w)
		}
	}

	if got := parseVMCliSnapshots(""); len(got) != 0 {
		t.Errorf("empty output: %+v", got)
	}
	if got := parseVMCliSnapshots("currentUID: 1\nhelperUID: 0\n"); len(got) != 0 {
		t.Errorf("no snapshots key: %+v", got)
	}
}

// The marker gates query results: vmcli exits 0 with unrelated text
// for input it does not recognize, so only the "snapshots:" line
// proves this is a real query document.
func TestParseVMCliQueryMarker(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"fixture", vmcliQueryFixture, true},
		{"markerOnly", "currentUID: 0\nsnapshots:\n", true}, // valid empty result
		{"noMarker", "currentUID: 1\nhelperUID: 0\n", false},
		{"garbage", "vmcli: I do not understand", false},
		{"empty", "", false},
	}
	for _, c := range cases {
		snaps, ok := parseVMCliQuery(c.in)
		if ok != c.want {
			t.Errorf("%s: ok = %v, want %v", c.name, ok, c.want)
		}
		if c.want && c.name == "fixture" && len(snaps) != 4 {
			t.Errorf("fixture entries = %d, want 4", len(snaps))
		}
		if c.name == "markerOnly" && len(snaps) != 0 {
			t.Errorf("marker-only entries = %+v, want none", snaps)
		}
	}
}

// --- locateVMCli -------------------------------------------------------

func TestLocateVMCliOverride(t *testing.T) {
	bin := t.TempDir() + "/vmcli"
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := locateVMCli(bin)
	if err != nil || got != bin {
		t.Errorf("locateVMCli(%q) = (%q, %v), want (%q, nil)", bin, got, err, bin)
	}

	if _, err := locateVMCli(bin + "-missing"); err == nil {
		t.Error("missing override must error")
	}
}

func TestLocateVMCliEnv(t *testing.T) {
	bin := t.TempDir() + "/vmcli"
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VMCTL_VMCLI", bin)
	got, err := locateVMCli("")
	if err != nil || got != bin {
		t.Errorf("locateVMCli via env = (%q, %v), want (%q, nil)", got, err, bin)
	}
}

func TestLocateVMCliAutoDetect(t *testing.T) {
	t.Setenv("VMCTL_VMCLI", "")
	bin, err := locateVMCli("")
	if err != nil {
		t.Skipf("vmcli not available on this machine: %v", err)
	}
	if bin == "" {
		t.Error("auto-detect returned empty path without error")
	}
}

// --- uid operations ----------------------------------------------------

// stubVMCli routes vmcli invocations (args[0] = vmx path, no -T
// prefix) into a capture function.
func stubVMCli(d *Driver, fake *fakeExec, respond func(args []string) (string, error)) *[]string {
	d.vmcliBin = "/fake/vmcli"
	var captured []string
	prev := fake.respond
	fake.respond = func(args []string) (string, error) {
		if len(args) > 1 && strings.HasSuffix(args[0], ".vmx") && args[1] == "Snapshot" {
			captured = append([]string(nil), args...)
			return respond(args)
		}
		return prev(args)
	}
	return &captured
}

func TestSnapshotDeleteByUID(t *testing.T) {
	d, fake, paths := newTestEnv(t)
	captured := stubVMCli(d, fake, func([]string) (string, error) { return "", nil })

	if err := d.SnapshotDelete(context.Background(), "demo-two", "Clone#3", true); err != nil {
		t.Fatal(err)
	}
	want := []string{paths["demo-two"], "Snapshot", "Delete", "-d", "3"}
	if strings.Join(*captured, " ") != strings.Join(want, " ") {
		t.Errorf("vmcli args = %q, want %q", *captured, want)
	}

	if err := d.SnapshotDelete(context.Background(), "demo-two", "Clone#3", false); err != nil {
		t.Fatal(err)
	}
	want = []string{paths["demo-two"], "Snapshot", "Delete", "3"}
	if strings.Join(*captured, " ") != strings.Join(want, " ") {
		t.Errorf("vmcli args = %q, want %q", *captured, want)
	}
}

func TestSnapshotRevertByUID(t *testing.T) {
	d, fake, paths := newTestEnv(t)
	captured := stubVMCli(d, fake, func([]string) (string, error) { return "", nil })

	if err := d.SnapshotRevert(context.Background(), "demo-two", "Clone#5"); err != nil {
		t.Fatal(err)
	}
	want := []string{paths["demo-two"], "Snapshot", "Revert", "5"}
	if strings.Join(*captured, " ") != strings.Join(want, " ") {
		t.Errorf("vmcli args = %q, want %q", *captured, want)
	}
}

func TestSnapshotUIDMissingVMCli(t *testing.T) {
	d, _, _ := newTestEnv(t)
	d.vmcliPath = "/definitely/missing/vmcli"

	err := d.SnapshotDelete(context.Background(), "demo-two", "Clone#3", false)
	if err == nil {
		t.Fatal("uid reference without vmcli must error")
	}
	if !strings.Contains(err.Error(), "vmcli") {
		t.Errorf("error should mention vmcli: %v", err)
	}
}

// --- duplicate-name pre-check -----------------------------------------

func TestSnapshotDeleteAmbiguousListsCandidates(t *testing.T) {
	d, fake, _ := newTestEnv(t)
	fake.respond = func(args []string) (string, error) {
		a := stripHostType(args)
		if len(a) > 0 && a[0] == "listSnapshots" {
			return "Total snapshots: 4\nClone\nClone\nbase\nClone\n", nil
		}
		if len(a) > 1 && strings.HasSuffix(a[0], ".vmx") && a[1] == "Snapshot" {
			return vmcliQueryFixture, nil
		}
		return "", nil
	}
	d.vmcliBin = "/fake/vmcli"

	err := d.SnapshotDelete(context.Background(), "demo-two", "Clone", false)
	if err == nil {
		t.Fatal("duplicate snapshot name must error")
	}
	// Same-named entries in the query fixture: uid 1, 2, 5.
	for _, want := range []string{"Clone#1", "Clone#2", "Clone#5"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should list candidate %s", err, want)
		}
	}
	// No delete must have been attempted.
	for _, c := range fake.calls {
		if a := stripHostType(c); len(a) > 0 && a[0] == "deleteSnapshot" {
			t.Errorf("deleteSnapshot must not run on ambiguity: %v", c)
		}
	}
}

func TestSnapshotRevertAmbiguousWithoutVMCli(t *testing.T) {
	d, fake, _ := newTestEnv(t)
	fake.respond = func(args []string) (string, error) {
		a := stripHostType(args)
		if len(a) > 0 && a[0] == "listSnapshots" {
			return "Total snapshots: 2\nsnap-a\nsnap-a\n", nil
		}
		return "", nil
	}
	d.vmcliPath = "/definitely/missing/vmcli" // vmcli unusable

	err := d.SnapshotRevert(context.Background(), "demo-two", "snap-a")
	if err == nil {
		t.Fatal("duplicate snapshot name must error")
	}
	if !strings.Contains(err.Error(), "snap-a") || !strings.Contains(err.Error(), "vmcli") {
		t.Errorf("error should name the snapshot and mention vmcli: %v", err)
	}
}

func TestSnapshotDeleteUniqueNamePassesThrough(t *testing.T) {
	d, fake, paths := newTestEnv(t)
	fake.respond = func(args []string) (string, error) {
		a := stripHostType(args)
		switch {
		case len(a) > 0 && a[0] == "listSnapshots":
			return "Total snapshots: 2\nsnap-a\nsnap-b\n", nil
		case len(a) > 0 && a[0] == "deleteSnapshot":
			return "", nil
		}
		return "", nil
	}
	d.vmcliPath = "/definitely/missing/vmcli" // must not matter: no duplicates

	if err := d.SnapshotDelete(context.Background(), "demo-two", "snap-a", false); err != nil {
		t.Fatal(err)
	}
	want := "deleteSnapshot " + paths["demo-two"] + " snap-a"
	if got := strings.Join(fake.lastArgs(), " "); got != want {
		t.Errorf("args = %q, want %q", got, want)
	}
}
