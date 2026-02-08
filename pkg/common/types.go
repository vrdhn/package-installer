// Package common provides shared types and utilities used across the pi tool.
// It includes definitions for system types (OS, Arch), package definitions,
// and execution results used for communication between components.
package common

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
)

// ExecutionResult represents the outcome of a pi operation.
// It can signal whether a command should be executed as a replacement of the
// current process (via syscall.Exec) or if it should simply return an exit code.
type ExecutionResult struct {
	// ExitCode is the status code to return when not launching a cave.
	ExitCode int

	// Sandbox contains the configuration for the sandbox wrapper (e.g. bubblewrap).
	// If this is not nil, it indicates that a process should be executed.
	Sandbox *SandboxConfig

	// Preparation details for the sandbox, used when Sandbox is nil but
	// we have enough info to resolve it later.
	Cave        *Cave
	Settings    *CaveSettings
	Preparation *PreparationResult
	Command     []string
}

// SandboxConfig holds the metadata for wrapping a command in a sandbox.
type SandboxConfig struct {
	// Cwd is the working directory for the cave process.
	Cwd string
	// Exe is the path to the executable to run within the cave.
	Exe string
	// Args contains the command-line arguments for the cave process.
	Args []string
	// Env contains environment variables to be set within the cave.
	Env []string

	// Binds defines the filesystem bindings for the sandbox.
	Binds []SandboxBind
	// Flags are extra arguments for the sandbox engine (e.g. --unshare-pid).
	Flags []string
	// UnsetEnvs are environment variables that should be explicitly unset.
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
	// ID is a unique identifier for the cave (usually a hash of the workspace path).
	ID string
	// Workspace is the host path to the project root.
	Workspace HostPath
	// HomePath is the host path to the isolated HOME directory for this cave.
	HomePath HostPath
	// Variant is the currently active configuration variant (e.g., "dev", "prod").
	Variant string
	// Config is the parsed pi.cave.json configuration.
	Config *CaveConfig
}

// CaveSettings defines the environment configuration for a sandbox.
// It includes package requirements and environment variables.
type CaveSettings struct {
	// Pkgs is a list of package references (e.g., "go@1.22") required for this environment.
	Pkgs []PkgRef `json:"pkgs"`

	// Env is a map of environment variables to set within the sandbox.
	Env map[string]string `json:"env,omitempty"`
}

// CaveConfig represents the structure of the project-local 'pi.cave.json' file.
// It defines the workspace name, isolated home location, and various environment variants.
type CaveConfig struct {
	// Name is a unique name for this workspace.
	Name string `json:"name"`
	// Workspace is the host path to the project root.
	Workspace HostPath `json:"workspace"`
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
	Workspace HostPath `json:"workspace"`
}

// Registry represents the structure of the global '$XDG_CONFIG_DIR/pi/cave.json' file.
// It maintains a list of all caves known to the system.
type Registry struct {
	// Caves is the list of registered cave entries.
	Caves []CaveEntry `json:"caves"`
}

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
		Pkgs: append([]PkgRef(nil), base.Pkgs...),
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

// Symlink represents a symlink that should be created in the cave.
type Symlink struct {
	Source string // Path on the host (target of the symlink)
	Target string // Relative path in the cave (the symlink itself)
}

// PreparationResult contains the outcome of package preparation.
type PreparationResult struct {
	Symlinks []Symlink
	Env      map[string]string
	PkgDir   string
	CacheDir string
}

// OSType represents a target operating system.
type OSType string

const (
	// OSLinux represents the Linux operating system.
	OSLinux OSType = "linux"
	// OSDarwin represents macOS/Darwin.
	OSDarwin OSType = "darwin"
	// OSWindows represents Microsoft Windows.
	OSWindows OSType = "windows"
	// OSUnknown is used when the operating system cannot be determined.
	OSUnknown OSType = "unknown"
)

// ArchType represents a target CPU architecture.
type ArchType string

const (
	// ArchX64 represents the x86_64/AMD64 architecture.
	ArchX64 ArchType = "x64"
	// ArchArm64 represents the AArch64/ARM64 architecture.
	ArchArm64 ArchType = "arm64"
	// ArchUnknown is used when the architecture cannot be determined.
	ArchUnknown ArchType = "unknown"
)

// HostPath represents a path on the host filesystem.
type HostPath = string

// PkgRef represents a package reference string, typically in the format "name@version".
type PkgRef = string

// ParseOS converts a string representation of an operating system into an OSType.
// It supports common aliases like "macos" or "osx" for Darwin.
func ParseOS(os string) (OSType, error) {
	switch strings.ToLower(os) {
	case "linux":
		return OSLinux, nil
	case "darwin", "macos", "osx":
		return OSDarwin, nil
	case "windows", "win":
		return OSWindows, nil
	case "unknown":
		return OSUnknown, nil
	default:
		return OSUnknown, fmt.Errorf("unsupported operating system: %s", os)
	}
}

// ParseArch converts a string representation of a CPU architecture into an ArchType.
// It supports common aliases like "amd64" for x64.
func ParseArch(arch string) (ArchType, error) {
	switch strings.ToLower(arch) {
	case "amd64", "x64", "x86_64":
		return ArchX64, nil
	case "arm64", "aarch64":
		return ArchArm64, nil
	case "unknown":
		return ArchUnknown, nil
	default:
		return ArchUnknown, fmt.Errorf("unsupported architecture: %s", arch)
	}
}

// String returns the string representation of the OSType.
func (o OSType) String() string {
	return string(o)
}

// String returns the string representation of the ArchType.
func (a ArchType) String() string {
	return string(a)
}

// PackageDefinition describes a specific build of a package for a particular platform.
// It contains metadata about the release, download information, and installation instructions.
type PackageDefinition struct {
	// RepoUUID is the unique identifier of the repository providing this package.
	RepoUUID uuid.UUID `json:"repo_uuid,omitempty"`
	// Name is the canonical name of the package.
	Name string `json:"name"`
	// Version is the version string of the package.
	Version string `json:"version"`
	// ReleaseStatus indicates the stability of the release (e.g., "stable", "lts").
	ReleaseStatus string `json:"release_status"`
	// ReleaseDate is when the package was released.
	ReleaseDate string `json:"release_date"`
	// OS is the target operating system for this build.
	OS OSType `json:"os"`
	// Arch is the target architecture for this build.
	Arch ArchType `json:"arch"`
	// URL is the download URL for the package archive.
	URL string `json:"url"`
	// Filename is the suggested name for the downloaded archive.
	Filename string `json:"filename"`
	// Checksum is the SHA256 checksum of the archive.
	Checksum string `json:"checksum"`
	// Env contains environment variables that should be set when using this package.
	Env map[string]string `json:"env,omitempty"`
	// Symlinks defines binary symlinks to be created in the cave (e.g., {"bin/node": "bin/node"}).
	Symlinks map[string]string `json:"symlinks,omitempty"`
}
