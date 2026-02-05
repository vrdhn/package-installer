package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"pi/pkg/bubblewrap"
	"pi/pkg/cave"
	"pi/pkg/config"
	"pi/pkg/disk"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	infoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
)

type DefaultHandlers struct{}

func (h *DefaultHandlers) Version(ctx context.Context, m *Managers, args *versionArgs, flags *versionFlags) (*ExecutionResult, error) {
	fmt.Println(config.GetBuildInfo())
	return &ExecutionResult{ExitCode: 0}, nil
}

func (h *DefaultHandlers) PkgInstall(ctx context.Context, m *Managers, args *pkgInstallArgs, flags *pkgInstallFlags) (*ExecutionResult, error) {
	return runInstall(ctx, m, args, flags)
}

func (h *DefaultHandlers) PkgList(ctx context.Context, m *Managers, args *pkgListArgs, flags *pkgListFlags) (*ExecutionResult, error) {
	return runPkgList(ctx, m, args, flags)
}

func (h *DefaultHandlers) RecipeRepl(ctx context.Context, m *Managers, args *recipeReplArgs, flags *recipeReplFlags) (*ExecutionResult, error) {
	return runRecipeRepl(ctx, m, args)
}

func (h *DefaultHandlers) CaveInfo(ctx context.Context, m *Managers, args *caveInfoArgs, flags *caveInfoFlags) (*ExecutionResult, error) {
	return runInfo(ctx, m)
}

func (h *DefaultHandlers) CaveList(ctx context.Context, m *Managers, args *caveListArgs, flags *caveListFlags) (*ExecutionResult, error) {
	return runCaveList(ctx, m)
}

func (h *DefaultHandlers) CaveUse(ctx context.Context, m *Managers, args *caveUseArgs, flags *caveUseFlags) (*ExecutionResult, error) {
	return runCaveUse(ctx, m, args)
}

func (h *DefaultHandlers) CaveRun(ctx context.Context, m *Managers, args *caveRunArgs, flags *caveRunFlags) (*ExecutionResult, error) {
	return runCaveCommand(ctx, m, args, flags)
}

func (h *DefaultHandlers) CaveEnter(ctx context.Context, m *Managers, args *caveEnterArgs, flags *caveEnterFlags) (*ExecutionResult, error) {
	return runCaveCommand(ctx, m, &caveRunArgs{Command: ""}, &caveRunFlags{globalFlags: flags.globalFlags})
}

func (h *DefaultHandlers) CaveInit(ctx context.Context, m *Managers, args *caveInitArgs, flags *caveInitFlags) (*ExecutionResult, error) {
	return runInit(ctx, m)
}

func (h *DefaultHandlers) CaveSync(ctx context.Context, m *Managers, args *caveSyncArgs, flags *caveSyncFlags) (*ExecutionResult, error) {
	fmt.Println("Syncing workspace...")
	return &ExecutionResult{ExitCode: 0}, nil
}

func (h *DefaultHandlers) CaveAddpkg(ctx context.Context, m *Managers, args *caveAddpkgArgs, flags *caveAddpkgFlags) (*ExecutionResult, error) {
	return runAddPkg(ctx, m, args)
}

func (h *DefaultHandlers) DiskInfo(ctx context.Context, m *Managers, args *diskInfoArgs, flags *diskInfoFlags) (*ExecutionResult, error) {
	return runDiskInfo(ctx, m)
}

func (h *DefaultHandlers) DiskClean(ctx context.Context, m *Managers, args *diskCleanArgs, flags *diskCleanFlags) (*ExecutionResult, error) {
	return runDiskClean(ctx, m)
}

func (h *DefaultHandlers) DiskUninstall(ctx context.Context, m *Managers, args *diskUninstallArgs, flags *diskUninstallFlags) (*ExecutionResult, error) {
	return runDiskUninstall(ctx, m, flags)
}

func (h *DefaultHandlers) RepoList(ctx context.Context, m *Managers, args *repoListArgs, flags *repoListFlags) (*ExecutionResult, error) {
	return runRepoList(ctx, m)
}

