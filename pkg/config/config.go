package config

import (
	"fmt"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/adrg/xdg"
)

// config holds the base directories and system info for pi.
// This struct is immutable after initialization.
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

// Config provides access to application-wide paths and system environment information.
type Config = *config

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

// Init initializes the configuration by detecting the system environment
// and setting up XDG-compliant base directories.
func Init() (Config, error) {
	osType, _ := ParseOS(runtime.GOOS)
	archType, _ := ParseArch(runtime.GOARCH)

	u, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	cacheDir := filepath.Join(xdg.CacheHome, "pi")
	configDir := filepath.Join(xdg.ConfigHome, "pi")
	stateDir := filepath.Join(xdg.StateHome, "pi")

	return &config{
		cacheDir:     cacheDir,
		configDir:    configDir,
		stateDir:     stateDir,
		pkgDir:       filepath.Join(cacheDir, "pkgs"),
		downloadDir:  filepath.Join(cacheDir, "downloads"),
		recipeDir:    filepath.Join(configDir, "recipes"),
		homeDir:      filepath.Join(stateDir, "homes"),
		discoveryDir: filepath.Join(cacheDir, "discovery"),
		os:           osType,
		arch:         archType,
		user:         u.Username,
		hostHome:     u.HomeDir,
	}, nil
}

//TODO: We need to split the Config into separated structs, separate cave
// data form cache data, so that NewTestConfig ignore the user/home

// NewTestConfig creates a configuration rooted at the specified base directory.
func NewTestConfig(basePath string) Config {
	osType, _ := ParseOS(runtime.GOOS)
	archType, _ := ParseArch(runtime.GOARCH)

	u, _ := user.Current()
	var username, home string
	if u != nil {
		username = u.Username
		home = u.HomeDir
	}

	absBase, _ := filepath.Abs(basePath)

	return &config{
		cacheDir:     absBase,
		configDir:    absBase,
		stateDir:     absBase,
		pkgDir:       filepath.Join(absBase, "pkgs"),
		downloadDir:  filepath.Join(absBase, "downloads"),
		recipeDir:    filepath.Join(absBase, "recipes"),
		homeDir:      filepath.Join(absBase, "homes"),
		discoveryDir: filepath.Join(absBase, "discovery"),
		os:           osType,
		arch:         archType,
		user:         username,
		hostHome:     home,
	}
}
