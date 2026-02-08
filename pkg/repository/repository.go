// Package repository manages the discovery and indexing of package recipes.
// It handles multiple recipe repositories (local or remote) and maintains
// a compiled index for fast package-to-recipe resolution.
package repository

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"pi/pkg/common"
	"pi/pkg/config"
	"pi/pkg/display"
	"pi/pkg/lazyjson"
	"pi/pkg/recipe"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// RepoConfig represents the configuration for a single recipe repository.
type RepoConfig struct {
	// Name is the display name of the repository.
	Name string `json:"name"`
	// URL is the location of the repository (local path or remote URL).
	URL string `json:"url"`
	// UUID is a unique identifier for the repository.
	UUID uuid.UUID `json:"uuid"`
}

// IndexEntry represents a mapping from a package pattern to a specific recipe and handler.
type IndexEntry struct {
	// RepoUUID is the identifier of the repository where the recipe is located.
	RepoUUID uuid.UUID `json:"repo_uuid"`
	// RepoName is the name of the repository (resolved at runtime).
	RepoName string `json:"-"`
	// RecipeName is the name of the Starlark recipe file (without .star).
	RecipeName string `json:"recipe_name"`
	// Pattern is the regex pattern that this handler supports.
	Pattern string `json:"pattern"`
	// Handler is the name of the Starlark function that handles this pattern.
	Handler string `json:"handler"`
}

type repoRegistry struct {
	Repos []RepoConfig `json:"repos"`
	Index []IndexEntry `json:"index"`
}

type resolvedRecipe struct {
	recipeName string
	regexKey   string
}

// Manager defines the operations for managing repositories and resolving packages.
type manager struct {
	recipes map[string]string // recipe name -> source
	repos   []RepoConfig
	disp    display.Display
	cfg     config.Config
	repoMgr *lazyjson.Manager[repoRegistry]

	// In-memory index of all packages
	index []IndexEntry
	// Cache for resolution: pkgName -> {recipeName, regexKey}
	resolveCache     map[string]resolvedRecipe
	compiledPatterns map[string]*regexp.Regexp
}

// Manager is a pointer to the internal manager implementation.
type Manager = *manager

// NewManager creates a new repository manager with the provided display and configuration.
func NewManager(disp display.Display, cfg config.Config) Manager {
	repoPath := filepath.Join(cfg.GetConfigDir(), "repo.json")
	m := &manager{
		recipes:          make(map[string]string),
		resolveCache:     make(map[string]resolvedRecipe),
		compiledPatterns: make(map[string]*regexp.Regexp),
		disp:             disp,
		cfg:              cfg,
		repoMgr:          lazyjson.New[repoRegistry](repoPath),
	}

	return m
}

func (m *manager) List(verbose bool) (*common.ExecutionResult, error) {
	m.disp.Close()

	entries, err := m.GetFullRegistryInfo(verbose)
	if err != nil {
		return nil, err
	}

	m.DisplayRegistryInfo(entries)

	return &common.ExecutionResult{ExitCode: 0}, nil
}

func (m *manager) Add(path string, verbose bool) (*common.ExecutionResult, error) {
	if err := m.AddLocalRepo(path, verbose); err != nil {
		return nil, err
	}
	slog.Info("Added repository", "path", path)
	return &common.ExecutionResult{ExitCode: 0}, nil
}

func (m *manager) SyncRepo(verbose bool) (*common.ExecutionResult, error) {
	if err := m.Sync(verbose); err != nil {
		return nil, err
	}
	slog.Info("Package index synchronized successfully")
	return &common.ExecutionResult{ExitCode: 0}, nil
}

func (m *manager) loadBuiltins() error {
	return fs.WalkDir(recipe.BuiltinRecipes, "recipes", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".star") {
			name := strings.TrimSuffix(filepath.Base(path), ".star")
			m.disp.Log(fmt.Sprintf("Loading builtin recipe: %s", name))
			content, err := fs.ReadFile(recipe.BuiltinRecipes, path)
			if err != nil {
				return err
			}
			m.recipes[name] = string(content)
		}
		return nil
	})
}

