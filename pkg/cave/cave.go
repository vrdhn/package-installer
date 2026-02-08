package cave

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"pi/pkg/common"
	"pi/pkg/config"
	"pi/pkg/display"
	"pi/pkg/pkgs"
	"strings"
)

// Cave represents an active sandbox context.
// Mutable
type Cave struct {
	ID        string
	Workspace config.HostPath
	HomePath  config.HostPath
	Variant   string
	Config    *CaveConfig
}

// Manager handles cave discovery and loading.
// Mutable
type manager struct {
	SysConfig config.Config
}

type Manager = *manager

// NewManager creates a new Cave Manager.
func NewManager(cfg config.Config) Manager {
	return &manager{
		SysConfig: cfg,
	}
}

// Find looks for a pi.cave.json.
// Priority:
// 1. PI_WORKSPACE environment variable
// 2. PI_CAVENAME environment variable (lookup in registry)
// 3. Walking up from cwd
func (m *manager) Find(cwd string) (*Cave, error) {
	var root string
	var variant string

	envWorkspace := os.Getenv("PI_WORKSPACE")
	envCaveName := os.Getenv("PI_CAVENAME")

	if envWorkspace != "" {
		root = envWorkspace
	}

	if envCaveName != "" {
		parts := strings.SplitN(envCaveName, ":", 2)
		name := parts[0]
		if len(parts) > 1 {
			variant = parts[1]
		}

		if root == "" {
			reg, err := LoadRegistry(m.SysConfig)
			if err == nil {
				for _, entry := range reg.Caves {
					if entry.Name == name {
						root = entry.Workspace
						break
					}
				}
			}
		}
	}

	if root == "" {
		var err error
		root, err = findWorkspaceRoot(cwd)
		if err != nil {
			return nil, err
		}
	}

	cfgPath := filepath.Join(root, "pi.cave.json")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load cave config: %w", err)
	}

	workspace := root
	if cfg.Workspace != "" {
		workspace = cfg.Workspace
	}

	var homePath string
	if cfg.Home != "" {
		if filepath.IsAbs(cfg.Home) {
			homePath = cfg.Home
		} else {
			homePath = filepath.Join(m.SysConfig.GetHomeDir(), cfg.Home)
		}
	} else {
		// Fallback to hash-based ID for backward compatibility
		id := generateID(root)
		homePath = filepath.Join(m.SysConfig.GetHomeDir(), id)
	}

	return &Cave{
		ID:        filepath.Base(homePath),
		Workspace: workspace,
		HomePath:  homePath,
		Variant:   variant,
		Config:    cfg,
	}, nil
}

