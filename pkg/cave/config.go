package cave

import (
	"encoding/json"
	"fmt"
	"os"
	"pi/pkg/config"
)

// CaveSettings defines the environment configuration for a sandbox.
// It includes package requirements and environment variables.
type CaveSettings struct {
	// Pkgs is a list of package references (e.g., "go@1.22") required for this environment.
	Pkgs []config.PkgRef `json:"pkgs"`

	// Env is a map of environment variables to set within the sandbox.
	Env map[string]string `json:"env,omitempty"`
}

// CaveConfig represents the structure of the project-local 'pi.cave.json' file.
// It defines the workspace name, isolated home location, and various environment variants.
type CaveConfig struct {
	// Name is a unique name for this workspace.
	Name string `json:"name"`
	// Workspace is the host path to the project root.
	Workspace config.HostPath `json:"workspace"`
	// Home is the name or path of the directory used as $HOME inside the cave.
	Home string `json:"home"`
	// Variants defines different environment configurations (e.g., "", "test", "dev").
	Variants map[string]CaveSettings `json:"variants"`
}

// CaveEntry is an entry in the global cave registry, used for quick lookup by name.
type CaveEntry struct {
	// Name is the unique name of the cave.
	Name string `json:"name"`
	// Workspace is the host path to the project.
	Workspace config.HostPath `json:"workspace"`
}

// Registry represents the structure of the global '$XDG_CONFIG_DIR/pi/cave.json' file.
// It maintains a list of all caves known to the system.
type Registry struct {
	// Caves is the list of registered cave entries.
	Caves []CaveEntry `json:"caves"`
}

// LoadConfig reads and parses a CaveConfig from the specified filesystem path.
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
