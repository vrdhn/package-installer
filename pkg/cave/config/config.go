package config

import (
	"encoding/json"
	"os"
)

// CaveConfig represents the content of pi.cave.json
type CaveConfig struct {
	// Packages is a list of package requirements (e.g. "nodejs@20", "go@1.21")
	Packages []string `json:"packages"`

	// Env is a map of environment variables to set in the cave
	Env map[string]string `json:"env,omitempty"`

	// Variants defines alternative configurations (e.g. "test", "prod")
	// These override or extend the base configuration
	Variants map[string]*CaveConfig `json:"variants,omitempty"`
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

// MergeVariant merges a variant config into the base config.
// The variant takes precedence.
func (c *CaveConfig) MergeVariant(v *CaveConfig) *CaveConfig {
	if v == nil {
		return c
	}

	merged := &CaveConfig{
		Packages: append([]string(nil), c.Packages...),
		Env:      make(map[string]string),
		Variants: c.Variants, // Variants are not recursively merged usually, but kept from base? Or just ignored.
	}

	// Copy base env
	for k, val := range c.Env {
		merged.Env[k] = val
	}

	// Append variant packages
	merged.Packages = append(merged.Packages, v.Packages...)

	// Overwrite env
	for k, val := range v.Env {
		merged.Env[k] = val
	}

	return merged
}
