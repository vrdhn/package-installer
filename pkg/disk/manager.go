package disk

import (
	"fmt"
	"os"
	"path/filepath"
	"pi/pkg/common"
	"pi/pkg/config"
	"strings"
)

// Manager handles disk operations for pi's XDG directories.
type manager struct {
	cfg config.Config
}

type Manager = *manager

// NewManager creates a new disk manager.
func NewManager(cfg config.Config) Manager {
	return &manager{cfg: cfg}
}

// Usage represents disk usage information for a specific type.
type Usage struct {
	Label string
	Size  int64
	Items int
	Path  string
}

// Info displays disk usage statistics for all pi directories.
func (m *manager) Info() (*common.ExecutionResult, error) {
	stats, total := m.GetInfo()
	fmt.Printf("%-15s %-10s %-10s %s\n", "Type", "Size", "Items", "Path")
	fmt.Println(strings.Repeat("-", 75))
	for _, s := range stats {
		fmt.Printf("%-15s %-10s %-10d %s\n", s.Label, FormatSize(s.Size), s.Items, s.Path)
	}
	fmt.Println(strings.Repeat("-", 75))
	fmt.Printf("%-15s %-10s\n", "Total", FormatSize(total))
	return &common.ExecutionResult{ExitCode: 0}, nil
}

// CleanDir removes temporary and cached data (packages, downloads, discovery).
func (m *manager) CleanDir() (*common.ExecutionResult, error) {
	cleaned := m.Clean()
	for _, dir := range cleaned {
		fmt.Printf("Cleaning %s...\n", dir)
	}
	fmt.Println("Clean complete.")
	return &common.ExecutionResult{ExitCode: 0}, nil
}

// UninstallData removes all pi-related XDG directories.
func (m *manager) UninstallData(force bool) (*common.ExecutionResult, error) {
	if !force {
		fmt.Print("This will delete ALL pi data (cache, config, state). Are you sure? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("Aborted.")
			return &common.ExecutionResult{ExitCode: 0}, nil
		}
	}
	removed := m.Uninstall()
	for _, dir := range removed {
		fmt.Printf("Removing %s...\n", dir)
	}
	fmt.Println("Uninstall complete. Local data removed.")
	return &common.ExecutionResult{ExitCode: 0}, nil
}

// GetInfo returns disk usage statistics for all pi directories.
func (m *manager) GetInfo() ([]Usage, int64) {
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
		size, count := DirSize(path)
		total += size
		stats = append(stats, Usage{
			Label: label,
			Size:  size,
			Items: count,
			Path:  path,
		})
	}
	return stats, total
}

// Clean removes temporary and cached data (packages, downloads, discovery).
func (m *manager) Clean() []string {
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
func (m *manager) Uninstall() []string {
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

// DirSize calculates the total size and item count of a directory.
func DirSize(path string) (int64, int) {
	var size int64
	var count int
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
			count++
		}
		return nil
	})
	return size, count
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
