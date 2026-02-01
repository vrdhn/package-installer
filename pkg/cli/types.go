package cli

import (
	"context"
)

type Flag struct {
	Name  string
	Short string
	Type  string // "bool", "string"
	Desc  string
	Value any // Populated during parsing
}

type Arg struct {
	Name  string
	Type  string
	Desc  string
	Value string // Populated during parsing
}

type Command struct {
	Name     string
	Desc     string
	Args     []*Arg
	Flags    []*Flag
	Subs     []*Command
	Parent   *Command
	Examples []string
}

type Topic struct {
	Name string
	Desc string
	Text string
}

type Invocation struct {
	Command *Command
	Args    map[string]string
	Flags   map[string]any
	Global  map[string]any
}

type Handler interface {
	Execute(ctx context.Context, inv *Invocation) error
}
