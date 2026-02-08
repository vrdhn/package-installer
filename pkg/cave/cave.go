// Package cave provides the sandbox (cave) management logic.
// It handles workspace discovery, sandbox initialization, and command execution
// within isolated environments using bubblewrap.
package cave

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"pi/pkg/common"
	"pi/pkg/config"
	"pi/pkg/display"
	"pi/pkg/lazyjson"
	"pi/pkg/pkgs"
	"strings"
)

// Manager defines the operations for discovering and managing caves.
type manager struct {
	Config config.Config
	Disp   display.Display
	regMgr *lazyjson.Manager[common.Registry]
}

// Manager is a pointer to the internal manager implementation.
type Manager = *manager

// NewManager creates a new Cave Manager.
func NewManager(cfg config.Config, disp display.Display) Manager {
	regPath := filepath.Join(cfg.GetConfigDir(), "cave.json")
	return &manager{
		Config: cfg,
		Disp:   disp,
		regMgr: lazyjson.New[common.Registry](regPath),
	}
}

// Find looks for a pi.cave.json.
// Priority:
// 1. PI_WORKSPACE environment variable
// 2. PI_CAVENAME environment variable (lookup in registry)
// 3. Walking up from cwd
func (m *manager) Find(cwd string) (*common.Cave, error) {
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
			reg, err := m.regMgr.Get()
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
	cfg, err := common.LoadCaveConfig(cfgPath)
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
			homePath = filepath.Join(m.Config.GetHomeDir(), cfg.Home)
		}
	} else {
		// Fallback to hash-based ID for backward compatibility
		id := generateID(root)
		homePath = filepath.Join(m.Config.GetHomeDir(), id)
	}

	return &common.Cave{
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
		m.Disp.Print("Current directory is not in a pi workspace.\n")
		return &common.ExecutionResult{ExitCode: 0}, nil
	}
	m.Disp.Print(fmt.Sprintf("Cave Name:  %s\n", c.Config.Name))
	if c.Variant != "" {
		m.Disp.Print(fmt.Sprintf("Variant:    %s\n", c.Variant))
	}
	m.Disp.Print(fmt.Sprintf("Workspace:  %s\n", c.Workspace))
	m.Disp.Print(fmt.Sprintf("Home Path:  %s\n", c.HomePath))
	settings, _ := c.Config.Resolve("")
	if len(settings.Pkgs) > 0 {
		m.Disp.Print(fmt.Sprintf("Packages:   %s\n", strings.Join(settings.Pkgs, ", ")))
	}
	return &common.ExecutionResult{ExitCode: 0}, nil
}

func (m *manager) List(ctx context.Context, disp display.Display) (*common.ExecutionResult, error) {
	reg, err := m.regMgr.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to load cave registry: %w", err)
	}

	m.Disp.Close()

	if len(reg.Caves) == 0 {
		m.Disp.Print("No caves registered.\n")
		return &common.ExecutionResult{ExitCode: 0}, nil
	}

	m.Disp.Print(fmt.Sprintf("%-20s %-30s %s\n", "NAME", "VARIANTS", "WORKSPACE"))
	m.Disp.Print(fmt.Sprintln(strings.Repeat("-", 80)))

	for _, entry := range reg.Caves {
		cfgPath := filepath.Join(entry.Workspace, "pi.cave.json")
		cfg, err := common.LoadCaveConfig(cfgPath)
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
		m.Disp.Print(fmt.Sprintf("%-20s %-30s %s\n", entry.Name, variants, entry.Workspace))
	}

	return &common.ExecutionResult{ExitCode: 0}, nil
}

func (m *manager) Use(ctx context.Context, pkgsMgr pkgs.Manager, target string) (*common.ExecutionResult, error) {
	if target == "" {
		return nil, fmt.Errorf("cave name required")
	}

	parts := strings.SplitN(target, ":", 2)
	name := parts[0]
	variant := ""
	if len(parts) > 1 {
		variant = parts[1]
	}

	reg, err := m.regMgr.Get()
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

	return m.RunCommand(ctx, pkgsMgr, variant, "")
}

func (m *manager) RunCommand(ctx context.Context, pkgsMgr pkgs.Manager, variant string, commandStr string) (*common.ExecutionResult, error) {
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

	return &common.ExecutionResult{
		Cave:        c,
		Settings:    settings,
		Preparation: prep,
		Command:     command,
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
	slog.Info("Initialized new workspace", "path", cwd)
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

	slog.Info("Syncing workspace", "name", c.Config.Name, "variant", c.Variant)

	// Ensure packages are installed
	_, err = pkgsMgr.Prepare(ctx, settings.Pkgs)
	if err != nil {
		return nil, fmt.Errorf("failed to sync packages: %w", err)
	}

	slog.Info("Workspace synchronized successfully")
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
		base = common.CaveSettings{}
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
		slog.Info("Added package", "package", pkgStr, "config", cfgPath)
	} else {
		slog.Info("Package already exists in configuration", "package", pkgStr)
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

	cfg := &common.CaveConfig{
		Name:      name,
		Workspace: absDir,
		Home:      name,
		Variants: map[string]common.CaveSettings{
			"": {
				Pkgs: []common.PkgRef{},
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

// SyncRegistry updates the global cave.json with the provided config.
func (m *manager) SyncRegistry(cfg *common.CaveConfig) error {
	err := m.regMgr.Modify(func(reg *common.Registry) error {
		found := false
		for i, entry := range reg.Caves {
			if entry.Name == cfg.Name {
				reg.Caves[i].Workspace = cfg.Workspace
				found = true
				break
			}
		}

		if !found {
			reg.Caves = append(reg.Caves, common.CaveEntry{
				Name:      cfg.Name,
				Workspace: cfg.Workspace,
			})
		}
		return nil
	})
	if err != nil {
		return err
	}

	return m.regMgr.Save()
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
