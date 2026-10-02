package cli

import (
	"strings"
	"testing"

	"gitee.com/normalcoder/vmctl/internal/config"
)

func TestResolveSettingsPriority(t *testing.T) {
	cfg := &config.Config{
		Default: "lab",
		Profiles: map[string]config.Profile{
			"lab": {
				Backend:   "vsphere",
				Endpoint:  "https://vc.example.com",
				User:      "admin@vsphere.local",
				Password:  "s3cret",
				VMRunPath: "/profile/vmrun",
				VMCliPath: "/profile/vmcli",
				Insecure:  true,
			},
			"local": {Backend: "vmrun"},
		},
	}

	cases := []struct {
		name string
		in   mergeInput
		want settings
	}{
		{
			name: "profile only",
			in:   mergeInput{},
			want: settings{
				Backend: "vsphere", Endpoint: "https://vc.example.com",
				User: "admin@vsphere.local", Password: "s3cret",
				VMRunPath: "/profile/vmrun", VMCliPath: "/profile/vmcli", Insecure: true,
			},
		},
		{
			name: "explicit profile beats default",
			in:   mergeInput{Profile: "local"},
			want: settings{Backend: "vmrun"},
		},
		{
			name: "explicit backend flag beats profile",
			in:   mergeInput{Backend: "vmrun", BackendSet: true},
			// Password etc. still come from the resolved profile.
			want: settings{
				Backend: "vmrun", Endpoint: "https://vc.example.com",
				User: "admin@vsphere.local", Password: "s3cret",
				VMRunPath: "/profile/vmrun", VMCliPath: "/profile/vmcli", Insecure: true,
			},
		},
		{
			name: "backend flag without Changed does not beat profile",
			in:   mergeInput{Backend: "vmrun"}, // default value, not set
			want: settings{
				Backend: "vsphere", Endpoint: "https://vc.example.com",
				User: "admin@vsphere.local", Password: "s3cret",
				VMRunPath: "/profile/vmrun", VMCliPath: "/profile/vmcli", Insecure: true,
			},
		},
		{
			name: "explicit vmrun flag beats profile",
			in:   mergeInput{VMRunPath: "/cli/vmrun", VMRunSet: true},
			want: settings{
				Backend: "vsphere", Endpoint: "https://vc.example.com",
				User: "admin@vsphere.local", Password: "s3cret",
				VMRunPath: "/cli/vmrun", VMCliPath: "/profile/vmcli", Insecure: true,
			},
		},
		{
			name: "explicit vmcli flag beats profile",
			in:   mergeInput{VMCliPath: "/cli/vmcli", VMCliSet: true},
			want: settings{
				Backend: "vsphere", Endpoint: "https://vc.example.com",
				User: "admin@vsphere.local", Password: "s3cret",
				VMRunPath: "/profile/vmrun", VMCliPath: "/cli/vmcli", Insecure: true,
			},
		},
		{
			name: "vmcli flag without Changed does not beat profile",
			in:   mergeInput{VMCliPath: "/cli/vmcli"}, // default value, not set
			want: settings{
				Backend: "vsphere", Endpoint: "https://vc.example.com",
				User: "admin@vsphere.local", Password: "s3cret",
				VMRunPath: "/profile/vmrun", VMCliPath: "/profile/vmcli", Insecure: true,
			},
		},
		{
			name: "built-in default when no config",
			in:   mergeInput{},
			want: settings{Backend: "vmrun"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := cfg
			if c.name == "built-in default when no config" {
				src = &config.Config{}
			}
			got, err := resolveSettings(src, c.in)
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Errorf("resolveSettings() = %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestResolveSettingsPasswordEnv(t *testing.T) {
	t.Setenv("VMCTL_TEST_PW", "from-env")
	cfg := &config.Config{Profiles: map[string]config.Profile{
		"lab": {Backend: "vsphere", PasswordEnv: "VMCTL_TEST_PW"},
	}}
	got, err := resolveSettings(cfg, mergeInput{Profile: "lab"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Password != "from-env" {
		t.Errorf("Password = %q, want from-env", got.Password)
	}
}

func TestResolveSettingsMissingProfile(t *testing.T) {
	cfg := &config.Config{Profiles: map[string]config.Profile{
		"lab": {}, "local": {},
	}}
	_, err := resolveSettings(cfg, mergeInput{Profile: "nope"})
	if err == nil {
		t.Fatal("unknown --profile must error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "nope") || !strings.Contains(msg, "lab") || !strings.Contains(msg, "local") {
		t.Errorf("error should name the profile and list available ones: %v", msg)
	}
}

func TestResolveSettingsMissingProfileNoList(t *testing.T) {
	_, err := resolveSettings(&config.Config{}, mergeInput{Profile: "nope"})
	if err == nil {
		t.Fatal("must error even with no profiles configured")
	}
}

func TestResolveSettingsBrokenDefault(t *testing.T) {
	cfg := &config.Config{Default: "ghost"}
	_, err := resolveSettings(cfg, mergeInput{})
	if err == nil {
		t.Fatal("default pointing at a missing profile must error")
	}
}

func TestResolveSettingsMissingPasswordEnv(t *testing.T) {
	t.Setenv("VMCTL_TEST_PW_MISSING", "")
	cfg := &config.Config{Profiles: map[string]config.Profile{
		"lab": {PasswordEnv: "VMCTL_TEST_PW_MISSING"},
	}}
	_, err := resolveSettings(cfg, mergeInput{Profile: "lab"})
	if err == nil {
		t.Fatal("unset password_env must error")
	}
}
