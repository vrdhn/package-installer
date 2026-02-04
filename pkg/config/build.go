package config

import (
	"fmt"
	"runtime"
	"time"

	"github.com/dustin/go-humanize"
)

// Build information.
// These variables are set at build time using ldflags.
var (
	BuildVersion   = "unknown"
	BuildTimestamp = "unknown"
)

// GetBuildInfo returns a formatted string with build details.
func GetBuildInfo() string {
	ts := BuildTimestamp
	if t, err := time.Parse(time.RFC3339, BuildTimestamp); err == nil {
		ts = fmt.Sprintf("%s, %s", BuildTimestamp, humanize.Time(t))
	}
	return fmt.Sprintf("pi %s (%s) %s/%s", BuildVersion, ts, runtime.GOOS, runtime.GOARCH)
}
