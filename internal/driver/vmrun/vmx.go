package vmrun

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf16"
)

// vmxConfig holds the .vmx fields VMCTL surfaces.
type vmxConfig struct {
	DisplayName string
	GuestOS     string
	MemSizeMB   int
	NumCPUs     int
	UUID        string
	Annotation  string
}

// readVMX parses a .vmx file, tolerating UTF-16 encoded files.
func readVMX(path string) (*vmxConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseVMX(decodeVMX(raw)), nil
}

// decodeVMX converts raw vmx bytes to text. VMware occasionally writes
// UTF-16; plain UTF-8/ASCII (including BOM) passes through.
func decodeVMX(raw []byte) io.Reader {
	trimmed := bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
	if isUTF16(trimmed) {
		return strings.NewReader(decodeUTF16LE(trimmed))
	}
	return bytes.NewReader(trimmed)
}

func isUTF16(b []byte) bool {
	if len(b) < 2 || len(b)%2 != 0 {
		return false
	}
	// Heuristic: NUL bytes in odd positions for ASCII-heavy content.
	zeros := 0
	for i := 1; i < len(b); i += 2 {
		if b[i] == 0 {
			zeros++
		}
	}
	return zeros > len(b)/4
}

func decodeUTF16LE(b []byte) string {
	u := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		u = append(u, uint16(b[i])|uint16(b[i+1])<<8)
	}
	return string(utf16.Decode(u))
}

// parseVMX extracts the subset of keys VMCTL needs. Values may be
// quoted or bare.
func parseVMX(r io.Reader) *vmxConfig {
	cfg := &vmxConfig{}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		val = unquote(strings.TrimSpace(val))
		switch key {
		case "displayname":
			cfg.DisplayName = val
		case "guestos":
			cfg.GuestOS = val
		case "memsize":
			cfg.MemSizeMB, _ = strconv.Atoi(val)
		case "numvcpus":
			cfg.NumCPUs, _ = strconv.Atoi(val)
		case "uuid.bios", "uuid":
			if cfg.UUID == "" {
				cfg.UUID = NormalizeUUID(val)
			}
		case "annotation":
			cfg.Annotation = val
		}
	}
	return cfg
}

func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// setVMXKeys writes key/value pairs into a .vmx file, preserving
// existing lines and formatting. Keys missing from the file are
// appended. UTF-16 encoded files are rejected (writing them back
// correctly is not implemented yet).
func setVMXKeys(path string, keys map[string]string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if isUTF16(raw) {
		return fmt.Errorf("%s: UTF-16 encoded vmx is not supported for editing", path)
	}
	trimmed := bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
	lines := splitLines(strings.TrimSuffix(string(trimmed), "\n"))
	if lines == nil {
		lines = []string{}
	}

	for key, val := range keys {
		matched := false
		lowerKey := strings.ToLower(key)
		for i, line := range lines {
			k, _, ok := strings.Cut(line, "=")
			if !ok || strings.ToLower(strings.TrimSpace(k)) != lowerKey {
				continue
			}
			lines[i] = strings.TrimSpace(k) + " = " + quote(val)
			matched = true
			break
		}
		if !matched {
			lines = append(lines, key+" = "+quote(val))
		}
	}

	mode := os.FileMode(0o644)
	if st, err := os.Stat(path); err == nil {
		mode = st.Mode().Perm()
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), mode)
}
