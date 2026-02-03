package cave

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	sysconfig "pi/pkg/config"
)

// Cave represents an active sandbox context.
// Mutable
type Cave struct {
	ID        string
	Workspace sysconfig.HostPath
	HomePath  sysconfig.HostPath
	Variant   string
	Config    *CaveConfig
}

// Manager handles cave discovery and loading.
// Mutable
type Manager struct {
	SysConfig sysconfig.ReadOnly
}

// NewManager creates a new Cave Manager.
func NewManager(sysCfg sysconfig.ReadOnly) *Manager {
	return &Manager{
		SysConfig: sysCfg,
	}
}

// Find looks for a pi.cave.json starting from cwd and walking up.
// If not found, it returns an error.
func (m *Manager) Find(cwd string) (*Cave, error) {
	root, err := findWorkspaceRoot(cwd)
	if err != nil {
		// If not found by walking up, we don't yet support registry-only lookup here
		// as we need a workspace context. But we could potentially find by Name later.
		return nil, err
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
		Config:    cfg,
	}, nil
}

// CreateInitConfig creates a default pi.cave.json in the specified directory.
func (m *Manager) CreateInitConfig(dir string) error {
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
				Pkgs: []sysconfig.PkgRef{},
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
func (m *Manager) SyncRegistry(cfg *CaveConfig) error {
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
