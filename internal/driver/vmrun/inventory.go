package vmrun

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// InventoryEntry is one registered VM from Fusion's vmInventory file.
type InventoryEntry struct {
	Path string // absolute vmx path
	Name string // display name
	UUID string // normalized (dashed, lowercase) UUID
}

// inventoryLine matches lines like: vmlist2.DisplayName = "my-vm"
var inventoryLine = regexp.MustCompile(`^vmlist(\d+)\.([A-Za-z0-9_]+) = "(.*)"$`)

// defaultInventoryPath returns the Fusion inventory file location.
// Empty string means the platform has no known inventory file (M1
// targets macOS Fusion; Workstation stores it elsewhere).
func defaultInventoryPath() string {
	if runtime.GOOS != "darwin" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Library", "Application Support", "VMware Fusion", "vmInventory")
}

// parseInventory decodes the vmInventory format (grouped vmlistN.* keys).
// Entries with an empty config path are skipped.
func parseInventory(r io.Reader) ([]InventoryEntry, error) {
	groups := map[int]map[string]string{}
	var order []int

	sc := bufio.NewScanner(r)
	for sc.Scan() {
		m := inventoryLine.FindStringSubmatch(strings.TrimRight(sc.Text(), "\r"))
		if m == nil {
			continue
		}
		var idx int
		if _, err := fmt.Sscanf(m[1], "%d", &idx); err != nil {
			continue
		}
		if groups[idx] == nil {
			groups[idx] = map[string]string{}
			order = append(order, idx)
		}
		groups[idx][m[2]] = m[3]
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	var entries []InventoryEntry
	for _, idx := range order {
		g := groups[idx]
		path := strings.TrimSpace(g["config"])
		if path == "" {
			continue
		}
		entries = append(entries, InventoryEntry{
			Path: path,
			Name: g["DisplayName"],
			UUID: NormalizeUUID(g["UUID"]),
		})
	}
	return entries, nil
}

// readInventory loads the inventory file. A missing file yields no
// entries and no error (VMs can still be discovered via vmrun list).
func readInventory(path string) ([]InventoryEntry, error) {
	if path == "" {
		return nil, nil
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	return parseInventory(f)
}

// NormalizeUUID turns vmInventory's spaced hex ("56 4d ...") into the
// canonical 8-4-4-4-12 form. Unrecognized input is returned trimmed.
func NormalizeUUID(s string) string {
	hex := strings.Map(func(r rune) rune {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
			return r
		default:
			return -1
		}
	}, s)
	hex = strings.ToLower(hex)
	if len(hex) != 32 {
		return strings.TrimSpace(s)
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s", hex[0:8], hex[8:12], hex[12:16], hex[16:20], hex[20:32])
}
