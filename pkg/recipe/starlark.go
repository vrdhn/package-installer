package recipe

import (
	"encoding/json"
	"fmt"
	"pi/pkg/archive"
	"pi/pkg/config"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

type StarlarkRecipe struct {
	Name    string
	Source  string
	thread  *starlark.Thread
	globals starlark.StringDict
}

func NewStarlarkRecipe(name, source string) (*StarlarkRecipe, error) {
	sr := &StarlarkRecipe{
		Name:   name,
		Source: source,
		thread: &starlark.Thread{Name: name},
	}

	// Define built-ins
	builtins := starlark.StringDict{
		"struct": starlark.NewBuiltin("struct", starlarkstruct.Make),
		"json":   starlarkstruct.FromStringDict(starlark.String("json"), jsonBuiltins()),
	}

	globals, err := starlark.ExecFile(sr.thread, name+".star", source, builtins)
	if err != nil {
		return nil, err
	}
	sr.globals = globals

	return sr, nil
}

func jsonBuiltins() starlark.StringDict {
	return starlark.StringDict{
		"decode": starlark.NewBuiltin("decode", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var s string
			if err := starlark.UnpackArgs("decode", args, kwargs, "data", &s); err != nil {
				return nil, err
			}
			var data any
			if err := json.Unmarshal([]byte(s), &data); err != nil {
				return nil, err
			}
			return toStarlark(data), nil
		}),
	}
}

func toStarlark(v any) starlark.Value {
	switch x := v.(type) {
	case bool:
		return starlark.Bool(x)
	case string:
		return starlark.String(x)
	case float64:
		return starlark.Float(x)
	case []any:
		var list []starlark.Value
		for _, item := range x {
			list = append(list, toStarlark(item))
		}
		return starlark.NewList(list)
	case map[string]any:
		dict := starlark.NewDict(len(x))
		for k, v := range x {
			dict.SetKey(starlark.String(k), toStarlark(v))
		}
		return dict
	default:
		return starlark.None
	}
}

func (sr *StarlarkRecipe) Discover(cfg *config.Config, versionQuery string) (string, string, error) {
	discover, ok := sr.globals["discover"]
	if !ok {
		return "", "", fmt.Errorf("discover function not found in recipe %s", sr.Name)
	}

	exts := archive.Extensions(cfg.OS)
	starlarkExts := starlark.NewList(nil)
	for _, ext := range exts {
		starlarkExts.Append(starlark.String(ext))
	}

	ctx := starlarkstruct.FromStringDict(starlark.String("context"), starlark.StringDict{
		"os":         starlark.String(cfg.OS),
		"arch":       starlark.String(cfg.Arch),
		"extensions": starlarkExts,
	})

	res, err := starlark.Call(sr.thread, discover, starlark.Tuple{starlark.String(versionQuery), ctx}, nil)
	if err != nil {
		return "", "", err
	}

	dict, ok := res.(*starlark.Dict)
	if !ok {
		return "", "", fmt.Errorf("discover must return a dict")
	}

	urlVal, ok, err := dict.Get(starlark.String("url"))
	if err != nil || !ok {
		return "", "", fmt.Errorf("discover result missing 'url'")
	}

	methodVal, ok, err := dict.Get(starlark.String("method"))
	method := "GET"
	if ok && methodVal != nil {
		method = methodVal.(starlark.String).GoString()
	}

	return urlVal.(starlark.String).GoString(), method, nil
}

func (sr *StarlarkRecipe) Parse(cfg *config.Config, data []byte, versionQuery string) ([]PackageDefinition, error) {
	parse, ok := sr.globals["parse"]
	if !ok {
		return nil, fmt.Errorf("parse function not found in recipe %s", sr.Name)
	}

	exts := archive.Extensions(cfg.OS)
	starlarkExts := starlark.NewList(nil)
	for _, ext := range exts {
		starlarkExts.Append(starlark.String(ext))
	}

	ctx := starlarkstruct.FromStringDict(starlark.String("context"), starlark.StringDict{
		"os":         starlark.String(cfg.OS),
		"arch":       starlark.String(cfg.Arch),
		"extensions": starlarkExts,
	})

	res, err := starlark.Call(sr.thread, parse, starlark.Tuple{starlark.String(string(data)), starlark.String(versionQuery), ctx}, nil)
	if err != nil {
		return nil, err
	}

	list, ok := res.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("parse must return a list")
	}

	var pkgs []PackageDefinition
	for i := 0; i < list.Len(); i++ {
		item := list.Index(i)
		dict, ok := item.(*starlark.Dict)
		if !ok {
			continue
		}

		pkg := PackageDefinition{}
		pkg.Name = getString(dict, "name")
		pkg.Version = getString(dict, "version")
		pkg.URL = getString(dict, "url")
		pkg.Filename = getString(dict, "filename")
		pkg.Checksum = getString(dict, "checksum")

		osStr := getString(dict, "os")
		archStr := getString(dict, "arch")

		pkg.OS, _ = config.ParseOS(osStr)
		pkg.Arch, _ = config.ParseArch(archStr)

		pkgs = append(pkgs, pkg)
	}

	return pkgs, nil
}

func getString(dict *starlark.Dict, key string) string {
	val, ok, _ := dict.Get(starlark.String(key))
	if ok {
		if s, ok := val.(starlark.String); ok {
			return s.GoString()
		}
	}
	return ""
}
