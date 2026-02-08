package pkgs

import (
	"pi/pkg/common"
	"pi/pkg/recipe"
	"sort"

	"github.com/google/uuid"
)

// PackageDefinition is an alias for common.PackageDefinition for convenience.
type PackageDefinition = common.PackageDefinition

type PackageRegistry struct {
	Versions []PackageDefinition `json:"versions"`
}

// UpdateVersions updates the package.json with new versions for specific RepoUUID and Name.
// It removes old entries for the same RepoUUID and Name before adding new ones.
func (m *manager) UpdateVersions(repoUUID uuid.UUID, name string, newVersions []recipe.PackageDefinition) error {
	err := m.pkgMgr.Modify(func(reg *PackageRegistry) error {
		// Filter out old entries for this name in this repo
		var filtered []PackageDefinition
		for _, v := range reg.Versions {
			if v.RepoUUID == repoUUID && v.Name == name {
				continue
			}
			filtered = append(filtered, v)
		}

		// Add new entries
		for _, nv := range newVersions {
			v := nv
			v.RepoUUID = repoUUID
			filtered = append(filtered, v)
		}

		reg.Versions = filtered
		return nil
	})
	if err != nil {
		return err
	}
	return m.pkgMgr.Save()
}

func SortPackageDefinitions(versions []PackageDefinition) {
	sort.Slice(versions, func(i, j int) bool {
		// Compare timestamps if available
		if versions[i].ReleaseDate != "" && versions[j].ReleaseDate != "" {
			return versions[i].ReleaseDate < versions[j].ReleaseDate
		}
		// Fallback to version string
		return versions[i].Version < versions[j].Version
	})
}
