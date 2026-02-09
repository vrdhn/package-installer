// Package installer handles the physical placement of packages on the host filesystem.
// It manages the download, extraction, and integrity verification of package archives.
package installer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"pi/pkg/config"
	"pi/pkg/recipe"
)

// Plan contains the full specification for a package installation.
type Plan struct {
	// Package is the metadata of the package being installed.
	Package recipe.PackageDefinition
	// DownloadPath is where the archive will be saved on the host.
	DownloadPath string
	// ExtractPath is a temporary location used during the extraction phase.
	ExtractPath string
	// InstallPath is the final destination directory for the package.
	InstallPath string
}

// Stage represents a single, atomic step in the multi-stage installation pipeline.
type Stage func(ctx context.Context, plan *Plan) error

// NewPlan calculates the filesystem paths and creates an installation plan
// based on the package definition and system configuration.
func NewPlan(cfg config.Config, pkg recipe.PackageDefinition) (*Plan, error) {
	// Create subdirectories if they don't exist
	if err := os.MkdirAll(cfg.GetDownloadDir(), 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.GetPkgDir(), 0755); err != nil {
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

	downloadPath := filepath.Join(cfg.GetDownloadDir(), fileName)

	// Extraction path
	folderName := fmt.Sprintf("%s-%s-%s-%s", pkg.Name, pkg.Version, pkg.OS, pkg.Arch)
	installPath := filepath.Join(cfg.GetPkgDir(), folderName)

	return &Plan{
		Package:      pkg,
		DownloadPath: downloadPath,
		InstallPath:  installPath,
	}, nil
}
