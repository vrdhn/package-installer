// Package pkgs handles the resolution, installation, and management of individual packages.
package pkgs

import (
	"context"
	"fmt"
	"log/slog"
	"pi/pkg/common"
	"pi/pkg/config"
	"pi/pkg/installer"
	"pi/pkg/recipe"
	"pi/pkg/repo"
	"pi/pkg/resolver"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
)

func (m *manager) SyncPkgs(ctx context.Context, query string) (*common.ExecutionResult, error) {
	if err := m.Repo.LoadRepos(); err != nil {
		return nil, err
	}
	if err := m.Sync(ctx, query); err != nil {
		return nil, err
	}

	versions, err := m.List(ctx, query)
	if err != nil {
		return nil, err
	}

	return m.renderPackageTable(versions), nil
}

func (m *manager) ListPkgs(ctx context.Context, query string, showAll bool) (*common.ExecutionResult, error) {
	if err := m.Repo.LoadRepos(); err != nil {
		return nil, err
	}
	versions, err := m.List(ctx, query)
	if err != nil {
		return nil, err
	}

	if !showAll {
		myOS := m.Config.GetOS()
		myArch := m.Config.GetArch()
		var filtered []PackageDefinition
		for _, v := range versions {
			if v.OS == myOS && v.Arch == myArch {
				filtered = append(filtered, v)
			}
		}
		versions = filtered
	}

	SortPackageDefinitions(versions)

	if len(versions) > 5 {
		versions = versions[len(versions)-5:]
	}

	return m.renderPackageTable(versions), nil
}

func (m *manager) renderPackageTable(versions []PackageDefinition) *common.ExecutionResult {
	table := &common.Table{
		Header: []string{"REPO", "NAME", "VERSION", "OS", "ARCH", "RELEASE", "DATE"},
	}
	for _, v := range versions {
		repo, _ := m.Repo.GetRepoByUUID(v.RepoUUID)
		table.Rows = append(table.Rows, []string{
			repo.Name, v.Name, v.Version, v.OS.String(), v.Arch.String(), v.ReleaseStatus, v.ReleaseDate,
		})
	}
	return &common.ExecutionResult{
		Output: &common.Output{
			Table: table,
		},
	}
}

func (m *manager) Prepare(ctx context.Context, pkgStrings []config.PkgRef) (*common.PreparationResult, error) {
	var allSymlinks []common.Symlink
	allEnv := make(map[string]string)
	var mu sync.Mutex

	g, ctx := errgroup.WithContext(ctx)

	for _, pkgStr := range pkgStrings {
		pkgStr := pkgStr
		g.Go(func() error {
			links, env, err := m.preparePkg(ctx, pkgStr)
			if err != nil {
				return err
			}
			mu.Lock()
			allSymlinks = append(allSymlinks, links...)
			for k, v := range env {
				allEnv[k] = v
			}
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return &common.PreparationResult{
		Symlinks: allSymlinks,
		Env:      allEnv,
		PkgDir:   m.Config.GetPkgDir(),
		CacheDir: m.Config.GetCacheDir(),
	}, nil
}

func (m *manager) preparePkg(ctx context.Context, pkgStr config.PkgRef) ([]common.Symlink, map[string]string, error) {
	p, err := Parse(pkgStr)
	if err != nil {
		return nil, nil, err
	}

	recipeName, regexKey, err := m.Repo.Resolve(p.Name, m.Config)
	if err != nil {
		return nil, nil, err
	}

	src, err := m.Repo.GetRecipe(recipeName)
	if err != nil {
		return nil, nil, fmt.Errorf("error loading recipe for %s: %v", recipeName, err)
	}

	slog.Info("Preparing package", "package", p.String())

	recipeObj, err := recipe.NewStarlarkRecipe(recipeName, src, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("error initializing recipe %s: %v", recipeName, err)
	}
	selected := recipe.NewPinnedRecipe(recipeObj, regexKey)

	pkgDef, err := resolver.Resolve(ctx, m.Config, selected, p.Name, p.Version)
	if err != nil {
		return nil, nil, fmt.Errorf("resolution failed for %s: %v", p.String(), err)
	}

	plan, err := installer.NewPlan(m.Config, *pkgDef)
	if err != nil {
		return nil, nil, fmt.Errorf("planning failed for %s: %v", p.String(), err)
	}

	if err := installer.Install(ctx, plan); err != nil {
		return nil, nil, fmt.Errorf("installation failed for %s: %v", p.String(), err)
	}

	links, err := DiscoverSymlinks(plan.InstallPath, pkgDef.Symlinks)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to discover symlinks for %s: %v", p.String(), err)
	}

	env := make(map[string]string)
	for k, v := range pkgDef.Env {
		env[k] = strings.ReplaceAll(v, "${PI_PKG_ROOT}", plan.InstallPath)
	}

	return links, env, nil
}

func (m *manager) ListFromSource(ctx context.Context, pkgStr string) ([]recipe.PackageDefinition, error) {
	p, err := Parse(pkgStr)
	if err != nil {
		return nil, err
	}

	recipeName, regexKey, err := m.Repo.Resolve(p.Name, m.Config)
	if err != nil {
		return nil, err
	}

	src, err := m.Repo.GetRecipe(recipeName)
	if err != nil {
		return nil, fmt.Errorf("error loading recipe for %s: %v", recipeName, err)
	}

	recipeObj, err := recipe.NewStarlarkRecipe(recipeName, src, nil)
	if err != nil {
		return nil, fmt.Errorf("error initializing recipe %s: %v", recipeName, err)
	}

	selected := recipe.NewPinnedRecipe(recipeObj, regexKey)
	return resolver.List(ctx, m.Config, selected, p.Name, p.Version)
}

func (m *manager) Sync(ctx context.Context, query string) error {
	matches, err := m.Repo.ResolveQuery(query)
	if err != nil {
		return err
	}

	if len(matches) == 0 {
		return fmt.Errorf("no packages matched query: %s", query)
	}

	for _, match := range matches {
		if err := m.syncMatch(ctx, match); err != nil {
			return err
		}
	}

	return nil
}

func (m *manager) syncMatch(ctx context.Context, match repo.ResolvedQuery) error {
	src, err := m.Repo.GetRecipe(match.RecipeName)
	if err != nil {
		return err
	}

	slog.Info("Syncing package", "repo", match.RepoName, "pattern", match.Pattern)

	recipeObj, err := recipe.NewStarlarkRecipe(match.RecipeName, src, nil)
	if err != nil {
		return err
	}

	selected := recipe.NewPinnedRecipe(recipeObj, match.Pattern)
	pkgs, err := resolver.List(ctx, m.Config, selected, match.Pattern, "")
	if err != nil {
		return err
	}

	slog.Debug("Found versions", "count", len(pkgs), "pattern", match.Pattern)

	return m.UpdateVersions(match.RepoUUID, match.Pattern, pkgs)
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

func (m *manager) List(ctx context.Context, query string) ([]PackageDefinition, error) {
	reg, err := m.pkgMgr.Get()
	if err != nil {
		return nil, err
	}
	all := reg.Versions

	matches := m.filterVersions(all, query)

	if len(matches) == 0 && query != "" {
		if err := m.Sync(ctx, query); err != nil {
			return nil, err
		}

		reg, err = m.pkgMgr.Get()
		if err != nil {
			return nil, err
		}
		all = reg.Versions
		matches = m.filterVersions(all, query)
	}

	return matches, nil
}

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

		patterns, err := recipeObj.Registry(m.Config)
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
