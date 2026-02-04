package recipe

import (
	"fmt"

	"go.starlark.net/starlark"
)

func newAddEcosystemBuiltin(sr *StarlarkRecipe) *starlark.Builtin {
	def := CommandDef{
		Name: "add_ecosystem",
		Desc: "Registers a new ecosystem handler.",
		Params: []ParamDef{
			{Name: "name", Type: "string", Desc: "Ecosystem name (e.g. 'npm', 'pip')"},
		},
	}

	return NewStrictBuiltin(def, func(kwargs map[string]starlark.Value) (starlark.Value, error) {
		if sr.currentCtx == nil {
			return nil, fmt.Errorf("add_ecosystem called without active context")
		}
		// Placeholder for future ecosystem logic
		return starlark.None, nil
	})
}
