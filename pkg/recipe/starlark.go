package recipe

import (
	"encoding/json"
	"fmt"
	"pi/pkg/archive"
	"pi/pkg/config"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/itchyny/gojq"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"golang.org/x/net/html"
)

type StarlarkRecipe struct {
	Name    string
	Source  string
	thread  *starlark.Thread
	globals starlark.StringDict
}

func NewStarlarkRecipe(name, source string, printFunc func(string)) (*StarlarkRecipe, error) {
	sr := &StarlarkRecipe{
		Name:   name,
		Source: source,
		thread: &starlark.Thread{
			Name: name,
			Print: func(thread *starlark.Thread, msg string) {
				if printFunc != nil {
					printFunc(msg)
				} else {
					fmt.Printf("[%s] %s\n", thread.Name, msg)
				}
			},
		},
	}

	// Define built-ins
	builtins := starlark.StringDict{
		"struct": starlark.NewBuiltin("struct", starlarkstruct.Make),
		"json":   starlarkstruct.FromStringDict(starlark.String("json"), jsonBuiltins()),
		"html":   starlarkstruct.FromStringDict(starlark.String("html"), htmlBuiltins()),
		"jq":     starlarkstruct.FromStringDict(starlark.String("jq"), jqBuiltins()),
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
		"encode": starlark.NewBuiltin("encode", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var v starlark.Value
			if err := starlark.UnpackArgs("encode", args, kwargs, "value", &v); err != nil {
				return nil, err
			}
			data, err := fromStarlark(v)
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
		"query": starlark.NewBuiltin("query", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var query string
			var v starlark.Value
			if err := starlark.UnpackArgs("query", args, kwargs, "query", &query, "value", &v); err != nil {
				return nil, err
			}

			data, err := fromStarlark(v)
			if err != nil {
				return nil, err
			}

			q, err := gojq.Parse(query)
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

func htmlBuiltins() starlark.StringDict {
	return starlark.StringDict{
		"parse": starlark.NewBuiltin("parse", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var s string
			if err := starlark.UnpackArgs("parse", args, kwargs, "data", &s); err != nil {
				return nil, err
			}
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(s))
			if err != nil {
				return nil, err
			}
			return &Selection{sel: doc.Selection}, nil
		}),
		"to_json": starlark.NewBuiltin("to_json", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var s string
			if err := starlark.UnpackArgs("to_json", args, kwargs, "data", &s); err != nil {
				return nil, err
			}
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(s))
			if err != nil {
				return nil, err
			}
			if doc.Selection.Length() == 0 {
				return starlark.None, nil
			}
			return toStarlark(nodeToMap(doc.Selection.Get(0))), nil
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

// Selection wraps goquery.Selection for Starlark
type Selection struct {
	sel *goquery.Selection
}

func (s *Selection) String() string        { return "html.selection" }
func (s *Selection) Type() string          { return "html.selection" }
func (s *Selection) Freeze()               {}
func (s *Selection) Truth() starlark.Bool  { return s.sel.Length() > 0 }
func (s *Selection) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: %s", s.Type()) }

func (s *Selection) Attr(name string) (starlark.Value, error) {
	switch name {
	case "text":
		return starlark.NewBuiltin("text", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			return starlark.String(s.sel.Text()), nil
		}), nil
	case "attr":
		return starlark.NewBuiltin("attr", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var name string
			if err := starlark.UnpackArgs("attr", args, kwargs, "name", &name); err != nil {
				return nil, err
			}
			val, _ := s.sel.Attr(name)
			return starlark.String(val), nil
		}), nil
	case "find":
		return starlark.NewBuiltin("find", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var selector string
			if err := starlark.UnpackArgs("find", args, kwargs, "selector", &selector); err != nil {
				return nil, err
			}
			return &Selection{sel: s.sel.Find(selector)}, nil
		}), nil
	case "each":
		return starlark.NewBuiltin("each", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var list []starlark.Value
			s.sel.Each(func(i int, gs *goquery.Selection) {
				list = append(list, &Selection{sel: gs})
			})
			return starlark.NewList(list), nil
		}), nil
	}
	return nil, nil
}

func (s *Selection) AttrNames() []string {
	return []string{"text", "attr", "find", "each"}
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

func (sr *StarlarkRecipe) Discover(cfg *config.Config, pkgName string, versionQuery string) (string, string, error) {
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

	res, err := starlark.Call(sr.thread, discover, starlark.Tuple{starlark.String(pkgName), starlark.String(versionQuery), ctx}, nil)
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

func (sr *StarlarkRecipe) Parse(cfg *config.Config, pkgName string, data []byte, versionQuery string) ([]PackageDefinition, error) {
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

	res, err := starlark.Call(sr.thread, parse, starlark.Tuple{starlark.String(pkgName), starlark.String(string(data)), starlark.String(versionQuery), ctx}, nil)
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
