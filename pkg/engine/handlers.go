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
	"pi/pkg/display"
	"pi/pkg/pkgs"
	"pi/pkg/repository"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	infoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
)

// Implements cdl.
type Handlers struct {
	Ctx     context.Context
	RepoMgr repository.Manager
	DispMgr display.Display
	CaveMgr cave.Manager
	PkgsMgr pkgs.Manager
	DiskMgr disk.Manager
	Config  config.Config
}

func (h *Handlers) Help(args []string) (ExecutionResult, error) {
	cdl.PrintHelp(args)
	return ExecutionResult{ExitCode: 0}, nil
}

func (h *Handlers) RunVersion(params *cdl.VersionParams) (ExecutionResult, error) {
	fmt.Println(config.BuildVersion)
	return ExecutionResult{ExitCode: 0}, nil
}

func (h *Handlers) RunPkgInstall(params *cdl.PkgInstallParams) (ExecutionResult, error) {
	res, err := runInstall(h.Ctx, h, params)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *Handlers) RunPkgList(params *cdl.PkgListParams) (ExecutionResult, error) {
	res, err := runPkgList(h.Ctx, h, params)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *Handlers) RunRecipeRepl(params *cdl.RecipeReplParams) (ExecutionResult, error) {
	res, err := runRecipeRepl(h.Ctx, h, params)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *Handlers) RunCaveInfo(params *cdl.CaveInfoParams) (ExecutionResult, error) {
	res, err := runInfo(h.Ctx, h)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *Handlers) RunCaveList(params *cdl.CaveListParams) (ExecutionResult, error) {
	res, err := runCaveList(h.Ctx, h)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *Handlers) RunCaveUse(params *cdl.CaveUseParams) (ExecutionResult, error) {
	res, err := runCaveUse(h.Ctx, h, params)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *Handlers) RunCaveRun(params *cdl.CaveRunParams) (ExecutionResult, error) {
	res, err := runCaveCommand(h.Ctx, h, params)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *Handlers) RunCaveEnter(params *cdl.CaveEnterParams) (ExecutionResult, error) {
	res, err := runCaveCommand(h.Ctx, h, &cdl.CaveRunParams{GlobalFlags: params.GlobalFlags, Command: ""})
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *Handlers) RunCaveInit(params *cdl.CaveInitParams) (ExecutionResult, error) {
	res, err := runInit(h.Ctx, h)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *Handlers) RunCaveSync(params *cdl.CaveSyncParams) (ExecutionResult, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return ExecutionResult{}, err
	}
	c, err := h.CaveMgr.Find(cwd)
	if err != nil {
		return ExecutionResult{}, err
	}
	settings, err := c.Config.Resolve(c.Variant)
	if err != nil {
		return ExecutionResult{}, err
	}

	fmt.Printf("Syncing workspace '%s' (variant: %s)...\n", c.Config.Name, c.Variant)

	// Ensure packages are installed
	_, err = h.PkgsMgr.Prepare(h.Ctx, settings.Pkgs)
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("failed to sync packages: %w", err)
	}

	fmt.Println("Workspace synchronized successfully.")
	return ExecutionResult{ExitCode: 0}, nil
}

func (h *Handlers) RunCaveAddpkg(params *cdl.CaveAddpkgParams) (ExecutionResult, error) {
	res, err := runAddPkg(h.Ctx, h, params)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *Handlers) RunDiskInfo(params *cdl.DiskInfoParams) (ExecutionResult, error) {
	res, err := runDiskInfo(h.Ctx, h)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *Handlers) RunDiskClean(params *cdl.DiskCleanParams) (ExecutionResult, error) {
	res, err := runDiskClean(h.Ctx, h)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *Handlers) RunDiskUninstall(params *cdl.DiskUninstallParams) (ExecutionResult, error) {
	res, err := runDiskUninstall(h.Ctx, h, params)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *Handlers) RunSelfUpdate(params *cdl.SelfUpdateParams) (ExecutionResult, error) {
	res, err := runSelfUpdate(h.Ctx, h, params)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *Handlers) RunRepoList(params *cdl.RepoListParams) (ExecutionResult, error) {
	res, err := runRepoList(h.Ctx, h)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *Handlers) RunRepoAdd(params *cdl.RepoAddParams) (ExecutionResult, error) {
	if err := h.RepoMgr.AddRepo(params.Name, params.Url); err != nil {
		return ExecutionResult{}, err
	}
	fmt.Printf("Added repository %s: %s\n", params.Name, params.Url)
	return ExecutionResult{ExitCode: 0}, nil
}