func (m *manager) LoadRepos() error {
	err := m.repoMgr.Modify(func(reg *repoRegistry) error {
		hasBuiltin := false
		for i := range reg.Repos {
			if reg.Repos[i].Name == "builtin" || reg.Repos[i].URL == "builtin://" {
				hasBuiltin = true
				if reg.Repos[i].UUID == uuid.Nil {
					reg.Repos[i].UUID = generateUUID()
				}
				break
			}
		}
		if !hasBuiltin {
			reg.Repos = append([]RepoConfig{{
				Name: "builtin",
				URL:  "builtin://",
				UUID: generateUUID(),
			}}, reg.Repos...)
		}
		return nil
	})
	if err != nil {
		return err
	}

	reg, err := m.repoMgr.Get()
	if err != nil {
		return err
	}

	m.repos = nil
	for _, repo := range reg.Repos {
		if repo.UUID != uuid.Nil && repo.Name != "" {
			m.repos = append(m.repos, repo)
		}
	}

	// TODO: Fetch recipes from URLs.
	if err := m.loadBuiltins(); err != nil {
		return err
	}

	for _, repo := range m.repos {
		if repo.URL == "builtin://" {
			continue // already loaded by loadBuiltins
		}
		if strings.HasPrefix(repo.URL, "http://") || strings.HasPrefix(repo.URL, "https://") {
			// Remote URL - WIP
			continue
		}
		// Assume local path if no remote scheme
		m.loadLocalRepo(repo)
	}

	return nil
}

func (m *manager) loadLocalRepo(repo RepoConfig) {
	err := filepath.Walk(repo.URL, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".star") {
			name := strings.TrimSuffix(filepath.Base(path), ".star")
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			m.recipes[name] = string(content)
		}
		return nil
	})
	if err != nil {
		m.disp.Log(fmt.Sprintf("Failed to load local repo %s: %v", repo.Name, err))
	}
}

func (m *manager) LoadIndex() error {
	reg, err := m.repoMgr.Get()
	if err != nil {
		return err
	}

	m.index = nil
	for _, entry := range reg.Index {
		e := entry
		e.RepoName = "unknown"
		for _, repo := range m.repos {
			if repo.UUID == e.RepoUUID {
				e.RepoName = repo.Name
				break
			}
		}
		m.index = append(m.index, e)
	}
	return nil
}

func (m *manager) ensureIndexLoaded() error {
	if err := m.LoadRepos(); err != nil {
		return err
	}
	if len(m.index) == 0 {
		if err := m.LoadIndex(); err != nil {
			return err
		}
		if len(m.index) == 0 {
			// Auto-sync if still empty (likely fresh system)
			if err := m.Sync(false); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *manager) Sync(verbose bool) error {
	if err := m.LoadRepos(); err != nil {
		return err
	}
	slog.Debug("Regenerating index")

	var entries []IndexEntry

	for _, repo := range m.repos {
		if repo.URL == "builtin://" {
			slog.Debug("Indexing builtin recipes")
			err := fs.WalkDir(recipe.BuiltinRecipes, "recipes", func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !d.IsDir() && strings.HasSuffix(path, ".star") {
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
				}
				return nil
			})
			if err != nil {
				return err
			}
		} else if strings.HasPrefix(repo.URL, "http://") || strings.HasPrefix(repo.URL, "https://") {
			return fmt.Errorf("remote url syncing is WIP: %s", repo.URL)
		} else {
			// Local path
			path := repo.URL
			if path == "" {
				// Fallback or skip? Assuming path is in URL field as per instructions
				continue
			}
			if verbose {
				slog.Debug("Indexing repo", "name", repo.Name, "path", path)
			}
			err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && strings.HasSuffix(p, ".star") {
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
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
	}

	err := m.repoMgr.Modify(func(reg *repoRegistry) error {
		reg.Index = entries
		return nil
	})
	if err != nil {
		return err
	}

	if err := m.repoMgr.Save(); err != nil {
		return err
	}

	m.index = entries
	// Clear caches as patterns might have changed
	m.compiledPatterns = make(map[string]*regexp.Regexp)
	m.resolveCache = make(map[string]resolvedRecipe)

	slog.Debug("Sync complete", "patterns", len(entries))
	return nil
}

func (m *manager) AddLocalRepo(path string, verbose bool) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	slog.Debug("Statting", "path", absPath)
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("repository path does not exist: %s", absPath)
	}
	if !info.IsDir() {
		return fmt.Errorf("repository path is not a directory: %s", absPath)
	}

	hasStar := false
	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".star") {
			hasStar = true
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return err
	}
	if !hasStar {
		return fmt.Errorf("no .star files found in %s", absPath)
	}

	for _, r := range m.repos {
		if r.URL == absPath {
			return fmt.Errorf("repository already exists with path: %s", absPath)
		}
	}

	uuid := generateUUID()

	name := filepath.Base(absPath)

	slog.Debug("Found repository", "path", absPath, "uuid", uuid, "name", name)

	newRepo := RepoConfig{
		Name: name,
		URL:  absPath,
		UUID: uuid,
	}

	err = m.repoMgr.Modify(func(reg *repoRegistry) error {
		reg.Repos = append(reg.Repos, newRepo)
		return nil
	})
	if err != nil {
		return err
	}

	if err := m.repoMgr.Save(); err != nil {
		return err
	}

	// Refresh in-memory list
	if err := m.LoadRepos(); err != nil {
		return err
	}

	// Update index
	return m.Sync(verbose)
}

