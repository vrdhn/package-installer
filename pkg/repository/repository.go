package repository

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"pi/pkg/display"
	"strings"

	"pi/pkg/recipe"
)

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
