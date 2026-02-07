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
		// Verify file exists and is executable
		info, err := os.Stat(res.Exe)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cave executable not found: %v\n", err)
			os.Exit(1)
		}
		if info.Mode()&0111 == 0 {
			fmt.Fprintf(os.Stderr, "Cave executable is not executable: %s\n", res.Exe)
			os.Exit(1)
		}

		if err := syscall.Exec(res.Exe, res.Args, res.Env); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to exec cave: %v\n", err)
			os.Exit(1)
		}
		// syscall.Exec never returns on success
	}
	os.Exit(res.ExitCode)
}

func PiEngine(ctx context.Context, args []string) (engine.ExecutionResult, error) {

	config, err := config.Init()
	if err != nil {
		return engine.ExecutionResult{}, fmt.Errorf("error initializing config: %w", err)
	}

	action, cmd, err := cdl.Parse[engine.ExecutionResult](args)
	if err != nil {
		return engine.ExecutionResult{}, err
	}
	if action == nil {
		return engine.ExecutionResult{}, fmt.Errorf("no action defined for command")
	}
	if !cmd.Safe {
		caveName, exists := os.LookupEnv("PI_CAVENAME")
		if exists {
			return engine.ExecutionResult{},
				fmt.Errorf("command can not be run from cave %s", caveName)
		}
	}

	dispMgr := display.NewConsole()
	defer dispMgr.Close()

	repoMgr, err := repository.NewManager(dispMgr, config)
	if err != nil {
		return engine.ExecutionResult{}, fmt.Errorf("error initializing repository: %w", err)
	}

	caveMgr := cave.NewManager(config)
	pkgsMgr := pkgs.NewManager(repoMgr, dispMgr, config)
	diskMgr := disk.NewManager(config)

	handlers := &engine.Handlers{
		Ctx:     ctx,
		RepoMgr: repoMgr,
		DispMgr: dispMgr,
		CaveMgr: caveMgr,
		PkgsMgr: pkgsMgr,
		DiskMgr: diskMgr,
		Config:  config,
	}

	return action(handlers)
}