func (m *manager) GetRecipeRegistryInfo(name, src string) (map[string]string, error) {
	sr, err := recipe.NewStarlarkRecipe(name, src, nil)
	if err != nil {
		return nil, err
	}
	return sr.GetRegistryInfo(m.cfg)
}

func (m *manager) GetFullRegistryInfo(verbose bool) ([]IndexEntry, error) {
	// If index is empty, try loading it
	if len(m.index) == 0 {
		if err := m.LoadIndex(); err != nil {
			return nil, err
		}
		// If still empty and we have repos/builtins, maybe we need to sync?
		if len(m.index) == 0 {
			if err := m.Sync(verbose); err != nil {
				return nil, err
			}
		}
	}

	// Make a copy to sort
	entries := make([]IndexEntry, len(m.index))
	copy(entries, m.index)

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].RepoName != entries[j].RepoName {
			return entries[i].RepoName < entries[j].RepoName
		}
		if entries[i].RecipeName != entries[j].RecipeName {
			return entries[i].RecipeName < entries[j].RecipeName
		}
		return entries[i].Pattern < entries[j].Pattern
	})

	return entries, nil
}

func (m *manager) DisplayRegistryInfo(entries []IndexEntry) {
	m.disp.Print(fmt.Sprintf("%-15s %-15s %-30s %s\n", "REPO", "RECIPE", "PATTERN", "HANDLER"))
	m.disp.Print(fmt.Sprintln(strings.Repeat("-", 80)))
	for _, e := range entries {
		m.disp.Print(fmt.Sprintf("%-15s %-15s %-30s %s\n", e.RepoName, e.RecipeName, e.Pattern, e.Handler))
	}
}

func (m *manager) ListRepos() []RepoConfig {
	return m.repos
}

func (m *manager) GetRepoByUUID(uuid uuid.UUID) (RepoConfig, bool) {
	for _, r := range m.repos {
		if r.UUID == uuid {
			return r, true
		}
	}
	return RepoConfig{}, false
}

func (m *manager) GetRepoByName(name string) (RepoConfig, bool) {
	for _, r := range m.repos {
		if r.Name == name {
			return r, true
		}
	}
	return RepoConfig{}, false
}

func (m *manager) GetRecipe(name string) (string, error) {
	content, ok := m.recipes[name]
	if !ok {
		return "", fmt.Errorf("recipe not found: %s", name)
	}
	return content, nil
}

func (m *manager) ListRecipes() []string {
	var list []string
	for name := range m.recipes {
		list = append(list, name)
	}
	return list
}

