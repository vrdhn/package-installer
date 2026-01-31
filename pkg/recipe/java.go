package recipe

import (
	"encoding/json"
	"pi/pkg/archive"
	"pi/pkg/config"
	"strings"
)

type foojayPackage struct {
	Filename        string `json:"filename"`
	JavaVersion     string `json:"java_version"`
	Distribution    string `json:"distribution"`
	OperatingSystem string `json:"operating_system"`
	Architecture    string `json:"architecture"`
	Links           struct {
		PkgDownloadRedirect string `json:"pkg_download_redirect"`
	} `json:"links"`
	Checksum string `json:"checksum"`
}

type foojayResponse struct {
	Result []foojayPackage `json:"result"`
}

func GetJavaRecipe() *Recipe {
	return &Recipe{
		Name: "java",
		// We use a broad discovery URL for foojay, or we might need to parameterize it later.
		// For now, let's use a query that returns some latest packages.
		DiscoveryURL: "https://api.foojay.io/disco/v3.0/packages?package_type=jdk&release_status=ga",
		Parser:       parseFoojay,
		Filter: func(cfg *config.Config, p PackageDefinition, version string) bool {
			if version != "latest" && version != "" && !strings.HasPrefix(p.Version, version) {
				return false
			}
			return archive.IsSupported(p.Filename)
		},
	}
}

func parseFoojay(data []byte) ([]PackageDefinition, error) {
	var resp foojayResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	var pkgs []PackageDefinition
	for _, p := range resp.Result {
		// Standardize OS/Arch labels
		osType, _ := config.ParseOS(p.OperatingSystem)
		archType, _ := config.ParseArch(p.Architecture)

		pkgs = append(pkgs, PackageDefinition{
			Name:     "java-" + p.Distribution,
			Version:  p.JavaVersion,
			OS:       osType,
			Arch:     archType,
			URL:      p.Links.PkgDownloadRedirect,
			Filename: p.Filename,
			Checksum: p.Checksum,
		})
	}
	return pkgs, nil
}
