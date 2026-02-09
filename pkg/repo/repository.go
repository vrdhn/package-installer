// Package repo manages the discovery and indexing of package recipes.
package repo

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"pi/pkg/common"
	"pi/pkg/config"
	"pi/pkg/lazyjson"
	"pi/pkg/recipe"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// Manager is a pointer to the internal manager implementation.
type Manager = *manager

// NewManager creates a new repository manager with the provided configuration.
func NewManager(cfg config.Config) Manager {
	repoPath := filepath.Join(cfg.GetConfigDir(), "repo.json")
	return &manager{
		recipes:          make(map[string]string),
		resolveCache:     make(map[string]resolvedRecipe),
		compiledPatterns: make(map[string]*regexp.Regexp),
		cfg:              cfg,
		repoMgr:          lazyjson.New[repoRegistry](repoPath),
	}
}

func (m *manager) List(verbose bool) (*common.ExecutionResult, error) {
	entries, err := m.GetFullRegistryInfo(verbose)
	if err != nil {
		return nil, err
	}
	return m.renderRegistryTable(entries), nil
}

func (m *manager) renderRegistryTable(entries []IndexEntry) *common.ExecutionResult {
	table := &common.Table{
		Header: []string{"REPO", "RECIPE", "PATTERN", "HANDLER"},
	}
	for _, e := range entries {
		table.Rows = append(table.Rows, []string{e.RepoName, e.RecipeName, e.Pattern, e.Handler})
	}
	return &common.ExecutionResult{
		Output: &common.Output{
			Table: table,
		},
	}
}

func (m *manager) Add(path string, verbose bool) (*common.ExecutionResult, error) {
	if err := m.AddLocalRepo(path, verbose); err != nil {
		return nil, err
	}
	return &common.ExecutionResult{
		Output: &common.Output{
			Message: fmt.Sprintf("Added repository at %s", path),
		},
	}, nil
}

func (m *manager) SyncRepo(verbose bool) (*common.ExecutionResult, error) {
	if err := m.Sync(verbose); err != nil {
		return nil, err
	}
	return &common.ExecutionResult{
		Output: &common.Output{
			Message: "Package index synchronized successfully",
		},
	}, nil
}

func (m *manager) Sync(verbose bool) error {
	if err := m.LoadRepos(); err != nil {
		return err
	}
	slog.Debug("Regenerating index")

	var entries []IndexEntry
	for _, repo := range m.repos {
		repoEntries, err := m.syncRepo(repo, verbose)
		if err != nil {
			return err
		}
		entries = append(entries, repoEntries...)
	}

	if err := m.updateIndex(entries); err != nil {
		return err
	}

	m.index = entries
	m.clearCaches()

	slog.Debug("Sync complete", "patterns", len(entries))
	return nil
}

func (m *manager) syncRepo(repo RepoConfig, verbose bool) ([]IndexEntry, error) {
	switch {
	case repo.URL == "builtin://":
		return m.syncBuiltinRepo(repo)
	case strings.HasPrefix(repo.URL, "http://") || strings.HasPrefix(repo.URL, "https://"):
		return nil, fmt.Errorf("remote url syncing is WIP: %s", repo.URL)
	default:
		return m.syncLocalRepo(repo, verbose)
	}
}

func (m *manager) syncBuiltinRepo(repo RepoConfig) ([]IndexEntry, error) {
	slog.Debug("Indexing builtin recipes")
	var entries []IndexEntry
	err := fs.WalkDir(recipe.BuiltinRecipes, "recipes", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".star") {
			return err
		}
		name := strings.TrimSuffix(filepath.Base(path), ".star")
		content, err := fs.ReadFile(recipe.BuiltinRecipes, path)
		if err != nil {
			return err
		}
		regInfo, err := m.GetRecipeRegistryInfo(name, string(content))
		if err != nil {
			return err
		}
		for p, h := range regInfo {
			entries = append(entries, IndexEntry{
				RepoUUID:   repo.UUID,
				RepoName:   repo.Name,
				RecipeName: name,
				Pattern:    p,
				Handler:    h,
			})
		}
		return nil
	})
	return entries, err
}