func (h *DefaultHandlers) RepoAdd(ctx context.Context, m *Managers, args *repoAddArgs, flags *repoAddFlags) (*ExecutionResult, error) {
	fmt.Printf("Adding repo %s: %s\n", args.Name, args.URL)
	return &ExecutionResult{ExitCode: 0}, nil
}
func runCaveCommand(ctx context.Context, m *Managers, args *caveRunArgs, flags *caveRunFlags) (*ExecutionResult, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	c, err := m.CaveMgr.Find(cwd)
	if err != nil {
		return nil, err
	}
	variant := flags.Variant
	if variant == "" {
		variant = c.Variant
	}
	c.Variant = variant
	settings, err := c.Config.Resolve(variant)
	if err != nil {
		return nil, err
	}
	// Ensure packages are installed and get symlinks
	prep, err := m.PkgsMgr.Prepare(ctx, settings.Pkgs)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare packages: %w", err)
	}
	// For 'run', we take the 'command' arg. For 'enter', it's empty.
	var command []string
	if args.Command != "" {
		command = strings.Fields(args.Command)
	}
	backend := bubblewrap.Create()
	cmd, err := backend.ResolveLaunch(ctx, m.SysCfg, c, settings, prep, command)
	if err != nil {
		return nil, err
	}
	return &ExecutionResult{
		IsCave: true,
		Exe:    cmd.Path,
		Args:   cmd.Args,
		Env:    cmd.Env,
	}, nil
}
func runInfo(ctx context.Context, m *Managers) (*ExecutionResult, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	c, err := m.CaveMgr.Find(cwd)
	if err != nil {
		fmt.Printf("Current directory is not in a pi workspace.\n")
		return &ExecutionResult{ExitCode: 0}, nil
	}
	fmt.Printf("Cave Name:  %s\n", c.Config.Name)
	if c.Variant != "" {
		fmt.Printf("Variant:    %s\n", c.Variant)
	}
	fmt.Printf("Workspace:  %s\n", c.Workspace)
	fmt.Printf("Home Path:  %s\n", c.HomePath)
	settings, _ := c.Config.Resolve("")
	if len(settings.Pkgs) > 0 {
		fmt.Printf("Packages:   %s\n", strings.Join(settings.Pkgs, ", "))
	}
	return &ExecutionResult{ExitCode: 0}, nil
}

func runCaveList(ctx context.Context, m *Managers) (*ExecutionResult, error) {
	reg, err := cave.LoadRegistry(m.SysCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load cave registry: %w", err)
	}

	m.Disp.Close()

	if len(reg.Caves) == 0 {
		fmt.Println("No caves registered.")
		return &ExecutionResult{ExitCode: 0}, nil
	}

	fmt.Printf("%-20s %-30s %s\n", "NAME", "VARIANTS", "WORKSPACE")
	fmt.Println(strings.Repeat("-", 80))

	for _, entry := range reg.Caves {
		cfgPath := filepath.Join(entry.Workspace, "pi.cave.json")
		cfg, err := cave.LoadConfig(cfgPath)
		variants := "-"
		if err == nil {
			var names []string
			for v := range cfg.Variants {
				if v != "" {
					names = append(names, v)
				}
			}
			if len(names) > 0 {
				variants = strings.Join(names, ", ")
			}
		}
		fmt.Printf("%-20s %-30s %s\n", entry.Name, variants, entry.Workspace)
	}

	return &ExecutionResult{ExitCode: 0}, nil
}

