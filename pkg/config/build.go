package config

import (
	"fmt"
	"runtime"
)

// Build information.
// These variables are set at build time using ldflags.
var (
	BuildVersion   = "unknown"
	BuildTimestamp = "unknown"
)

// GetBuildInfo returns a formatted string with build details.
func GetBuildInfo() string {
	return fmt.Sprintf("pi %s (%s) %s/%s", BuildVersion, BuildTimestamp, runtime.GOOS, runtime.GOARCH)
}
