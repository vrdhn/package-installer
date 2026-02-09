package repo

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"pi/pkg/recipe"
	"sort"
	"strings"

	"github.com/google/uuid"
)

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

func (m *manager) GetRecipeRegistryInfo(name, src string) (map[string]string, error) {
	sr, err := recipe.NewStarlarkRecipe(name, src, nil)
	if err != nil {
		return nil, err
	}
	return sr.GetRegistryInfo(m.cfg)
}

func (m *manager) GetFullRegistryInfo(verbose bool) ([]IndexEntry, error) {
	if len(m.index) == 0 {
		if err := m.LoadIndex(); err != nil {
			return nil, err
		}
		if len(m.index) == 0 {
			if err := m.Sync(verbose); err != nil {
				return nil, err
			}
		}
	}

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

func (m *manager) LoadRepos() error {
	if err := m.ensureBuiltinRepo(); err != nil {
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

	if err := m.loadBuiltins(); err != nil {
		return err
	}

	for _, repo := range m.repos {
		if repo.URL == "builtin://" {
			continue
		}
		if !strings.HasPrefix(repo.URL, "http://") && !strings.HasPrefix(repo.URL, "https://") {
			m.loadLocalRepo(repo)
		}
	}

	return nil
}

func (m *manager) ensureBuiltinRepo() error {
	return m.repoMgr.Modify(func(reg *repoRegistry) error {
		for _, r := range reg.Repos {
			if r.Name == "builtin" || r.URL == "builtin://" {
				if r.UUID == uuid.Nil {
					r.UUID = generateUUID()
				}
				return nil
			}
		}
		reg.Repos = append([]RepoConfig{{
			Name: "builtin",
			URL:  "builtin://",
			UUID: generateUUID(),
		}}, reg.Repos...)
		return nil
	})
}

func (m *manager) loadBuiltins() error {
	return fs.WalkDir(recipe.BuiltinRecipes, "recipes", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".star") {
			return err
		}
		name := strings.TrimSuffix(filepath.Base(path), ".star")
		m.disp.Log(fmt.Sprintf("Loading builtin recipe: %s", name))
		content, err := fs.ReadFile(recipe.BuiltinRecipes, path)
		if err != nil {
			return err
		}
		m.recipes[name] = string(content)
		return nil
	})
}

func (m *manager) loadLocalRepo(repo RepoConfig) {
	err := filepath.Walk(repo.URL, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".star") {
			return err
		}
		name := strings.TrimSuffix(filepath.Base(path), ".star")
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		m.recipes[name] = string(content)
		return nil
	})
	if err != nil {
		m.disp.Log(fmt.Sprintf("Failed to load local repo %s: %v", repo.Name, err))
	}
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
			return m.Sync(false)
		}
	}
	return nil
}
