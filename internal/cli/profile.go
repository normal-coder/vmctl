package cli

import (
	"fmt"
	"strings"

	"gitee.com/normalcoder/vmctl/internal/config"
	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/i18n"
)

// mergeInput is the CLI side of profile resolution: values plus a flag
// telling whether the user set them explicitly (pflag Changed). Only
// explicit flags win over profile fields.
type mergeInput struct {
	// Profile is the --profile value ("" = not given).
	Profile string
	// Backend / VMRunPath mirror the persistent flags; the *Set
	// booleans mark them as explicitly passed on the command line.
	Backend    string
	BackendSet bool
	VMRunPath  string
	VMRunSet   bool
}

// settings is the merged result of CLI flags, profile fields and
// built-in defaults (CLI > profile > default).
type settings struct {
	Backend   string
	VMRunPath string
	// Endpoint, User, Password and Insecure are consumed once the
	// vsphere backend lands; they are resolved here so profile
	// behavior is fully covered by tests today.
	Endpoint string
	User     string
	Password string
	Insecure bool
}

// resolveSettings picks the profile (--profile > config default) and
// merges its fields under any explicitly set CLI flags. Pure function
// so the priority matrix is testable without cobra or a real file.
func resolveSettings(cfg *config.Config, in mergeInput) (settings, error) {
	name := in.Profile
	if name == "" {
		name = cfg.Default
	}

	var p config.Profile
	if name != "" {
		var ok bool
		if p, ok = cfg.Lookup(name); !ok {
			if names := cfg.Names(); len(names) > 0 {
				return settings{}, fmt.Errorf(i18n.T("err.config.profile"), name, strings.Join(names, ", "))
			}
			return settings{}, fmt.Errorf(i18n.T("err.config.profileNoList"), name)
		}
	}

	s := settings{
		Backend:   p.Backend,
		VMRunPath: p.VMRunPath,
		Endpoint:  p.Endpoint,
		User:      p.User,
		Insecure:  p.Insecure,
	}
	if s.Backend == "" {
		s.Backend = "vmrun" // built-in default
	}
	if in.BackendSet {
		s.Backend = in.Backend
	}
	if in.VMRunSet {
		s.VMRunPath = in.VMRunPath
	}
	if name != "" {
		pw, err := p.Secret()
		if err != nil {
			return settings{}, err
		}
		s.Password = pw
	}
	return s, nil
}

// openDriver loads the configuration, resolves the effective settings
// and constructs the selected backend.
func openDriver() (driver.Driver, error) {
	path, mustExist := config.Path()
	cfg, err := config.Load(path, mustExist)
	if err != nil {
		return nil, err
	}

	in := mergeInput{Profile: flagProfile, Backend: flagBackend, VMRunPath: flagVMRun}
	if rootFlags != nil {
		in.BackendSet = rootFlags.Changed("backend")
		in.VMRunSet = rootFlags.Changed("vmrun")
	}
	s, err := resolveSettings(cfg, in)
	if err != nil {
		return nil, err
	}
	return driver.Open(s.Backend, driver.Options{
		VMRunPath: s.VMRunPath,
		Endpoint:  s.Endpoint,
		User:      s.User,
		Password:  s.Password,
		Insecure:  s.Insecure,
	})
}
