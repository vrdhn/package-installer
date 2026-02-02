package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// CaveSettings defines the configuration for a specific environment (default or variant).
type CaveSettings struct {
	// Packages is a list of package requirements (e.g. "nodejs@20", "go@1.21")
	Packages []string `json:"packages"`

	// Env is a map of environment variables to set in the cave
	Env map[string]string `json:"env,omitempty"`
}

// CaveConfig represents the content of pi.cave.json
type CaveConfig struct {
	// Cave holds the default configuration (variant "")
	Cave CaveSettings `json:"cave"`

	// Variants defines alternative configurations
	// These override or extend the base configuration
	Variants map[string]CaveSettings `json:"variants,omitempty"`
}

// Load reads a CaveConfig from a file path.
func Load(path string) (*CaveConfig, error) {
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

// Resolve returns the settings for a specific variant, merging with the default.
// If variant is "", returns the default configuration.
func (c *CaveConfig) Resolve(variant string) (*CaveSettings, error) {
	// Start with a copy of the base config
	merged := &CaveSettings{
		Packages: append([]string(nil), c.Cave.Packages...),
		Env:      make(map[string]string),
	}
	for k, v := range c.Cave.Env {
		merged.Env[k] = v
	}

	if variant == "" {
		return merged, nil
	}

	vConfig, ok := c.Variants[variant]
	if !ok {
		return nil, fmt.Errorf("variant not found: %s", variant)
	}

	// Merge variant packages
	merged.Packages = append(merged.Packages, vConfig.Packages...)

	// Overwrite/Append env
	for k, v := range vConfig.Env {
		merged.Env[k] = v
	}

	return merged, nil
}
