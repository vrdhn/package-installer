package cli

import (
	"context"
	"pi/pkg/cave"
	"pi/pkg/config"
	"pi/pkg/disk"
	"pi/pkg/display"
	"pi/pkg/pkgs"
	"pi/pkg/repository"
)

// Managers holds the various system managers needed for command execution.
// Immutable.
type Managers struct {
	Repo    *repository.Manager
	Disp    display.Display
	CaveMgr *cave.Manager
	PkgsMgr *pkgs.Manager
	DiskMgr *disk.Manager
	SysCfg  config.ReadOnly
}

// Action is a lambda that executes a command using the provided managers.
// The structs for global flags, cmd flags and cmd arguments have already
// been captured in this lamda.
type Action func(ctx context.Context, m *Managers) (*ExecutionResult, error)

// Mutable
// Returned at top level, to figure out if to execute bwrap
// or just exit with error code. The curses library should have
// cleaned up when this is returned, so exec is safe.
type ExecutionResult struct {
	IsCave   bool
	ExitCode int

	// Cave Launch details
	Cwd  string
	Exe  string
	Args []string
	Env  []string
}
