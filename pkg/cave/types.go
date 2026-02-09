package cave

import (
	"pi/pkg/config"
	"pi/pkg/lazyjson"
)

// manager defines the internal state for managing caves.
type manager struct {
	Config config.Config
	regMgr lazyjson.Manager[Registry]
}

// Manager is a pointer to the internal manager implementation.
type Manager = *manager

// HostPath represents a path on the host filesystem.
type HostPath = string

// CaveSettings defines the environment configuration for a sandbox.
type CaveSettings struct {
	Pkgs []string          `json:"pkgs"`
	Env  map[string]string `json:"env,omitempty"`
}

// CaveConfig represents the structure of the project-local 'pi.cave.json' file.
type CaveConfig struct {
	Name      string                  `json:"name"`
	Workspace HostPath                `json:"workspace"`
	Home      string                  `json:"home"`
	Variants  map[string]CaveSettings `json:"variants"`
}

// Cave represents an active sandbox context and its configuration.
type Cave struct {
	ID        string
	Workspace HostPath
	HomePath  HostPath
	Variant   string
	Config    *CaveConfig
}

// CaveEntry is an entry in the global cave registry.
type CaveEntry struct {
	Name      string   `json:"name"`
	Workspace HostPath `json:"workspace"`
}

// Registry represents the structure of the global cave registry file.
type Registry struct {
	Caves []CaveEntry `json:"caves"`
}
