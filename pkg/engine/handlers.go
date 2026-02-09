// Package engine implements the core command handlers for the pi CLI.
package engine

import (
	"context"
	"pi/pkg/cave"
	"pi/pkg/cdl"
	"pi/pkg/common"
	"pi/pkg/config"
	"pi/pkg/disk"
	"pi/pkg/pkgs"
	"pi/pkg/repo"
)

// Handlers implements the cdl.Handlers interface, providing the logic for all pi CLI commands.
type Handlers struct {
	// Ctx is the global execution context.
	Ctx context.Context
	// RepoMgr manages recipe repositories and package indexing.
	RepoMgr repo.Manager
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
	return ExecutionResult{
		ExitCode: 0,
		Output: &common.Output{
			Message: config.BuildVersion,
		},
	}, nil
}
