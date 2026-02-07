package resolver

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"pi/pkg/archive"
	"pi/pkg/cache"
	"pi/pkg/config"
	"pi/pkg/display"
	"pi/pkg/recipe"
	"strings"
	"time"
)

// List returns all available packages for the given recipe and version query.
func List(ctx context.Context, cfg config.Config, r recipe.Recipe, pkgName string, version string, task display.Task) ([]recipe.PackageDefinition, error) {
	task.SetStage("List", r.GetName())

	dCtx := &recipe.DiscoveryContext{
		Config:       cfg,
		PkgName:      pkgName,
		VersionQuery: version,
		Download: func(url string) ([]byte, error) {
			return fetchData(ctx, cfg, url, task)
		},
	}

	return r.Execute(dCtx)
}

// Resolve finds the best matching package for the given recipe and version constraint.
func Resolve(ctx context.Context, cfg config.Config, r recipe.Recipe, pkgName string, version string, task display.Task) (*recipe.PackageDefinition, error) {
	pkgs, err := List(ctx, cfg, r, pkgName, version, task)
	if err != nil {
		return nil, err
	}

	task.SetStage("Resolve", r.GetName())

	// Filter by OS/Arch and Version and Extension
	targetOS := cfg.GetOS()
	targetArch := cfg.GetArch()
	allowedExts := archive.Extensions(targetOS)

	var bestMatch *recipe.PackageDefinition

	for i := range pkgs {
		p := &pkgs[i]
		if p.OS != targetOS || p.Arch != targetArch {
			continue
		}

		// Basic filtering
		if version == "stable" {
			if p.ReleaseStatus != "stable" && p.ReleaseStatus != "lts" {
				continue
			}
		} else if version == "lts" {
			if p.ReleaseStatus != "lts" {
				continue
			}
		} else if version != "latest" && version != "" && !strings.HasPrefix(p.Version, version) {
			continue
		}

		// Extension filtering
		supported := false
		for _, ext := range allowedExts {
			if strings.HasSuffix(p.Filename, ext) {
				supported = true
				break
			}
		}
		if !supported {
			continue
		}

		// First match is considered best (assume sorted by discovery source)
		bestMatch = p
		break
	}

	if bestMatch == nil {
		return nil, fmt.Errorf("no matching package found for %s version %s on %s/%s", r.GetName(), version, targetOS, targetArch)
	}

	task.Log(fmt.Sprintf("Resolved %s to version %s", r.GetName(), bestMatch.Version))
	return bestMatch, nil
}

func fetchData(ctx context.Context, cfg config.Config, url string, task display.Task) ([]byte, error) {
	// 1. Create discovery directory
	if err := os.MkdirAll(cfg.GetDiscoveryDir(), 0755); err != nil {
		return nil, err
	}

	// 2. Generate cache filename (sanitized URL)
	safeURL := strings.ReplaceAll(url, "/", "_")
	safeURL = strings.ReplaceAll(safeURL, ":", "_")
	cachePath := filepath.Join(cfg.GetDiscoveryDir(), safeURL+".json")

	// 3. Check TTL (1 hour)
	if info, err := os.Stat(cachePath); err == nil {
		if time.Since(info.ModTime()) < 1*time.Hour {
			return os.ReadFile(cachePath)
		}
	}

	// 4. Lock and Fetch
	unlock, err := cache.Lock(cachePath)
	if err != nil {
		return nil, err
	}
	defer unlock()

	// Re-check after locking
	if info, err := os.Stat(cachePath); err == nil {
		if time.Since(info.ModTime()) < 1*time.Hour {
			return os.ReadFile(cachePath)
		}
	}

	task.Log(fmt.Sprintf("Fetching data from %s", url))
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch data: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Write to cache
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return nil, err
	}

	return data, nil
}
