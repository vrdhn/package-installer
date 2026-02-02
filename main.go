package main

import (
	"context"
	"fmt"
	"os"

	"pi/pkg/cave"
	"pi/pkg/cli"
	"pi/pkg/config"
	"pi/pkg/display"
	"pi/pkg/repository"
)

func main() {
	disp := display.NewConsole()
	defer disp.Close()

	// Simple global flag check before engine parsing to enable early logs
	for _, arg := range os.Args {
		if arg == "--verbose" || arg == "-v" {
			disp.SetVerbose(true)
			break
		}
	}

	sysCfg, err := config.Init()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
		os.Exit(1)
	}

	engine, err := cli.NewEngine(cli.DefaultDSL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing CLI definition: %v\n", err)
		os.Exit(1)
	}

	repo, err := repository.NewManager(disp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing repository: %v\n", err)
		os.Exit(1)
	}

	caveMgr := cave.NewManager(sysCfg)

	handler := &cli.DefaultHandler{
		Repo:    repo,
		Disp:    disp,
		CaveMgr: caveMgr,
	}

	// Register the same handler for all paths, it internally switches
	registerAll(engine, engine.Commands, handler)

	ctx := context.Background()
	if err := engine.Run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
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
