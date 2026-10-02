package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitee.com/normalcoder/vmctl/internal/i18n"
)

// runComplete executes the root command in __complete mode and
// returns the captured output.
func runComplete(t *testing.T, args ...string) string {
	t.Helper()
	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(append([]string{"__complete"}, args...))
	if err := root.Execute(); err != nil {
		t.Fatalf("__complete %v: %v", args, err)
	}
	return buf.String()
}

func TestCompleteBackendFlag(t *testing.T) {
	out := runComplete(t, "--backend", "")
	for _, want := range []string{"vmrun", "vsphere"} {
		if !strings.Contains(out, want) {
			t.Errorf("output %q should offer %s", out, want)
		}
	}
}

func TestCompleteLangFlag(t *testing.T) {
	out := runComplete(t, "--lang", "")
	for _, want := range []string{"zh", "en"} {
		if !strings.Contains(out, want) {
			t.Errorf("output %q should offer %s", out, want)
		}
	}
}

func TestCompleteProfileFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	yaml := "profiles:\n  lab:\n    backend: vmrun\n  dev:\n    backend: vsphere\n"
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VMCTL_CONFIG", path)

	out := runComplete(t, "--profile", "")
	for _, want := range []string{"lab", "dev"} {
		if !strings.Contains(out, want) {
			t.Errorf("output %q should offer profile %s", out, want)
		}
	}
}

func TestCompleteProfileFlagMissingConfig(t *testing.T) {
	t.Setenv("VMCTL_CONFIG", filepath.Join(t.TempDir(), "missing.yaml"))
	// Must not fail the shell — an unreadable config just yields nothing.
	_ = runComplete(t, "--profile", "")
}

func TestCompleteVMRef(t *testing.T) {
	prev := vmNameList
	vmNameList = func(context.Context) ([]string, error) {
		return []string{"alpha", "alpine", "beta"}, nil
	}
	t.Cleanup(func() { vmNameList = prev })

	out := runComplete(t, "stop", "al")
	for _, want := range []string{"alpha", "alpine"} {
		if !strings.Contains(out, want) {
			t.Errorf("output %q should offer %s", out, want)
		}
	}
	if strings.Contains(out, "beta") {
		t.Errorf("output %q must not offer non-matching beta", out)
	}
}

func TestCompleteVMRefBackendErrorSilent(t *testing.T) {
	prev := vmNameList
	vmNameList = func(context.Context) ([]string, error) {
		return nil, errors.New("backend offline")
	}
	t.Cleanup(func() { vmNameList = prev })

	// Completion swallows backend errors: no crash, no error line.
	out := runComplete(t, "stop", "")
	if strings.Contains(out, "backend offline") {
		t.Errorf("backend errors must not leak into completion output: %q", out)
	}
}

func TestCompletionCommandLocalized(t *testing.T) {
	cases := []struct {
		lang string
		want string
	}{
		{i18n.ZH, "生成 shell 自动补全脚本"},
		{i18n.EN, "Generate shell autocompletion scripts"},
	}
	for _, c := range cases {
		t.Run(c.lang, func(t *testing.T) {
			i18n.SetLang(c.lang)
			t.Cleanup(func() { i18n.SetLang(i18n.ZH) })

			root := NewRootCmd()
			found := false
			for _, cmd := range root.Commands() {
				if cmd.Name() != "completion" {
					continue
				}
				found = true
				if cmd.Short != c.want {
					t.Errorf("completion short = %q, want %q", cmd.Short, c.want)
				}
			}
			if !found {
				t.Fatal("completion command not registered")
			}
		})
	}
}
