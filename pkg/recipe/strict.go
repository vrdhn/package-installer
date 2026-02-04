package recipe

import (
	"fmt"
	"strings"

	"go.starlark.net/starlark"
)

// ParamDef defines a single parameter for a strict builtin.
type ParamDef struct {
	Name string
	Type string
	Desc string
}

// CommandDef defines the schema for a strict builtin function.
type CommandDef struct {
	Name   string
	Desc   string
	Params []ParamDef
}

// StrictAction is the implementation of a strict builtin.
type StrictAction func(kwargs map[string]starlark.Value) (starlark.Value, error)

// NewStrictBuiltin creates a Starlark builtin that mandates keyword-only arguments.
func NewStrictBuiltin(def CommandDef, action StrictAction) *starlark.Builtin {
	return starlark.NewBuiltin(def.Name, func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if len(args) > 0 {
			return nil, fmt.Errorf("%s: takes keyword-only arguments\n%s", def.Name, generateUsage(def))
		}

		kwMap := make(map[string]starlark.Value)
		for _, pair := range kwargs {
			kwMap[pair[0].(starlark.String).GoString()] = pair[1]
		}

		if err := validateArgs(def, kwMap); err != nil {
			return nil, fmt.Errorf("%s: %w\n%s", def.Name, err, generateUsage(def))
		}

		return action(kwMap)
	})
}

func validateArgs(def CommandDef, kwMap map[string]starlark.Value) error {
	var missing []string
	for _, p := range def.Params {
		if _, ok := kwMap[p.Name]; !ok {
			missing = append(missing, p.Name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing mandatory arguments: %v", missing)
	}

	for k := range kwMap {
		found := false
		for _, p := range def.Params {
			if p.Name == k {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unknown argument '%s'", k)
		}
	}
	return nil
}

func generateUsage(def CommandDef) string {
	var sb strings.Builder
	sb.WriteString("\nDescription:\n  " + def.Desc + "\n")
	sb.WriteString("\nUsage:\n  " + def.Name + "(\n")
	for _, p := range def.Params {
		sb.WriteString(fmt.Sprintf("    %-15s # (%s) %s\n", p.Name+"=", p.Type, p.Desc))
	}
	sb.WriteString("  )\n")
	return sb.String()
}
