// Package disk provides utilities for managing the local storage used by pi.
package disk

import (
	"fmt"
	"log/slog"
	"os"
	"pi/pkg/common"
)

func (m *manager) Info() (*common.ExecutionResult, error) {
	stats, total := m.GetInfo()
	table := &common.Table{
		Header: []string{"Type", "Size", "Items", "Path"},
	}
	for _, s := range stats {
		table.Rows = append(table.Rows, []string{s.Label, FormatSize(s.Size), fmt.Sprintf("%d", s.Items), s.Path})
	}

	return &common.ExecutionResult{
		Output: &common.Output{
			Table:   table,
			Message: fmt.Sprintf("Total: %s", FormatSize(total)),
		},
	}, nil
}

func (m *manager) CleanDir() (*common.ExecutionResult, error) {
	cleaned := m.Clean()
	for _, dir := range cleaned {
		slog.Info("Cleaning", "path", dir)
	}
	return &common.ExecutionResult{
		Output: &common.Output{
			Message: "Clean complete",
		},
	}, nil
}

func (m *manager) UninstallData() (*common.ExecutionResult, error) {
	removed := m.Uninstall()
	for _, dir := range removed {
		slog.Info("Removing", "path", dir)
	}
	return &common.ExecutionResult{
		Output: &common.Output{
			Message: "Uninstall complete. Local data removed",
		},
	}, nil
}

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
