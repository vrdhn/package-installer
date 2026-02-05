package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type cdlTop struct {
	GlobalFlags  []*flag
	GlobalParams map[string]value
	Commands     []*command
	Topics       []*topic
	AppName      string
	Tagline      string
}

type flag struct {
	Name  string
	Short string
	Type  string
	Desc  string
}

type arg struct {
	Name string
	Type string
	Desc string
}

type command struct {
	Name     string
	Desc     string
	Params   map[string]value
	Args     []*arg
	Flags    []*flag
	Subs     []*command
	Parent   *command
	Examples []string
}

type topic struct {
	Name string
	Desc string
	Text string
}

type value struct {
	Kind string
	Bool bool
	Str  string
	Int  int
}

type param struct {
	Name string
	Kind string
}

func goTypeForFlag(t string) string {
	switch t {
	case "bool":
		return "bool"
	case "string":
		return "string"
	default:
		return "string"
	}
}

func goTypeForArg(t string) string {
	return "string"
}

func goTypeForParam(kind string) string {
	switch kind {
	case "bool":
		return "bool"
	case "string":
		return "string"
	case "int":
		return "int"
	default:
		return "string"
	}
}

func collectLeafCommands(cmds []*command) []*command {
	var out []*command
	walkCommands(cmds, func(c *command) {
		if len(c.Subs) == 0 {
			out = append(out, c)
		}
	})
	return out
}

func walkCommands(cmds []*command, fn func(*command)) {
	for _, c := range cmds {
		fn(c)
		if len(c.Subs) > 0 {
			walkCommands(c.Subs, fn)
		}
	}
}

func cmdPath(c *command) string {
	if c.Parent == nil {
		return c.Name
	}
	return cmdPath(c.Parent) + "/" + c.Name
}

func goNameForPath(path string) string {
	parts := strings.Split(path, "/")
	var out string
	for _, p := range parts {
		out += goFieldName(p)
	}
	return out
}

func goFieldName(name string) string {
	parts := splitIdent(name)
	var out string
	for _, p := range parts {
		if p == "" {
			continue
		}
		out += strings.ToUpper(p[:1]) + p[1:]
	}
	if out == "" {
		return "X"
	}
	if out[0] >= '0' && out[0] <= '9' {
		return "X" + out
	}
	return out
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func splitIdent(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		switch r {
		case '-', '_', '.', ':':
			return true
		default:
			return false
		}
	})
}

func isValidIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !(r == '_' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')) {
				return false
			}
			continue
		}
		if !(r == '_' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

func collectParams(cdl *cdlTop) ([]param, error) {
	kinds := map[string]string{}
	for name, pv := range cdl.GlobalParams {
		kinds[name] = pv.Kind
	}
	var walk func(cmds []*command) error
	walk = func(cmds []*command) error {
		for _, c := range cmds {
			for name, pv := range c.Params {
				if existing, ok := kinds[name]; ok && existing != pv.Kind {
					return fmt.Errorf("param %q has conflicting kinds: %s vs %s", name, existing, pv.Kind)
				}
				kinds[name] = pv.Kind
			}
			if len(c.Subs) > 0 {
				if err := walk(c.Subs); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(cdl.Commands); err != nil {
		return nil, err
	}
	if len(kinds) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(kinds))
	for k := range kinds {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]param, 0, len(keys))
	for _, k := range keys {
		out = append(out, param{Name: k, Kind: kinds[k]})
	}
	return out, nil
}

func hasParam(params []param, name string) bool {
	for _, d := range params {
		if d.Name == name {
			return true
		}
	}
	return false
}

func resolveParam(cdl *cdlTop, cmd *command, name string) (value, bool) {
	if cmd != nil && cmd.Params != nil {
		if v, ok := cmd.Params[name]; ok {
			return v, true
		}
	}
	if cdl != nil && cdl.GlobalParams != nil {
		if v, ok := cdl.GlobalParams[name]; ok {
			return v, true
		}
	}
	return value{}, false
}

func emitParamLiteral(v value, kind string) string {
	switch kind {
	case "bool":
		if v.Kind == "bool" {
			if v.Bool {
				return "true"
			}
			return "false"
		}
		return "false"
	case "string":
		if v.Kind == "string" {
			return strconv.Quote(v.Str)
		}
		return `""`
	case "int":
		if v.Kind == "int" {
			return fmt.Sprintf("%d", v.Int)
		}
		return "0"
	default:
		return "false"
	}
}
