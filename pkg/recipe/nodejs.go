package recipe

import (
	"encoding/json"
	"fmt"
	"pi/pkg/archive"
	"pi/pkg/config"
	"strings"
)

type nodejsVersion struct {
	Version string   `json:"version"`
	Files   []string `json:"files"`
}

func GetNodejsRecipe() *Recipe {
	return &Recipe{
		Name:         "nodejs",
		DiscoveryURL: "https://nodejs.org/dist/index.json",
		Parser:       parseNodejs,
		Filter: func(cfg *config.Config, p PackageDefinition, version string) bool {
			if version != "latest" && version != "" && !strings.HasPrefix(p.Version, version) {
				return false
			}
			return archive.IsSupported(p.Filename)
		},
	}
}

func parseNodejs(data []byte) ([]PackageDefinition, error) {
	var versions []nodejsVersion
	if err := json.Unmarshal(data, &versions); err != nil {
		return nil, err
	}

	var pkgs []PackageDefinition
	for _, v := range versions {
		version := strings.TrimPrefix(v.Version, "v")
		for _, file := range v.Files {
			osStr, archStr, ok := mapNodejsFile(file)
			if !ok {
				continue
			}

			osType, _ := config.ParseOS(osStr)
			archType, _ := config.ParseArch(archStr)

			// Construct URL
			// Example: https://nodejs.org/dist/v20.11.1/node-v20.11.1-linux-x64.tar.gz
			ext := ".tar.gz"
			if osType == config.OSWindows {
				ext = ".zip"
			}
			filename := fmt.Sprintf("node-v%s-%s%s", version, file, ext)
			url := fmt.Sprintf("https://nodejs.org/dist/v%s/%s", version, filename)

			pkgs = append(pkgs, PackageDefinition{
				Name:     "nodejs",
				Version:  version,
				OS:       osType,
				Arch:     archType,
				URL:      url,
				Filename: filename,
			})
		}
	}
	return pkgs, nil
}

func mapNodejsFile(file string) (os, arch string, ok bool) {
	switch file {
	case "linux-x64":
		return "linux", "x64", true
	case "linux-arm64":
		return "linux", "arm64", true
	case "osx-x64-tar":
		return "darwin", "x64", true
	case "osx-arm64-tar":
		return "darwin", "arm64", true
	case "win-x64-zip":
		return "windows", "x64", true
	case "win-arm64-zip":
		return "windows", "arm64", true
	default:
		return "", "", false
	}
}