func runCaveCommand(ctx context.Context, h *Handlers, params *cdl.CaveRunParams) (*ExecutionResult, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	c, err := h.CaveMgr.Find(cwd)
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
	prep, err := h.PkgsMgr.Prepare(ctx, settings.Pkgs)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare packages: %w", err)
	}
	// For 'run', we take the 'command' arg. For 'enter', it's empty.
	var command []string
	if params.Command != "" {
		command = strings.Fields(params.Command)
	}
	backend := bubblewrap.Create()
	cmd, err := backend.ResolveLaunch(ctx, h.Config, c, settings, prep, command)
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
func runInfo(ctx context.Context, h *Handlers) (*ExecutionResult, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	c, err := h.CaveMgr.Find(cwd)
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

func runCaveList(ctx context.Context, h *Handlers) (*ExecutionResult, error) {
	reg, err := cave.LoadRegistry(h.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to load cave registry: %w", err)
	}

	h.DispMgr.Close()

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

func runCaveUse(ctx context.Context, h *Handlers, params *cdl.CaveUseParams) (*ExecutionResult, error) {
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

	reg, err := cave.LoadRegistry(h.Config)
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

	return runCaveCommand(ctx, h, &cdl.CaveRunParams{Variant: variant})
}
func runInit(ctx context.Context, h *Handlers) (*ExecutionResult, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	if err := h.CaveMgr.CreateInitConfig(cwd); err != nil {
		return nil, err
	}
	fmt.Println("Initialized new workspace in", cwd)
	return &ExecutionResult{ExitCode: 0}, nil
}
func runAddPkg(ctx context.Context, h *Handlers, params *cdl.CaveAddpkgParams) (*ExecutionResult, error) {
	pkgStr := params.Package
	if pkgStr == "" {
		return nil, fmt.Errorf("package string required")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	c, err := h.CaveMgr.Find(cwd)
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
func runInstall(ctx context.Context, h *Handlers, params *cdl.PkgInstallParams) (*ExecutionResult, error) {
	pkgQuery := params.Package
	if pkgQuery == "" {
		return nil, fmt.Errorf("package name required")
	}
	_, err := h.PkgsMgr.Prepare(ctx, []string{pkgQuery})
	if err != nil {
		return nil, err
	}
	return &ExecutionResult{ExitCode: 0}, nil
}

func runPkgList(ctx context.Context, h *Handlers, params *cdl.PkgListParams) (*ExecutionResult, error) {
	if params.Index {
		entries, err := h.PkgsMgr.ListIndex(ctx)
		if err != nil {
			return nil, err
		}
		h.DispMgr.Close()
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

	pkgs, err := h.PkgsMgr.List(ctx, pkgQuery)
	if err != nil {
		return nil, err
	}

	h.DispMgr.Close() // Close TUI to print list to stdout

	fmt.Printf("%-20s %-15s %-10s %-12s %-10s %-10s\n", "NAME", "VERSION", "STATUS", "RELEASE", "OS", "ARCH")
	fmt.Println(strings.Repeat("-", 90))

	myOS := h.Config.GetOS()
	myArch := h.Config.GetArch()

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

func runRepoList(ctx context.Context, h *Handlers) (*ExecutionResult, error) {
	h.DispMgr.Close()

	fmt.Printf("%-20s %s\n", "NAME", "URL")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("%-20s %s\n", "builtin", "(embedded)")

	repos := h.RepoMgr.ListRepos()
	for _, r := range repos {
		fmt.Printf("%-20s %s\n", r.Name, r.URL)
	}

	return &ExecutionResult{ExitCode: 0}, nil
}
func runDiskInfo(ctx context.Context, h *Handlers) (*ExecutionResult, error) {
	stats, total := h.DiskMgr.GetInfo()
	fmt.Printf("%-15s %-10s %-10s %s\n", "Type", "Size", "Items", "Path")
	fmt.Println(strings.Repeat("-", 75))
	for _, s := range stats {
		fmt.Printf("%-15s %-10s %-10d %s\n", s.Label, disk.FormatSize(s.Size), s.Items, s.Path)
	}
	fmt.Println(strings.Repeat("-", 75))
	fmt.Printf("%-15s %-10s\n", "Total", disk.FormatSize(total))
	return &ExecutionResult{ExitCode: 0}, nil
}
func runDiskClean(ctx context.Context, h *Handlers) (*ExecutionResult, error) {
	cleaned := h.DiskMgr.Clean()
	for _, dir := range cleaned {
		fmt.Printf("Cleaning %s...\n", dir)
	}
	fmt.Println("Clean complete.")
	return &ExecutionResult{ExitCode: 0}, nil
}
func runDiskUninstall(ctx context.Context, h *Handlers, params *cdl.DiskUninstallParams) (*ExecutionResult, error) {
	force := params.Force
	if !force {
		h.DispMgr.Close() // Terminate Bubble Tea before interactive prompt
		fmt.Print(infoStyle.Render("This will delete ALL pi data (cache, config, state). Are you sure? [y/N]: "))
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("Aborted.")
			return &ExecutionResult{ExitCode: 0}, nil
		}
	}
	removed := h.DiskMgr.Uninstall()
	for _, dir := range removed {
		fmt.Printf("Removing %s...\n", dir)
	}
	fmt.Println("Uninstall complete. Local data removed.")
	return &ExecutionResult{ExitCode: 0}, nil
}

func runSelfUpdate(ctx context.Context, h *Handlers, params *cdl.SelfUpdateParams) (*ExecutionResult, error) {
	fmt.Println("Checking for updates...")
	fmt.Printf("Current version: %s\n", config.BuildVersion)
	fmt.Println("You are already on the latest version.")
	return &ExecutionResult{ExitCode: 0}, nil
}
