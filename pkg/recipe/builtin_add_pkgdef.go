package recipe

import (
	"fmt"

	"go.starlark.net/starlark"
)

func newAddPkgdefBuiltin(sr *StarlarkRecipe) *starlark.Builtin {
	return starlark.NewBuiltin("add_pkgdef", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var regex string
		var handler starlark.Callable

		if len(args) > 2 {
			return nil, fmt.Errorf("add_pkgdef: expected 2 arguments, got %d", len(args))
		}

		if len(args) == 2 {
			rs, ok := args[0].(starlark.String)
			if !ok {
				return nil, fmt.Errorf("add_pkgdef: regex must be a string")
			}
			regex = rs.GoString()

			h, ok := args[1].(starlark.Callable)
			if !ok {
				return nil, fmt.Errorf("add_pkgdef: handler must be callable")
			}
			handler = h
		}

		for _, kv := range kwargs {
			key := kv[0].(starlark.String).GoString()
			switch key {
			case "regex":
				rs, ok := kv[1].(starlark.String)
				if !ok {
					return nil, fmt.Errorf("add_pkgdef: regex must be a string")
				}
				regex = rs.GoString()
			case "handler":
				h, ok := kv[1].(starlark.Callable)
				if !ok {
					return nil, fmt.Errorf("add_pkgdef: handler must be callable")
				}
				handler = h
			default:
				return nil, fmt.Errorf("add_pkgdef: unknown argument '%s'", key)
			}
		}

		if regex == "" || handler == nil {
			return nil, fmt.Errorf("add_pkgdef: requires regex and handler")
		}

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
