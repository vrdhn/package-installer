package pkgs

import (
	"pi/pkg/recipe"

	"github.com/google/uuid"
)

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
