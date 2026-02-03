package pkgs

import (
	"fmt"
	"os"
	"path/filepath"
	sysconfig "pi/pkg/config"
	"strings"
)

// Package represents a parsed package requirement.
// Immutable
type Package struct {
	Ecosystem string
	Name      string
	Version   string
}

// Symlink represents a symlink that should be created in the cave.
// Immutable
type Symlink struct {
	Source string // Path on the host (target of the symlink)
	Target string // Relative path in the cave (the symlink itself)
}

// Result contains the outcome of package preparation.
// Immutable
type Result struct {
	Symlinks []Symlink
	Env      map[string]string
}

// Parse parses a package string in the format [ecosystem:]name[=version]
func Parse(s sysconfig.PkgRef) (*Package, error) {
	p := &Package{}

	// Extract ecosystem if present
	if colonIdx := strings.Index(s, ":"); colonIdx != -1 {
		p.Ecosystem = s[:colonIdx]
		s = s[colonIdx+1:]
	}

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

func (p *Package) String() string {
	res := ""
	if p.Ecosystem != "" {
		res += p.Ecosystem + ":"
	}
	res += p.Name
	if p.Version != "" {
		res += "=" + p.Version
	}
	return res
}

// DiscoverSymlinks finds all binaries in the install path and returns them as symlinks.
// For now, it just looks into the "bin" directory.
func DiscoverSymlinks(installPath string) ([]Symlink, error) {
	binDir := filepath.Join(installPath, "bin")
	entries, err := os.ReadDir(binDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var symlinks []Symlink
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		// Skip hidden files
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		symlinks = append(symlinks, Symlink{
			Source: filepath.Join(binDir, entry.Name()),
			Target: filepath.Join(".local", "bin", entry.Name()),
		})
	}

	return symlinks, nil
}

// CreateSymlinks creates the symlinks in the specified home directory.
func CreateSymlinks(homePath string, symlinks []Symlink) error {
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
