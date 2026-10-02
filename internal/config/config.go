// Package config loads vmctl's profile configuration.
//
// The file lives at os.UserConfigDir()/vmctl/config.yaml, overridable
// via VMCTL_CONFIG. A missing file at the default location is fine
// (behaves like having no config at all); a missing file at an
// explicit VMCTL_CONFIG path is an error so a typo cannot silently
// fall back to defaults.
package config

import (
	"os"
	"path/filepath"
	"sort"
)

// Profile is one named environment: which backend to use and how to
// reach it. Fields a backend ignores (e.g. Endpoint for vmrun) are
// simply not consulted.
type Profile struct {
	// Backend selects the driver ("vmrun", "vsphere"). Empty = vmrun.
	Backend string `yaml:"backend"`
	// Endpoint is the vCenter/ESXi URL (vsphere).
	Endpoint string `yaml:"endpoint"`
	// User is the vSphere login name (vsphere).
	User string `yaml:"user"`
	// Password is the credential. Takes precedence over PasswordEnv.
	Password string `yaml:"password"`
	// PasswordEnv names an environment variable holding the
	// credential, for keeping secrets out of the file.
	PasswordEnv string `yaml:"password_env"`
	// Insecure skips TLS certificate verification (vsphere).
	Insecure bool `yaml:"insecure"`
	// VMRunPath overrides auto-detection of the vmrun binary.
	VMRunPath string `yaml:"vmrun_path"`
}

// Config is the whole configuration file.
type Config struct {
	// Default names the profile used when --profile is not given.
	Default  string             `yaml:"default"`
	Profiles map[string]Profile `yaml:"profiles"`
}

// Path returns the config file location and whether it was set
// explicitly via VMCTL_CONFIG (which makes a missing file an error).
func Path() (path string, mustExist bool) {
	if p := os.Getenv("VMCTL_CONFIG"); p != "" {
		return p, true
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		// No home directory: behave as if no config exists.
		return "", false
	}
	return filepath.Join(dir, "vmctl", "config.yaml"), false
}

// Lookup returns the named profile.
func (c *Config) Lookup(name string) (Profile, bool) {
	p, ok := c.Profiles[name]
	return p, ok
}

// Names lists the configured profile names in sorted order, for
// "did you mean" error messages.
func (c *Config) Names() []string {
	names := make([]string, 0, len(c.Profiles))
	for n := range c.Profiles {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
