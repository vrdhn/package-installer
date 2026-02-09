package recipe

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"go.starlark.net/starlark"
)

func htmlBuiltins() starlark.StringDict {
	return starlark.StringDict{
		"parse": NewStrictBuiltin(CommandDef{
			Name: "html.parse",
			Desc: "Parses an HTML string into a queryable document.",
			Params: []ParamDef{
				{Name: "data", Type: "string", Desc: "The HTML string to parse"},
			},
		}, func(kwargs map[string]starlark.Value) (starlark.Value, error) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(asString(kwargs["data"])))
			if err != nil {
				return nil, err
			}
			return &Selection{sel: doc.Selection}, nil
		}),
		"to_json": NewStrictBuiltin(CommandDef{
			Name: "html.to_json",
			Desc: "Converts an HTML string into a nested map structure.",
			Params: []ParamDef{
				{Name: "data", Type: "string", Desc: "The HTML string to convert"},
			},
		}, func(kwargs map[string]starlark.Value) (starlark.Value, error) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(asString(kwargs["data"])))
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
