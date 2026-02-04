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
	"pi/pkg/display"
	"pi/pkg/pkgs"
	"pi/pkg/repository"
	"strings"
)

// Mutable
type DefaultHandler struct {
	Repo    *repository.Manager
	Disp    display.Display
	CaveMgr *cave.Manager
	PkgsMgr *pkgs.Manager
	DiskMgr *disk.Manager
	SysCfg  config.ReadOnly
	Theme   *Theme
}

func (h *DefaultHandler) Execute(ctx context.Context, inv *Invocation) (*ExecutionResult, error) {
	if v, ok := inv.Global["verbose"].(bool); ok {
		h.Disp.SetVerbose(v)
	}
	path := getCmdPath(inv.Command)

	// Restriction: if already in a cave, only allow 'cave info'
	if envCave := os.Getenv("PI_CAVENAME"); envCave != "" {
		if path != "cave/info" {
			return nil, fmt.Errorf("already in cave %s", envCave)
		}
	}

	switch path {
	case "pkg/install":
		return h.runInstall(ctx, inv)
	case "pkg/list":
		return h.runPkgList(ctx, inv)
	case "cave/info":
		return h.runInfo(ctx, inv)
	case "cave/list":
		return h.runCaveList(ctx, inv)
	case "cave/use":
		return h.runCaveUse(ctx, inv)
	case "cave/run":
		return h.runCaveCommand(ctx, inv)
	case "cave/sync":
		fmt.Println("Syncing workspace...")
	case "cave/init":
		return h.runInit(ctx, inv)
	case "cave/addpkg":
		return h.runAddPkg(ctx, inv)
	case "cave/enter":
		return h.runCaveCommand(ctx, inv)
	case "disk/info":
		return h.runDiskInfo(ctx, inv)
	case "disk/clean":
		return h.runDiskClean(ctx, inv)
	case "disk/uninstall":
		return h.runDiskUninstall(ctx, inv)
	case "remote/list":
		fmt.Println("Listing remotes...")
	case "remote/add":
		fmt.Printf("Adding remote %s: %s\n", inv.Args["name"], inv.Args["url"])
	default:
		panic("unreachable")
	}
	return &ExecutionResult{ExitCode: 0}, nil
}
func (h *DefaultHandler) runCaveCommand(ctx context.Context, inv *Invocation) (*ExecutionResult, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	c, err := h.CaveMgr.Find(cwd)
	if err != nil {
		return nil, err
	}
	variant, _ := inv.Flags["variant"].(string)
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
	if cmd, ok := inv.Args["command"]; ok && cmd != "" {
		command = strings.Fields(cmd)
	}
	backend := bubblewrap.Create()
	cmd, err := backend.ResolveLaunch(ctx, h.SysCfg, c, settings, prep, command)
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
func (h *DefaultHandler) runInfo(ctx context.Context, inv *Invocation) (*ExecutionResult, error) {
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

func (h *DefaultHandler) runCaveList(ctx context.Context, inv *Invocation) (*ExecutionResult, error) {
	reg, err := cave.LoadRegistry(h.SysCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load cave registry: %w", err)
	}

	h.Disp.Close()

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

func (h *DefaultHandler) runCaveUse(ctx context.Context, inv *Invocation) (*ExecutionResult, error) {
	target := inv.Args["cave"]
	if target == "" {
		return nil, fmt.Errorf("cave name required")
	}

	parts := strings.SplitN(target, ":", 2)
	name := parts[0]
	variant := ""
	if len(parts) > 1 {
		variant = parts[1]
	}

	reg, err := cave.LoadRegistry(h.SysCfg)
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

	// Override variant flag for runCaveCommand
	if variant != "" {
		if inv.Flags == nil {
			inv.Flags = make(map[string]any)
		}
		inv.Flags["variant"] = variant
	}

	return h.runCaveCommand(ctx, inv)
}
func (h *DefaultHandler) runInit(ctx context.Context, inv *Invocation) (*ExecutionResult, error) {
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
func (h *DefaultHandler) runAddPkg(ctx context.Context, inv *Invocation) (*ExecutionResult, error) {
	pkgStr := inv.Args["package"]
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
func (h *DefaultHandler) runInstall(ctx context.Context, inv *Invocation) (*ExecutionResult, error) {
	pkgQuery := inv.Args["package"]
	if pkgQuery == "" {
		return nil, fmt.Errorf("package name required")
	}
	_, err := h.PkgsMgr.Prepare(ctx, []string{pkgQuery})
	if err != nil {
		return nil, err
	}
	return &ExecutionResult{ExitCode: 0}, nil
}

func (h *DefaultHandler) runPkgList(ctx context.Context, inv *Invocation) (*ExecutionResult, error) {
	pkgQuery := inv.Args["package"]
	if pkgQuery == "" {
		return nil, fmt.Errorf("package name required")
	}
	showAll, _ := inv.Flags["all"].(bool)

	pkgs, err := h.PkgsMgr.List(ctx, pkgQuery)
	if err != nil {
		return nil, err
	}

	h.Disp.Close() // Close TUI to print list to stdout

	fmt.Printf("%-20s %-15s %-10s %-10s %-10s\n", "NAME", "VERSION", "STATUS", "OS", "ARCH")
	fmt.Println(strings.Repeat("-", 75))

	myOS := h.SysCfg.GetOS()
	myArch := h.SysCfg.GetArch()

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
		fmt.Printf("%-20s %-15s %-10s %-10s %-10s\n", p.Name, p.Version, status, p.OS, p.Arch)
	}
	return &ExecutionResult{ExitCode: 0}, nil
}
func (h *DefaultHandler) runDiskInfo(ctx context.Context, inv *Invocation) (*ExecutionResult, error) {
	stats, total := h.DiskMgr.GetInfo()
	fmt.Printf("%-15s %-10s %s\n", "Type", "Size", "Path")
	fmt.Println(strings.Repeat("-", 60))
	for _, s := range stats {
		fmt.Printf("%-15s %-10s %s\n", s.Label, disk.FormatSize(s.Size), s.Path)
	}
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("%-15s %-10s\n", "Total", disk.FormatSize(total))
	return &ExecutionResult{ExitCode: 0}, nil
}
func (h *DefaultHandler) runDiskClean(ctx context.Context, inv *Invocation) (*ExecutionResult, error) {
	cleaned := h.DiskMgr.Clean()
	for _, dir := range cleaned {
		fmt.Printf("Cleaning %s...\n", dir)
	}
	fmt.Println("Clean complete.")
	return &ExecutionResult{ExitCode: 0}, nil
}
func (h *DefaultHandler) runDiskUninstall(ctx context.Context, inv *Invocation) (*ExecutionResult, error) {
	force, _ := inv.Flags["force"].(bool)
	if !force {
		h.Disp.Close() // Terminate Bubble Tea before interactive prompt
		fmt.Print(h.Theme.Styled(h.Theme.Red, "This will delete ALL pi data (cache, config, state). Are you sure? [y/N]: "))
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
