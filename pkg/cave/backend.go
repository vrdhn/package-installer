package cave

import (
	"context"
	"os/exec"
	sysconfig "pi/pkg/config"
	"pi/pkg/pkgs"
)

// Backend is the interface for sandbox execution.
type Backend interface {
	// ResolveLaunch prepares a command to be executed inside the sandbox.
	ResolveLaunch(ctx context.Context, cfg sysconfig.ReadOnly, c *Cave, settings *CaveSettings, prep *pkgs.Result, command []string) (*exec.Cmd, error)
}
