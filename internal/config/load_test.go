package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadMissingDefaultIsSilent(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.yaml"), false)
	if err != nil {
		t.Fatalf("missing default file should be silent, got %v", err)
	}
	if len(cfg.Profiles) != 0 {
		t.Errorf("want empty config, got %v", cfg)
	}
}

func TestLoadMissingExplicitErrors(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nope.yaml"), true)
	if err == nil {
		t.Fatal("missing explicit file must error")
	}
}

func TestLoadEmptyPath(t *testing.T) {
	cfg, err := Load("", false)
	if err != nil || cfg.Default != "" || len(cfg.Profiles) != 0 {
		t.Errorf("Load(\"\") = %v, %v", cfg, err)
	}
}

func TestLoadValid(t *testing.T) {
	path := writeFile(t, `
default: lab
profiles:
  lab:
    backend: vsphere
    endpoint: https://vcenter.example.com
    user: administrator@vsphere.local
    password_env: VC_PASS
    insecure: true
  local:
    backend: vmrun
    vmrun_path: /opt/vmware/bin/vmrun
`)
	cfg, err := Load(path, true)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Default != "lab" {
		t.Errorf("Default = %q", cfg.Default)
	}
	lab, ok := cfg.Lookup("lab")
	if !ok {
		t.Fatal("profile lab missing")
	}
	if lab.Backend != "vsphere" || lab.Endpoint == "" || !lab.Insecure || lab.PasswordEnv != "VC_PASS" {
		t.Errorf("lab profile = %+v", lab)
	}
	local, ok := cfg.Lookup("local")
	if !ok || local.VMRunPath != "/opt/vmware/bin/vmrun" {
		t.Errorf("local profile = %+v", local)
	}
}

func TestLoadUnknownFieldRejected(t *testing.T) {
	path := writeFile(t, "profiles:\n  a:\n    backendt: vmrun\n")
	if _, err := Load(path, true); err == nil {
		t.Fatal("unknown key must be rejected (KnownFields)")
	}
}

func TestLoadSyntaxError(t *testing.T) {
	path := writeFile(t, "profiles: [oops")
	if _, err := Load(path, true); err == nil {
		t.Fatal("syntax error must be reported")
	}
}

func TestLoadEmptyFile(t *testing.T) {
	cfg, err := Load(writeFile(t, ""), true)
	if err != nil {
		t.Fatalf("empty file should load as empty config: %v", err)
	}
	if cfg.Profiles == nil {
		t.Error("Profiles map must be non-nil")
	}
}

func TestNames(t *testing.T) {
	cfg := &Config{Profiles: map[string]Profile{"z": {}, "a": {}, "m": {}}}
	got := cfg.Names()
	want := []string{"a", "m", "z"}
	if len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("Names() = %v, want %v", got, want)
	}
}

func TestSecret(t *testing.T) {
	t.Run("inline password wins", func(t *testing.T) {
		t.Setenv("P_ENV", "from-env")
		got, err := Profile{Password: "inline", PasswordEnv: "P_ENV"}.Secret()
		if err != nil || got != "inline" {
			t.Errorf("Secret() = %q, %v", got, err)
		}
	})
	t.Run("env fallback", func(t *testing.T) {
		t.Setenv("P_ENV", "from-env")
		got, err := Profile{PasswordEnv: "P_ENV"}.Secret()
		if err != nil || got != "from-env" {
			t.Errorf("Secret() = %q, %v", got, err)
		}
	})
	t.Run("neither set", func(t *testing.T) {
		got, err := Profile{}.Secret()
		if err != nil || got != "" {
			t.Errorf("Secret() = %q, %v", got, err)
		}
	})
	t.Run("env missing", func(t *testing.T) {
		t.Setenv("P_ENV", "")
		_, err := Profile{PasswordEnv: "P_ENV"}.Secret()
		if err == nil {
			t.Fatal("unset password_env must error")
		}
		if !strings.Contains(err.Error(), "P_ENV") {
			t.Errorf("error should name the variable: %v", err)
		}
	})
}

func TestPath(t *testing.T) {
	t.Run("explicit", func(t *testing.T) {
		t.Setenv("VMCTL_CONFIG", "/tmp/custom.yaml")
		path, mustExist := Path()
		if path != "/tmp/custom.yaml" || !mustExist {
			t.Errorf("Path() = %q, %v", path, mustExist)
		}
	})
	t.Run("default", func(t *testing.T) {
		t.Setenv("VMCTL_CONFIG", "")
		path, mustExist := Path()
		if mustExist {
			t.Error("default path must not require existence")
		}
		if !strings.HasSuffix(path, filepath.Join("vmctl", "config.yaml")) {
			t.Errorf("unexpected default path %q", path)
		}
	})
}
