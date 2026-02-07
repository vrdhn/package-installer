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
	"sync"

	"golang.org/x/sync/errgroup"
)

// Mutable
type manager struct {
	Repo      repository.Manager
	Disp      display.Display
	SysConfig sysconfig.Config
}

type Manager = *manager

// IndexEntry represents a lazy recipe registration entry.
type IndexEntry struct {
	Recipe   string
	Patterns []string
	Legacy   bool
}

func NewManager(repo repository.Manager, disp display.Display, config sysconfig.Config) Manager {
	return &manager{
		Repo:      repo,
		Disp:      disp,
		SysConfig: config,
	}
}

// Prepare ensures all packages are installed and returns the required symlinks.
func (m *manager) Prepare(ctx context.Context, pkgStrings []sysconfig.PkgRef) (*Result, error) {
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
			selected := recipe.NewSelectedRecipe(recipeObj, regexKey)

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

	selected := recipe.NewSelectedRecipe(recipeObj, regexKey)
	pkgs, err := resolver.List(ctx, m.SysConfig, selected, p.Name, p.Version, task)
	task.Done()
	return pkgs, err
}

// Sync retrieves all versions for matching packages and saves them to repo.csv.
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

		selected := recipe.NewSelectedRecipe(recipeObj, match.Pattern)
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

func (m *manager) filterVersions(all []PackageVersion, query string) []PackageVersion {
	repoFilter := ""
	pkgFilter := query
	if strings.Contains(query, "/") {
		parts := strings.SplitN(query, "/", 2)
		repoFilter = parts[0]
		pkgFilter = parts[1]
	}

	var matches []PackageVersion
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

// List returns the matching versions from repo.csv.
func (m *manager) List(ctx context.Context, query string) ([]PackageVersion, error) {
	all, err := m.loadRepoCSV()
	if err != nil {
		return nil, err
	}

	matches := m.filterVersions(all, query)

	// Auto-sync if no matches found and a query was provided
	if len(matches) == 0 && query != "" {
		if err := m.Sync(ctx, query); err != nil {
			// If Sync fails, we return the error (e.g. no recipe found)
			return nil, err
		}

		// Reload after sync
		all, err = m.loadRepoCSV()
		if err != nil {
			return nil, err
		}
		matches = m.filterVersions(all, query)
	}

	return matches, nil
}

// ListIndex returns the registered package definitions for all recipes without executing handlers.
func (m *manager) ListIndex(ctx context.Context) ([]IndexEntry, error) {
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
