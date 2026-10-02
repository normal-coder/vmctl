package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"

	"gitee.com/normalcoder/vmctl/internal/i18n"
)

// Load reads the configuration at path. A missing file only yields an
// empty config when mustExist is false (the default location); an
// explicit VMCTL_CONFIG path must exist. YAML is decoded in strict
// mode so unknown keys are reported instead of ignored.
func Load(path string, mustExist bool) (*Config, error) {
	empty := func() *Config { return &Config{Profiles: map[string]Profile{}} }

	if path == "" {
		return empty(), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !mustExist {
			return empty(), nil
		}
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf(i18n.T("err.config.notExist"), path)
		}
		return nil, fmt.Errorf(i18n.T("err.config.read"), err)
	}

	var cfg Config
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf(i18n.T("err.config.parse"), err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	return &cfg, nil
}

// Secret resolves the profile's password: an inline password wins,
// otherwise the environment variable named by PasswordEnv. Both empty
// means no password (backends may treat that as "try unauthenticated").
func (p Profile) Secret() (string, error) {
	if p.Password != "" {
		return p.Password, nil
	}
	if p.PasswordEnv == "" {
		return "", nil
	}
	if v := os.Getenv(p.PasswordEnv); v != "" {
		return v, nil
	}
	return "", fmt.Errorf(i18n.T("err.config.envEmpty"), p.PasswordEnv)
}
