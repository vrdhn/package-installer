// pi is a universal, workspace-based package installer that manages dependencies
// across various languages and runtimes using isolated sandboxes.
package main

import (
	"context"
	"fmt"
	"log/slog"
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

// main is the entry point for the pi CLI.
// It initializes logging, parses global flags, and executes the command engine.
// If the command requires a cave (sandbox), it performs the final syscall.Exec
// to transition the process into the isolated environment.
func main() {
	// Setup logging
	var gf cdl.GlobalFlags
	cdl.ProcessGlobalFlags(os.Args[1:], &gf)

	level := slog.LevelInfo
	if gf.Verbose {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

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

// PiEngine bootstraps the pi environment and executes a command.
// It initializes the configuration, command parser, and various managers,
// returning an ExecutionResult that describes the outcome or a request to launch a cave.
func PiEngine(ctx context.Context, args []string) (engine.ExecutionResult, error) {

	config, err := config.Init()
	if err != nil {
		return engine.ExecutionResult{}, fmt.Errorf("error initializing config: %w", err)
	}

	// contract: err is nil;, or both action and cmd are nil.
	action, cmd, err := cdl.Parse[engine.ExecutionResult](args)

	if action == nil || cmd == nil { // implies err != nil
		return engine.ExecutionResult{}, err
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

	repoMgr := repository.NewManager(dispMgr, config)
	caveMgr := cave.NewManager(config, dispMgr)
	pkgsMgr := pkgs.NewManager(repoMgr, dispMgr, config)
	diskMgr := disk.NewManager(config, dispMgr)

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
