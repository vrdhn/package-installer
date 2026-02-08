package recipe

import (
	"embed"
	"pi/pkg/common"
	"pi/pkg/config"
)

//go:embed recipes/*.star
var BuiltinRecipes embed.FS

// PackageDefinition is an alias for common.PackageDefinition.
type PackageDefinition = common.PackageDefinition

// Fetcher is a function that retrieves data from a URL.
type Fetcher func(url string) ([]byte, error)

// Recipe defines how to discover and resolve packages.
type Recipe interface {
	GetName() string
	Execute(cfg config.Config, pkgName string, versionQuery string, fetch Fetcher) ([]PackageDefinition, error)
}

// PinnedRecipe wraps a Starlark recipe and pins it to a specific regex.
// Immutable.
type PinnedRecipe struct {
	base  *StarlarkRecipe
	regex string
}

func NewPinnedRecipe(base *StarlarkRecipe, regex string) *PinnedRecipe {
	return &PinnedRecipe{base: base, regex: regex}
}

func (r *PinnedRecipe) GetName() string { return r.base.GetName() }

func (r *PinnedRecipe) Execute(cfg config.Config, pkgName string, versionQuery string, fetch Fetcher) ([]PackageDefinition, error) {
	return r.base.ExecuteRegex(cfg, pkgName, versionQuery, fetch, r.regex)
}

func (sr *StarlarkRecipe) GetName() string { return sr.Name }
