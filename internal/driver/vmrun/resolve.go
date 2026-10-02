package vmrun

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/i18n"
)

// resolve maps a user-supplied reference (path, name or UUID) to an
// absolute vmx path. Accepts:
//  1. a direct path to a .vmx file or .vmwarevm bundle
//  2. a display name from the inventory (case-insensitive exact match)
//  3. a UUID / UUID prefix from the inventory
//  4. a unique substring of a known name (helpful for short refs)
func (d *Driver) resolve(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", driver.WrapNotFound(i18n.T("err.resolve.empty"))
	}

	// 1. direct path
	if p := expandPath(ref); isVMXFile(p) {
		return p, nil
	}
	if p := expandPath(ref); isVMBundleDir(p) {
		if vmx, ok := findVMXInBundle(p); ok {
			return vmx, nil
		}
	}

	entries, err := d.inventory()
	if err != nil {
		return "", err
	}

	// 2. exact name (case-insensitive)
	var nameHits []InventoryEntry
	for _, e := range entries {
		if strings.EqualFold(e.Name, ref) {
			nameHits = append(nameHits, e)
		}
	}
	switch len(nameHits) {
	case 1:
		return nameHits[0].Path, nil
	case 0:
		// fall through
	default:
		return "", fmt.Errorf(i18n.T("err.resolve.ambiguous"), ref, len(nameHits))
	}

	// 3. UUID: exact, dashed prefix or bare-hex prefix (>= 8 chars)
	if refHex := compactHex(ref); len(refHex) >= 8 {
		var uuidHits []InventoryEntry
		for _, e := range entries {
			if strings.HasPrefix(compactHex(e.UUID), refHex) {
				uuidHits = append(uuidHits, e)
			}
		}
		if len(uuidHits) == 1 {
			return uuidHits[0].Path, nil
		}
		if len(uuidHits) > 1 {
			return "", fmt.Errorf(i18n.T("err.resolve.ambiguous"), ref, len(uuidHits))
		}
	}

	// 4. unique substring match on display names
	var partial []InventoryEntry
	lower := strings.ToLower(ref)
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.Name), lower) {
			partial = append(partial, e)
		}
	}
	if len(partial) == 1 {
		return partial[0].Path, nil
	}
	if len(partial) > 1 {
		return "", fmt.Errorf(i18n.T("err.resolve.ambiguousNames"), ref, joinNames(partial))
	}

	return "", driver.WrapNotFound(fmt.Sprintf(i18n.T("err.resolve.notFound"), ref, joinNames(entries)))
}

// compactHex strips separators from s and lowercases it. Returns "" if
// any character is not a hex digit (so plain names never hit the UUID
// matching path).
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

func joinNames(entries []InventoryEntry) string {
	if len(entries) == 0 {
		return "none"
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Name != "" {
			names = append(names, e.Name)
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, ", ")
}

func expandPath(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~/"))
		}
	}
	return p
}

func isVMXFile(p string) bool {
	return strings.EqualFold(filepath.Ext(p), ".vmx")
}

func isVMBundleDir(p string) bool {
	return strings.EqualFold(filepath.Ext(p), ".vmwarevm")
}

func findVMXInBundle(dir string) (string, bool) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.vmx"))
	if err != nil || len(matches) == 0 {
		return "", false
	}
	return matches[0], true
}
