package recipe

import (
	"fmt"
	"log/slog"

	"go.starlark.net/starlark"
)

func newAddPkgdefBuiltin(sr *StarlarkRecipe) *starlark.Builtin {
	return NewStrictBuiltin(CommandDef{
		Name: "add_pkgdef",
		Desc: "Registers a package definition handler matching a regex.",
		Params: []ParamDef{
			{Name: "regex", Type: "string", Desc: "The anchored regex to match package names"},
			{Name: "handler", Type: "callable", Desc: "Function to call when regex matches: handler(pkg_name)"},
		},
	}, func(kwargs map[string]starlark.Value) (starlark.Value, error) {
		regex := asString(kwargs["regex"])
		handler := kwargs["handler"].(starlark.Callable)

		slog.Debug("add_pkgdef", "recipe", sr.Name, "regex", regex, "handler", handler.Name())

		if sr.registry == nil {
			sr.registry = make(map[string]starlark.Callable)
		}
		if _, exists := sr.registry[regex]; exists {
			return nil, fmt.Errorf("add_pkgdef: duplicate regex '%s'", regex)
		}
		sr.registry[regex] = handler

		return starlark.None, nil
	})
}