func (m *manager) syncLocalRepo(repo RepoConfig, verbose bool) ([]IndexEntry, error) {
	path := repo.URL
	if path == "" {
		return nil, nil
	}
	if verbose {
		slog.Debug("Indexing repo", "name", repo.Name, "path", path)
	}
	var entries []IndexEntry
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".star") {
			return err
		}
		name := strings.TrimSuffix(filepath.Base(p), ".star")
		content, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		regInfo, err := m.GetRecipeRegistryInfo(name, string(content))
		if err != nil {
			return err
		}
		for pat, h := range regInfo {
			entries = append(entries, IndexEntry{
				RepoUUID:   repo.UUID,
				RepoName:   repo.Name,
				RecipeName: name,
				Pattern:    pat,
				Handler:    h,
			})
		}
		return nil
	})
	return entries, err
}

func (m *manager) updateIndex(entries []IndexEntry) error {
	err := m.repoMgr.Modify(func(reg *repoRegistry) error {
		reg.Index = entries
		return nil
	})
	if err != nil {
		return err
	}
	return m.repoMgr.Save()
}

func (m *manager) clearCaches() {
	m.compiledPatterns = make(map[string]*regexp.Regexp)
	m.resolveCache = make(map[string]resolvedRecipe)
}

func (m *manager) AddLocalRepo(path string, verbose bool) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	if err := m.validateLocalRepo(absPath); err != nil {
		return err
	}

	if err := m.checkDuplicateRepo(absPath); err != nil {
		return err
	}

	newRepo := RepoConfig{
		Name: filepath.Base(absPath),
		URL:  absPath,
		UUID: generateUUID(),
	}

	if err := m.registerRepo(newRepo); err != nil {
		return err
	}

	return m.Sync(verbose)
}

func (m *manager) checkDuplicateRepo(path string) error {
	for _, r := range m.repos {
		if r.URL == path {
			return fmt.Errorf("repository already exists with path: %s", path)
		}
	}
	return nil
}

func (m *manager) validateLocalRepo(path string) error {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("repository path is not a valid directory: %s", path)
	}

	hasStar := false
	err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil || hasStar {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(p, ".star") {
			hasStar = true
			return filepath.SkipAll
		}
		return nil
	})
	if !hasStar {
		return fmt.Errorf("no .star files found in %s", path)
	}
	return err
}

func (m *manager) registerRepo(repo RepoConfig) error {
	err := m.repoMgr.Modify(func(reg *repoRegistry) error {
		reg.Repos = append(reg.Repos, repo)
		return nil
	})
	if err != nil {
		return err
	}
	if err := m.repoMgr.Save(); err != nil {
		return err
	}
	return m.LoadRepos()
}

func (m *manager) GetRecipe(name string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	content, ok := m.recipes[name]
	if !ok {
		return "", fmt.Errorf("recipe not found: %s", name)
	}
	return content, nil
}

func (m *manager) ListRecipes() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []string
	for name := range m.recipes {
		list = append(list, name)
	}
	return list
}

func (m *manager) Resolve(pkgName string, cfg config.Config) (string, string, error) {
	if err := m.ensureIndexLoaded(); err != nil {
		return "", "", err
	}

	m.mu.RLock()
	if res, ok := m.resolveCache[pkgName]; ok {
		m.mu.RUnlock()
		return res.recipeName, res.regexKey, nil
	}
	m.mu.RUnlock()

	originalPkgName := pkgName
	repoFilter, pkgName := m.parsePackageQuery(pkgName)

	matches, err := m.findMatches(repoFilter, pkgName)
	if err != nil {
		return "", "", err
	}

	if len(matches) == 0 {
		return "", "", fmt.Errorf("no recipe matched: %s", originalPkgName)
	}
	if len(matches) > 1 {
		return "", "", m.ambiguityError(originalPkgName, matches)
	}

	m.mu.Lock()
	m.resolveCache[originalPkgName] = resolvedRecipe{
		recipeName: matches[0].recipe,
		regexKey:   matches[0].regex,
	}
	m.mu.Unlock()

	return matches[0].recipe, matches[0].regex, nil
}

