package config

import (
	"pi/pkg/common"
)

type OSType = common.OSType

const (
	OSLinux   OSType = common.OSLinux
	OSDarwin  OSType = common.OSDarwin
	OSWindows OSType = common.OSWindows
	OSUnknown OSType = common.OSUnknown
)

type ArchType = common.ArchType

const (
	ArchX64     ArchType = common.ArchX64
	ArchArm64   ArchType = common.ArchArm64
	ArchUnknown ArchType = common.ArchUnknown
)

// HostPath represents a path on the host filesystem.
type HostPath = common.HostPath

// PkgRef represents a package reference (e.g. "nodejs@20").
type PkgRef = common.PkgRef

func ParseOS(os string) (OSType, error) {
	return common.ParseOS(os)
}

func ParseArch(arch string) (ArchType, error) {
	return common.ParseArch(arch)
}
