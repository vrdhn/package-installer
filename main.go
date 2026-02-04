package main

import (
	"context"
	"fmt"
	"os"
	"pi/pkg/cave"
	"pi/pkg/cli"
	"pi/pkg/config"
	"pi/pkg/disk"
	"pi/pkg/display"
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
func PiEngine(ctx context.Context, args []string) (*cli.ExecutionResult, error) {
	// 1. Parse cli.def
	cliEngine, err := cli.MakeEngine()
	if err != nil {
		return nil, fmt.Errorf("INTERNAL ERROR:  parsing CLI definition: %w", err)
	}
	cliEngine.Binder = cli.Binder

	// 2. Parse command line arguments
	pr := cliEngine.Parse(args)

	// 3. If inside cave, check restrictions
	if envCave := os.Getenv("PI_CAVENAME"); envCave != "" {
		if pr.Command != nil {
			if !pr.Command.SafeInCave {
				return nil, fmt.Errorf("already in cave %s", envCave)
			}
		}
	}

	// 4. Initialize console, setup verbosity, theme etc.
	disp := display.NewConsole()
	defer disp.Close()

	if pr.GlobalFlags != nil && pr.GlobalFlags.Verbose {
		disp.SetVerbose(true)
	}

	// 5. Generate any errors etc for the command line parsing
	if pr.Error != nil {
		return nil, pr.Error
	}
	if pr.Help {
		cliEngine.PrintHelp(pr.HelpArgs...)
		return &cli.ExecutionResult{ExitCode: 0}, nil
	}

	// 6. Execute commands
	sysCfg, err := config.Init()
	if err != nil {
		return nil, fmt.Errorf("error initializing config: %w", err)
	}

	repo, err := repository.NewManager(disp)
	if err != nil {
		return nil, fmt.Errorf("error initializing repository: %w", err)
	}

	caveMgr := cave.NewManager(sysCfg)
	pkgsMgr := pkgs.NewManager(repo, disp, sysCfg)
	diskMgr := disk.NewManager(sysCfg)

	managers := &cli.Managers{
		Repo:    repo,
		Disp:    disp,
		CaveMgr: caveMgr,
		PkgsMgr: pkgsMgr,
		DiskMgr: diskMgr,
		SysCfg:  sysCfg,
	}

	if pr.Action == nil {
		return nil, fmt.Errorf("no action defined for command")
	}

	return pr.Action(ctx, managers)
}