func (m *manager) parsePackageQuery(query string) (string, string) {
	parts := strings.Split(query, "/")
	if len(parts) >= 2 {
		if _, isRepo := m.GetRepoByName(parts[0]); isRepo {
			return parts[0], strings.Join(parts[1:], "/")
		}
	}
	return "", query
}

type match struct {
	repo   string
	recipe string
	regex  string
}

func (m *manager) findMatches(repoFilter, pkgName string) ([]match, error) {
	var matches []match
	for _, entry := range m.index {
		if repoFilter != "" && entry.RepoName != repoFilter {
			continue
		}
		re, err := m.getCompiledPattern(entry.Pattern)
		if err != nil {
			continue
		}
		if re.MatchString(pkgName) {
			matches = append(matches, match{
				repo:   entry.RepoName,
				recipe: entry.RecipeName,
				regex:  entry.Pattern,
			})
		}
	}
	return matches, nil
}

func (m *manager) getCompiledPattern(pattern string) (*regexp.Regexp, error) {
	m.mu.RLock()
	if re, ok := m.compiledPatterns[pattern]; ok {
		m.mu.RUnlock()
		return re, nil
	}
	m.mu.RUnlock()

	re, err := recipe.CompileAnchored(pattern)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.compiledPatterns[pattern] = re
	m.mu.Unlock()

	return re, nil
}

func (m *manager) ambiguityError(query string, matches []match) error {
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].repo != matches[j].repo {
			return matches[i].repo < matches[j].repo
		}
		if matches[i].recipe != matches[j].recipe {
			return matches[i].recipe < matches[j].recipe
		}
		return matches[i].regex < matches[j].regex
	})
	var lines []string
	for _, m := range matches {
		lines = append(lines, fmt.Sprintf("  %s/%s  %s", m.repo, m.recipe, m.regex))
	}
	return fmt.Errorf("ambiguous package match for %s:\n%s", query, strings.Join(lines, "\n"))
}

func (m *manager) ResolveQuery(query string) ([]ResolvedQuery, error) {
	if err := m.ensureIndexLoaded(); err != nil {
		return nil, err
	}

	repoFilter, pkgFilter := m.parsePackageQuery(query)
	var results []ResolvedQuery
	for _, entry := range m.index {
		if repoFilter != "" && entry.RepoName != repoFilter {
			continue
		}

		if pkgFilter != "" {
			re, err := m.getCompiledPattern(entry.Pattern)
			if err != nil {
				continue
			}
			if re.MatchString(pkgFilter) {
				results = append(results, m.toResolvedQuery(entry))
			}
		} else {
			results = append(results, m.toResolvedQuery(entry))
		}
	}

	if repoFilter == "" && pkgFilter != "" && m.hasAmbiguousRepos(results) {
		return nil, m.queryAmbiguityError(query, results)
	}

	return results, nil
}

func (m *manager) toResolvedQuery(entry IndexEntry) ResolvedQuery {
	return ResolvedQuery{
		RepoUUID:   entry.RepoUUID,
		RepoName:   entry.RepoName,
		RecipeName: entry.RecipeName,
		Pattern:    entry.Pattern,
	}
}

func (m *manager) hasAmbiguousRepos(results []ResolvedQuery) bool {
	if len(results) <= 1 {
		return false
	}
	repos := make(map[string]bool)
	for _, r := range results {
		repos[r.RepoName] = true
	}
	return len(repos) > 1
}

func (m *manager) queryAmbiguityError(query string, results []ResolvedQuery) error {
	var lines []string
	for _, r := range results {
		lines = append(lines, fmt.Sprintf("  %s/%s (%s)", r.RepoName, r.RecipeName, r.Pattern))
	}
	return fmt.Errorf("ambiguous package match for %q:\n%s", query, strings.Join(lines, "\n"))
}

func generateUUID() uuid.UUID {
	return uuid.New()
}
