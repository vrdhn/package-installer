package pkgs

import (
	"fmt"
	"os"
	"path/filepath"
	"pi/pkg/common"
	"pi/pkg/config"
	"strings"
)

// Package represents a parsed package requirement.
// Immutable
type Package struct {
	Name    string
	Version string
}

// Parse parses a package string in the format name[=version]
// Note: name can contain colons (e.g., pip:numpy)
func Parse(s config.PkgRef) (*Package, error) {
	p := &Package{}

	// Extract version if present
	if equalIdx := strings.Index(s, "="); equalIdx != -1 {
		p.Name = s[:equalIdx]
		p.Version = s[equalIdx+1:]
	} else {
		p.Name = s
		p.Version = "latest"
	}

	if p.Name == "" {
		return nil, fmt.Errorf("invalid package string: %q (missing name)", s)
	}

	return p, nil
}

// String returns the string representation of the package (e.g., "name=version").
func (p *Package) String() string {
	res := p.Name
	if p.Version != "" {
		res += "=" + p.Version
	}
	return res
}

// DiscoverSymlinks finds files within the installPath that match specified patterns
// and returns them as a slice of Symlink definitions.
// Patterns support directory expansion via "/*" suffixes (e.g., {"bin/*": ".local/bin"}).
func DiscoverSymlinks(installPath string, patterns map[string]string) ([]common.Symlink, error) {
	if len(patterns) == 0 {
		// Default behavior: bin/* -> .local/bin
		patterns = map[string]string{
			"bin/*": ".local/bin",
		}
	}

	var symlinks []common.Symlink
	for srcPattern, destDir := range patterns {
		if strings.HasSuffix(srcPattern, "/*") {
			// Directory expansion
			subDir := strings.TrimSuffix(srcPattern, "/*")
			absSubDir := filepath.Join(installPath, subDir)

			entries, err := os.ReadDir(absSubDir)
			if err != nil {
				return nil, fmt.Errorf("failed to read expansion directory %s: %w", absSubDir, err)
			}

			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				// Skip hidden files
				if strings.HasPrefix(entry.Name(), ".") {
					continue
				}

				symlinks = append(symlinks, common.Symlink{
					Source: filepath.Join(absSubDir, entry.Name()),
					Target: filepath.Join(destDir, entry.Name()),
				})
			}
		} else {
			// Direct file/dir mapping
			symlinks = append(symlinks, common.Symlink{
				Source: filepath.Join(installPath, srcPattern),
				Target: destDir,
			})
		}
	}

	return symlinks, nil
}

// CreateSymlinks physically creates symlinks on the host filesystem within the homePath.
// It ensures parent directories exist and replaces any existing files at the target location.
func CreateSymlinks(homePath string, symlinks []common.Symlink) error {
	for _, s := range symlinks {
		linkPath := filepath.Join(homePath, s.Target)

		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
			return err
		}

		// Remove existing link/file if it exists
		if _, err := os.Lstat(linkPath); err == nil {
			if err := os.Remove(linkPath); err != nil {
				return err
			}
		}

		// Create the symlink
		if err := os.Symlink(s.Source, linkPath); err != nil {
			return err
		}
	}
	return nil
}
