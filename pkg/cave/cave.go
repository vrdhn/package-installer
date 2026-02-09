// Package cave provides the sandbox (cave) management logic.
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
	"pi/pkg/lazyjson"
	"pi/pkg/pkgs"
	"strings"
)

// NewManager creates a new Cave Manager.
func NewManager(cfg config.Config) Manager {
	regPath := filepath.Join(cfg.GetConfigDir(), "cave.json")
	return &manager{
		Config: cfg,
		regMgr: lazyjson.New[Registry](regPath),
	}
}

// Find looks for a pi.cave.json.
func (m *manager) Find(cwd string) (*Cave, error) {
	root, variant := m.findRootFromEnv()

	if root == "" {
		var err error
		root, err = findWorkspaceRoot(cwd)
		if err != nil {
			return nil, err
		}
	}

	cfgPath := filepath.Join(root, "pi.cave.json")
	cfg, err := LoadCaveConfig(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load cave config: %w", err)
	}

	workspace := root
	if cfg.Workspace != "" {
		workspace = cfg.Workspace
	}

	homePath := m.resolveHomePath(root, cfg)

	return &Cave{
		ID:        filepath.Base(homePath),
		Workspace: workspace,
		HomePath:  homePath,
		Variant:   variant,
		Config:    cfg,
	}, nil
}

func (m *manager) findRootFromEnv() (string, string) {
	root := os.Getenv("PI_WORKSPACE")
	envCaveName := os.Getenv("PI_CAVENAME")
	variant := ""

	if envCaveName != "" {
		parts := strings.SplitN(envCaveName, ":", 2)
		name := parts[0]
		if len(parts) > 1 {
			variant = parts[1]
		}

		if root == "" {
			root = m.findRootByName(name)
		}
	}
	return root, variant
}

func (m *manager) findRootByName(name string) string {
	reg, err := m.regMgr.Get()
	if err != nil {
		return ""
	}
	for _, entry := range reg.Caves {
		if entry.Name == name {
			return entry.Workspace
		}
	}
	return ""
}

func (m *manager) resolveHomePath(root string, cfg *CaveConfig) string {
	if cfg.Home != "" {
		if filepath.IsAbs(cfg.Home) {
			return cfg.Home
		}
		return filepath.Join(m.Config.GetHomeDir(), cfg.Home)
	}
	// Fallback to hash-based ID for backward compatibility
	id := generateID(root)
	return filepath.Join(m.Config.GetHomeDir(), id)
}

func (m *manager) Info(ctx context.Context) (*common.ExecutionResult, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	c, err := m.Find(cwd)
	if err != nil {
		return &common.ExecutionResult{
			Output: &common.Output{
				Message: "Current directory is not in a pi workspace.",
			},
		}, nil
	}

	out := &common.Output{
		KV: []common.KeyValue{
			{Key: "Cave Name", Value: c.Config.Name},
			{Key: "Workspace", Value: c.Workspace},
			{Key: "Home Path", Value: c.HomePath},
		},
	}

	if c.Variant != "" {
		out.KV = append(out.KV, common.KeyValue{Key: "Variant", Value: c.Variant})
	}

	settings, _ := c.Config.Resolve(c.Variant)
	if len(settings.Pkgs) > 0 {
		out.KV = append(out.KV, common.KeyValue{Key: "Packages", Value: strings.Join(settings.Pkgs, ", ")})
	}

	return &common.ExecutionResult{
		Output: out,
	}, nil
}

func (m *manager) List(ctx context.Context) (*common.ExecutionResult, error) {
	reg, err := m.regMgr.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to load cave registry: %w", err)
	}

	if len(reg.Caves) == 0 {
		return &common.ExecutionResult{
			Output: &common.Output{
				Message: "No caves registered.",
			},
		}, nil
	}

	table := &common.Table{
		Header: []string{"NAME", "VARIANTS", "WORKSPACE"},
	}

	for _, entry := range reg.Caves {
		cfgPath := filepath.Join(entry.Workspace, "pi.cave.json")
		cfg, err := LoadCaveConfig(cfgPath)
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
		table.Rows = append(table.Rows, []string{entry.Name, variants, entry.Workspace})
	}

	return &common.ExecutionResult{
		Output: &common.Output{
			Table: table,
		},
	}, nil
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

	// Create symlinks for packages in host Cave Home
	if err := pkgs.CreateSymlinks(c.HomePath, prep.Symlinks); err != nil {
		return nil, fmt.Errorf("failed to create symlinks: %w", err)
	}

	// For 'run', we take the 'command' arg. For 'enter', it's empty.
	var command []string
	if commandStr != "" {
		command = strings.Fields(commandStr)
	}

	caveName := c.Config.Name
	if c.Variant != "" {
		caveName = fmt.Sprintf("%s:%s", caveName, c.Variant)
	}

	return &common.ExecutionResult{
		SandboxInfo: &common.SandboxInfo{
			ID:        c.ID,
			Workspace: c.Workspace,
			HomePath:  c.HomePath,
			CaveName:  caveName,
			Env:       settings.Env,
		},
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
	return &common.ExecutionResult{
		Output: &common.Output{
			Message: fmt.Sprintf("Initialized new workspace at %s", cwd),
		},
	}, nil
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

	return &common.ExecutionResult{
		Output: &common.Output{
			Message: "Workspace synchronized successfully",
		},
	}, nil
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
		return &common.ExecutionResult{
			Output: &common.Output{
				Message: fmt.Sprintf("Added package %s to %s", pkgStr, cfgPath),
			},
		}, nil
	} else {
		return &common.ExecutionResult{
			Output: &common.Output{
				Message: fmt.Sprintf("Package %s already exists in configuration", pkgStr),
			},
		}, nil
	}
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
				Pkgs: []string{},
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
func (m *manager) SyncRegistry(cfg *CaveConfig) error {
	err := m.regMgr.Modify(func(reg *Registry) error {
		found := false
		for i, entry := range reg.Caves {
			if entry.Name == cfg.Name {
				reg.Caves[i].Workspace = cfg.Workspace
				found = true
				break
			}
		}

		if !found {
			reg.Caves = append(reg.Caves, CaveEntry{
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
