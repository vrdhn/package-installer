package disk

import (
	"fmt"
	"os"
	"path/filepath"
	"pi/pkg/config"
)

// Manager handles disk operations for pi's XDG directories.
type Manager struct {
	cfg config.ReadOnly
}

// NewManager creates a new disk manager.
func NewManager(cfg config.ReadOnly) *Manager {
	return &Manager{cfg: cfg}
}

// Usage represents disk usage information for a specific type.
type Usage struct {
	Label string
	Size  int64
	Path  string
}

// GetInfo returns disk usage statistics for all pi directories.
func (m *Manager) GetInfo() ([]Usage, int64) {
	paths := map[string]string{
		"Packages":  m.cfg.GetPkgDir(),
		"Downloads": m.cfg.GetDownloadDir(),
		"Discovery": m.cfg.GetDiscoveryDir(),
		"Recipes":   m.cfg.GetRecipeDir(),
		"Homes":     m.cfg.GetHomeDir(),
	}

	var total int64
	var stats []Usage
	for label, path := range paths {
		size, _ := DirSize(path)
		total += size
		stats = append(stats, Usage{
			Label: label,
			Size:  size,
			Path:  path,
		})
	}
	return stats, total
}

// Clean removes temporary and cached data (packages, downloads, discovery).
func (m *Manager) Clean() []string {
	dirs := []string{
		m.cfg.GetPkgDir(),
		m.cfg.GetDownloadDir(),
		m.cfg.GetDiscoveryDir(),
	}

	var cleaned []string
	for _, dir := range dirs {
		if _, err := os.Stat(dir); err == nil {
			os.RemoveAll(dir)
			os.MkdirAll(dir, 0755)
			cleaned = append(cleaned, dir)
		}
	}
	return cleaned
}

// Uninstall removes all pi-related XDG directories.
func (m *Manager) Uninstall() []string {
	dirs := []string{
		m.cfg.GetCacheDir(),
		m.cfg.GetConfigDir(),
		m.cfg.GetStateDir(),
	}

	var removed []string
	for _, dir := range dirs {
		if _, err := os.Stat(dir); err == nil {
			os.RemoveAll(dir)
			removed = append(removed, dir)
		}
	}
	return removed
}

// DirSize calculates the total size of a directory.
func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// FormatSize converts bytes to a human-readable string.
func FormatSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
