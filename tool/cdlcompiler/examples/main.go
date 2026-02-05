package main

import (
	"fmt"
	"os"

	"pi/tool/cdlcompiler/examples/cli"
)

//go:generate go run .. ./cli/example.cdl cli

type ExecutionResult struct {
	ExitCode int
}

type handlerfuncs struct {
	// Some values initialized when Parse is called.
}

func (h *handlerfuncs) Help(args []string) (ExecutionResult, error) {
	cli.PrintHelp(args)
	return ExecutionResult{ExitCode: 0}, nil
}

func (h *handlerfuncs) RunUserAdd(params *cli.UserAddParams) (ExecutionResult, error) {
	if params.Verbose {
		fmt.Println("Verbose: enabled")
	}
	fmt.Printf("Add user: %s\n", params.Name)
	return ExecutionResult{ExitCode: 0}, nil
}

func (h *handlerfuncs) RunProjectInit(params *cli.ProjectInitParams) (ExecutionResult, error) {
	if params.Verbose {
		fmt.Println("Verbose: enabled")
	}
	fmt.Printf("Init project at: %s\n", params.Path)
	return ExecutionResult{ExitCode: 0}, nil
}

func main() {
	h := &handlerfuncs{}
	action, _, err := cli.Parse[ExecutionResult](h, os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	res, err := action()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	os.Exit(res.ExitCode)
}
