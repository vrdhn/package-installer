package repository

import (
	"crypto/rand"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"pi/pkg/config"
	"pi/pkg/display"
	"pi/pkg/recipe"
	"regexp"
	"sort"
	"strings"
)

type RepoConfig struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	UUID string `json:"uuid"`
}

type repoRegistry struct {
	Repos []RepoConfig `json:"repos"`
}

type RegistryEntry struct {
	RepoUUID   string
	RepoName   string
	RecipeName string
	Pattern    string
	Handler    string
}

// Mutable
type manager struct {
	recipes map[string]string // recipe name -> source
	repos   []RepoConfig
	disp    display.Display
	cfg     config.Config

	// In-memory index of all packages
	index []RegistryEntry
	// Cache for resolution: pkgName -> {recipeName, regexKey}
	resolveCache     map[string]resolvedRecipe
	compiledPatterns map[string]*regexp.Regexp
}

type Manager = *manager

type resolvedRecipe struct {
	recipeName string
	regexKey   string
}

func NewManager(disp display.Display, cfg config.Config) (Manager, error) {
	m := &manager{
		recipes:          make(map[string]string),
		resolveCache:     make(map[string]resolvedRecipe),
		compiledPatterns: make(map[string]*regexp.Regexp),
		disp:             disp,
		cfg:              cfg,
	}

	if err := m.loadBuiltins(); err != nil {
		return nil, err
	}

	if err := m.LoadRepos(); err != nil {
		return nil, err
	}

	if err := m.LoadIndex(); err != nil {
		// If loading index fails (e.g. file missing), we'll just have an empty index
		// and Resolve will fail or fallback if we implement fallback.
		// For now, we assume index is critical for performance but not strictly fatal for startup if missing (just Resolve will fail).
		// Ideally we should auto-sync if missing, but we will leave that for 'repo sync' command.
		m.disp.Log(fmt.Sprintf("Warning: Failed to load package index: %v", err))
	}

	return m, nil
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
	path := filepath.Join(m.cfg.GetConfigDir(), "repos.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var reg repoRegistry
	if err := json.Unmarshal(data, &reg); err != nil {
		return err
	}
	m.repos = reg.Repos

	// TODO: Fetch recipes from URLs. For now, we only support local paths as URLs.
	for _, repo := range m.repos {
		if strings.HasPrefix(repo.URL, "/") || strings.HasPrefix(repo.URL, "./") {
			m.loadLocalRepo(repo)
		}
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
	path := filepath.Join(m.cfg.GetConfigDir(), "packages.csv")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	r := csv.NewReader(f)
	m.index = nil

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if len(record) != 4 {
			continue
		}

		uuid := record[0]
		recipeName := record[1]
		pattern := record[2]
		handler := record[3]

		repoName := "unknown"
		if uuid == "builtin" {
			repoName = "builtin"
		} else {
			for _, repo := range m.repos {
				if repo.UUID == uuid {
					repoName = repo.Name
					break
				}
			}
		}

		m.index = append(m.index, RegistryEntry{
			RepoUUID:   uuid,
			RepoName:   repoName,
			RecipeName: recipeName,
			Pattern:    pattern,
			Handler:    handler,
		})
	}
	return nil
}

func (m *manager) ensureIndexLoaded() error {
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
	csvPath := filepath.Join(m.cfg.GetConfigDir(), "packages.csv")
	if verbose {
		fmt.Printf("Regenerating index at %s\n", csvPath)
	}

	if err := os.MkdirAll(filepath.Dir(csvPath), 0755); err != nil {
		return err
	}

	f, err := os.Create(csvPath)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)

	var entries []RegistryEntry

	// 1. Builtins
	if verbose {
		fmt.Println("Indexing builtin recipes...")
	}
	err = fs.WalkDir(recipe.BuiltinRecipes, "recipes", func(path string, d fs.DirEntry, err error) error {
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
				if err := w.Write([]string{"builtin", name, p, h}); err != nil {
					return err
				}
				entries = append(entries, RegistryEntry{
					RepoUUID:   "builtin",
					RepoName:   "builtin",
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

	// 2. Repos
	for _, repo := range m.repos {
		if verbose {
			fmt.Printf("Indexing repo: %s (%s)\n", repo.Name, repo.URL)
		}
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
				regInfo, err := m.GetRecipeRegistryInfo(name, string(content))
				if err != nil {
					return err
				}
				for p, h := range regInfo {
					if err := w.Write([]string{repo.UUID, name, p, h}); err != nil {
						return err
					}
					entries = append(entries, RegistryEntry{
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
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}

	m.index = entries
	// Clear caches as patterns might have changed
	m.compiledPatterns = make(map[string]*regexp.Regexp)
	m.resolveCache = make(map[string]resolvedRecipe)

	if verbose {
		fmt.Printf("Sync complete. Indexed %d patterns.\n", len(entries))
	}
	return nil
}

func (m *manager) AddLocalRepo(path string, verbose bool) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	if verbose {
		fmt.Printf("Statting %s\n", absPath)
	}
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

	b := make([]byte, 16)
	_, _ = rand.Read(b)
	uuid := fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])

	name := filepath.Base(absPath)

	if verbose {
		fmt.Printf("Found repository at %s\n", absPath)
		fmt.Printf("Generated UUID: %s\n", uuid)
		fmt.Printf("Repo Name: %s\n", name)
	}

	newRepo := RepoConfig{
		Name: name,
		URL:  absPath,
		UUID: uuid,
	}

	m.repos = append(m.repos, newRepo)
	if err := m.saveRepos(verbose); err != nil {
		return err
	}

	// Also load it into memory
	m.loadLocalRepo(newRepo)

	// Update index
	return m.Sync(verbose)
}

func (m *manager) saveRepos(verbose bool) error {
	path := filepath.Join(m.cfg.GetConfigDir(), "repos.json")
	if verbose {
		fmt.Printf("Saving repos to %s\n", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(repoRegistry{Repos: m.repos}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (m *manager) GetRecipeRegistryInfo(name, src string) (map[string]string, error) {
	sr, err := recipe.NewStarlarkRecipe(name, src, nil)
	if err != nil {
		return nil, err
	}
	return sr.GetRegistryInfo(m.cfg)
}

func (m *manager) GetFullRegistryInfo(verbose bool) ([]RegistryEntry, error) {
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
	entries := make([]RegistryEntry, len(m.index))
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

func DisplayRegistryInfo(entries []RegistryEntry) {
	fmt.Printf("%-15s %-15s %-30s %s\n", "REPO", "RECIPE", "PATTERN", "HANDLER")
	fmt.Println(strings.Repeat("-", 80))
	for _, e := range entries {
		fmt.Printf("%-15s %-15s %-30s %s\n", e.RepoName, e.RecipeName, e.Pattern, e.Handler)
	}
}

func (m *manager) ListRepos() []RepoConfig {
	return m.repos
}

func (m *manager) GetRepoByUUID(uuid string) (RepoConfig, bool) {
	if uuid == "builtin" {
		return RepoConfig{Name: "builtin", UUID: "builtin"}, true
	}
	for _, r := range m.repos {
		if r.UUID == uuid {
			return r, true
		}
	}
	return RepoConfig{}, false
}

func (m *manager) GetRepoByName(name string) (RepoConfig, bool) {
	if name == "builtin" {
		return RepoConfig{Name: "builtin", UUID: "builtin"}, true
	}
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
		isRepo := false
		if first == "builtin" {
			isRepo = true
		} else {
			for _, r := range m.repos {
				if r.Name == first {
					isRepo = true
					break
				}
			}
		}

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

type ResolvedQuery struct {
	RepoUUID   string
	RepoName   string
	RecipeName string
	Pattern    string
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
