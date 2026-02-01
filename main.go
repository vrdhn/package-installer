package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	"pi/pkg/cli"
)

//go:embed pkg/cli/cli.def
var cliDSL string

func main() {
	engine, err := cli.NewEngine(cliDSL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing CLI definition: %v\n", err)
		os.Exit(1)
	}

	handler := &cli.DefaultHandler{}

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
