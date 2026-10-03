package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// TestVersionText: `vmctl version` prints "vmctl <version>" on stdout.
func TestVersionText(t *testing.T) {
	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatalf("version: %v", err)
	}
	if got := buf.String(); got != "vmctl "+Version+"\n" {
		t.Errorf("stdout = %q, want %q", got, "vmctl "+Version+"\n")
	}
}

// TestVersionJSON: `vmctl version --json` emits exactly
// {"version": "<version>"}. NewRootCmd re-binds the global flagJSON
// via BoolVar, so running this after other tests is safe.
func TestVersionJSON(t *testing.T) {
	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"version", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("version --json: %v", err)
	}
	var got map[string]string
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("decode %q: %v", buf.String(), err)
	}
	if len(got) != 1 || got["version"] != Version {
		t.Errorf("payload = %v, want exactly {version:%s}", got, Version)
	}
	// Plain text mode must not have leaked into the JSON document.
	if strings.Contains(buf.String(), "vmctl "+Version) {
		t.Errorf("JSON must not contain the text form: %q", buf.String())
	}
}
