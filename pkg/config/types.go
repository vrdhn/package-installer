// Package config manages application-wide settings and directory structures.
// It follows XDG specifications for storing cache, configuration, and state.
package config

import (
	"pi/pkg/common"
)

// OSType represents a target operating system.
type OSType = common.OSType

const (
	// OSLinux represents the Linux operating system.
	OSLinux OSType = common.OSLinux
	// OSDarwin represents macOS/Darwin.
	OSDarwin OSType = common.OSDarwin
	// OSWindows represents Microsoft Windows.
	OSWindows OSType = common.OSWindows
	// OSUnknown is used when the operating system cannot be determined.
	OSUnknown OSType = common.OSUnknown
)

// ArchType represents a target CPU architecture.
type ArchType = common.ArchType

const (
	// ArchX64 represents the x86_64/AMD64 architecture.
	ArchX64 ArchType = common.ArchX64
	// ArchArm64 represents the AArch64/ARM64 architecture.
	ArchArm64 ArchType = common.ArchArm64
	// ArchUnknown is used when the architecture cannot be determined.
	ArchUnknown ArchType = common.ArchUnknown
)

// HostPath represents a path on the host filesystem.
type HostPath = common.HostPath

// PkgRef represents a package reference (e.g., "nodejs@20").
type PkgRef = common.PkgRef

// ParseOS converts a string representation of an operating system into an OSType.
func ParseOS(os string) (OSType, error) {
	return common.ParseOS(os)
}

// ParseArch converts a string representation of a CPU architecture into an ArchType.
func ParseArch(arch string) (ArchType, error) {
	return common.ParseArch(arch)
}
