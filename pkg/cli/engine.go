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
	Theme       *Theme
}

func NewEngine(dsl string) (*Engine, error) {
	e := &Engine{
		Handlers: make(map[string]Handler),
		Theme:    DefaultTheme(),
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

type ParseResult struct {
	Invocation *Invocation
	Help       bool
	HelpArgs   []string
	Error      error
}

func (e *Engine) Run(ctx context.Context, args []string) (*ExecutionResult, error) {
	res := e.Parse(args)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.Help {
		e.PrintHelp(res.HelpArgs...)
		return &ExecutionResult{ExitCode: 0}, nil
	}
	return e.Execute(ctx, res.Invocation)
}

func (e *Engine) Parse(args []string) *ParseResult {
	res := &ParseResult{
		Invocation: &Invocation{
			Args:   make(map[string]string),
			Flags:  make(map[string]any),
			Global: make(map[string]any),
		},
	}
	var remaining []string
	// Parse global flags and help
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--help" || arg == "-h" {
			res.Help = true
			continue
		}
		found := false
		for _, gf := range e.GlobalFlags {
			if arg == "--"+gf.Name || arg == "-"+gf.Short {
				if gf.Type == "bool" {
					res.Invocation.Global[gf.Name] = true
					found = true
				} else if gf.Type == "string" && i+1 < len(args) {
					res.Invocation.Global[gf.Name] = args[i+1]
					i++
					found = true
				}
			}
		}
		if !found {
			remaining = append(remaining, arg)
		}
	}

	if res.Help {
		res.HelpArgs = remaining
		return res
	}

	if len(remaining) == 0 {
		res.Help = true
		return res
	}

	inv, err := e.resolve(res.Invocation, e.Commands, remaining)
	if err != nil {
		res.Error = err
		return res
	}
	res.Invocation = inv
	return res
}

func (e *Engine) Execute(ctx context.Context, inv *Invocation) (*ExecutionResult, error) {
	path := getCmdPath(inv.Command)
	if h, ok := e.Handlers[path]; ok {
		return h.Execute(ctx, inv)
	}
	return nil, fmt.Errorf("no handler registered for command: %s", path)
}

