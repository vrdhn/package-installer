package pkgs

import (
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"pi/pkg/recipe"
	"sort"
)

type PackageVersion struct {
	RepoUUID    string
	Name        string
	Version     string
	Arch        string
	OS          string
	FileHash    string
	FileURL     string
	Timestamp   string
	ReleaseType string
}

func (m *manager) loadRepoCSV() ([]PackageVersion, error) {
	path := filepath.Join(m.SysConfig.GetConfigDir(), "repo.csv")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	var versions []PackageVersion
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(record) < 9 {
			continue
		}
		versions = append(versions, PackageVersion{
			RepoUUID:    record[0],
			Name:        record[1],
			Version:     record[2],
			Arch:        record[3],
			OS:          record[4],
			FileHash:    record[5],
			FileURL:     record[6],
			Timestamp:   record[7],
			ReleaseType: record[8],
		})
	}
	return versions, nil
}

func (m *manager) saveRepoCSV(versions []PackageVersion) error {
	path := filepath.Join(m.SysConfig.GetConfigDir(), "repo.csv")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	for _, v := range versions {
		err := w.Write([]string{
			v.RepoUUID,
			v.Name,
			v.Version,
			v.Arch,
			v.OS,
			v.FileHash,
			v.FileURL,
			v.Timestamp,
			v.ReleaseType,
		})
		if err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

// UpdateVersions updates the repo.csv with new versions for specific RepoUUID and Name.
// It removes old entries for the same RepoUUID and Name before adding new ones.
func (m *manager) UpdateVersions(repoUUID string, name string, newVersions []recipe.PackageDefinition) error {
	all, err := m.loadRepoCSV()
	if err != nil {
		return err
	}

	// Filter out old entries for this name in this repo
	var filtered []PackageVersion
	for _, v := range all {
		if v.RepoUUID == repoUUID && v.Name == name {
			continue
		}
		filtered = append(filtered, v)
	}

	// Add new entries
	for _, nv := range newVersions {
		filtered = append(filtered, PackageVersion{
			RepoUUID:    repoUUID,
			Name:        nv.Name,
			Version:     nv.Version,
			Arch:        string(nv.Arch),
			OS:          string(nv.OS),
			FileHash:    nv.Checksum,
			FileURL:     nv.URL,
			Timestamp:   nv.ReleaseDate,
			ReleaseType: nv.ReleaseStatus,
		})
	}

	return m.saveRepoCSV(filtered)
}

func SortPackageVersions(versions []PackageVersion) {
	sort.Slice(versions, func(i, j int) bool {
		// Compare timestamps if available
		if versions[i].Timestamp != "" && versions[j].Timestamp != "" {
			return versions[i].Timestamp < versions[j].Timestamp
		}
		// Fallback to version string
		return versions[i].Version < versions[j].Version
	})
}
