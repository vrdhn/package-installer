package recipe

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"pi/pkg/cache"
	"pi/pkg/config"
	"regexp"
	"sort"
	"strings"
	"time"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// Immutable
type StarlarkRecipe struct {
	Name    string
	Source  string
	thread  *starlark.Thread
	globals starlark.StringDict

	registry       map[string]starlark.Callable
	regexCache     map[string]*regexp.Regexp
	registryLoaded bool
}

func NewStarlarkRecipe(name, source string, printFunc func(string)) (*StarlarkRecipe, error) {
	return &StarlarkRecipe{
		Name:   name,
		Source: source,
		thread: &starlark.Thread{
			Name: name,
			Print: func(thread *starlark.Thread, msg string) {
				if printFunc != nil {
					printFunc(msg)
				} else {
					slog.Info(msg, "recipe", thread.Name)
				}
			},
		},
	}, nil
}

const (
	keyFetcher   = "fetcher"
	keyCollector = "collector"
	keyConfig    = "config"
)

// Execute identifies and runs the appropriate handler for a given package name.
func (sr *StarlarkRecipe) Execute(cfg config.Config, pkgName string, versionQuery string, fetch Fetcher) ([]PackageDefinition, error) {
	if err := sr.ensureRegistryLoaded(); err != nil {
		return nil, err
	}

	handler, regexKey, err := sr.matchHandler(pkgName)
	if err != nil {
		return nil, err
	}
	if handler == nil {
		return nil, fmt.Errorf("recipe not applicable: %s", sr.Name)
	}

	return sr.executeHandler(cfg, pkgName, versionQuery, fetch, regexKey, handler)
}

// ExecuteRegex runs the specific handler registered for the provided regex pattern.
func (sr *StarlarkRecipe) ExecuteRegex(cfg config.Config, pkgName string, versionQuery string, fetch Fetcher, regexKey string) ([]PackageDefinition, error) {
	if err := sr.ensureRegistryLoaded(); err != nil {
		return nil, err
	}

	handler, ok := sr.registry[regexKey]
	if !ok {
		return nil, fmt.Errorf("recipe handler not found for regex '%s' in %s", regexKey, sr.Name)
	}

	return sr.executeHandler(cfg, pkgName, versionQuery, fetch, regexKey, handler)
}

func (sr *StarlarkRecipe) ensureRegistryLoaded() error {
	if !sr.registryLoaded {
		if err := sr.loadRegistry(); err != nil {
			return err
		}
	}
	if len(sr.registry) == 0 {
		return fmt.Errorf("recipe does not define any package handlers: %s", sr.Name)
	}
	return nil
}

// Registry returns a sorted list of all regex patterns registered by this recipe.
func (sr *StarlarkRecipe) Registry(cfg config.Config) ([]string, error) {
	if err := sr.ensureRegistryLoaded(); err != nil {
		return nil, err
	}

	var keys []string
	for k := range sr.registry {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, nil
}

// GetRegistryInfo returns a map of registered patterns to the names of their Starlark handler functions.
func (sr *StarlarkRecipe) GetRegistryInfo(cfg config.Config) (map[string]string, error) {
	if _, err := sr.Registry(cfg); err != nil {
		return nil, err
	}
	info := make(map[string]string)
	for k, v := range sr.registry {
		info[k] = v.Name()
	}
	return info, nil
}

func (sr *StarlarkRecipe) loadRegistry() error {
	sr.registry = make(map[string]starlark.Callable)
	sr.regexCache = make(map[string]*regexp.Regexp)

	builtins := starlark.StringDict{
		"struct":                   starlark.NewBuiltin("struct", starlarkstruct.Make),
		"json":                     starlarkstruct.FromStringDict(starlark.String("json"), jsonBuiltins()),
		"html":                     starlarkstruct.FromStringDict(starlark.String("html"), htmlBuiltins()),
		"jq":                       starlarkstruct.FromStringDict(starlark.String("jq"), jqBuiltins()),
		"download":                 newDownloadBuiltin(sr),
		"download_github_releases": newDownloadGitHubReleasesBuiltin(sr),
		"add_version":              newAddVersionBuiltin(sr),
		"add_pkgdef":               newAddPkgdefBuiltin(sr),
		"get_os":                   newGetOSBuiltin(sr),
		"get_arch":                 newGetArchBuiltin(sr),
	}

	globals, err := starlark.ExecFile(sr.thread, sr.Name+".star", sr.Source, builtins)
	if err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			return fmt.Errorf("failed to load recipe %s:\n%s", sr.Name, evalErr.Backtrace())
		}
		return err
	}
	sr.globals = globals
	sr.registryLoaded = true
	return nil
}

