package recipe

import (
	"fmt"
	"pi/pkg/config"

	"go.starlark.net/starlark"
)

func newAddVersionBuiltin(sr *StarlarkRecipe) *starlark.Builtin {
	def := CommandDef{
		Name: "add_version",
		Desc: "Registers a new package version into the discovery context.",
		Params: []ParamDef{
			{Name: "name", Type: "string", Desc: "Internal package name (e.g. 'nodejs')"},
			{Name: "version", Type: "string", Desc: "Version string (e.g. '20.11.0')"},
			{Name: "release_status", Type: "string", Desc: "Status: 'stable', 'lts', 'current', 'rc', 'ea'"},
			{Name: "release_date", Type: "string", Desc: "Release date (e.g. '2024-01-12')"},
			{Name: "os", Type: "string", Desc: "Target OS: 'linux', 'darwin', 'windows'"},
			{Name: "arch", Type: "string", Desc: "Target Arch: 'x64', 'arm64'"},
			{Name: "url", Type: "string", Desc: "Download URL for the archive"},
			{Name: "filename", Type: "string", Desc: "Local filename for the archive"},
			{Name: "checksum", Type: "string", Desc: "Optional SHA256 checksum"},
			{Name: "env", Type: "dict", Desc: "Optional environment variables"},
			{Name: "symlinks", Type: "dict", Desc: "Optional symlink mappings"},
		},
	}

	return NewStrictBuiltin(def, func(kwargs map[string]starlark.Value) (starlark.Value, error) {
		ctx := sr.currentCtx
		if ctx == nil || ctx.AddVersion == nil {
			return nil, fmt.Errorf("add_version called without active context")
		}

		nonNullable := []string{
			"name",
			"version",
			"release_status",
			"os",
			"arch",
			"url",
			"filename",
			"checksum",
			"env",
			"symlinks",
		}
		for _, key := range nonNullable {
			if isNone(kwargs[key]) {
				return nil, fmt.Errorf("%s cannot be None", key)
			}
		}

		pkg := PackageDefinition{
			Name:          asString(kwargs["name"]),
			Version:       asString(kwargs["version"]),
			ReleaseStatus: asString(kwargs["release_status"]),
			ReleaseDate:   asString(kwargs["release_date"]),
			URL:           asString(kwargs["url"]),
			Filename:      asString(kwargs["filename"]),
			Checksum:      asString(kwargs["checksum"]),
		}

		osType, err := config.ParseOS(asString(kwargs["os"]))
		if err != nil {
			return nil, err
		}
		archType, err := config.ParseArch(asString(kwargs["arch"]))
		if err != nil {
			return nil, err
		}
		pkg.OS = osType
		pkg.Arch = archType

		fmt.Printf("[%s] add_version: name=%s version=%s os=%s arch=%s\n", sr.Name, pkg.Name, pkg.Version, pkg.OS, pkg.Arch)

		if env, ok := kwargs["env"].(*starlark.Dict); ok {
			pkg.Env = make(map[string]string)
			for _, k := range env.Keys() {
				pkg.Env[asString(k)] = asString(mustGet(env, k))
			}
		} else {
			return nil, fmt.Errorf("env must be a dict")
		}

		if syms, ok := kwargs["symlinks"].(*starlark.Dict); ok {
			pkg.Symlinks = make(map[string]string)
			for _, k := range syms.Keys() {
				pkg.Symlinks[asString(k)] = asString(mustGet(syms, k))
			}
		} else {
			return nil, fmt.Errorf("symlinks must be a dict")
		}

		ctx.AddVersion(pkg)

		return starlark.None, nil
	})
}

func asString(v starlark.Value) string {
	if v == nil {
		return ""
	}
	if v == starlark.None {
		return ""
	}
	if s, ok := v.(starlark.String); ok {
		return s.GoString()
	}
	return fmt.Sprintf("%v", v)
}

func isNone(v starlark.Value) bool {
	if v == nil {
		return true
	}
	return v == starlark.None
}

func mustGet(d *starlark.Dict, k starlark.Value) starlark.Value {
	v, ok, _ := d.Get(k)
	if !ok {
		return starlark.None
	}
	return v
}
