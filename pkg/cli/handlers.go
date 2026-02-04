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
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	infoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
)

func boolFlag(flags map[string]any, name string) bool {
	if v, ok := flags[name]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func stringFlag(flags map[string]any, name string) string {
	if v, ok := flags[name]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func Binder(inv *Invocation, global *GlobalFlags) (Action, error) {
	path := getCmdPath(inv.Command)

	switch path {
	case "version":
		return func(ctx context.Context, m *Managers) (*ExecutionResult, error) {
			fmt.Println(config.GetBuildInfo())
			return &ExecutionResult{ExitCode: 0}, nil
		}, nil

	case "pkg/install":
		args := &PkgInstallArgs{Package: inv.Args["package"]}
		flags := &PkgInstallFlags{Force: boolFlag(inv.Flags, "force")}
		return func(ctx context.Context, m *Managers) (*ExecutionResult, error) {
			return runInstall(ctx, m, args, flags)
		}, nil

	case "pkg/list":
		args := &PkgListArgs{Package: inv.Args["package"]}
		flags := &PkgListFlags{
			All:   boolFlag(inv.Flags, "all"),
			Index: boolFlag(inv.Flags, "index"),
		}
		return func(ctx context.Context, m *Managers) (*ExecutionResult, error) {
			return runPkgList(ctx, m, args, flags)
		}, nil

	case "cave/info":
		return func(ctx context.Context, m *Managers) (*ExecutionResult, error) {
			return runInfo(ctx, m)
		}, nil

	case "cave/list":
		return func(ctx context.Context, m *Managers) (*ExecutionResult, error) {
			return runCaveList(ctx, m)
		}, nil

	case "cave/use":
		args := &CaveUseArgs{Cave: inv.Args["cave"]}
		return func(ctx context.Context, m *Managers) (*ExecutionResult, error) {
			return runCaveUse(ctx, m, args)
		}, nil

	case "cave/run":
		args := &CaveRunArgs{Command: inv.Args["command"]}
		flags := &CaveRunFlags{Variant: stringFlag(inv.Flags, "variant")}
		return func(ctx context.Context, m *Managers) (*ExecutionResult, error) {
			return runCaveCommand(ctx, m, args, flags)
		}, nil

	case "cave/enter":
		args := &CaveRunArgs{Command: ""}
		flags := &CaveRunFlags{Variant: ""}
		return func(ctx context.Context, m *Managers) (*ExecutionResult, error) {
			return runCaveCommand(ctx, m, args, flags)
		}, nil

	case "cave/init":
		return func(ctx context.Context, m *Managers) (*ExecutionResult, error) {
			return runInit(ctx, m)
		}, nil

	case "cave/sync":
		return func(ctx context.Context, m *Managers) (*ExecutionResult, error) {
			fmt.Println("Syncing workspace...")
			return &ExecutionResult{ExitCode: 0}, nil
		}, nil

	case "cave/addpkg":
		args := &CaveAddPkgArgs{Package: inv.Args["package"]}
		return func(ctx context.Context, m *Managers) (*ExecutionResult, error) {
			return runAddPkg(ctx, m, args)
		}, nil

	case "disk/info":
		return func(ctx context.Context, m *Managers) (*ExecutionResult, error) {
			return runDiskInfo(ctx, m)
		}, nil

	case "disk/clean":
		return func(ctx context.Context, m *Managers) (*ExecutionResult, error) {
			return runDiskClean(ctx, m)
		}, nil

	case "disk/uninstall":
		flags := &DiskUninstallFlags{Force: boolFlag(inv.Flags, "force")}
		return func(ctx context.Context, m *Managers) (*ExecutionResult, error) {
			return runDiskUninstall(ctx, m, flags)
		}, nil

	case "remote/list":
		return func(ctx context.Context, m *Managers) (*ExecutionResult, error) {
			fmt.Println("Listing remotes...")
			return &ExecutionResult{ExitCode: 0}, nil
		}, nil

	case "remote/add":
		args := &RemoteAddArgs{Name: inv.Args["name"], URL: inv.Args["url"]}
		return func(ctx context.Context, m *Managers) (*ExecutionResult, error) {
			fmt.Printf("Adding remote %s: %s\n", args.Name, args.URL)
			return &ExecutionResult{ExitCode: 0}, nil
		}, nil

	default:
		return nil, fmt.Errorf("no binder for command: %s", path)
	}
}
func runCaveCommand(ctx context.Context, m *Managers, args *CaveRunArgs, flags *CaveRunFlags) (*ExecutionResult, error) {
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

func runCaveUse(ctx context.Context, m *Managers, args *CaveUseArgs) (*ExecutionResult, error) {
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

	cf := &CaveRunFlags{Variant: variant}
	ca := &CaveRunArgs{Command: ""}

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
func runAddPkg(ctx context.Context, m *Managers, args *CaveAddPkgArgs) (*ExecutionResult, error) {
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
func runInstall(ctx context.Context, m *Managers, args *PkgInstallArgs, flags *PkgInstallFlags) (*ExecutionResult, error) {
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

func runPkgList(ctx context.Context, m *Managers, args *PkgListArgs, flags *PkgListFlags) (*ExecutionResult, error) {
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

	fmt.Printf("%-20s %-15s %-10s %-10s %-10s\n", "NAME", "VERSION", "STATUS", "OS", "ARCH")
	fmt.Println(strings.Repeat("-", 75))

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
		fmt.Printf("%-20s %-15s %-10s %-10s %-10s\n", p.Name, p.Version, status, p.OS, p.Arch)
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
func runDiskUninstall(ctx context.Context, m *Managers, flags *DiskUninstallFlags) (*ExecutionResult, error) {
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
