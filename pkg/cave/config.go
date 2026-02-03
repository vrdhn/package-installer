package cave

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	sysconfig "pi/pkg/config"
)

// CaveSettings defines the configuration for a specific environment.
type CaveSettings struct {
	// Pkgs is a list of package requirements.
	Pkgs []sysconfig.PkgRef `json:"pkgs"`

	// Env is a map of environment variables (e.g. "DEBUG": "1").
	Env map[string]string `json:"env,omitempty"`
}

// CaveConfig represents the content of pi.cave.json.
type CaveConfig struct {
	Name      string                  `json:"name"`
	Workspace sysconfig.HostPath      `json:"workspace"`
	Home      string                  `json:"home"`
	Variants  map[string]CaveSettings `json:"variants"`
}

// RegistryEntry represents an entry in the global caves.json.
type RegistryEntry struct {
	Name      string             `json:"name"`
	Workspace sysconfig.HostPath `json:"workspace"`
}

// Registry represents the content of $XDG_CONFIG_DIR/pi/caves.json.
type Registry struct {
	Caves []RegistryEntry `json:"caves"`
}

// LoadConfig reads a CaveConfig from a file path.
func LoadConfig(path string) (*CaveConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg CaveConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save writes the CaveConfig to a file path.
func (c *CaveConfig) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Resolve returns the settings for a specific variant, merging with the default ("").
func (c *CaveConfig) Resolve(variant string) (*CaveSettings, error) {
	base, ok := c.Variants[""]
	if !ok {
		base = CaveSettings{}
	}

	merged := &CaveSettings{
		Pkgs: append([]sysconfig.PkgRef(nil), base.Pkgs...),
		Env:  make(map[string]string),
	}
	for k, v := range base.Env {
		merged.Env[k] = v
	}

	if variant == "" {
		return merged, nil
	}

	vConfig, ok := c.Variants[variant]
	if !ok {
		return nil, fmt.Errorf("variant not found: %s", variant)
	}

	merged.Pkgs = append(merged.Pkgs, vConfig.Pkgs...)
	for k, v := range vConfig.Env {
		merged.Env[k] = v
	}

	return merged, nil
}

// LoadRegistry reads the global registry.
func LoadRegistry(cfg sysconfig.ReadOnly) (*Registry, error) {
	path := filepath.Join(cfg.GetConfigDir(), "caves.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Registry{Caves: []RegistryEntry{}}, nil
		}
		return nil, err
	}
	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, err
	}
	return &reg, nil
}

// SaveRegistry writes the global registry.
func (r *Registry) Save(cfg sysconfig.ReadOnly) error {
	dir := cfg.GetConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, "caves.json")
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
