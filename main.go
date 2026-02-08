// pi is a universal, workspace-based package installer that manages dependencies
// across various languages and runtimes using isolated sandboxes.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"pi/pkg/bubblewrap"
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
	if res.Sandbox != nil {
		var exe string
		var args []string
		var env []string

		// The decision to use bubblewrap is made here in main.
		if os.Getenv("PI_NO_SANDBOX") == "" {
			cmd := bubblewrap.CmdFromSandbox(res.Sandbox)
			exe = cmd.Path
			args = cmd.Args
			env = cmd.Env
		} else {
			exe = res.Sandbox.Exe
			args = append([]string{res.Sandbox.Exe}, res.Sandbox.Args...)
			env = res.Sandbox.Env
		}

		// Verify file exists and is executable
		info, err := os.Stat(exe)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Executable not found: %v\n", err)
			os.Exit(1)
		}
		if info.Mode()&0111 == 0 {
			fmt.Fprintf(os.Stderr, "Executable is not executable: %s\n", exe)
			os.Exit(1)
		}

		if err := syscall.Exec(exe, args, env); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to exec: %v\n", err)
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
