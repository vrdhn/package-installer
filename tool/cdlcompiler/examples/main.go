package main

import (
	"fmt"
	"os"
)

//go:generate go run .. ./example.cdl main

type ExecutionResult struct {
	ExitCode int
}

type Action func() (*ExecutionResult, error)

type handlers struct{}

func (h *handlers) UserAdd(params *userAddParams) Action {
	return func() (*ExecutionResult, error) {
		if params.Verbose {
			fmt.Println("Verbose: enabled")
		}
		fmt.Printf("Add user: %s\n", params.Name)
		return &ExecutionResult{ExitCode: 0}, nil
	}
}

func (h *handlers) ProjectInit(params *projectInitParams) Action {
	return func() (*ExecutionResult, error) {
		if params.Verbose {
			fmt.Println("Verbose: enabled")
		}
		fmt.Printf("Init project at: %s\n", params.Path)
		return &ExecutionResult{ExitCode: 0}, nil
	}
}

func main() {
	action, err := Parse(&handlers{}, os.Args[1:])
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