func (e *Engine) resolve(inv *Invocation, cmds []*Command, args []string) (*Invocation, error) {
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
			// This is a hack because our parser returns help requested
			return nil, fmt.Errorf("help requested")
		}
		currArgs := args[1:]
		// If help requested for this command
		if len(currArgs) > 0 && (currArgs[0] == "--help" || currArgs[0] == "-h") {
			return nil, fmt.Errorf("help requested for %s", cmd.Name)
		}
		// Recurse to subcommands if possible
		if len(currArgs) > 0 && len(cmd.Subs) > 0 {
			subInv, err := e.resolve(inv, cmd.Subs, currArgs)
			if err == nil {
				return subInv, nil
			}
			// Propagate error unless it's "unknown command"
			if !strings.HasPrefix(err.Error(), "unknown command") {
				return nil, err
			}
		}
		// If no subcommands matched but command has subcommands, it's an error in this new flow
		// because we want precise parsing. Actually, if we want help, we should handle it.
		if len(cmd.Subs) > 0 {
			return nil, fmt.Errorf("unknown command: %s %s", cmd.Name, currArgs[0])
		}
		inv.Command = cmd
		if err := e.parseParams(inv, cmd, currArgs); err != nil {
			return nil, err
		}
		return inv, nil
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
			if err := e.parseParams(inv, s, args[1:]); err != nil {
				return nil, err
			}
			return inv, nil
		}
	}
	return nil, fmt.Errorf("unknown command: %s", word)
}
func (e *Engine) parseParams(inv *Invocation, cmd *Command, args []string) error {
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

	// Check for missing required arguments
	if argIdx < len(cmd.Args) {
		return fmt.Errorf("argument %s is missing", cmd.Args[argIdx].Name)
	}
	return nil
}
func (e *Engine) PrintHelp(args ...string) {
	t := e.Theme
	if len(args) > 0 {
		subject := args[0]
		// Try topic
		for _, topic := range e.Topics {
			if topic.Name == subject || strings.HasPrefix(topic.Name, subject) {
				e.PrintTopicHelp(topic)
				return
			}
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
		if found != nil {
			e.PrintCommandHelp(found)
			return
		}
	}
	fmt.Printf("%s\n", t.Styled(t.Cyan.Copy().Bold(true), "pi - Universal Package Installer"))
	fmt.Printf("\n%s\n", t.Styled(t.Bold, "Usage:"))
	fmt.Printf("  pi %s\n", t.Styled(t.Yellow, "[flags] <command>"))
	fmt.Printf("\n%s\n", t.Styled(t.Bold, "Global Flags:"))
	fmt.Printf("  %-12s %s\n", t.Styled(t.Cyan, "--help, -h"), t.Styled(t.Dim, "Show help [command | topic]"))
	for _, f := range e.GlobalFlags {
		short := ""
		if f.Short != "" {
			short = ", -" + f.Short
		}
		fmt.Printf("  %-12s %s\n", t.Styled(t.Cyan, "--"+f.Name+short), t.Styled(t.Dim, f.Desc))
	}
	// Categorize commands
	categories := []struct {
		name string
		icon string
		cmds []string
	}{
		{"PACKAGE", t.IconPkg, []string{"pkg"}},
		{"CAVE", t.IconCave, []string{"cave"}},
		{"DISK", t.IconDisk, []string{"disk"}},
		{"REMOTE", t.IconWorld, []string{"remote"}},
	}
	shown := make(map[string]bool)
	fmt.Println() // Space before commands
	for _, cat := range categories {
		for _, name := range cat.cmds {
			for _, c := range e.Commands {
				if c.Name == name {
					e.printCommandTree(c, "", true, cat.icon)
					fmt.Println() // Newline after each top-level category
					shown[c.Name] = true
				}
			}
		}
	}
	// Show remaining commands under MISC
	var misc []*Command
	for _, c := range e.Commands {
		if !shown[c.Name] && c.Name != "help" {
			misc = append(misc, c)
		}
	}
	if len(misc) > 0 {
		fmt.Printf("%s %s\n", t.Bullet, t.Styled(t.Bold, "MISC"))
		for i, c := range misc {
			e.printCommandTree(c, "", i == len(misc)-1, "")
		}
		fmt.Println()
	}
	fmt.Printf("%s %s\n", t.IconHelp, t.Styled(t.Bold, "Topics:"))
	for _, topic := range e.Topics {
		name := t.Styled(t.Cyan, topic.Name)
		padding := e.getPadding(topic.Name, 20)
		fmt.Printf("  %s %s %s\n", name, padding, t.Styled(t.Dim, topic.Desc))
	}
	fmt.Printf("\nType '%s' for more details.\n", t.Styled(t.Yellow, "pi help <command>"))
}
func (e *Engine) getPadding(name string, target int) string {
	t := e.Theme
	dots := target - len(name)
	if dots < 2 {
		dots = 2
	}
	return t.Styled(t.Dim, strings.Repeat(".", dots))
}
func (e *Engine) printCommandTree(c *Command, indent string, isLast bool, icon string) {
	t := e.Theme
	prefix := t.BoxTree
	if isLast {
		prefix = t.BoxLast
	}
	namePart := indent + prefix + " "
	if icon != "" {
		namePart += icon + " "
	}
	namePart += t.Styled(t.Cyan, c.Name)
	// Calculate padding based on visual length
	visualLen := len(indent) + 4 // 3 for box prefix + 1 for space
	if icon != "" {
		visualLen += 3 // 2 for icon (assumed) + 1 for space
	}
	visualLen += len(c.Name)
	padding := e.getPadding(strings.Repeat(" ", visualLen), 30)
	line := fmt.Sprintf("%s %s %s", namePart, padding, t.Styled(t.Dim, c.Desc))
	fmt.Println(line)
	newIndent := indent
	if isLast {
		newIndent += "    "
	} else {
		newIndent += t.BoxItem + " "
	}
	for i, s := range c.Subs {
		e.printCommandTree(s, newIndent, i == len(c.Subs)-1, "")
	}
}
func (e *Engine) PrintCommandHelp(c *Command) {
	t := e.Theme
	fmt.Printf("\n%s %s\n", t.Styled(t.Bold, "Command:"), t.Styled(t.Cyan, getCmdPath(c)))
	fmt.Printf("%s %s\n", t.Styled(t.Bold, "Description:"), t.Styled(t.Dim, c.Desc))
	fmt.Println()
	if len(c.Subs) > 0 {
		fmt.Printf("%s\n", t.Styled(t.Bold, "Subcommands:"))
		for i, s := range c.Subs {
			prefix := t.BoxTree
			if i == len(c.Subs)-1 {
				prefix = t.BoxLast
			}
			fmt.Printf("  %s %-12s %s\n", prefix, t.Styled(t.Cyan, s.Name), t.Styled(t.Dim, s.Desc))
		}
		fmt.Println()
	}
	if len(c.Args) > 0 {
		fmt.Printf("%s\n", t.Styled(t.Bold, "Arguments:"))
		for _, a := range c.Args {
			fmt.Printf("  %-15s %s\n", t.Styled(t.Yellow, "<"+a.Name+">"), t.Styled(t.Dim, a.Desc))
		}
		fmt.Println()
	}
	if len(c.Flags) > 0 {
		fmt.Printf("%s\n", t.Styled(t.Bold, "Flags:"))
		for _, f := range c.Flags {
			short := ""
			if f.Short != "" {
				short = ", -" + f.Short
			}
			fmt.Printf("  %-15s %s\n", t.Styled(t.Cyan, "--"+f.Name+short), t.Styled(t.Dim, f.Desc))
		}
		fmt.Println()
	}
	if len(c.Examples) > 0 {
		fmt.Printf("%s\n", t.Styled(t.Bold, "Examples:"))
		for _, ex := range c.Examples {
			fmt.Printf("  %s %s\n", t.Styled(t.Green, "$"), ex)
		}
		fmt.Println()
	}
}
func (e *Engine) PrintTopicHelp(topic *Topic) {
	t := e.Theme
	fmt.Printf("\n%s %s\n", t.Styled(t.Bold, "Topic:"), t.Styled(t.Cyan, topic.Name))
	fmt.Printf("%s %s\n", t.Styled(t.Bold, "Description:"), t.Styled(t.Dim, topic.Desc))
	fmt.Println()
	fmt.Printf("%s\n\n", topic.Text)
}
func getCmdPath(c *Command) string {
	if c.Parent == nil {
		return c.Name
	}
	return getCmdPath(c.Parent) + "/" + c.Name
}
