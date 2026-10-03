package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"gitee.com/normalcoder/vmctl/internal/i18n"
	"gitee.com/normalcoder/vmctl/internal/model"
)

func TestPrintSnapshots(t *testing.T) {
	snaps := []model.Snapshot{
		{Name: "base", Depth: 0, UID: "42"},
		{Name: "child", Depth: 1, UID: "7"},
		{Name: "legacy", Depth: 0, UID: ""}, // backend without uid info
	}
	var buf bytes.Buffer
	if err := PrintSnapshots(&buf, "my-vm", snaps, false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"NAME", "UID", "base", "child", "legacy", "42", "7"} {
		if !strings.Contains(out, want) {
			t.Errorf("output %q should contain %q", out, want)
		}
	}
	// Nested entries are indented by Depth.
	if !strings.Contains(out, "  child") {
		t.Errorf("output %q should indent the nested snapshot", out)
	}
	// A missing uid renders as a dash on that row.
	found := false
	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, "legacy") {
			continue
		}
		found = true
		if !strings.Contains(line, "-") {
			t.Errorf("legacy row should render the empty uid as -: %q", line)
		}
		break
	}
	if !found {
		t.Fatalf("legacy row missing: %q", out)
	}
	// The uid column comes after the name column on the base row.
	if i := strings.Index(out, "base"); i < 0 || !strings.Contains(out[i:], "42") {
		t.Errorf("uid must follow the name: %q", out)
	}
}

func TestPrintSnapshotsJSON(t *testing.T) {
	snaps := []model.Snapshot{
		{Name: "s1", UID: "5"},
		{Name: "s2"}, // uid omitted
	}
	var buf bytes.Buffer
	if err := PrintSnapshots(&buf, "my-vm", snaps, true); err != nil {
		t.Fatal(err)
	}
	var payload struct {
		VM        string           `json:"vm"`
		Snapshots []model.Snapshot `json:"snapshots"`
	}
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("decode %q: %v", buf.String(), err)
	}
	if payload.VM != "my-vm" || len(payload.Snapshots) != 2 {
		t.Fatalf("payload = %+v", payload)
	}
	if payload.Snapshots[0].UID != "5" || payload.Snapshots[1].UID != "" {
		t.Errorf("uids = %q / %q, want 5 / empty",
			payload.Snapshots[0].UID, payload.Snapshots[1].UID)
	}
	// omitempty: exactly one "uid" key in the document.
	if n := strings.Count(buf.String(), `"uid"`); n != 1 {
		t.Errorf(`"uid" appears %d times, want 1 (omitempty): %s`, n, buf.String())
	}
}

func TestPrintSnapshotsEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintSnapshots(&buf, "lonely-vm", nil, false); err != nil {
		t.Fatal(err)
	}
	want := i18n.T("output.noSnapshots", "lonely-vm")
	if !strings.Contains(buf.String(), want) {
		t.Errorf("output %q, want %q", buf.String(), want)
	}
}

func TestPrintVersion(t *testing.T) {
	t.Run("text", func(t *testing.T) {
		var buf bytes.Buffer
		if err := PrintVersion(&buf, "1.2.3", false); err != nil {
			t.Fatal(err)
		}
		if buf.String() != "vmctl 1.2.3\n" {
			t.Errorf("output = %q, want %q", buf.String(), "vmctl 1.2.3\n")
		}
	})
	t.Run("json", func(t *testing.T) {
		var buf bytes.Buffer
		if err := PrintVersion(&buf, "1.2.3", true); err != nil {
			t.Fatal(err)
		}
		var got map[string]string
		if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
			t.Fatalf("decode %q: %v", buf.String(), err)
		}
		if len(got) != 1 || got["version"] != "1.2.3" {
			t.Errorf("payload = %v, want exactly {version:1.2.3}", got)
		}
	})
}
