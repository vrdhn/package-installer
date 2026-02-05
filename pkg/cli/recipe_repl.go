package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"pi/pkg/config"
	"pi/pkg/pkgs"
	"pi/pkg/recipe"
	"pi/pkg/resolver"
	"sort"
	"strings"
)

const (
	replPrompt        = "recipe> "
	replListLimit     = 20
	replMismatchLimit = 5
)

type recipeRepl struct {
	in   *bufio.Reader
	out  io.Writer
	err  io.Writer
	cfg  config.ReadOnly
	path string

	name     string
	source   string
	patterns []string
	legacy   bool
}

func runRecipeRepl(ctx context.Context, m *Managers, args *recipeReplArgs) (*ExecutionResult, error) {
	if args.File == "" {
		return nil, fmt.Errorf("recipe file required")
	}

	repl := &recipeRepl{
		in:   bufio.NewReader(os.Stdin),
		out:  os.Stdout,
		err:  os.Stderr,
		cfg:  m.SysCfg,
		path: args.File,
	}

	if err := repl.Run(ctx); err != nil {
		return nil, err
	}
	return &ExecutionResult{ExitCode: 0}, nil
}

func (r *recipeRepl) Run(ctx context.Context) error {
	if err := r.reload(); err != nil {
		return err
	}

	fmt.Fprintf(r.out, "Recipe REPL: %s\n", r.path)
	r.printSummary()
	r.printHelp()

	for {
		if _, err := fmt.Fprint(r.out, replPrompt); err != nil {
			return err
		}
		line, err := r.in.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if err := r.handleLine(ctx, line); err != nil {
			if err == io.EOF {
				return nil
			}
			fmt.Fprintf(r.err, "Error: %v\n", err)
		}
	}
}

func (r *recipeRepl) reload() error {
	absPath, err := filepath.Abs(r.path)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return err
	}
	r.path = absPath
	r.source = string(data)
	r.name = strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath))
	return r.loadPatterns()
}

func (r *recipeRepl) loadPatterns() error {
	sr, err := recipe.NewStarlarkRecipe(r.name, r.source, nil)
	if err != nil {
		return err
	}
	patterns, legacy, err := sr.Registry(r.cfg)
	if err != nil {
		return err
	}
	r.patterns = patterns
	r.legacy = legacy
	return nil
}

func (r *recipeRepl) handleLine(ctx context.Context, line string) error {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil
	}
	cmd := strings.ToLower(fields[0])

	switch cmd {
	case "help", "?":
		r.printHelp()
		return nil
	case "show", "patterns":
		r.printPatterns()
		return nil
	case "reload":
		if err := r.reload(); err != nil {
			return err
		}
		r.printSummary()
		return nil
	case "exit", "quit":
		return io.EOF
	case "run":
		if len(fields) < 2 {
			return fmt.Errorf("usage: run <pkg>")
		}
		return r.runRecipe(ctx, fields[1], "")
	case "run-regex":
		if len(fields) < 3 {
			return fmt.Errorf("usage: run-regex <regex> <pkg>")
		}
		return r.runRecipe(ctx, fields[2], fields[1])
	default:
		return fmt.Errorf("unknown command: %s (try 'help')", cmd)
	}
}

func (r *recipeRepl) runRecipe(ctx context.Context, pkgStr string, regexOverride string) error {
	p, err := pkgs.Parse(pkgStr)
	if err != nil {
		return err
	}
	fullName := p.Name
	if p.Ecosystem != "" {
		fullName = p.Ecosystem + ":" + p.Name
	}

	sr, err := recipe.NewStarlarkRecipe(r.name, r.source, func(msg string) {
		fmt.Fprintf(r.out, "[starlark] %s\n", msg)
	})
	if err != nil {
		return err
	}

	patterns, legacy, err := sr.Registry(r.cfg)
	if err != nil {
		return err
	}

	if legacy {
		fmt.Fprintln(r.out, "Legacy recipe detected (no regex registry).")
		return r.executeLegacy(ctx, sr, p, fullName)
	}

	regex, err := selectRegex(patterns, fullName, regexOverride)
	if err != nil {
		return err
	}

	selected := recipe.NewSelectedRecipe(sr, regex)
	task := newReplTask(fmt.Sprintf("%s (%s)", r.name, fullName), r.out)
	pkgs, err := resolver.List(ctx, r.cfg, selected, p.Ecosystem, p.Name, p.Version, task)
	if err != nil {
		return err
	}

	r.printRunSummary(fullName, p.Name, p.Version, regex, pkgs)
	return nil
}

func (r *recipeRepl) executeLegacy(ctx context.Context, sr *recipe.StarlarkRecipe, p *pkgs.Package, fullName string) error {
	task := newReplTask(fmt.Sprintf("%s (%s)", r.name, fullName), r.out)
	pkgs, err := resolver.List(ctx, r.cfg, sr, p.Ecosystem, p.Name, p.Version, task)
	if err != nil {
		return err
	}
	r.printRunSummary(fullName, p.Name, p.Version, "(legacy)", pkgs)
	return nil
}

