package config

import (
	"path/filepath"
	"runtime"

	"github.com/adrg/xdg"
)

// Config holds the base directories and system info for pi.
type Config struct {
	CacheDir  string // XDG_CACHE_HOME/pi
	ConfigDir string // XDG_CONFIG_HOME/pi
	StateDir  string // XDG_STATE_HOME/pi

	// Derived paths
	PkgDir       string // CacheDir/pkgs
	DownloadDir  string // CacheDir/downloads
	RecipeDir    string // ConfigDir/recipes
	HomeDir      string // StateDir/homes
	DiscoveryDir string // CacheDir/discovery

	// System info
	OS   OSType
	Arch ArchType
}

// Init initializes the configuration using XDG base directories.
func Init() (*Config, error) {
	osType, _ := ParseOS(runtime.GOOS)
	archType, _ := ParseArch(runtime.GOARCH)

	c := &Config{
		CacheDir:  filepath.Join(xdg.CacheHome, "pi"),
		ConfigDir: filepath.Join(xdg.ConfigHome, "pi"),
		StateDir:  filepath.Join(xdg.StateHome, "pi"),
		OS:        osType,
		Arch:      archType,
	}

	c.PkgDir = filepath.Join(c.CacheDir, "pkgs")
	c.DownloadDir = filepath.Join(c.CacheDir, "downloads")
	c.RecipeDir = filepath.Join(c.ConfigDir, "recipes")
	c.HomeDir = filepath.Join(c.StateDir, "homes")
	c.DiscoveryDir = filepath.Join(c.CacheDir, "discovery")

	return c, nil
}
