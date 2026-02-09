package config

import "pi/pkg/common"

type OSType = common.OSType
type ArchType = common.ArchType

const (
	OSLinux   = common.OSLinux
	OSDarwin  = common.OSDarwin
	OSWindows = common.OSWindows
	OSUnknown = common.OSUnknown
)

const (
	ArchX64     = common.ArchX64
	ArchArm64   = common.ArchArm64
	ArchUnknown = common.ArchUnknown
)

var ParseOS = common.ParseOS
var ParseArch = common.ParseArch

type PkgRef = common.PkgRef
