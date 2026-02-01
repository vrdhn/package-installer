package repository

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

//go:embed recipes/*.star
var recipesFS embed.FS

type Manager struct {
	recipes map[string]string
}

func NewManager() (*Manager, error) {
	m := &Manager{
		recipes: make(map[string]string),
	}

	err := fs.WalkDir(recipesFS, "recipes", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".star") {
			content, err := recipesFS.ReadFile(path)
			if err != nil {
				return err
			}
			name := strings.TrimSuffix(filepath.Base(path), ".star")
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
