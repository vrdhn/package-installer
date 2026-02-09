package pkgs

import (
	"fmt"
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
func Parse(s config.PkgRef) (Package, error) {
	var p Package

	// Extract version if present
	if equalIdx := strings.Index(s, "="); equalIdx != -1 {
		p.Name = s[:equalIdx]
		p.Version = s[equalIdx+1:]
	} else {
		p.Name = s
		p.Version = "latest"
	}

	if p.Name == "" {
		return Package{}, fmt.Errorf("invalid package string: %q (missing name)", s)
	}

	return p, nil
}

// String returns the string representation of the package (e.g., "name=version").
func (p Package) String() string {
	res := p.Name
	if p.Version != "" {
		res += "=" + p.Version
	}
	return res
}
