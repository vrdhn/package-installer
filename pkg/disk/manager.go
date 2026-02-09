// Package disk provides utilities for managing the local storage used by pi.
package disk

import (
	"fmt"
	"log/slog"
	"os"
	"pi/pkg/common"
	"strings"
)

func (m *manager) Info() (*common.ExecutionResult, error) {
	stats, total := m.GetInfo()
	m.Disp.Print(fmt.Sprintf("%-15s %-10s %-10s %s\n", "Type", "Size", "Items", "Path"))
	m.Disp.Print(fmt.Sprintln(strings.Repeat("-", 75)))
	for _, s := range stats {
		m.Disp.Print(fmt.Sprintf("%-15s %-10s %-10d %s\n", s.Label, FormatSize(s.Size), s.Items, s.Path))
	}
	m.Disp.Print(fmt.Sprintln(strings.Repeat("-", 75)))
	m.Disp.Print(fmt.Sprintf("%-15s %-10s\n", "Total", FormatSize(total)))
	return &common.ExecutionResult{ExitCode: 0}, nil
}

func (m *manager) CleanDir() (*common.ExecutionResult, error) {
	cleaned := m.Clean()
	for _, dir := range cleaned {
		slog.Info("Cleaning", "path", dir)
	}
	slog.Info("Clean complete")
	return &common.ExecutionResult{ExitCode: 0}, nil
}

func (m *manager) UninstallData(force bool) (*common.ExecutionResult, error) {
	if !force {
		m.Disp.Print("This will delete ALL pi data (cache, config, state). Are you sure? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			m.Disp.Print("Aborted.\n")
			return &common.ExecutionResult{ExitCode: 0}, nil
		}
	}
	removed := m.Uninstall()
	for _, dir := range removed {
		slog.Info("Removing", "path", dir)
	}
	slog.Info("Uninstall complete. Local data removed")
	return &common.ExecutionResult{ExitCode: 0}, nil
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
