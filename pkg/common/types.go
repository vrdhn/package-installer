// Package common provides shared types and utilities used across the pi tool.
// It includes definitions for system types (OS, Arch), package definitions,
// and execution results used for communication between components.
package common

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ExecutionResult represents the outcome of a pi operation.
// It can signal whether a command should be executed within a "cave" (sandbox)
// or if it should simply return an exit code.
type ExecutionResult struct {
	// IsCave indicates if the result requires launching a sandboxed environment.
	IsCave bool
	// ExitCode is the status code to return when not launching a cave.
	ExitCode int

	// Cwd is the working directory for the cave process.
	Cwd string
	// Exe is the path to the executable to run within the cave.
	Exe string
	// Args contains the command-line arguments for the cave process.
	Args []string
	// Env contains environment variables to be set within the cave.
	Env []string
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
