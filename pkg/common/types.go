package common

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ExecutionResult is returned from operations.
type ExecutionResult struct {
	IsCave   bool
	ExitCode int

	// Cave Launch details
	Cwd  string
	Exe  string
	Args []string
	Env  []string
}

type OSType string

const (
	OSLinux   OSType = "linux"
	OSDarwin  OSType = "darwin"
	OSWindows OSType = "windows"
	OSUnknown OSType = "unknown"
)

type ArchType string

const (
	ArchX64     ArchType = "x64"
	ArchArm64   ArchType = "arm64"
	ArchUnknown ArchType = "unknown"
)

// HostPath represents a path on the host filesystem.
type HostPath = string

// PkgRef represents a package reference (e.g. "nodejs@20").
type PkgRef = string

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

func (o OSType) String() string {
	return string(o)
}

func (a ArchType) String() string {
	return string(a)
}

// PackageDefinition describes a specific build of a package.
// It is used both for discovery results and the persistent index.
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
