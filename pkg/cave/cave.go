package cave

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	sysconfig "pi/pkg/config"
	"strings"
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
type manager struct {
	SysConfig sysconfig.Config
}

type Manager = *manager

// NewManager creates a new Cave Manager.
func NewManager(config sysconfig.Config) Manager {
	return &manager{
		SysConfig: config,
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
