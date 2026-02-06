package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"pi/pkg/bubblewrap"
	"pi/pkg/cave"
	"pi/pkg/cdl"
	"pi/pkg/config"
	"pi/pkg/disk"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	infoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
)

type DefaultHandlers struct {
	Ctx context.Context
	Mgr *Managers
}

func (h *DefaultHandlers) Help(args []string) (ExecutionResult, error) {
	cdl.PrintHelp(args)
	return ExecutionResult{ExitCode: 0}, nil
}

func (h *DefaultHandlers) RunVersion(params *cdl.VersionParams) (ExecutionResult, error) {
	fmt.Println(config.GetBuildInfo())
	return ExecutionResult{ExitCode: 0}, nil
}

func (h *DefaultHandlers) RunPkgInstall(params *cdl.PkgInstallParams) (ExecutionResult, error) {
	res, err := runInstall(h.Ctx, h.Mgr, params)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *DefaultHandlers) RunPkgList(params *cdl.PkgListParams) (ExecutionResult, error) {
	res, err := runPkgList(h.Ctx, h.Mgr, params)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *DefaultHandlers) RunRecipeRepl(params *cdl.RecipeReplParams) (ExecutionResult, error) {
	res, err := runRecipeRepl(h.Ctx, h.Mgr, params)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *DefaultHandlers) RunCaveInfo(params *cdl.CaveInfoParams) (ExecutionResult, error) {
	res, err := runInfo(h.Ctx, h.Mgr)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *DefaultHandlers) RunCaveList(params *cdl.CaveListParams) (ExecutionResult, error) {
	res, err := runCaveList(h.Ctx, h.Mgr)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *DefaultHandlers) RunCaveUse(params *cdl.CaveUseParams) (ExecutionResult, error) {
	res, err := runCaveUse(h.Ctx, h.Mgr, params)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *DefaultHandlers) RunCaveRun(params *cdl.CaveRunParams) (ExecutionResult, error) {
	res, err := runCaveCommand(h.Ctx, h.Mgr, params)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *DefaultHandlers) RunCaveEnter(params *cdl.CaveEnterParams) (ExecutionResult, error) {
	res, err := runCaveCommand(h.Ctx, h.Mgr, &cdl.CaveRunParams{GlobalFlags: params.GlobalFlags, Command: ""})
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *DefaultHandlers) RunCaveInit(params *cdl.CaveInitParams) (ExecutionResult, error) {
	res, err := runInit(h.Ctx, h.Mgr)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *DefaultHandlers) RunCaveSync(params *cdl.CaveSyncParams) (ExecutionResult, error) {
	fmt.Println("Syncing workspace...")
	return ExecutionResult{ExitCode: 0}, nil
}

func (h *DefaultHandlers) RunCaveAddpkg(params *cdl.CaveAddpkgParams) (ExecutionResult, error) {
	res, err := runAddPkg(h.Ctx, h.Mgr, params)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *DefaultHandlers) RunDiskInfo(params *cdl.DiskInfoParams) (ExecutionResult, error) {
	res, err := runDiskInfo(h.Ctx, h.Mgr)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *DefaultHandlers) RunDiskClean(params *cdl.DiskCleanParams) (ExecutionResult, error) {
	res, err := runDiskClean(h.Ctx, h.Mgr)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *DefaultHandlers) RunDiskUninstall(params *cdl.DiskUninstallParams) (ExecutionResult, error) {
	res, err := runDiskUninstall(h.Ctx, h.Mgr, params)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *DefaultHandlers) RunRepoList(params *cdl.RepoListParams) (ExecutionResult, error) {
	res, err := runRepoList(h.Ctx, h.Mgr)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *DefaultHandlers) RunRepoAdd(params *cdl.RepoAddParams) (ExecutionResult, error) {
	if err := h.Mgr.Repo.AddRepo(params.Name, params.Url); err != nil {
		return ExecutionResult{}, err
	}
	fmt.Printf("Added repository %s: %s\n", params.Name, params.Url)
	return ExecutionResult{ExitCode: 0}, nil
}

func runCaveCommand(ctx context.Context, m *Managers, params *cdl.CaveRunParams) (*ExecutionResult, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	c, err := m.CaveMgr.Find(cwd)
	if err != nil {
		return nil, err
	}
	variant := params.Variant
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
	if params.Command != "" {
		command = strings.Fields(params.Command)
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

func runCaveUse(ctx context.Context, m *Managers, params *cdl.CaveUseParams) (*ExecutionResult, error) {
	target := params.Cave
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

	return runCaveCommand(ctx, m, &cdl.CaveRunParams{Variant: variant})
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
func runAddPkg(ctx context.Context, m *Managers, params *cdl.CaveAddpkgParams) (*ExecutionResult, error) {
	pkgStr := params.Package
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
func runInstall(ctx context.Context, m *Managers, params *cdl.PkgInstallParams) (*ExecutionResult, error) {
	pkgQuery := params.Package
	if pkgQuery == "" {
		return nil, fmt.Errorf("package name required")
	}
	_, err := m.PkgsMgr.Prepare(ctx, []string{pkgQuery})
	if err != nil {
		return nil, err
	}
	return &ExecutionResult{ExitCode: 0}, nil
}

func runPkgList(ctx context.Context, m *Managers, params *cdl.PkgListParams) (*ExecutionResult, error) {
	if params.Index {
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

	pkgQuery := params.Package
	if pkgQuery == "" {
		return nil, fmt.Errorf("package name required")
	}
	showAll := params.All

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
	m.Disp.Close()

	fmt.Printf("%-20s %s\n", "NAME", "URL")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("%-20s %s\n", "builtin", "(embedded)")

	repos := m.Repo.ListRepos()
	for _, r := range repos {
		fmt.Printf("%-20s %s\n", r.Name, r.URL)
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
func runDiskUninstall(ctx context.Context, m *Managers, params *cdl.DiskUninstallParams) (*ExecutionResult, error) {
	force := params.Force
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
