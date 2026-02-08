// Package pkgs handles the resolution, installation, and management of individual packages.
// It coordinates between repositories, recipes, and the installer to ensure
// packages are correctly placed on disk and ready for use in sandboxes.
package pkgs

import (
	"context"
	"fmt"
	"path/filepath"
	"pi/pkg/common"
	"pi/pkg/config"
	"pi/pkg/display"
	"pi/pkg/installer"
	"pi/pkg/lazyjson"
	"pi/pkg/recipe"
	"pi/pkg/repository"
	"pi/pkg/resolver"
	"strings"
	"sync"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

// Manager defines the operations for managing package installations and synchronization.
type Manager interface {
	// SyncPkgs synchronizes package information from repositories matching the query.
	SyncPkgs(ctx context.Context, query string) (*common.ExecutionResult, error)
	// ListPkgs returns a list of installed or available packages matching the query.
	ListPkgs(ctx context.Context, query string, showAll bool) (*common.ExecutionResult, error)
	// Prepare ensures a set of packages are installed and returns instructions for sandboxing.
	Prepare(ctx context.Context, pkgStrings []config.PkgRef) (*Result, error)
	// ListFromSource executes a recipe to find all available versions of a package.
	ListFromSource(ctx context.Context, pkgStr string) ([]recipe.PackageDefinition, error)
	// Sync retrieves and indexes all versions for packages matching the query.
	Sync(ctx context.Context, query string) error
	// List returns the matching versions from the local package index.
	List(ctx context.Context, query string) ([]PackageDefinition, error)
	// ListIndex returns supported patterns for all known recipes without executing them.
	ListIndex(ctx context.Context) ([]RecipeIndexEntry, error)
	// UpdateVersions updates the local index for a specific repository and pattern.
	UpdateVersions(repoUUID uuid.UUID, pattern string, versions []recipe.PackageDefinition) error
}

type manager struct {
	Repo      repository.Manager
	Disp      display.Display
	SysConfig config.Config
	pkgMgr    *lazyjson.Manager[PackageRegistry]
}

// RecipeIndexEntry represents a discovery entry for a recipe and its supported patterns.
type RecipeIndexEntry struct {
	// Recipe is the name of the recipe file.
	Recipe string
	// Patterns is the list of package patterns supported by the recipe.
	Patterns []string
}

// NewManager creates a new package manager with the given repository manager and system config.
func NewManager(repo repository.Manager, disp display.Display, cfg config.Config) Manager {
	pkgPath := filepath.Join(cfg.GetConfigDir(), "package.json")
	return &manager{
		Repo:      repo,
		Disp:      disp,
		SysConfig: cfg,
		pkgMgr:    lazyjson.New[PackageRegistry](pkgPath),
	}
}

func (m *manager) SyncPkgs(ctx context.Context, query string) (*common.ExecutionResult, error) {
	if err := m.Repo.LoadRepos(); err != nil {
		return nil, err
	}
	err := m.Sync(ctx, query)
	if err != nil {
		return nil, err
	}

	// Print updated records
	versions, err := m.List(ctx, query)
	if err != nil {
		return nil, err
	}

	m.Disp.Close()
	SortPackageDefinitions(versions)

	m.Disp.Print(fmt.Sprintf("%-10s %-15s %-15s %-8s %-8s %-20s %s\n", "REPO", "NAME", "VERSION", "OS", "ARCH", "RELEASE", "DATE"))
	m.Disp.Print(fmt.Sprintln(strings.Repeat("-", 100)))
	for _, v := range versions {
		repo, _ := m.Repo.GetRepoByUUID(v.RepoUUID)
		m.Disp.Print(fmt.Sprintf("%-10s %-15s %-15s %-8s %-8s %-20s %s\n", repo.Name, v.Name, v.Version, v.OS, v.Arch, v.ReleaseStatus, v.ReleaseDate))
	}

	return &common.ExecutionResult{ExitCode: 0}, nil
}

func (m *manager) ListPkgs(ctx context.Context, query string, showAll bool) (*common.ExecutionResult, error) {
	if err := m.Repo.LoadRepos(); err != nil {
		return nil, err
	}
	versions, err := m.List(ctx, query)
	if err != nil {
		return nil, err
	}

	m.Disp.Close()

	myOS := m.SysConfig.GetOS()
	myArch := m.SysConfig.GetArch()

	if !showAll {
		var filtered []PackageDefinition
		for _, v := range versions {
			if v.OS == myOS && v.Arch == myArch {
				filtered = append(filtered, v)
			}
		}
		versions = filtered
	}

	SortPackageDefinitions(versions)

	// Show only most recent 5
	if len(versions) > 5 {
		versions = versions[len(versions)-5:]
	}

	m.Disp.Print(fmt.Sprintf("%-10s %-15s %-15s %-8s %-8s %-20s %s\n", "REPO", "NAME", "VERSION", "OS", "ARCH", "RELEASE", "DATE"))
	m.Disp.Print(fmt.Sprintln(strings.Repeat("-", 100)))
	for _, v := range versions {
		repo, _ := m.Repo.GetRepoByUUID(v.RepoUUID)
		m.Disp.Print(fmt.Sprintf("%-10s %-15s %-15s %-8s %-8s %-20s %s\n", repo.Name, v.Name, v.Version, v.OS, v.Arch, v.ReleaseStatus, v.ReleaseDate))
	}

	return &common.ExecutionResult{ExitCode: 0}, nil
}

// Prepare ensures all packages are installed and returns the required symlinks.
func (m *manager) Prepare(ctx context.Context, pkgStrings []config.PkgRef) (*Result, error) {
	var allSymlinks []Symlink
	allEnv := make(map[string]string)
	var mu sync.Mutex

	g, ctx := errgroup.WithContext(ctx)

	for _, pkgStr := range pkgStrings {
		pkgStr := pkgStr // capture for goroutine
		g.Go(func() error {
			p, err := Parse(pkgStr)
			if err != nil {
				return err
			}

			recipeName, regexKey, err := m.Repo.Resolve(p.Name, m.SysConfig)
			if err != nil {
				return err
			}

			src, err := m.Repo.GetRecipe(recipeName)
			if err != nil {
				return fmt.Errorf("error loading recipe for %s: %v", recipeName, err)
			}

			task := m.Disp.StartTask(p.String())
			recipeObj, err := recipe.NewStarlarkRecipe(recipeName, src, task.Log)
			if err != nil {
				task.Done()
				return fmt.Errorf("error initializing recipe %s: %v", recipeName, err)
			}
			selected := recipe.NewPinnedRecipe(recipeObj, regexKey)

			// Resolve
			pkgDef, err := resolver.Resolve(ctx, m.SysConfig, selected, p.Name, p.Version, task)
			if err != nil {
				task.Done()
				return fmt.Errorf("resolution failed for %s: %v", p.String(), err)
			}

			// Plan
			plan, err := installer.NewPlan(m.SysConfig, *pkgDef)
			if err != nil {
				task.Done()
				return fmt.Errorf("planning failed for %s: %v", p.String(), err)
			}

			// Install
			if err := installer.Install(ctx, plan, task); err != nil {
				task.Done()
				return fmt.Errorf("installation failed for %s: %v", p.String(), err)
			}

			// Discover symlinks
			links, err := DiscoverSymlinks(plan.InstallPath, pkgDef.Symlinks)
			if err != nil {
				task.Done()
				return fmt.Errorf("failed to discover symlinks for %s: %v", p.String(), err)
			}

			mu.Lock()
			allSymlinks = append(allSymlinks, links...)

			// Handle environment variables
			for k, v := range pkgDef.Env {
				// Replace ${PI_PKG_ROOT} with the actual install path
				// Inside the cave, we will bind the package directory to its host path.
				resolvedVal := strings.ReplaceAll(v, "${PI_PKG_ROOT}", plan.InstallPath)
				allEnv[k] = resolvedVal
			}
			mu.Unlock()

			task.Done()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return &Result{
		Symlinks: allSymlinks,
		Env:      allEnv,
		PkgDir:   m.SysConfig.GetPkgDir(),
		CacheDir: m.SysConfig.GetCacheDir(),
	}, nil
}

// ListFromSource returns all versions of a package by executing the recipe.
func (m *manager) ListFromSource(ctx context.Context, pkgStr string) ([]recipe.PackageDefinition, error) {
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

	selected := recipe.NewPinnedRecipe(recipeObj, regexKey)
	pkgs, err := resolver.List(ctx, m.SysConfig, selected, p.Name, p.Version, task)
	task.Done()
	return pkgs, err
}

// Sync retrieves all versions for matching packages and saves them to package.json.
func (m *manager) Sync(ctx context.Context, query string) error {
	matches, err := m.Repo.ResolveQuery(query)
	if err != nil {
		return err
	}

	if len(matches) == 0 {
		return fmt.Errorf("no packages matched query: %s", query)
	}

	for _, match := range matches {
		src, err := m.Repo.GetRecipe(match.RecipeName)
		if err != nil {
			return err
		}

		task := m.Disp.StartTask(fmt.Sprintf("Syncing %s/%s", match.RepoName, match.Pattern))
		recipeObj, err := recipe.NewStarlarkRecipe(match.RecipeName, src, task.Log)
		if err != nil {
			task.Done()
			return err
		}

		selected := recipe.NewPinnedRecipe(recipeObj, match.Pattern)
		// List with empty version to get all
		pkgs, err := resolver.List(ctx, m.SysConfig, selected, match.Pattern, "", task)
		if err != nil {
			task.Done()
			return err
		}

		task.Log(fmt.Sprintf("Found %d versions for %s", len(pkgs), match.Pattern))

		if err := m.UpdateVersions(match.RepoUUID, match.Pattern, pkgs); err != nil {
			task.Done()
			return err
		}
		task.Done()
	}

	return nil
}

func (m *manager) filterVersions(all []PackageDefinition, query string) []PackageDefinition {
	repoFilter := ""
	pkgFilter := query
	if strings.Contains(query, "/") {
		parts := strings.SplitN(query, "/", 2)
		repoFilter = parts[0]
		pkgFilter = parts[1]
	}

	var matches []PackageDefinition
	for _, v := range all {
		if repoFilter != "" {
			repo, ok := m.Repo.GetRepoByUUID(v.RepoUUID)
			if !ok || repo.Name != repoFilter {
				continue
			}
		}

		if pkgFilter != "" && v.Name != pkgFilter {
			continue
		}

		matches = append(matches, v)
	}
	return matches
}

// List returns the matching versions from package.json.
func (m *manager) List(ctx context.Context, query string) ([]PackageDefinition, error) {
	reg, err := m.pkgMgr.Get()
	if err != nil {
		return nil, err
	}
	all := reg.Versions

	matches := m.filterVersions(all, query)

	// Auto-sync if no matches found and a query was provided
	if len(matches) == 0 && query != "" {
		if err := m.Sync(ctx, query); err != nil {
			// If Sync fails, we return the error (e.g. no recipe found)
			return nil, err
		}

		// Reload after sync
		reg, err = m.pkgMgr.Get()
		if err != nil {
			return nil, err
		}
		all = reg.Versions
		matches = m.filterVersions(all, query)
	}

	return matches, nil
}

// ListIndex returns the registered package definitions for all recipes without executing handlers.
func (m *manager) ListIndex(ctx context.Context) ([]RecipeIndexEntry, error) {
	var entries []RecipeIndexEntry
	for _, name := range m.Repo.ListRecipes() {
		src, err := m.Repo.GetRecipe(name)
		if err != nil {
			return nil, err
		}
		recipeObj, err := recipe.NewStarlarkRecipe(name, src, nil)
		if err != nil {
			return nil, err
		}

		patterns, err := recipeObj.Registry(m.SysConfig)
		if err != nil {
			return nil, err
		}

		entries = append(entries, RecipeIndexEntry{
			Recipe:   name,
			Patterns: patterns,
		})
	}
	return entries, nil
}
