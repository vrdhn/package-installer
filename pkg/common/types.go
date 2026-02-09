// Package common provides shared types used across the pi tool.
package common

import (
	"github.com/google/uuid"
)

// ExecutionResult represents the outcome of a pi operation.
type ExecutionResult struct {
	// ExitCode is the status code to return when not launching a cave.
	ExitCode int

	// Sandbox contains the configuration for the sandbox wrapper (e.g. bubblewrap).
	// If this is not nil, it indicates that a process should be executed.
	Sandbox *SandboxConfig
}

// SandboxConfig holds the metadata for wrapping a command in a sandbox.
type SandboxConfig struct {
	Cwd  string
	Exe  string
	Args []string
	Env  []string

	Binds     []SandboxBind
	Flags     []string
	UnsetEnvs []string
}

// SandboxBind represents a filesystem mount or virtual filesystem in the sandbox.
type SandboxBind struct {
	Source string
	Target string
	Type   string // e.g., "--bind", "--ro-bind", "--proc", "--tmpfs"
}

// Cave represents an active sandbox context and its configuration.
type Cave struct {
	ID        string
	Workspace HostPath
	HomePath  HostPath
	Variant   string
	Config    *CaveConfig
}

// CaveSettings defines the environment configuration for a sandbox.
type CaveSettings struct {
	Pkgs []PkgRef          `json:"pkgs"`
	Env  map[string]string `json:"env,omitempty"`
}

// CaveConfig represents the structure of the project-local 'pi.cave.json' file.
type CaveConfig struct {
	Name      string                  `json:"name"`
	Workspace HostPath                `json:"workspace"`
	Home      string                  `json:"home"`
	Variants  map[string]CaveSettings `json:"variants"`
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

// Symlink represents a symlink that should be created in the cave.
type Symlink struct {
	Source string // Path on the host
	Target string // Relative path in the cave
}

// PreparationResult contains the outcome of package preparation.
type PreparationResult struct {
	Symlinks []Symlink
	Env      map[string]string
	PkgDir   string
	CacheDir string
}

// HostPath represents a path on the host filesystem.
type HostPath = string

// PkgRef represents a package reference string, typically in the format "name@version".
type PkgRef = string

// PackageDefinition describes a specific build of a package for a particular platform.
type PackageDefinition struct {
	RepoUUID      uuid.UUID         `json:"repo_uuid,omitempty"`
	Name          string            `json:"name"`
	Version       string            `json:"version"`
	ReleaseStatus string            `json:"release_status"`
	ReleaseDate   string            `json:"release_date"`
	OS            OSType            `json:"os"`
	Arch          ArchType          `json:"arch"`
	URL           string            `json:"url"`
	Filename      string            `json:"filename"`
	Checksum      string            `json:"checksum"`
	Env           map[string]string `json:"env,omitempty"`
	Symlinks      map[string]string `json:"symlinks,omitempty"`
}
