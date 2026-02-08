// Package engine implements the core command handlers for the pi CLI.
// It orchestrates the various managers (repo, cave, pkgs, disk) to fulfill
// user requests initiated via the command line.
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

// Handlers implements the cdl.Handlers interface, providing the logic for all pi CLI commands.
type Handlers struct {
	// Ctx is the global execution context.
	Ctx context.Context
	// RepoMgr manages recipe repositories and package indexing.
	RepoMgr repository.Manager
	// DispMgr handles user feedback and progress visualization.
	DispMgr display.Display
	// CaveMgr manages sandboxed project environments.
	CaveMgr cave.Manager
	// PkgsMgr handles package resolution and installation.
	PkgsMgr pkgs.Manager
	// DiskMgr provides utilities for managing local storage usage.
	DiskMgr disk.Manager
	// Config provides access to application-wide configuration and system info.
	Config config.Config
}

// Help displays help information for pi commands and topics.
func (h *Handlers) Help(args []string) (ExecutionResult, error) {
	cdl.PrintHelp(args)
	return ExecutionResult{ExitCode: 0}, nil
}

// RunVersion outputs the current build version of the pi tool.
func (h *Handlers) RunVersion(params *cdl.VersionParams) (ExecutionResult, error) {
	h.DispMgr.Print(fmt.Sprintln(config.BuildVersion))
	return ExecutionResult{ExitCode: 0}, nil
}

// RunSelfUpdate triggers an update of the pi tool to the latest available version.
func (h *Handlers) RunSelfUpdate(params *cdl.SelfUpdateParams) (ExecutionResult, error) {
	err := h.Config.SelfUpdate()
	return ExecutionResult{ExitCode: 0}, err
}
