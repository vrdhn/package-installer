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
	"pi/pkg/common"
	"pi/pkg/config"
	"pi/pkg/disk"
	"pi/pkg/display"
	"pi/pkg/engine"
	"pi/pkg/pkgs"
	"pi/pkg/repo"
	"syscall"
)

// main is the entry point for the pi CLI.
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
	if res.Output != nil {
		dispMgr := display.NewConsole()
		dispMgr.RenderOutput(res.Output)
	}
	if res.Sandbox != nil {
		os.Exit(runSandBox(res.Sandbox))
	}
	os.Exit(res.ExitCode)
}

func runSandBox(s *common.SandboxConfig) int {
	var exe string
	var args []string
	var env []string

	if os.Getenv("PI_NO_SANDBOX") == "" {
		cmd := bubblewrap.CmdFromSandbox(s)
		exe = cmd.Path
		args = cmd.Args
		env = cmd.Env
	} else {
		exe = s.Exe
		args = append([]string{s.Exe}, s.Args...)
		env = s.Env
	}

	info, err := os.Stat(exe)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Executable not found: %v\n", err)
		return 1
	}
	if info.Mode()&0111 == 0 {
		fmt.Fprintf(os.Stderr, "Executable is not executable: %s\n", exe)
		return 1
	}

	if err := syscall.Exec(exe, args, env); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to exec: %v\n", err)
		return 1
	}
	return 1
}

// PiEngine bootstraps the pi environment and executes a command.
func PiEngine(ctx context.Context, args []string) (engine.ExecutionResult, error) {
	config, err := config.Init()
	if err != nil {
		return engine.ExecutionResult{}, fmt.Errorf("error initializing config: %w", err)
	}

	action, cmd, err := cdl.Parse[engine.ExecutionResult](args)
	if action == nil || cmd == nil {
		return engine.ExecutionResult{}, err
	}
	if !cmd.Safe {
		if caveName, exists := os.LookupEnv("PI_CAVENAME"); exists {
			return engine.ExecutionResult{},
				fmt.Errorf("command can not be run from cave %s", caveName)
		}
	}

	repoMgr := repo.NewManager(config)
	caveMgr := cave.NewManager(config)
	pkgsMgr := pkgs.NewManager(repoMgr, config)
	diskMgr := disk.NewManager(config)

	handlers := &engine.Handlers{
		Ctx:     ctx,
		RepoMgr: repoMgr,
		CaveMgr: caveMgr,
		PkgsMgr: pkgsMgr,
		DiskMgr: diskMgr,
		Config:  config,
	}

	res, err := action(handlers)
	return res, err
}
