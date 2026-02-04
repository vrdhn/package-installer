package cli

import (
	"context"
)

// Immutable
type Flag struct {
	Name  string
	Short string
	Type  string // "bool", "string"
	Desc  string
	Value any // Populated during parsing
}

// Immutable
type Arg struct {
	Name  string
	Type  string
	Desc  string
	Value string // Populated during parsing
}

// Immutable
type Command struct {
	Name       string
	Desc       string
	SafeInCave bool
	Args       []*Arg
	Flags      []*Flag
	Subs       []*Command
	Parent     *Command
	Examples   []string
}

// Immutable
type Topic struct {
	Name string
	Desc string
	Text string
}

// Mutable
type Invocation struct {
	Command *Command
	Args    map[string]string
	Flags   map[string]any
	Global  map[string]any
}

// Mutable
type ExecutionResult struct {
	IsCave   bool
	ExitCode int

	// Cave Launch details
	Exe  string
	Args []string
	Env  []string
}

type Handler interface {
	Execute(ctx context.Context, inv *Invocation) (*ExecutionResult, error)
}
