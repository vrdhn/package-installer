package recipe

import (
	"encoding/json"
	"fmt"
	"pi/pkg/config"
	"strings"

	"github.com/itchyny/gojq"
	"go.starlark.net/starlark"
	"golang.org/x/net/html"
)

func jsonBuiltins() starlark.StringDict {
	return starlark.StringDict{
		"decode": NewStrictBuiltin(CommandDef{
			Name: "json.decode",
			Desc: "Decodes a JSON string into Starlark values.",
			Params: []ParamDef{
				{Name: "data", Type: "string", Desc: "The JSON string to decode"},
			},
		}, func(kwargs map[string]starlark.Value) (starlark.Value, error) {
			var data any
			if err := json.Unmarshal([]byte(asString(kwargs["data"])), &data); err != nil {
				return nil, err
			}
			return toStarlark(data), nil
		}),
		"encode": NewStrictBuiltin(CommandDef{
			Name: "json.encode",
			Desc: "Encodes a Starlark value into a JSON string.",
			Params: []ParamDef{
				{Name: "value", Type: "any", Desc: "The value to encode"},
			},
		}, func(kwargs map[string]starlark.Value) (starlark.Value, error) {
			data, err := fromStarlark(kwargs["value"])
			if err != nil {
				return nil, err
			}
			bArr, err := json.MarshalIndent(data, "", "  ")
			if err != nil {
				return nil, err
			}
			return starlark.String(string(bArr)), nil
		}),
	}
}

func jqBuiltins() starlark.StringDict {
	return starlark.StringDict{
		"query": NewStrictBuiltin(CommandDef{
			Name: "jq.query",
			Desc: "Executes a JQ query on a value.",
			Params: []ParamDef{
				{Name: "query", Type: "string", Desc: "The JQ filter string"},
				{Name: "value", Type: "any", Desc: "The value to query"},
			},
		}, func(kwargs map[string]starlark.Value) (starlark.Value, error) {
			data, err := fromStarlark(kwargs["value"])
			if err != nil {
				return nil, err
			}

			q, err := gojq.Parse(asString(kwargs["query"]))
			if err != nil {
				return nil, err
			}

			iter := q.Run(data)
			var results []starlark.Value
			for {
				res, ok := iter.Next()
				if !ok {
					break
				}
				if err, ok := res.(error); ok {
					return nil, err
				}
				results = append(results, toStarlark(res))
			}

			if len(results) == 1 {
				return results[0], nil
			}
			return starlark.NewList(results), nil
		}),
	}
}

func nodeToMap(n *html.Node) any {
	if n == nil {
		return nil
	}

	if n.Type == html.TextNode {
		txt := strings.TrimSpace(n.Data)
		if txt == "" {
			return nil
		}
		return txt
	}

	if n.Type != html.ElementNode && n.Type != html.DocumentNode {
		return nil
	}

	m := make(map[string]any)
	if n.Type == html.ElementNode {
		m["tag"] = n.Data
		attrs := make(map[string]string)
		for _, a := range n.Attr {
			attrs[a.Key] = a.Val
		}
		m["attr"] = attrs
	} else {
		m["tag"] = "#document"
	}

	var children []any
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		child := nodeToMap(c)
		if child != nil {
			children = append(children, child)
		}
	}
	m["children"] = children

	var sb strings.Builder
	var flattenText func(*html.Node)
	flattenText = func(node *html.Node) {
		if node.Type == html.TextNode {
			sb.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			flattenText(c)
		}
	}
	flattenText(n)
	m["text"] = strings.TrimSpace(sb.String())

	return m
}

// fromStarlark converts a Starlark value to a plain Go value
func fromStarlark(v starlark.Value) (any, error) {
	switch x := v.(type) {
	case starlark.NoneType:
		return nil, nil
	case starlark.Bool:
		return bool(x), nil
	case starlark.String:
		return string(x), nil
	case starlark.Int:
		i, _ := x.Int64()
		return i, nil
	case starlark.Float:
		return float64(x), nil
	case *starlark.List:
		var list []any
		for i := 0; i < x.Len(); i++ {
			val, err := fromStarlark(x.Index(i))
			if err != nil {
				return nil, err
			}
			list = append(list, val)
		}
		return list, nil
	case *starlark.Dict:
		dict := make(map[string]any)
		for _, key := range x.Keys() {
			k, ok := key.(starlark.String)
			if !ok {
				continue
			}
			val, _, _ := x.Get(key)
			v, err := fromStarlark(val)
			if err != nil {
				return nil, err
			}
			dict[string(k)] = v
		}
		return dict, nil
	default:
		return nil, fmt.Errorf("cannot convert %T to go", v)
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
	case int64:
		return starlark.MakeInt64(x)
	case int:
		return starlark.MakeInt(x)
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
	case map[string]string:
		dict := starlark.NewDict(len(x))
		for k, v := range x {
			dict.SetKey(starlark.String(k), starlark.String(v))
		}
		return dict
	default:
		return starlark.None
	}
}

func getString(dict *starlark.Dict, key string) string {
	val, ok, err := dict.Get(starlark.String(key))
	if err != nil || !ok || val == nil {
		return ""
	}

	str, ok := val.(starlark.String)
	if !ok {
		return ""
	}

	return str.GoString()
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

func (sr *StarlarkRecipe) executeHandler(cfg config.Config, pkgName string, versionQuery string, fetch Fetcher, regexKey string, handler starlark.Callable) ([]PackageDefinition, error) {
	if cached, ok, err := sr.loadHandlerCache(cfg, pkgName, versionQuery, regexKey); err != nil {
		return nil, err
	} else if ok {
		return cached, nil
	}

	pkgs := &[]PackageDefinition{}
	sr.thread.SetLocal(keyFetcher, fetch)
	sr.thread.SetLocal(keyCollector, pkgs)
	sr.thread.SetLocal(keyConfig, cfg)

	_, err := starlark.Call(sr.thread, handler, starlark.Tuple{
		starlark.String(pkgName),
	}, nil)
	if err != nil {
		return nil, mungeEvalError(sr.Name, err)
	}

	if err := sr.storeHandlerCache(cfg, pkgName, versionQuery, regexKey, *pkgs); err != nil {
		return nil, err
	}

	return *pkgs, nil
}

func mungeEvalError(name string, err error) error {
	if evalErr, ok := err.(*starlark.EvalError); ok {
		return fmt.Errorf("recipe handler error in %s:\n%s", name, evalErr.Backtrace())
	}
	return fmt.Errorf("recipe handler error: %s %w", name, err)
}
