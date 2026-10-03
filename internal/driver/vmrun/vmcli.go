package vmrun

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"gitee.com/normalcoder/vmctl/internal/i18n"
)

// vmcli is VMware Fusion's higher-level CLI (13.5+). Unlike vmrun it
// addresses snapshots by uid, which lets vmctl disambiguate snapshots
// that share a name — vmrun only ever matches by name.

// splitSnapshotRef splits a "<name>#<uid>" snapshot reference. The
// suffix must be all digits and >= 1 (helperUID 0 is meaningless);
// anything else is returned as a plain name so names containing "#"
// keep working.
func splitSnapshotRef(ref string) (name string, uid int, ok bool) {
	i := strings.LastIndex(ref, "#")
	if i < 1 {
		return ref, 0, false
	}
	suffix := ref[i+1:]
	if suffix == "" {
		return ref, 0, false
	}
	for _, c := range suffix {
		if c < '0' || c > '9' {
			return ref, 0, false
		}
	}
	n, err := strconv.Atoi(suffix)
	if err != nil || n < 1 {
		return ref, 0, false
	}
	return ref[:i], n, true
}

// vmcliSnapshot is one node of a vmcli query tree.
type vmcliSnapshot struct {
	Name      string
	ParentUID int
	UID       int
}

// parseVMCliQuery decodes `vmcli <vmx> Snapshot query` output
// (default YAML-ish style):
//
//	currentUID: 5
//	helperUID: 0
//	snapshots:
//	  - displayName: Clone
//	    parentUID: 0
//	    uid: 1
//
// ok reports whether the document contained the "snapshots:" marker
// — the only reliable signal that this is a real query result. vmcli
// happily exits 0 with unrelated text for input it does not
// recognize, so absence of the marker must trigger a fallback.
func parseVMCliQuery(out string) (snaps []vmcliSnapshot, ok bool) {
	var cur *vmcliSnapshot
	inList := false
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if trimmed == "snapshots:" {
			inList = true
			ok = true
			continue
		}
		if !inList {
			continue
		}
		if rest, isEntry := strings.CutPrefix(trimmed, "- "); isEntry {
			trimmed = rest
			snaps = append(snaps, vmcliSnapshot{})
			cur = &snaps[len(snaps)-1]
		}
		if cur == nil {
			continue
		}
		key, val, hasKV := strings.Cut(trimmed, ":")
		if !hasKV {
			continue
		}
		val = strings.TrimSpace(val)
		switch strings.TrimSpace(key) {
		case "displayName":
			cur.Name = val
		case "parentUID":
			if n, err := strconv.Atoi(val); err == nil {
				cur.ParentUID = n
			}
		case "uid":
			if n, err := strconv.Atoi(val); err == nil {
				cur.UID = n
			}
		}
	}
	return snaps, ok
}

// parseVMCliSnapshots is parseVMCliQuery without the marker check,
// for callers that only need the entries.
func parseVMCliSnapshots(out string) []vmcliSnapshot {
	snaps, _ := parseVMCliQuery(out)
	return snaps
}

// vmcliBinary lazily locates and caches the vmcli binary.
func (d *Driver) vmcliBinary() (string, error) {
	if d.vmcliBin != "" {
		return d.vmcliBin, nil
	}
	bin, err := locateVMCli(d.vmcliPath)
	if err != nil {
		return "", err
	}
	d.vmcliBin = bin
	return bin, nil
}

// vmcliExec runs one vmcli invocation (the vmx path comes first,
// then the subcommand). Output and errors are vmcli's own English
// text and are passed through untranslated.
func (d *Driver) vmcliExec(ctx context.Context, path string, args ...string) (string, error) {
	bin, err := d.vmcliBinary()
	if err != nil {
		return "", err
	}
	full := append([]string{path}, args...)
	out, err := d.runner.execFn(ctx, bin, full...)
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text != "" {
			return "", fmt.Errorf("%s", text)
		}
		return "", err
	}
	return text, nil
}

// vmcliRefsFor returns "<name>#<uid>" candidates for snapshots named
// name, joined for an error message. It fails when vmcli is missing
// or the query output cannot be parsed — callers then fall back to a
// hint without candidates.
func (d *Driver) vmcliRefsFor(ctx context.Context, path, name string) (string, error) {
	out, err := d.vmcliExec(ctx, path, "Snapshot", "query")
	if err != nil {
		return "", err
	}
	var refs []string
	for _, s := range parseVMCliSnapshots(out) {
		if s.Name == name && s.UID >= 1 {
			refs = append(refs, fmt.Sprintf("%s#%d", s.Name, s.UID))
		}
	}
	if len(refs) == 0 {
		return "", fmt.Errorf("no uid for snapshot %q", name)
	}
	return strings.Join(refs, ", "), nil
}

// errVMCliAmbiguous builds the duplicate-name error for delete/revert,
// listing "<name>#<uid>" candidates when vmcli can provide them.
func (d *Driver) errVMCliAmbiguous(ctx context.Context, path, name string) error {
	if refs, err := d.vmcliRefsFor(ctx, path, name); err == nil {
		return fmt.Errorf(i18n.T("err.snapshot.ambiguous"), name, refs)
	}
	return fmt.Errorf(i18n.T("err.snapshot.ambiguousNoVmcli"), name)
}
