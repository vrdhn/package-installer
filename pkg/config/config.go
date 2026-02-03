package config

import (
	"fmt"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/adrg/xdg"
)

// ReadOnly defines the read-only interface for Config.
// Immutable
type ReadOnly interface {
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
	Freeze()
	Checkout() Writable
}

// Writable defines the writable interface for Config.
// Mutable
type Writable interface {
	ReadOnly
	SetCacheDir(string)
	SetConfigDir(string)
	SetStateDir(string)
}

// Config holds the base directories and system info for pi.
// Mutable
type Config struct {
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

	frozen bool
	edited bool
}

var _ ReadOnly = (*Config)(nil)
var _ Writable = (*Config)(nil)

func (c *Config) GetCacheDir() string     { return c.cacheDir }
func (c *Config) GetConfigDir() string    { return c.configDir }
func (c *Config) GetStateDir() string     { return c.stateDir }
func (c *Config) GetPkgDir() string       { return c.pkgDir }
func (c *Config) GetDownloadDir() string  { return c.downloadDir }
func (c *Config) GetRecipeDir() string    { return c.recipeDir }
func (c *Config) GetHomeDir() string      { return c.homeDir }
func (c *Config) GetDiscoveryDir() string { return c.discoveryDir }
func (c *Config) GetOS() OSType           { return c.os }
func (c *Config) GetArch() ArchType       { return c.arch }
func (c *Config) GetUser() string         { return c.user }
func (c *Config) GetHostHome() string     { return c.hostHome }

func (c *Config) SetCacheDir(s string) {
	if c.frozen {
		panic("cannot modify frozen config")
	}
	c.cacheDir = s
	c.updateDerived()
}

func (c *Config) SetConfigDir(s string) {
	if c.frozen {
		panic("cannot modify frozen config")
	}
	c.configDir = s
	c.updateDerived()
}

func (c *Config) SetStateDir(s string) {
	if c.frozen {
		panic("cannot modify frozen config")
	}
	c.stateDir = s
	c.updateDerived()
}

func (c *Config) Freeze() {
	c.frozen = true
}

func (c *Config) Checkout() Writable {
	if c.frozen {
		panic("cannot checkout from frozen config")
	}
	if c.edited {
		panic("config already checked out")
	}
	c.edited = true
	return c
}

func (c *Config) updateDerived() {
	c.pkgDir = filepath.Join(c.cacheDir, "pkgs")
	c.downloadDir = filepath.Join(c.cacheDir, "downloads")
	c.recipeDir = filepath.Join(c.configDir, "recipes")
	c.homeDir = filepath.Join(c.stateDir, "homes")
	c.discoveryDir = filepath.Join(c.cacheDir, "discovery")
}

// Init initializes the configuration using XDG base directories.
func Init() (ReadOnly, error) {
	osType, _ := ParseOS(runtime.GOOS)
	archType, _ := ParseArch(runtime.GOARCH)

	u, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	c := &Config{
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
