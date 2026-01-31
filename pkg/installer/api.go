package installer

import (
	"context"
	"fmt"
	"pi/pkg/config"
	"pi/pkg/display"
	"pi/pkg/recipe"
	"os"
	"path/filepath"
)

// Plan contains all information needed to perform an installation.
type Plan struct {
	Package      recipe.PackageDefinition
	DownloadPath string
	ExtractPath  string
	// The final directory where the package is "installed"
	InstallPath string
}

// Stage represents a single step in the installation process.
type Stage func(ctx context.Context, plan *Plan, task display.Task) error

// NewPlan creates an installation plan for a package definition using provided config.
func NewPlan(cfg *config.Config, pkg recipe.PackageDefinition) (*Plan, error) {
	// Create subdirectories if they don't exist
	if err := os.MkdirAll(cfg.DownloadDir, 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.PkgDir, 0755); err != nil {
		return nil, err
	}

	// Determine download filename
	fileName := pkg.Filename
	if fileName == "" {
		fileName = filepath.Base(pkg.URL)
	}
	// If fileName is still empty or just /, use a fallback
	if fileName == "" || fileName == "." || fileName == "/" {
		fileName = fmt.Sprintf("%s-%s-%s-%s.bin", pkg.Name, pkg.Version, pkg.OS, pkg.Arch)
	}

	downloadPath := filepath.Join(cfg.DownloadDir, fileName)

	// Extraction path
	folderName := fmt.Sprintf("%s-%s-%s-%s", pkg.Name, pkg.Version, pkg.OS, pkg.Arch)
	installPath := filepath.Join(cfg.PkgDir, folderName)

	return &Plan{
		Package:      pkg,
		DownloadPath: downloadPath,
		InstallPath:  installPath,
	}, nil
}
