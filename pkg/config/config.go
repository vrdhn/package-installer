package config

import (
	"fmt"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/adrg/xdg"
)

// Config holds the base directories and system info for pi.
// Mutable
type config struct {
	cacheDir  string
	configDir string
	stateDir  string

	pkgDir       string
	downloadDir  string
	recipeDir    string
	homeDir      string
	discoveryDir string

	os   OSType
	arch ArchType

	user     string
	hostHome string
}

type Config interface {
	GetCacheDir() string
	GetConfigDir() string
	GetStateDir() string
	GetPkgDir() string
	GetDownloadDir() string
	GetRecipeDir() string
	GetHomeDir() string
	GetDiscoveryDir() string
	GetOS() OSType
	GetArch() ArchType
	GetUser() string
	GetHostHome() string
}

func (c *config) GetCacheDir() string     { return c.cacheDir }
func (c *config) GetConfigDir() string    { return c.configDir }
func (c *config) GetStateDir() string     { return c.stateDir }
func (c *config) GetPkgDir() string       { return c.pkgDir }
func (c *config) GetDownloadDir() string  { return c.downloadDir }
func (c *config) GetRecipeDir() string    { return c.recipeDir }
func (c *config) GetHomeDir() string      { return c.homeDir }
func (c *config) GetDiscoveryDir() string { return c.discoveryDir }
func (c *config) GetOS() OSType           { return c.os }
func (c *config) GetArch() ArchType       { return c.arch }
func (c *config) GetUser() string         { return c.user }
func (c *config) GetHostHome() string     { return c.hostHome }

func (c *config) updateDerived() {
	c.pkgDir = filepath.Join(c.cacheDir, "pkgs")
	c.downloadDir = filepath.Join(c.cacheDir, "downloads")
	c.recipeDir = filepath.Join(c.configDir, "recipes")
	c.homeDir = filepath.Join(c.stateDir, "homes")
	c.discoveryDir = filepath.Join(c.cacheDir, "discovery")
}

// Init initializes the configuration using XDG base directories.
func Init() (Config, error) {
	osType, _ := ParseOS(runtime.GOOS)
	archType, _ := ParseArch(runtime.GOARCH)

	u, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	c := &config{
		cacheDir:  filepath.Join(xdg.CacheHome, "pi"),
		configDir: filepath.Join(xdg.ConfigHome, "pi"),
		stateDir:  filepath.Join(xdg.StateHome, "pi"),
		os:        osType,
		arch:      archType,
		user:      u.Username,
		hostHome:  u.HomeDir,
	}

	c.updateDerived()

	return c, nil

}
