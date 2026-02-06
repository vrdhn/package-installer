package pkgs

import (
	"context"
	"fmt"
	sysconfig "pi/pkg/config"
	"pi/pkg/display"
	"pi/pkg/installer"
	"pi/pkg/recipe"
	"pi/pkg/repository"
	"pi/pkg/resolver"
	"strings"
)

// Mutable
type Manager struct {
	Repo      *repository.Manager
	Disp      display.Display
	SysConfig sysconfig.ReadOnly
}

// IndexEntry represents a lazy recipe registration entry.
type IndexEntry struct {
	Recipe   string
	Patterns []string
	Legacy   bool
}

func NewManager(repo *repository.Manager, disp display.Display, sysCfg sysconfig.ReadOnly) *Manager {
	return &Manager{
		Repo:      repo,
		Disp:      disp,
		SysConfig: sysCfg,
	}
}

// Prepare ensures all packages are installed and returns the required symlinks.
func (m *Manager) Prepare(ctx context.Context, pkgStrings []sysconfig.PkgRef) (*Result, error) {
	var allSymlinks []Symlink
	allEnv := make(map[string]string)

	for _, pkgStr := range pkgStrings {
		p, err := Parse(pkgStr)
		if err != nil {
			return nil, err
		}

		recipeName, regexKey, err := m.Repo.Resolve(p.Name, m.SysConfig)
		if err != nil {
			return nil, err
		}

		src, err := m.Repo.GetRecipe(recipeName)
		if err != nil {
			return nil, fmt.Errorf("error loading recipe for %s: %v", recipeName, err)
		}

		task := m.Disp.StartTask(p.String())
		recipeObj, err := recipe.NewStarlarkRecipe(recipeName, src, task.Log)
		if err != nil {
			task.Done()
			return nil, fmt.Errorf("error initializing recipe %s: %v", recipeName, err)
		}
		selected := recipe.NewSelectedRecipe(recipeObj, regexKey)

		// Resolve
		pkgDef, err := resolver.Resolve(ctx, m.SysConfig, selected, p.Name, p.Version, task)
		if err != nil {
			task.Done()
			return nil, fmt.Errorf("resolution failed for %s: %v", p.String(), err)
		}

		// Plan
		plan, err := installer.NewPlan(m.SysConfig, *pkgDef)
		if err != nil {
			task.Done()
			return nil, fmt.Errorf("planning failed for %s: %v", p.String(), err)
		}

		// Install
		if err := installer.Install(ctx, plan, task); err != nil {
			task.Done()
			return nil, fmt.Errorf("installation failed for %s: %v", p.String(), err)
		}

		// Discover symlinks
		links, err := DiscoverSymlinks(plan.InstallPath, pkgDef.Symlinks)
		if err != nil {
			task.Done()
			return nil, fmt.Errorf("failed to discover symlinks for %s: %v", p.String(), err)
		}
		allSymlinks = append(allSymlinks, links...)

		// Handle environment variables
		for k, v := range pkgDef.Env {
			// Replace ${PI_PKG_ROOT} with the actual install path
			// Inside the cave, we will bind the package directory to its host path.
			resolvedVal := strings.ReplaceAll(v, "${PI_PKG_ROOT}", plan.InstallPath)
			allEnv[k] = resolvedVal
		}
		task.Done()
	}

	return &Result{
		Symlinks: allSymlinks,
		Env:      allEnv,
		PkgDir:   m.SysConfig.GetPkgDir(),
		CacheDir: m.SysConfig.GetCacheDir(),
	}, nil
}

// List returns all versions of a package.
func (m *Manager) List(ctx context.Context, pkgStr string) ([]recipe.PackageDefinition, error) {
	p, err := Parse(pkgStr)
	if err != nil {
		return nil, err
	}

	recipeName, regexKey, err := m.Repo.Resolve(p.Name, m.SysConfig)
	if err != nil {
		return nil, err
	}

	src, err := m.Repo.GetRecipe(recipeName)
	if err != nil {
		return nil, fmt.Errorf("error loading recipe for %s: %v", recipeName, err)
	}

	task := m.Disp.StartTask("Listing " + p.String())
	recipeObj, err := recipe.NewStarlarkRecipe(recipeName, src, task.Log)
	if err != nil {
		task.Done()
		return nil, fmt.Errorf("error initializing recipe %s: %v", recipeName, err)
	}

	selected := recipe.NewSelectedRecipe(recipeObj, regexKey)
	pkgs, err := resolver.List(ctx, m.SysConfig, selected, p.Name, p.Version, task)
	task.Done()
	return pkgs, err
}

// ListIndex returns the registered package definitions for all recipes without executing handlers.
func (m *Manager) ListIndex(ctx context.Context) ([]IndexEntry, error) {
	var entries []IndexEntry
	for _, name := range m.Repo.ListRecipes() {
		src, err := m.Repo.GetRecipe(name)
		if err != nil {
			return nil, err
		}
		recipeObj, err := recipe.NewStarlarkRecipe(name, src, nil)
		if err != nil {
			return nil, err
		}

		patterns, legacy, err := recipeObj.Registry(m.SysConfig)
		if err != nil {
			return nil, err
		}

		entries = append(entries, IndexEntry{
			Recipe:   name,
			Patterns: patterns,
			Legacy:   legacy,
		})
	}
	return entries, nil
}
