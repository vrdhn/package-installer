// Package recipe provides the core types for Starlark-based package discovery.
package recipe

import (
	"embed"
	"pi/pkg/common"
	"pi/pkg/config"
)

// BuiltinRecipes contains the Starlark recipes bundled with the pi binary.
//
//go:embed recipes/*.star
var BuiltinRecipes embed.FS

// PackageDefinition is a type alias for the common package definition.
type PackageDefinition = common.PackageDefinition

// Fetcher is a function type used by recipes to retrieve remote content (e.g., HTML or JSON)
// during the package discovery process.
type Fetcher func(url string) ([]byte, error)

// PinnedRecipe wraps a StarlarkRecipe and restricts its execution to a specific
// regex pattern that was matched during the repository resolution phase.
type PinnedRecipe struct {
	base  *StarlarkRecipe
	regex string
}

// NewPinnedRecipe creates a new PinnedRecipe for the given base recipe and regex pattern.
func NewPinnedRecipe(base *StarlarkRecipe, regex string) *PinnedRecipe {
	return &PinnedRecipe{base: base, regex: regex}
}

// GetName returns the name of the pinned recipe.
func (r *PinnedRecipe) GetName() string { return r.base.GetName() }

// Execute runs the pinned recipe's handler for the specified package and version query.
func (r *PinnedRecipe) Execute(cfg config.Config, pkgName string, versionQuery string, fetch Fetcher) ([]PackageDefinition, error) {
	return r.base.ExecuteRegex(cfg, pkgName, versionQuery, fetch, r.regex)
}

// GetName returns the name of the Starlark recipe.
func (sr *StarlarkRecipe) GetName() string { return sr.Name }
