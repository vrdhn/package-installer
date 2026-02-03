package cli

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
)

//go:embed cli.def
var DefaultDSL string

// Mutable
type Engine struct {
	GlobalFlags []*Flag
	Commands    []*Command
	Topics      []*Topic
	Handlers    map[string]Handler
}

func NewEngine(dsl string) (*Engine, error) {
	e := &Engine{
		Handlers: make(map[string]Handler),
	}
	if err := e.parseDSL(dsl); err != nil {
		return nil, err
	}
	e.Commands = append(e.Commands, &Command{
		Name: "help",
		Desc: "Show help information",
	})
	return e, nil
}

func (e *Engine) Register(cmdPath string, h Handler) {
	e.Handlers[cmdPath] = h
}

func (e *Engine) parseDSL(dsl string) error {
	p := newParser(dsl, e)
	return p.parse()
}

func (e *Engine) Run(ctx context.Context, args []string) (*ExecutionResult, error) {
	inv := &Invocation{
		Args:   make(map[string]string),
		Flags:  make(map[string]any),
		Global: make(map[string]any),
	}

	var remaining []string
	helpRequested := false

	// Parse global flags and help
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--help" || arg == "-h" {
			helpRequested = true
			continue
		}

		found := false
		for _, gf := range e.GlobalFlags {
			if arg == "--"+gf.Name || arg == "-"+gf.Short {
				if gf.Type == "bool" {
					inv.Global[gf.Name] = true
					found = true
				} else if gf.Type == "string" && i+1 < len(args) {
					inv.Global[gf.Name] = args[i+1]
					i++
					found = true
				}
			}
		}

		if !found {
			remaining = append(remaining, arg)
		}
	}

	if helpRequested {
		e.PrintHelp(remaining...)
		return &ExecutionResult{ExitCode: 0}, nil
	}

	if len(remaining) == 0 {
		e.PrintHelp()
		return &ExecutionResult{ExitCode: 0}, nil
	}

	return e.dispatch(ctx, inv, e.Commands, remaining)
}

func (e *Engine) dispatch(ctx context.Context, inv *Invocation, cmds []*Command, args []string) (*ExecutionResult, error) {
	word := args[0]

	// Command match
	var matches []*Command
	for _, c := range cmds {
		if c.Name == word {
			matches = []*Command{c}
			break
		}
		if strings.HasPrefix(c.Name, word) {
			matches = append(matches, c)
		}
	}

	if len(matches) > 1 {
		var names []string
		for _, m := range matches {
			names = append(names, m.Name)
		}
		return nil, fmt.Errorf("ambiguous command: %s (candidates: %s)", word, strings.Join(names, ", "))
	}

	if len(matches) == 1 {
		cmd := matches[0]

		// Special case for built-in help
		if cmd.Name == "help" {
			e.PrintHelp(args[1:]...)
			return &ExecutionResult{ExitCode: 0}, nil
		}

		currArgs := args[1:]

		// If help requested for this command
		if len(currArgs) > 0 && (currArgs[0] == "--help" || currArgs[0] == "-h") {
			e.PrintCommandHelp(cmd)
			return &ExecutionResult{ExitCode: 0}, nil
		}

		// Recurse to subcommands if possible
		if len(currArgs) > 0 && len(cmd.Subs) > 0 {
			res, err := e.dispatch(ctx, inv, cmd.Subs, currArgs)
			if err == nil {
				return res, nil
			}

			// If error is ambiguity, propagate it
			if strings.HasPrefix(err.Error(), "ambiguous") {
				return nil, err
			}
		}

		// If no subcommands matched but command has subcommands, show help
		if len(cmd.Subs) > 0 {
			e.PrintCommandHelp(cmd)
			return &ExecutionResult{ExitCode: 0}, nil
		}

		inv.Command = cmd
		e.parseParams(inv, cmd, currArgs)
		path := getCmdPath(cmd)
		if h, ok := e.Handlers[path]; ok {
			return h.Execute(ctx, inv)
		}
		return nil, fmt.Errorf("no handler registered for command: %s", path)
	}

	// Omitted parent support
	if len(cmds) == len(e.Commands) {
		var subMatches []*Command
		for _, c := range cmds {
			for _, s := range c.Subs {
				if s.Name == word || strings.HasPrefix(s.Name, word) {
					subMatches = append(subMatches, s)
				}
			}
		}

		if len(subMatches) > 1 {
			var names []string
			for _, m := range subMatches {
				names = append(names, getCmdPath(m))
			}
			return nil, fmt.Errorf("ambiguous command: %s (candidates: %s)", word, strings.Join(names, ", "))
		}

		if len(subMatches) == 1 {
			s := subMatches[0]
			inv.Command = s
			e.parseParams(inv, s, args[1:])
			path := getCmdPath(s)
			if h, ok := e.Handlers[path]; ok {
				return h.Execute(ctx, inv)
			}
		}
	}
	return nil, fmt.Errorf("unknown command: %s", word)
}