func selectRegex(patterns []string, fullName string, override string) (string, error) {
	if override != "" {
		for _, p := range patterns {
			if p == override {
				return override, nil
			}
		}
		return "", fmt.Errorf("regex not found in registry: %s", override)
	}

	var matches []string
	for _, pattern := range patterns {
		re, err := recipe.CompileAnchored(pattern)
		if err != nil {
			return "", fmt.Errorf("invalid regex '%s': %w", pattern, err)
		}
		if re.MatchString(fullName) {
			matches = append(matches, pattern)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no regex matched %s", fullName)
	}
	if len(matches) > 1 {
		sort.Strings(matches)
		return "", fmt.Errorf("multiple regexes matched %s: %s (use run-regex)", fullName, strings.Join(matches, ", "))
	}
	return matches[0], nil
}

func (r *recipeRepl) printSummary() {
	fmt.Fprintf(r.out, "Recipe: %s\n", r.name)
	if r.legacy {
		fmt.Fprintln(r.out, "Registry: legacy (no regex patterns)")
		return
	}
	fmt.Fprintf(r.out, "Registry patterns: %d\n", len(r.patterns))
}

func (r *recipeRepl) printPatterns() {
	if r.legacy {
		fmt.Fprintln(r.out, "Legacy recipe (no regex patterns).")
		return
	}
	if len(r.patterns) == 0 {
		fmt.Fprintln(r.out, "No regex patterns registered.")
		return
	}
	fmt.Fprintln(r.out, "Regex patterns:")
	for _, p := range r.patterns {
		fmt.Fprintf(r.out, "  %s\n", p)
	}
}

func (r *recipeRepl) printHelp() {
	fmt.Fprintln(r.out, "Commands:")
	fmt.Fprintln(r.out, "  show | patterns         List regex patterns")
	fmt.Fprintln(r.out, "  run <pkg>               Execute matching handler")
	fmt.Fprintln(r.out, "  run-regex <regex> <pkg> Execute a specific regex")
	fmt.Fprintln(r.out, "  reload                  Reload recipe file")
	fmt.Fprintln(r.out, "  exit | quit             Exit the REPL")
}

func (r *recipeRepl) printRunSummary(fullName string, queryName string, version string, regex string, pkgs []recipe.PackageDefinition) {
	stats := summarizeRecipeMatches(fullName, queryName, regex, pkgs)

	fmt.Fprintln(r.out, "")
	fmt.Fprintf(r.out, "Query: %s (version query: %s)\n", fullName, version)
	fmt.Fprintf(r.out, "Regex: %s\n", regex)
	fmt.Fprintf(r.out, "Regex matches query: %t\n", stats.RegexMatchesQuery)
	fmt.Fprintf(r.out, "Versions returned: %d\n", stats.Total)
	fmt.Fprintf(r.out, "Name matches query: %d/%d\n", stats.NameMatches, stats.Total)
	fmt.Fprintf(r.out, "Name matches regex: %d/%d\n", stats.RegexMatchesName, stats.Total)

	if len(stats.NameMismatches) > 0 {
		fmt.Fprintln(r.out, "Name mismatches (sample):")
		printPackageSample(r.out, stats.NameMismatches, replMismatchLimit)
	}
	if len(stats.RegexMismatches) > 0 {
		fmt.Fprintln(r.out, "Regex mismatches (sample):")
		printPackageSample(r.out, stats.RegexMismatches, replMismatchLimit)
	}

	if len(pkgs) > 0 {
		fmt.Fprintln(r.out, "Versions (sample):")
		printPackageSample(r.out, pkgs, replListLimit)
	}
	fmt.Fprintln(r.out, "")
}

type recipeReplStats struct {
	Total             int
	NameMatches       int
	RegexMatchesName  int
	RegexMatchesQuery bool
	NameMismatches    []recipe.PackageDefinition
	RegexMismatches   []recipe.PackageDefinition
}

func summarizeRecipeMatches(fullName string, queryName string, regex string, pkgs []recipe.PackageDefinition) recipeReplStats {
	stats := recipeReplStats{Total: len(pkgs)}
	re, err := recipe.CompileAnchored(regex)
	if err == nil {
		stats.RegexMatchesQuery = re.MatchString(fullName)
	}

	for _, p := range pkgs {
		if p.Name == queryName {
			stats.NameMatches++
		} else {
			stats.NameMismatches = append(stats.NameMismatches, p)
		}
		if err == nil && re.MatchString(p.Name) {
			stats.RegexMatchesName++
		} else {
			stats.RegexMismatches = append(stats.RegexMismatches, p)
		}
	}
	return stats
}

func printPackageSample(w io.Writer, pkgs []recipe.PackageDefinition, limit int) {
	if limit <= 0 {
		limit = len(pkgs)
	}
	if len(pkgs) < limit {
		limit = len(pkgs)
	}
	fmt.Fprintf(w, "%-20s %-15s %-10s %-12s %-10s %-10s\n", "NAME", "VERSION", "STATUS", "RELEASE", "OS", "ARCH")
	for i := 0; i < limit; i++ {
		p := pkgs[i]
		status := p.ReleaseStatus
		if status == "" {
			status = "unknown"
		}
		releaseDate := p.ReleaseDate
		if releaseDate == "" {
			releaseDate = "-"
		}
		fmt.Fprintf(w, "%-20s %-15s %-10s %-12s %-10s %-10s\n", p.Name, p.Version, status, releaseDate, p.OS, p.Arch)
	}
	if len(pkgs) > limit {
		fmt.Fprintf(w, "... (%d more)\n", len(pkgs)-limit)
	}
}

type replTask struct {
	name string
	out  io.Writer
}

func newReplTask(name string, out io.Writer) *replTask {
	return &replTask{name: name, out: out}
}

func (t *replTask) Log(msg string) {
	fmt.Fprintf(t.out, "[%s] %s\n", t.name, msg)
}

func (t *replTask) SetStage(name string, target string) {
	if name == "" {
		return
	}
	if target != "" {
		fmt.Fprintf(t.out, "[%s] %s: %s\n", t.name, name, target)
	} else {
		fmt.Fprintf(t.out, "[%s] %s\n", t.name, name)
	}
}

func (t *replTask) Progress(percent int, message string) {
	if message == "" {
		return
	}
	fmt.Fprintf(t.out, "[%s] %d%% %s\n", t.name, percent, message)
}

func (t *replTask) Done() {}
