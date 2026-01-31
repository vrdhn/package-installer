package recipe

import "pi/pkg/config"

// PackageDefinition describes a specific build of a package.
type PackageDefinition struct {
	Name     string
	Version  string
	OS       config.OSType
	Arch     config.ArchType
	URL      string
	Filename string
	Checksum string
}

// Recipe defines how to discover and resolve packages for an ecosystem.
type Recipe struct {
	Name         string
	DiscoveryURL string
	Parser       func(data []byte) ([]PackageDefinition, error)
	// Filter returns true if the package matches requested version and preferences.
	Filter func(cfg *config.Config, p PackageDefinition, version string) bool
}
