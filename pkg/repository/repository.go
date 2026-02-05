package repository

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"pi/pkg/config"
	"pi/pkg/display"
	"sort"
	"strings"

	"pi/pkg/recipe"
)

// Mutable
type Manager struct {
	recipes map[string]string
	disp    display.Display
}

func NewManager(disp display.Display) (*Manager, error) {
	m := &Manager{
		recipes: make(map[string]string),
		disp:    disp,
	}

	err := fs.WalkDir(recipe.BuiltinRecipes, "recipes", func(path string, d fs.DirEntry, err error) error {
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

	if err != nil {
		return nil, err
	}

	return m, nil
}

func (m *Manager) GetRecipe(name string) (string, error) {
	content, ok := m.recipes[name]
	if !ok {
		return "", fmt.Errorf("recipe not found: %s", name)
	}
	return content, nil
}

func (m *Manager) ListRecipes() []string {
	var list []string
	for name := range m.recipes {
		list = append(list, name)
	}
	return list
}

// Resolve selects the single matching recipe/regex for a package identifier.
// pkgName must be [ecosystem:]name (no version).
func (m *Manager) Resolve(pkgName string, cfg config.ReadOnly) (string, string, error) {
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

	return matches[0].recipe, matches[0].regex, nil
}