// Resolve selects the single matching recipe/regex for a package identifier.
func (m *manager) Resolve(pkgName string, cfg config.Config) (string, string, error) {
	if err := m.ensureIndexLoaded(); err != nil {
		return "", "", err
	}

	if res, ok := m.resolveCache[pkgName]; ok {
		return res.recipeName, res.regexKey, nil
	}

	originalPkgName := pkgName
	repoFilter := ""
	parts := strings.Split(pkgName, "/")

	// Check for repo/prefix:name or repo/name
	if len(parts) >= 2 {
		first := parts[0]
		// Check if 'first' is a known repo name
		_, isRepo := m.GetRepoByName(first)

		if isRepo {
			repoFilter = first
			pkgName = strings.Join(parts[1:], "/")
		}
	}

	type match struct {
		repo   string
		recipe string
		regex  string
	}

	var matches []match

	// Optimization: Use in-memory index
	for _, entry := range m.index {
		if repoFilter != "" && entry.RepoName != repoFilter {
			continue
		}

		re, ok := m.compiledPatterns[entry.Pattern]
		if !ok {
			var err error
			re, err = recipe.CompileAnchored(entry.Pattern)
			if err != nil {
				// Warn?
				continue
			}
			m.compiledPatterns[entry.Pattern] = re
		}

		if re.MatchString(pkgName) {
			matches = append(matches, match{
				repo:   entry.RepoName,
				recipe: entry.RecipeName,
				regex:  entry.Pattern,
			})
		}
	}

	if len(matches) == 0 {
		return "", "", fmt.Errorf("no recipe matched: %s", originalPkgName)
	}
	if len(matches) > 1 {
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
		return "", "", fmt.Errorf("ambiguous package match for %s:\n%s", originalPkgName, strings.Join(lines, "\n"))
	}

	m.resolveCache[originalPkgName] = resolvedRecipe{
		recipeName: matches[0].recipe,
		regexKey:   matches[0].regex,
	}

	return matches[0].recipe, matches[0].regex, nil
}

// ResolvedQuery represents a matched index entry for a package query.
type ResolvedQuery struct {
	// RepoUUID is the identifier of the repository.
	RepoUUID uuid.UUID
	// RepoName is the display name of the repository.
	RepoName string
	// RecipeName is the name of the recipe.
	RecipeName string
	// Pattern is the matched regex pattern.
	Pattern string
}

func (m *manager) ResolveQuery(query string) ([]ResolvedQuery, error) {
	if err := m.ensureIndexLoaded(); err != nil {
		return nil, err
	}

	repoFilter := ""
	pkgFilter := query

	if strings.Contains(query, "/") {
		parts := strings.SplitN(query, "/", 2)
		repoFilter = parts[0]
		pkgFilter = parts[1]
	}

	var results []ResolvedQuery
	for _, entry := range m.index {
		if repoFilter != "" && entry.RepoName != repoFilter {
			continue
		}

		if pkgFilter != "" {
			// Exact match or prefix match if specified in requirements?
			// User says: [repo/][prefix:]pkg
			// "prefix:pkg" is the name used in patterns.
			// Patterns in index are regexes.
			// But user says: "There is no wildcard/regexp matching."
			// This means the user input 'pkgFilter' should match a pattern exactly
			// if we treat patterns as names?
			// Actually, recipes use add_pkgdef("name", ...) or add_pkgdef(r"regex", ...)

			// If pkgFilter is non-empty, we want to match it.
			// For now, let's use the regex matching but anchored to start/end.
			re, ok := m.compiledPatterns[entry.Pattern]
			if !ok {
				var err error
				re, err = recipe.CompileAnchored(entry.Pattern)
				if err != nil {
					continue
				}
				m.compiledPatterns[entry.Pattern] = re
			}

			if re.MatchString(pkgFilter) {
				results = append(results, ResolvedQuery{
					RepoUUID:   entry.RepoUUID,
					RepoName:   entry.RepoName,
					RecipeName: entry.RecipeName,
					Pattern:    entry.Pattern,
				})
			}
		} else {
			// pkgFilter is empty, means we want all in repo
			results = append(results, ResolvedQuery{
				RepoUUID:   entry.RepoUUID,
				RepoName:   entry.RepoName,
				RecipeName: entry.RecipeName,
				Pattern:    entry.Pattern,
			})
		}
	}

	if repoFilter == "" && pkgFilter != "" && len(results) > 1 {
		// Check for ambiguity: if the SAME pkg name matches in multiple repos
		// Wait, if it's multiple patterns in same repo it's also maybe ambiguous?
		// User: "If repo is not given, and package is found in multiple repo, ambiguity should be reported."
		repos := make(map[string]bool)
		for _, r := range results {
			repos[r.RepoName] = true
		}
		if len(repos) > 1 {
			var lines []string
			for _, r := range results {
				lines = append(lines, fmt.Sprintf("  %s/%s (%s)", r.RepoName, r.RecipeName, r.Pattern))
			}
			return nil, fmt.Errorf("ambiguous package match for %q:\n%s", query, strings.Join(lines, "\n"))
		}
	}

	return results, nil
}

func generateUUID() uuid.UUID {
	return uuid.New()
}