func (sr *StarlarkRecipe) matchHandler(pi string) (starlark.Callable, string, error) {
	if len(sr.registry) == 0 {
		return nil, "", nil
	}
	var keys []string
	for k := range sr.registry {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		re, err := sr.getRegex(key)
		if err != nil {
			return nil, "", err
		}
		if re.MatchString(pi) {
			return sr.registry[key], key, nil
		}
	}
	return nil, "", nil
}

func (sr *StarlarkRecipe) getRegex(key string) (*regexp.Regexp, error) {
	if re, ok := sr.regexCache[key]; ok {
		return re, nil
	}
	re, err := CompileAnchored(key)
	if err != nil {
		return nil, fmt.Errorf("invalid regex '%s' in recipe %s: %w", key, sr.Name, err)
	}
	sr.regexCache[key] = re
	return re, nil
}

func (sr *StarlarkRecipe) handlerCachePath(cfg config.Config, pkgName string, versionQuery string, regexKey string) (string, error) {
	key := fmt.Sprintf("%s|%s|%s|%s|%s|%s", sr.Name, regexKey, pkgName, versionQuery, cfg.GetOS(), cfg.GetArch())
	sum := sha256.Sum256([]byte(key))
	fileName := fmt.Sprintf("handler_%x.json", sum[:])
	return filepath.Join(cfg.GetDiscoveryDir(), fileName), nil
}

func (sr *StarlarkRecipe) loadHandlerCache(cfg config.Config, pkgName string, versionQuery string, regexKey string) ([]PackageDefinition, bool, error) {
	path, err := sr.handlerCachePath(cfg, pkgName, versionQuery, regexKey)
	if err != nil {
		return nil, false, err
	}
	if !cache.IsFresh(path, time.Hour) {
		return nil, false, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}
	var pkgs []PackageDefinition
	if err := json.Unmarshal(data, &pkgs); err != nil {
		return nil, false, err
	}
	return pkgs, true, nil
}

func (sr *StarlarkRecipe) storeHandlerCache(cfg config.Config, pkgName string, versionQuery string, regexKey string, pkgs []PackageDefinition) error {
	path, err := sr.handlerCachePath(cfg, pkgName, versionQuery, regexKey)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(pkgs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (sr *StarlarkRecipe) Test(cfg config.Config) error {
	if err := sr.ensureRegistryLoaded(); err != nil {
		return err
	}

	var testFuncs []string
	for name, val := range sr.globals {
		if strings.HasPrefix(name, "test_") {
			if _, ok := val.(starlark.Callable); ok {
				testFuncs = append(testFuncs, name)
			}
		}
	}
	sort.Strings(testFuncs)

	if len(testFuncs) == 0 {
		return fmt.Errorf("no test functions found (starting with 'test_')")
	}

	for _, name := range testFuncs {
		slog.Info("Running test", "name", name)
		if _, err := starlark.Call(sr.thread, sr.globals[name], nil, nil); err != nil {
			slog.Error("Test failed", "name", name, "error", err)
			return mungeEvalError(name, err)
		}
		slog.Info("Test OK", "name", name)
	}

	return nil
}
