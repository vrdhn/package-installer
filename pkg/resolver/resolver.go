// Package resolver implements the logic for selecting the best build of a package
// from the set of versions discovered by a recipe.
package resolver

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"pi/pkg/archive"
	"pi/pkg/cache"
	"pi/pkg/config"
	"pi/pkg/downloader"
	"pi/pkg/recipe"
	"strings"
	"time"
)

// List returns all available builds for a package provided by a specific recipe.
// It executes the recipe's discovery logic and returns the raw results.
func List(ctx context.Context, cfg config.Config, r *recipe.PinnedRecipe, pkgName string, version string) ([]recipe.PackageDefinition, error) {
	slog.Debug("Listing package versions", "recipe", r.GetName(), "package", pkgName)

	return r.Execute(cfg, pkgName, version, func(url string) ([]byte, error) {
		return fetchData(ctx, cfg, url)
	})
}

// Resolve finds the optimal package build matching the user's requirements.
// It filters the discovered versions based on the current system's OS and architecture,
// release status keywords (latest, stable, lts), and archive extension compatibility.
func Resolve(ctx context.Context, cfg config.Config, r *recipe.PinnedRecipe, pkgName string, version string) (*recipe.PackageDefinition, error) {
	pkgs, err := List(ctx, cfg, r, pkgName, version)
	if err != nil {
		return nil, err
	}

	slog.Debug("Resolving best match", "recipe", r.GetName(), "package", pkgName, "version", version)

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

	slog.Info("Resolved package", "recipe", r.GetName(), "version", bestMatch.Version)
	return bestMatch, nil
}

func fetchData(ctx context.Context, cfg config.Config, url string) ([]byte, error) {
	// 1. Create discovery directory
	if err := os.MkdirAll(cfg.GetDiscoveryDir(), 0755); err != nil {
		return nil, err
	}

	// 2. Generate cache filename (sanitized URL)
	safeURL := strings.ReplaceAll(url, "/", "_")
	safeURL = strings.ReplaceAll(safeURL, ":", "_")
	cachePath := filepath.Join(cfg.GetDiscoveryDir(), safeURL+".json")

	// 3. Ensure with TTL (1 hour)
	err := cache.EnsureWithTTL(cachePath, 1*time.Hour, func() error {
		slog.Info("Fetching data", "url", url)
		d := downloader.NewDefaultDownloader()

		f, err := os.Create(cachePath)
		if err != nil {
			return err
		}
		defer f.Close()

		return d.Download(ctx, url, f)
	})

	if err != nil {
		return nil, err
	}

	return os.ReadFile(cachePath)
}
