package main

import (
	"context"
	"fmt"
	"os"
	"pi/pkg/cave"
	"pi/pkg/cdl"
	"pi/pkg/config"
	"pi/pkg/disk"
	"pi/pkg/display"
	"pi/pkg/engine"
	"pi/pkg/pkgs"
	"pi/pkg/repository"
	"syscall"
)

func main() {
	res, err := PiEngine(context.Background(), os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if res.IsCave {
		if err := syscall.Exec(res.Exe, res.Args, res.Env); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to exec cave: %v\n", err)
			os.Exit(1)
		}
		// syscall.Exec never returns on success
	}
	os.Exit(res.ExitCode)
}

func PiEngine(ctx context.Context, args []string) (engine.ExecutionResult, error) {
	// Initialize handlers with context (Managers populated later)
	handlers := &engine.DefaultHandlers{Ctx: ctx}

	// 1. Parse command line arguments with generics
	action, _, err := cdl.Parse[engine.ExecutionResult](handlers, args)
	if err != nil {
		return engine.ExecutionResult{}, err
	}

	// 2. Initialize console, setup verbosity, etc.
	disp := display.NewConsole()
	defer disp.Close()

	// 3. Execute commands
	sysCfg, err := config.Init()
	if err != nil {
		return engine.ExecutionResult{}, fmt.Errorf("error initializing config: %w", err)
	}

	repo, err := repository.NewManager(disp, sysCfg)
	if err != nil {
		return engine.ExecutionResult{}, fmt.Errorf("error initializing repository: %w", err)
	}

	caveMgr := cave.NewManager(sysCfg)
	pkgsMgr := pkgs.NewManager(repo, disp, sysCfg)
	diskMgr := disk.NewManager(sysCfg)

	managers := &engine.Managers{
		Repo:    repo,
		Disp:    disp,
		CaveMgr: caveMgr,
		PkgsMgr: pkgsMgr,
		DiskMgr: diskMgr,
		SysCfg:  sysCfg,
	}

	// Update handlers with managers
	handlers.Mgr = managers

	if action == nil {
		return engine.ExecutionResult{}, fmt.Errorf("no action defined for command")
	}

	return action()
}
