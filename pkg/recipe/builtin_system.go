package recipe

import (
	"fmt"
	"pi/pkg/config"

	"go.starlark.net/starlark"
)

func newGetOSBuiltin(sr *StarlarkRecipe) *starlark.Builtin {
	def := CommandDef{
		Name: "get_os",
		Desc: "Returns the target operating system.",
	}

	return NewStrictBuiltin(def, func(kwargs map[string]starlark.Value) (starlark.Value, error) {
		val := sr.thread.Local(keyConfig)
		if val == nil {
			return nil, fmt.Errorf("get_os called without active config")
		}
		cfg := val.(config.Config)
		return starlark.String(cfg.GetOS()), nil
	})
}

func newGetArchBuiltin(sr *StarlarkRecipe) *starlark.Builtin {
	def := CommandDef{
		Name: "get_arch",
		Desc: "Returns the target architecture.",
	}

	return NewStrictBuiltin(def, func(kwargs map[string]starlark.Value) (starlark.Value, error) {
		val := sr.thread.Local(keyConfig)
		if val == nil {
			return nil, fmt.Errorf("get_arch called without active config")
		}
		cfg := val.(config.Config)
		return starlark.String(cfg.GetArch()), nil
	})
}
