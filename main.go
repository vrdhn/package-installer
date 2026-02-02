package main

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"pi/pkg/cave"
	"pi/pkg/cli"
	"pi/pkg/config"
	"pi/pkg/display"
	"pi/pkg/pkgs"
	"pi/pkg/repository"
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
	disp := display.NewConsole()
	defer disp.Close()

	// Simple global flag check before engine parsing to enable early logs
	for _, arg := range args {
		if arg == "--verbose" || arg == "-v" {
			disp.SetVerbose(true)
			break
		}
	}

	sysCfg, err := config.Init()
	if err != nil {
		return nil, fmt.Errorf("error initializing config: %w", err)
	}

	engine, err := cli.NewEngine(cli.DefaultDSL)
	if err != nil {
		return nil, fmt.Errorf("error parsing CLI definition: %w", err)
	}

	repo, err := repository.NewManager(disp)
	if err != nil {
		return nil, fmt.Errorf("error initializing repository: %w", err)
	}

	caveMgr := cave.NewManager(sysCfg)
	pkgsMgr := pkgs.NewManager(repo, disp, sysCfg)

	handler := &cli.DefaultHandler{
		Repo:    repo,
		Disp:    disp,
		CaveMgr: caveMgr,
		PkgsMgr: pkgsMgr,
		SysCfg:  sysCfg,
	}

	// Register the same handler for all paths, it internally switches
	registerAll(engine, engine.Commands, handler)

	return engine.Run(ctx, args)
}

func registerAll(e *cli.Engine, cmds []*cli.Command, h cli.Handler) {
	for _, c := range cmds {
		path := getCmdPath(c)
		e.Register(path, h)
		registerAll(e, c.Subs, h)
	}
}

func getCmdPath(c *cli.Command) string {
	if c.Parent == nil {
		return c.Name
	}
	return getCmdPath(c.Parent) + "/" + c.Name
}
