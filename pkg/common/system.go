package common

import (
	"fmt"
	"strings"
)

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

// ParseOS converts a string representation of an operating system into an OSType.
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
