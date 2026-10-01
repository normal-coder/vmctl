package vmrun

import (
	"bufio"
	"crypto/rand"
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

// --- inventory mutation ------------------------------------------------

// generateUUID returns a VMware-style spaced BIOS UUID
// ("56 4d xx xx xx xx xx xx xx-xx xx xx xx xx xx xx xx") with a
// random body. The "56 4d" prefix mimics UUIDs VMware itself issues.
func generateUUID() (string, error) {
	b := make([]byte, 16)
	b[0], b[1] = 0x56, 0x4d
	if _, err := rand.Read(b[2:]); err != nil {
		return "", err
	}
	pairs := make([]string, len(b))
	for i, v := range b {
		pairs[i] = fmt.Sprintf("%02x", v)
	}
	return strings.Join(pairs[:8], " ") + "-" + strings.Join(pairs[8:], " "), nil
}

// fillCloneUUID gives a freshly cloned vmx a stable identity. vmrun
// blanks uuid.bios when cloning; Fusion would only regenerate it on
// first boot, leaving the VM without an ID until then. Returns the
// UUID in inventory (spaced) form, or "" on best-effort failure.
func fillCloneUUID(vmxPath string) string {
	cfg, err := readVMX(vmxPath)
	if err != nil {
		return ""
	}
	if cfg.UUID != "" {
		// Already has an identity (e.g. source value copied through).
		return cfg.UUID
	}
	uuid, err := generateUUID()
	if err != nil {
		return ""
	}
	if err := setVMXKeys(vmxPath, map[string]string{"uuid.bios": uuid}); err != nil {
		return ""
	}
	return uuid
}

// registerVM appends an inventory entry for vmxPath so the VM shows up
// in vmctl list. Best effort: a missing inventory file (Fusion never
// registered anything) is not an error. Existing entries for the same
// path are updated instead.
func registerVM(invPath, vmxPath, name, uuid string) error {
	if invPath == "" {
		return nil
	}
	raw, err := os.ReadFile(invPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	lines := splitLines(string(raw))

	// Update in place if the path is already registered.
	if idx, ok := findEntryIndex(lines, vmxPath); ok {
		lines = setEntryField(lines, idx, "DisplayName", name)
		if uuid != "" {
			lines = setEntryField(lines, idx, "UUID", uuid)
		}
		return writeInventory(invPath, lines)
	}

	// Append under the next free index.
	maxIdx := 0
	for _, m := range vmlistIndex.FindAllStringSubmatch(string(raw), -1) {
		var n int
		if _, err := fmt.Sscanf(m[1], "%d", &n); err == nil && n > maxIdx {
			maxIdx = n
		}
	}
	newIdx := maxIdx + 1
	lines = append(lines,
		fmt.Sprintf("vmlist%d.config = %s", newIdx, quote(vmxPath)),
		fmt.Sprintf("vmlist%d.DisplayName = %s", newIdx, quote(name)),
	)
	if uuid != "" {
		lines = append(lines, fmt.Sprintf("vmlist%d.UUID = %s", newIdx, quote(uuid)))
	}
	return writeInventory(invPath, lines)
}

// unregisterVM removes any inventory entry pointing at vmxPath.
func unregisterVM(invPath, vmxPath string) error {
	if invPath == "" {
		return nil
	}
	raw, err := os.ReadFile(invPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	idx, ok := findEntryIndex(splitLines(string(raw)), vmxPath)
	if !ok {
		return nil
	}
	var kept []string
	for _, line := range splitLines(string(raw)) {
		if !strings.HasPrefix(line, fmt.Sprintf("vmlist%d.", idx)) {
			kept = append(kept, line)
		}
	}
	return writeInventory(invPath, kept)
}

// renameInInventory updates the DisplayName of an existing entry.
// Unregistered VMs are left untouched (renaming must not register them).
func renameInInventory(invPath, vmxPath, name string) error {
	if invPath == "" {
		return nil
	}
	raw, err := os.ReadFile(invPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	idx, ok := findEntryIndex(splitLines(string(raw)), vmxPath)
	if !ok {
		return nil
	}
	return writeInventory(invPath, setEntryField(splitLines(string(raw)), idx, "DisplayName", name))
}

// vmlistIndex matches the leading vmlist<N>. of an inventory line.
// The (?m) flag makes ^ match at the start of every line when the
// regex is applied to the whole file.
var vmlistIndex = regexp.MustCompile(`(?m)^vmlist(\d+)\.`)

// findEntryIndex returns the vmlist index whose config is vmxPath.
func findEntryIndex(lines []string, vmxPath string) (int, bool) {
	for _, line := range lines {
		m := vmlistIndex.FindStringSubmatch(line)
		if m == nil || !strings.HasSuffix(line, ".config = "+quote(vmxPath)) {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(m[1], "%d", &n); err == nil {
			return n, true
		}
	}
	return 0, false
}

// setEntryField rewrites one vmlist<idx>.<field> value in place.
func setEntryField(lines []string, idx int, field, value string) []string {
	prefix := fmt.Sprintf("vmlist%d.%s = ", idx, field)
	for i, line := range lines {
		if strings.HasPrefix(line, prefix) {
			lines[i] = prefix + quote(value)
			return lines
		}
	}
	return append(lines, prefix+quote(value))
}

func quote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

func splitLines(s string) []string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func writeInventory(path string, lines []string) error {
	mode := os.FileMode(0o644)
	if st, err := os.Stat(path); err == nil {
		mode = st.Mode().Perm()
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), mode)
}