func runCaveUse(ctx context.Context, m *Managers, args *caveUseArgs) (*ExecutionResult, error) {
	target := args.Cave
	if target == "" {
		return nil, fmt.Errorf("cave name required")
	}

	parts := strings.SplitN(target, ":", 2)
	name := parts[0]
	variant := ""
	if len(parts) > 1 {
		variant = parts[1]
	}

	reg, err := cave.LoadRegistry(m.SysCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load cave registry: %w", err)
	}

	var workspace string
	for _, entry := range reg.Caves {
		if entry.Name == name {
			workspace = entry.Workspace
			break
		}
	}

	if workspace == "" {
		return nil, fmt.Errorf("cave %s not found in registry", name)
	}

	if err := os.Chdir(workspace); err != nil {
		return nil, fmt.Errorf("failed to change directory to workspace %s: %w", workspace, err)
	}

	cf := &caveRunFlags{Variant: variant}
	ca := &caveRunArgs{Command: ""}

	return runCaveCommand(ctx, m, ca, cf)
}
func runInit(ctx context.Context, m *Managers) (*ExecutionResult, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	if err := m.CaveMgr.CreateInitConfig(cwd); err != nil {
		return nil, err
	}
	fmt.Println("Initialized new workspace in", cwd)
	return &ExecutionResult{ExitCode: 0}, nil
}
func runAddPkg(ctx context.Context, m *Managers, args *caveAddpkgArgs) (*ExecutionResult, error) {
	pkgStr := args.Package
	if pkgStr == "" {
		return nil, fmt.Errorf("package string required")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	c, err := m.CaveMgr.Find(cwd)
	if err != nil {
		return nil, err
	}
	// Add package to default variant config
	base, ok := c.Config.Variants[""]
	if !ok {
		base = cave.CaveSettings{}
	}
	found := false
	for _, p := range base.Pkgs {
		if p == pkgStr {
			found = true
			break
		}
	}
	if !found {
		base.Pkgs = append(base.Pkgs, pkgStr)
		c.Config.Variants[""] = base
		cfgPath := filepath.Join(c.Workspace, "pi.cave.json")
		if err := c.Config.Save(cfgPath); err != nil {
			return nil, fmt.Errorf("failed to save cave config: %w", err)
		}
		fmt.Printf("Added package %s to %s\n", pkgStr, cfgPath)
	} else {
		fmt.Printf("Package %s already exists in configuration\n", pkgStr)
	}
	return &ExecutionResult{ExitCode: 0}, nil
}
func runInstall(ctx context.Context, m *Managers, args *pkgInstallArgs, flags *pkgInstallFlags) (*ExecutionResult, error) {
	pkgQuery := args.Package
	if pkgQuery == "" {
		return nil, fmt.Errorf("package name required")
	}
	_, err := m.PkgsMgr.Prepare(ctx, []string{pkgQuery})
	if err != nil {
		return nil, err
	}
	return &ExecutionResult{ExitCode: 0}, nil
}

func runPkgList(ctx context.Context, m *Managers, args *pkgListArgs, flags *pkgListFlags) (*ExecutionResult, error) {
	if flags.Index {
		entries, err := m.PkgsMgr.ListIndex(ctx)
		if err != nil {
			return nil, err
		}
		m.Disp.Close()
		fmt.Printf("%-20s %-50s %s\n", "RECIPE", "PATTERNS", "MODE")
		fmt.Println(strings.Repeat("-", 90))
		for _, entry := range entries {
			mode := "lazy"
			if entry.Legacy {
				mode = "legacy"
			}
			patterns := "-"
			if len(entry.Patterns) > 0 {
				patterns = strings.Join(entry.Patterns, ", ")
			}
			fmt.Printf("%-20s %-50s %s\n", entry.Recipe, patterns, mode)
		}
		return &ExecutionResult{ExitCode: 0}, nil
	}

	pkgQuery := args.Package
	if pkgQuery == "" {
		return nil, fmt.Errorf("package name required")
	}
	showAll := flags.All

	pkgs, err := m.PkgsMgr.List(ctx, pkgQuery)
	if err != nil {
		return nil, err
	}

	m.Disp.Close() // Close TUI to print list to stdout

	fmt.Printf("%-20s %-15s %-10s %-12s %-10s %-10s\n", "NAME", "VERSION", "STATUS", "RELEASE", "OS", "ARCH")
	fmt.Println(strings.Repeat("-", 90))

	myOS := m.SysCfg.GetOS()
	myArch := m.SysCfg.GetArch()

	for _, p := range pkgs {
		if !showAll {
			if p.OS != myOS || p.Arch != myArch {
				continue
			}
		}
		status := p.ReleaseStatus
		if status == "" {
			status = "unknown"
		}
		releaseDate := p.ReleaseDate
		if releaseDate == "" {
			releaseDate = "-"
		}
		fmt.Printf("%-20s %-15s %-10s %-12s %-10s %-10s\n", p.Name, p.Version, status, releaseDate, p.OS, p.Arch)
	}
	return &ExecutionResult{ExitCode: 0}, nil
}

func runRepoList(ctx context.Context, m *Managers) (*ExecutionResult, error) {
	entries, err := m.PkgsMgr.ListIndex(ctx)
	if err != nil {
		return nil, err
	}

	m.Disp.Close()

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Recipe < entries[j].Recipe
	})

	fmt.Println("builtin:")
	for _, entry := range entries {
		patterns := entry.Patterns
		if entry.Legacy || len(patterns) == 0 {
			patterns = []string{"legacy"}
		}
		fmt.Printf("  %s.star(%s)\n", entry.Recipe, strings.Join(patterns, ", "))
	}

	return &ExecutionResult{ExitCode: 0}, nil
}
func runDiskInfo(ctx context.Context, m *Managers) (*ExecutionResult, error) {
	stats, total := m.DiskMgr.GetInfo()
	fmt.Printf("%-15s %-10s %s\n", "Type", "Size", "Path")
	fmt.Println(strings.Repeat("-", 60))
	for _, s := range stats {
		fmt.Printf("%-15s %-10s %s\n", s.Label, disk.FormatSize(s.Size), s.Path)
	}
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("%-15s %-10s\n", "Total", disk.FormatSize(total))
	return &ExecutionResult{ExitCode: 0}, nil
}
func runDiskClean(ctx context.Context, m *Managers) (*ExecutionResult, error) {
	cleaned := m.DiskMgr.Clean()
	for _, dir := range cleaned {
		fmt.Printf("Cleaning %s...\n", dir)
	}
	fmt.Println("Clean complete.")
	return &ExecutionResult{ExitCode: 0}, nil
}
func runDiskUninstall(ctx context.Context, m *Managers, flags *diskUninstallFlags) (*ExecutionResult, error) {
	force := flags.Force
	if !force {
		m.Disp.Close() // Terminate Bubble Tea before interactive prompt
		fmt.Print(infoStyle.Render("This will delete ALL pi data (cache, config, state). Are you sure? [y/N]: "))
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("Aborted.")
			return &ExecutionResult{ExitCode: 0}, nil
		}
	}
	removed := m.DiskMgr.Uninstall()
	for _, dir := range removed {
		fmt.Printf("Removing %s...\n", dir)
	}
	fmt.Println("Uninstall complete. Local data removed.")
	return &ExecutionResult{ExitCode: 0}, nil
}
