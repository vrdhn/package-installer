package cave

import (
	"context"
	"os/exec"
	"pi/pkg/cave/config"
	"pi/pkg/pkgs"
)

// Backend is the interface for sandbox execution.
type Backend interface {
	// ResolveLaunch prepares a command to be executed inside the sandbox.
	ResolveLaunch(ctx context.Context, c *Cave, settings *config.CaveSettings, symlinks []pkgs.Symlink, command []string) (*exec.Cmd, error)
}