func (m *manager) Info(ctx context.Context) (*common.ExecutionResult, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	c, err := m.Find(cwd)
	if err != nil {
		fmt.Printf("Current directory is not in a pi workspace.\n")
		return &common.ExecutionResult{ExitCode: 0}, nil
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
	return &common.ExecutionResult{ExitCode: 0}, nil
}

func (m *manager) List(ctx context.Context, disp display.Display) (*common.ExecutionResult, error) {
	reg, err := LoadRegistry(m.SysConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load cave registry: %w", err)
	}

	disp.Close()

	if len(reg.Caves) == 0 {
		fmt.Println("No caves registered.")
		return &common.ExecutionResult{ExitCode: 0}, nil
	}

	fmt.Printf("%-20s %-30s %s\n", "NAME", "VARIANTS", "WORKSPACE")
	fmt.Println(strings.Repeat("-", 80))

	for _, entry := range reg.Caves {
		cfgPath := filepath.Join(entry.Workspace, "pi.cave.json")
		cfg, err := LoadConfig(cfgPath)
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

	return &common.ExecutionResult{ExitCode: 0}, nil
}

func (m *manager) Use(ctx context.Context, backend Backend, pkgsMgr pkgs.Manager, target string) (*common.ExecutionResult, error) {
	if target == "" {
		return nil, fmt.Errorf("cave name required")
	}

	parts := strings.SplitN(target, ":", 2)
	name := parts[0]
	variant := ""
	if len(parts) > 1 {
		variant = parts[1]
	}

	reg, err := LoadRegistry(m.SysConfig)
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

	return m.RunCommand(ctx, backend, pkgsMgr, variant, "")
}

func (m *manager) RunCommand(ctx context.Context, backend Backend, pkgsMgr pkgs.Manager, variant string, commandStr string) (*common.ExecutionResult, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	c, err := m.Find(cwd)
	if err != nil {
		return nil, err
	}
	if variant == "" {
		variant = c.Variant
	}
	c.Variant = variant
	settings, err := c.Config.Resolve(variant)
	if err != nil {
		return nil, err
	}
	// Ensure packages are installed and get symlinks
	prep, err := pkgsMgr.Prepare(ctx, settings.Pkgs)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare packages: %w", err)
	}
	// For 'run', we take the 'command' arg. For 'enter', it's empty.
	var command []string
	if commandStr != "" {
		command = strings.Fields(commandStr)
	}
	cmd, err := backend.ResolveLaunch(ctx, m.SysConfig, c, settings, prep, command)
	if err != nil {
		return nil, err
	}
	return &common.ExecutionResult{
		IsCave: true,
		Exe:    cmd.Path,
		Args:   cmd.Args,
		Env:    cmd.Env,
	}, nil
}

func (m *manager) Init(ctx context.Context) (*common.ExecutionResult, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	if err := m.CreateInitConfig(cwd); err != nil {
		return nil, err
	}
	fmt.Println("Initialized new workspace in", cwd)
	return &common.ExecutionResult{ExitCode: 0}, nil
}

func (m *manager) Sync(ctx context.Context, pkgsMgr pkgs.Manager) (*common.ExecutionResult, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	c, err := m.Find(cwd)
	if err != nil {
		return nil, err
	}
	settings, err := c.Config.Resolve(c.Variant)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Syncing workspace '%s' (variant: %s)...\n", c.Config.Name, c.Variant)

	// Ensure packages are installed
	_, err = pkgsMgr.Prepare(ctx, settings.Pkgs)
	if err != nil {
		return nil, fmt.Errorf("failed to sync packages: %w", err)
	}

	fmt.Println("Workspace synchronized successfully.")
	return &common.ExecutionResult{ExitCode: 0}, nil
}

func (m *manager) AddPkg(ctx context.Context, pkgStr string) (*common.ExecutionResult, error) {
	if pkgStr == "" {
		return nil, fmt.Errorf("package string required")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	c, err := m.Find(cwd)
	if err != nil {
		return nil, err
	}
	// Add package to default variant config
	base, ok := c.Config.Variants[""]
	if !ok {
		base = CaveSettings{}
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
	return &common.ExecutionResult{ExitCode: 0}, nil
}

// CreateInitConfig creates a default pi.cave.json in the specified directory.
func (m *manager) CreateInitConfig(dir string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	name := filepath.Base(absDir)

	cfg := &CaveConfig{
		Name:      name,
		Workspace: absDir,
		Home:      name,
		Variants: map[string]CaveSettings{
			"": {
				Pkgs: []config.PkgRef{},
			},
		},
	}

	path := filepath.Join(absDir, "pi.cave.json")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("pi.cave.json already exists in %s", absDir)
	}

	if err := cfg.Save(path); err != nil {
		return err
	}

	return m.SyncRegistry(cfg)
}

// SyncRegistry updates the global caves.json with the provided config.
func (m *manager) SyncRegistry(cfg *CaveConfig) error {
	reg, err := LoadRegistry(m.SysConfig)
	if err != nil {
		return err
	}

	found := false
	for i, entry := range reg.Caves {
		if entry.Name == cfg.Name {
			reg.Caves[i].Workspace = cfg.Workspace
			found = true
			break
		}
	}

	if !found {
		reg.Caves = append(reg.Caves, RegistryEntry{
			Name:      cfg.Name,
			Workspace: cfg.Workspace,
		})
	}

	return reg.Save(m.SysConfig)
}

func findWorkspaceRoot(start string) (string, error) {
	curr := start
	for {
		if _, err := os.Stat(filepath.Join(curr, "pi.cave.json")); err == nil {
			return curr, nil
		}

		parent := filepath.Dir(curr)
		if parent == curr {
			return "", fmt.Errorf("pi.cave.json not found in %s or any parent directory", start)
		}
		curr = parent
	}
}

func generateID(path string) string {
	hash := sha256.Sum256([]byte(path))
	return hex.EncodeToString(hash[:])[:12]
}
