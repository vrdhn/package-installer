package engine

import (
	"context"
	"fmt"
	"pi/pkg/cave"
	"pi/pkg/cdl"
	"pi/pkg/config"
	"pi/pkg/disk"
	"pi/pkg/display"
	"pi/pkg/pkgs"
	"pi/pkg/repository"
)

// Implements cdl.
type Handlers struct {
	Ctx     context.Context
	RepoMgr repository.Manager
	DispMgr display.Display
	CaveMgr cave.Manager
	PkgsMgr pkgs.Manager
	DiskMgr disk.Manager
	Config  config.Config
}

func (h *Handlers) Help(args []string) (ExecutionResult, error) {
	cdl.PrintHelp(args)
	return ExecutionResult{ExitCode: 0}, nil
}

func (h *Handlers) RunVersion(params *cdl.VersionParams) (ExecutionResult, error) {
	fmt.Println(config.BuildVersion)
	return ExecutionResult{ExitCode: 0}, nil
}

func (h *Handlers) RunSelfUpdate(params *cdl.SelfUpdateParams) (ExecutionResult, error) {
	err := h.Config.SelfUpdate()
	return ExecutionResult{ExitCode: 0}, err
}
