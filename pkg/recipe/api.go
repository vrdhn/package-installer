package recipe

import (
	"embed"
	"pi/pkg/config"
)

//go:embed recipes/*.star
var BuiltinRecipes embed.FS

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
type Recipe interface {
	GetName() string
	Discover(cfg *config.Config, pkgName string, versionQuery string) (url string, method string, err error)
	Parse(cfg *config.Config, pkgName string, data []byte, versionQuery string) ([]PackageDefinition, error)
}

// GoRecipe is the legacy Go-based implementation
type GoRecipe struct {
	Name         string
	DiscoveryURL string
	Parser       func(data []byte) ([]PackageDefinition, error)
}

func (r *GoRecipe) GetName() string { return r.Name }
func (r *GoRecipe) Discover(cfg *config.Config, pkgName string, versionQuery string) (string, string, error) {
	return r.DiscoveryURL, "GET", nil
}
func (r *GoRecipe) Parse(cfg *config.Config, pkgName string, data []byte, versionQuery string) ([]PackageDefinition, error) {
	return r.Parser(data)
}

func (sr *StarlarkRecipe) GetName() string { return sr.Name }
