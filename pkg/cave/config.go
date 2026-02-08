package cave

import (
	"encoding/json"
	"fmt"
	"os"
	"pi/pkg/config"
)

// CaveSettings defines the configuration for a specific environment.
type CaveSettings struct {
	// Pkgs is a list of package requirements.
	Pkgs []config.PkgRef `json:"pkgs"`

	// Env is a map of environment variables (e.g. "DEBUG": "1").
	Env map[string]string `json:"env,omitempty"`
}

// CaveConfig represents the content of pi.cave.json.
type CaveConfig struct {
	Name      string                  `json:"name"`
	Workspace config.HostPath         `json:"workspace"`
	Home      string                  `json:"home"`
	Variants  map[string]CaveSettings `json:"variants"`
}

// CaveEntry represents an entry in the global caves.json.
type CaveEntry struct {
	Name      string          `json:"name"`
	Workspace config.HostPath `json:"workspace"`
}

// Registry represents the content of $XDG_CONFIG_DIR/pi/cave.json.
type Registry struct {
	Caves []CaveEntry `json:"caves"`
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
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &cfg, nil
}

// Validate ensures the configuration is valid.
func (c *CaveConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("missing 'name'")
	}
	if c.Workspace == "" {
		return fmt.Errorf("missing 'workspace'")
	}
	return nil
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
		Pkgs: append([]config.PkgRef(nil), base.Pkgs...),
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
