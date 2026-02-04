package recipe

import (
	"embed"
	"pi/pkg/config"
)

//go:embed recipes/*.star
var BuiltinRecipes embed.FS

// PackageDefinition describes a specific build of a package.
// Immutable
type PackageDefinition struct {
	Name          string
	Version       string
	ReleaseStatus string
	OS            config.OSType
	Arch          config.ArchType
	URL           string
	Filename      string
	Checksum      string
	Env           map[string]string
	Symlinks      map[string]string
}

// DiscoveryContext provides the environment for recipe execution.
type DiscoveryContext struct {
	Config       config.ReadOnly
	PkgName      string
	VersionQuery string
	Download     func(url string) ([]byte, error)
	AddVersion   func(p PackageDefinition)
}

// Recipe defines how to discover and resolve packages for an ecosystem.
type Recipe interface {
	GetName() string
	Execute(ctx *DiscoveryContext) ([]PackageDefinition, error)
}

// SelectedRecipe wraps a Starlark recipe and pins it to a specific regex.
// Immutable.
type SelectedRecipe struct {
	base  *StarlarkRecipe
	regex string
}

func NewSelectedRecipe(base *StarlarkRecipe, regex string) *SelectedRecipe {
	return &SelectedRecipe{base: base, regex: regex}
}

func (r *SelectedRecipe) GetName() string { return r.base.GetName() }

func (r *SelectedRecipe) Execute(ctx *DiscoveryContext) ([]PackageDefinition, error) {
	return r.base.ExecuteRegex(ctx, r.regex)
}

// GoRecipe is the legacy Go-based implementation
// Immutable
type GoRecipe struct {
	Name         string
	DiscoveryURL string
	Parser       func(data []byte) ([]PackageDefinition, error)
}

func (r *GoRecipe) GetName() string { return r.Name }
func (r *GoRecipe) Execute(ctx *DiscoveryContext) ([]PackageDefinition, error) {
	data, err := ctx.Download(r.DiscoveryURL)
	if err != nil {
		return nil, err
	}
	return r.Parser(data)
}

func (sr *StarlarkRecipe) GetName() string { return sr.Name }