func (e *Engine) parseParams(inv *Invocation, cmd *Command, args []string) {
	argIdx := 0
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			found := false
			for _, f := range cmd.Flags {
				if arg == "--"+f.Name || arg == "-"+f.Short {
					if f.Type == "bool" {
						inv.Flags[f.Name] = true
						found = true
					} else if f.Type == "string" && i+1 < len(args) {
						inv.Flags[f.Name] = args[i+1]
						i++
						found = true
					}
					break
				}
			}
			if !found {
				if argIdx < len(cmd.Args) {
					inv.Args[cmd.Args[argIdx].Name] = arg
					argIdx++
				}
			}
		} else {
			if argIdx < len(cmd.Args) {
				inv.Args[cmd.Args[argIdx].Name] = arg
				argIdx++
			}
		}
	}
}

func (e *Engine) PrintHelp(args ...string) {
	if len(args) > 0 {
		subject := args[0]
		// Try topic
		for _, t := range e.Topics {
			if t.Name == subject || strings.HasPrefix(t.Name, subject) {
				e.PrintTopicHelp(t)
				return
			}
		}
		if subject == "topic" || subject == "topics" {
			fmt.Println("\nHelp Topics:")
			for _, t := range e.Topics {
				fmt.Printf("  %-15s %s\n", t.Name, t.Desc)
			}
			return
		}

		// Find command in hierarchy
		curr := e.Commands
		var found *Command
		for _, arg := range args {
			var match *Command
			for _, c := range curr {
				if c.Name == arg || strings.HasPrefix(c.Name, arg) {
					match = c
					break
				}
			}
			if match == nil {
				break
			}
			found = match
			curr = match.Subs
		}

		// Try omitted parent
		if found == nil && len(args) == 1 {
			word := args[0]
			for _, c := range e.Commands {
				for _, s := range c.Subs {
					if s.Name == word || strings.HasPrefix(s.Name, word) {
						found = s
						break
					}
				}
				if found != nil {
					break
				}
			}
		}

		if found != nil {
			e.PrintCommandHelp(found)
			return
		}
	}

	fmt.Println("pi - Universal Package Installer")
	fmt.Println("\nGlobal Flags:")
	fmt.Printf("  --%-10s -%s  %s\n", "help", "h", "Show help [command | topic]")
	for _, f := range e.GlobalFlags {
		fmt.Printf("  --%-10s -%s  %s\n", f.Name, f.Short, f.Desc)
	}
	fmt.Println("\nHelp Topics:")
	for _, t := range e.Topics {
		fmt.Printf("  %-15s %s\n", t.Name, t.Desc)
	}
	fmt.Println("\nCommands:")
	for _, c := range e.Commands {
		fmt.Printf("  %-15s %s\n", c.Name, c.Desc)
		for _, s := range c.Subs {
			fmt.Printf("    %-13s %s\n", s.Name, s.Desc)
		}
	}
}

func (e *Engine) PrintCommandHelp(c *Command) {
	fmt.Printf("Command: %s\nDescription: %s\n", getCmdPath(c), c.Desc)
	if len(c.Subs) > 0 {
		fmt.Println("\nSubcommands:")
		for _, s := range c.Subs {
			fmt.Printf("  %-15s %s\n", s.Name, s.Desc)
		}
	}
	if len(c.Args) > 0 {
		fmt.Println("\nArguments:")
		for _, a := range c.Args {
			fmt.Printf("  %-15s %s\n", a.Name, a.Desc)
		}
	}
	if len(c.Flags) > 0 {
		fmt.Println("\nFlags:")
		for _, f := range c.Flags {
			fmt.Printf("  --%-10s -%s  %s\n", f.Name, f.Short, f.Desc)
		}
	}
	if len(c.Examples) > 0 {
		fmt.Println("\nExamples:")
		for _, ex := range c.Examples {
			fmt.Printf("  %s\n", ex)
		}
	}
}

func (e *Engine) PrintTopicHelp(t *Topic) {
	fmt.Printf("Topic: %s\nDescription: %s\n\n%s\n", t.Name, t.Desc, t.Text)
}

func getCmdPath(c *Command) string {
	if c.Parent == nil {
		return c.Name
	}
	return getCmdPath(c.Parent) + "/" + c.Name
}
