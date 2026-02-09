package repo

import (
	"pi/pkg/config"
	"pi/pkg/display"
	"pi/pkg/lazyjson"
	"regexp"

	"github.com/google/uuid"
)

// RepoConfig represents the configuration for a single recipe repository.
type RepoConfig struct {
	// Name is the display name of the repository.
	Name string `json:"name"`
	// URL is the location of the repository (local path or remote URL).
	URL string `json:"url"`
	// UUID is a unique identifier for the repository.
	UUID uuid.UUID `json:"uuid"`
}

// IndexEntry represents a mapping from a package pattern to a specific recipe and handler.
type IndexEntry struct {
	// RepoUUID is the identifier of the repository where the recipe is located.
	RepoUUID uuid.UUID `json:"repo_uuid"`
	// RepoName is the name of the repository (resolved at runtime).
	RepoName string `json:"-"`
	// RecipeName is the name of the Starlark recipe file (without .star).
	RecipeName string `json:"recipe_name"`
	// Pattern is the regex pattern that this handler supports.
	Pattern string `json:"pattern"`
	// Handler is the name of the Starlark function that handles this pattern.
	Handler string `json:"handler"`
}

type repoRegistry struct {
	Repos []RepoConfig `json:"repos"`
	Index []IndexEntry `json:"index"`
}

type resolvedRecipe struct {
	recipeName string
	regexKey   string
}

// Manager is the implementation of repository management.
type manager struct {
	recipes map[string]string // recipe name -> source
	repos   []RepoConfig
	disp    display.Display
	cfg     config.Config
	repoMgr lazyjson.Manager[repoRegistry]

	// In-memory index of all packages
	index []IndexEntry
	// Cache for resolution: pkgName -> {recipeName, regexKey}
	resolveCache     map[string]resolvedRecipe
	compiledPatterns map[string]*regexp.Regexp
}

// ResolvedQuery represents a matched index entry for a package query.
type ResolvedQuery struct {
	// RepoUUID is the identifier of the repository.
	RepoUUID uuid.UUID
	// RepoName is the display name of the repository.
	RepoName string
	// RecipeName is the name of the recipe.
	RecipeName string
	// Pattern is the matched regex pattern.
	Pattern string
}
