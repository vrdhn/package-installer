package devel

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"pi/pkg/common"
	"pi/pkg/config"
	"pi/pkg/downloader"
	"pi/pkg/recipe"
	"strings"
)

// Run executes a recipe test for a specific package in a temporary work directory.
func Test(ctx context.Context, recipePath string, packageName string) (*common.Output, error) {
	workDir := "pi-work"
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, err
	}

	cfg := config.NewTestConfig(workDir)

	source, err := os.ReadFile(recipePath)
	if err != nil {
		return nil, err
	}

	recipeName := filepath.Base(recipePath)
	recipeName = strings.TrimSuffix(recipeName, filepath.Ext(recipeName))

	r, err := recipe.NewStarlarkRecipe(recipeName, string(source), func(msg string) {
		slog.Info(msg, "recipe", recipeName)
	})
	if err != nil {
		return nil, err
	}

	dl := downloader.NewDefaultDownloader()
	fetcher := func(url string) ([]byte, error) {
		var buf bytes.Buffer
		if err := dl.Download(ctx, url, &buf); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	pkgs, err := r.Execute(cfg, packageName, "latest", fetcher)
	if err != nil {
		return nil, err
	}

	table := &common.Table{
		Header: []string{"Version", "OS", "Arch", "Filename"},
	}

	for _, p := range pkgs {
		table.Rows = append(table.Rows, []string{
			p.Version,
			string(p.OS),
			string(p.Arch),
			p.Filename,
		})
	}

	return &common.Output{
		Message: fmt.Sprintf("Found %d versions for %s", len(pkgs), packageName),
		Table:   table,
	}, nil
}
