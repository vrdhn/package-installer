package repository

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"pi/pkg/config"
	"pi/pkg/display"
	"sort"
	"strings"

	"pi/pkg/recipe"
)

type RepoConfig struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type repoRegistry struct {
	Repos []RepoConfig `json:"repos"`
}

// Mutable
type manager struct {
	recipes map[string]string // recipe name -> source
	repos   []RepoConfig
	disp    display.Display
	cfg     config.Config

	// Cache for resolution: pkgName -> {recipeName, regexKey}
	resolveCache map[string]resolvedRecipe
}

type Manager = *manager

type resolvedRecipe struct {
	recipeName string
	regexKey   string
}

func NewManager(disp display.Display, cfg config.Config) (Manager, error) {
	m := &manager{
		recipes:      make(map[string]string),
		resolveCache: make(map[string]resolvedRecipe),
		disp:         disp,
		cfg:          cfg,
	}

	if err := m.loadBuiltins(); err != nil {
		return nil, err
	}

	if err := m.LoadRepos(); err != nil {
		return nil, err
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

func (m *manager) AddRepo(name, url string) error {
	for _, r := range m.repos {
		if r.Name == name {
			return fmt.Errorf("repository already exists: %s", name)
		}
	}

	m.repos = append(m.repos, RepoConfig{Name: name, URL: url})
	path := filepath.Join(m.cfg.GetConfigDir(), "repos.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(repoRegistry{Repos: m.repos}, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}

	// Try to load it immediately
	if strings.HasPrefix(url, "/") || strings.HasPrefix(url, "./") {
		m.loadLocalRepo(RepoConfig{Name: name, URL: url})
	}

	return nil
}

func (m *manager) ListRepos() []RepoConfig {
	return m.repos
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
// pkgName can be 'name' or 'prefix:name' (no version).
func (m *manager) Resolve(pkgName string, cfg config.Config) (string, string, error) {
	if res, ok := m.resolveCache[pkgName]; ok {
		return res.recipeName, res.regexKey, nil
	}

	type match struct {
		repo   string
		recipe string
		regex  string
	}

	var matches []match
	for _, recipeName := range m.ListRecipes() {
		src, err := m.GetRecipe(recipeName)
		if err != nil {
			return "", "", err
		}
		sr, err := recipe.NewStarlarkRecipe(recipeName, src, nil)
		if err != nil {
			return "", "", err
		}
		patterns, legacy, err := sr.Registry(cfg)
		if err != nil {
			return "", "", err
		}
		if legacy {
			continue
		}
		for _, pattern := range patterns {
			re, err := recipe.CompileAnchored(pattern)
			if err != nil {
				return "", "", fmt.Errorf("invalid regex '%s' in recipe %s: %w", pattern, recipeName, err)
			}
			if re.MatchString(pkgName) {
				matches = append(matches, match{
					repo:   "builtin",
					recipe: recipeName,
					regex:  pattern,
				})
			}
		}
	}

	if len(matches) == 0 {
		return "", "", fmt.Errorf("no recipe matched: %s", pkgName)
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
		return "", "", fmt.Errorf("ambiguous package match for %s:\n%s", pkgName, strings.Join(lines, "\n"))
	}

	m.resolveCache[pkgName] = resolvedRecipe{
		recipeName: matches[0].recipe,
		regexKey:   matches[0].regex,
	}

	return matches[0].recipe, matches[0].regex, nil
}
