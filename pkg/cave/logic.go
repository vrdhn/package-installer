package cave

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadCaveConfig reads and parses a CaveConfig from the specified filesystem path.
func LoadCaveConfig(path string) (*CaveConfig, error) {
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

// Validate checks the CaveConfig for required fields.
func (c *CaveConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("missing 'name'")
	}
	if c.Workspace == "" {
		return fmt.Errorf("missing 'workspace'")
	}
	return nil
}

// Save serializes the CaveConfig and writes it to the specified path.
func (c *CaveConfig) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Resolve merges the settings for a specific variant with the default base settings (empty variant).
// Variant settings override or append to the base settings.
func (c *CaveConfig) Resolve(variant string) (*CaveSettings, error) {
	base, ok := c.Variants[""]
	if !ok {
		base = CaveSettings{}
	}

	merged := &CaveSettings{
		Pkgs: append([]string(nil), base.Pkgs...),
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
