package pkgs

import (
	"path/filepath"
	"pi/pkg/common"
	"pi/pkg/config"
	"pi/pkg/display"
	"pi/pkg/lazyjson"
	"pi/pkg/repo"
	"sort"
)

// manager defines the internal state for managing package installations.
type manager struct {
	Repo   repo.Manager
	Disp   display.Display
	Config config.Config
	pkgMgr lazyjson.Manager[PackageRegistry]
}

// Manager is a pointer to the internal manager implementation.
type Manager = *manager

// RecipeIndexEntry represents a discovery entry for a recipe and its supported patterns.
type RecipeIndexEntry struct {
	// Recipe is the name of the recipe file.
	Recipe string
	// Patterns is the list of package patterns supported by the recipe.
	Patterns []string
}

// PackageDefinition is an alias for common.PackageDefinition for convenience.
type PackageDefinition = common.PackageDefinition

type PackageRegistry struct {
	Versions []PackageDefinition `json:"versions"`
}

// NewManager creates a new package manager with the given repository manager and system config.
func NewManager(repo repo.Manager, disp display.Display, cfg config.Config) Manager {
	pkgPath := filepath.Join(cfg.GetConfigDir(), "package.json")
	return &manager{
		Repo:   repo,
		Disp:   disp,
		Config: cfg,
		pkgMgr: lazyjson.New[PackageRegistry](pkgPath),
	}
}

func SortPackageDefinitions(versions []PackageDefinition) {
	sort.Slice(versions, func(i, j int) bool {
		// Compare timestamps if available
		if versions[i].ReleaseDate != "" && versions[j].ReleaseDate != "" {
			return versions[i].ReleaseDate < versions[j].ReleaseDate
		}
		// Fallback to version string
		return versions[i].Version < versions[j].Version
	})
}
