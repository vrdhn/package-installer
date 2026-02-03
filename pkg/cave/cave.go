package cave

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"pi/pkg/cave/config"
	sysconfig "pi/pkg/config"
)

// Cave represents an active sandbox context.
// Mutable
type Cave struct {
	ID        string
	Workspace string
	HomePath  string
	Variant   string
	Config    *config.CaveConfig
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
		return nil, err
	}

	cfgPath := filepath.Join(root, "pi.cave.json")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load cave config: %w", err)
	}

	// Generate ID based on workspace path
	id := generateID(root)

	// Determine HomePath
	homePath := filepath.Join(m.SysConfig.GetHomeDir(), id)

	return &Cave{
		ID:        id,
		Workspace: root,
		HomePath:  homePath,
		Config:    cfg,
	}, nil
}

// CreateInitConfig creates a default pi.cave.json in the specified directory.
func (m *Manager) CreateInitConfig(dir string) error {
	cfg := &config.CaveConfig{
		Cave: config.CaveSettings{
			Packages: []string{},
			Env:      make(map[string]string),
		},
		Variants: make(map[string]config.CaveSettings),
	}
	path := filepath.Join(dir, "pi.cave.json")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("pi.cave.json already exists in %s", dir)
	}
	return cfg.Save(path)
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
	return hex.EncodeToString(hash[:])[:12] // Short hash is usually enough
}
