package resolver

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"pi/pkg/cache"
	"pi/pkg/config"
	"pi/pkg/display"
	"pi/pkg/recipe"
	"strings"
	"time"
)

// Resolve finds the best matching package for the given recipe and version constraint.
func Resolve(ctx context.Context, cfg *config.Config, r *recipe.Recipe, version string, task display.Task) (*recipe.PackageDefinition, error) {
	task.SetStage("Resolve", r.Name)

	data, err := fetchDiscoveryData(ctx, cfg, r, task)
	if err != nil {
		return nil, err
	}

	pkgs, err := r.Parser(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse discovery data: %w", err)
	}

	// Filter by OS/Arch and Version
	targetOS := cfg.OS
	targetArch := cfg.Arch

	var bestMatch *recipe.PackageDefinition

	for i := range pkgs {
		p := &pkgs[i]
		if p.OS != targetOS || p.Arch != targetArch {
			continue
		}

		if r.Filter != nil {
			if !r.Filter(cfg, *p, version) {
				continue
			}
		} else {
			// Fallback basic filtering if no Filter function provided
			if version != "latest" && version != "" && !strings.HasPrefix(p.Version, version) {
				continue
			}
		}

		// First match is considered best (assume sorted by discovery source)
		bestMatch = p
		break
	}

	if bestMatch == nil {
		return nil, fmt.Errorf("no matching package found for %s version %s on %s/%s", r.Name, version, targetOS, targetArch)
	}

	task.Log(fmt.Sprintf("Resolved %s to version %s", r.Name, bestMatch.Version))
	return bestMatch, nil
}

func fetchDiscoveryData(ctx context.Context, cfg *config.Config, r *recipe.Recipe, task display.Task) ([]byte, error) {
	// 1. Create discovery directory
	if err := os.MkdirAll(cfg.DiscoveryDir, 0755); err != nil {
		return nil, err
	}

	// 2. Generate cache filename (sanitized URL)
	safeURL := strings.ReplaceAll(r.DiscoveryURL, "/", "_")
	safeURL = strings.ReplaceAll(safeURL, ":", "_")
	cachePath := filepath.Join(cfg.DiscoveryDir, safeURL+".json")

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

	task.Log(fmt.Sprintf("Fetching discovery data from %s", r.DiscoveryURL))
	req, err := http.NewRequestWithContext(ctx, "GET", r.DiscoveryURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch discovery data: %s", resp.Status)
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