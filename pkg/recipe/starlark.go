package recipe

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"pi/pkg/config"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/itchyny/gojq"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"golang.org/x/net/html"
)

// Immutable
type StarlarkRecipe struct {
	Name    string
	Source  string
	thread  *starlark.Thread
	globals starlark.StringDict

	registry       map[string]starlark.Callable
	regexCache     map[string]*regexp.Regexp
	registryLoaded bool
	legacy         bool
	legacyPkgs     []PackageDefinition

	currentCtx *DiscoveryContext
}

func NewStarlarkRecipe(name, source string, printFunc func(string)) (*StarlarkRecipe, error) {
	return &StarlarkRecipe{
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
	}, nil
}

func (sr *StarlarkRecipe) Execute(ctx *DiscoveryContext) ([]PackageDefinition, error) {
	if ctx == nil {
		return nil, fmt.Errorf("missing discovery context")
	}

	var pkgs []PackageDefinition

	// Local add_version to collect results while also calling the context's callback
	ctxWrapper := *ctx
	ctxWrapper.AddVersion = func(p PackageDefinition) {
		pkgs = append(pkgs, p)
		if ctx.AddVersion != nil {
			ctx.AddVersion(p)
		}
	}

	sr.currentCtx = &ctxWrapper

	if !sr.registryLoaded {
		if err := sr.loadRegistry(); err != nil {
			return nil, err
		}
		if ctx.Download != nil {
			fmt.Printf("[%s] Registry loaded, %d handlers found\n", sr.Name, len(sr.registry))
		}
		if len(sr.registry) == 0 {
			sr.legacy = true
			sr.legacyPkgs = pkgs
			return pkgs, nil
		}
	}

	if sr.legacy {
		return sr.legacyPkgs, nil
	}

	pi := ctx.PkgName
	handler, regexKey, err := sr.matchHandler(pi)
	if err != nil {
		return nil, err
	}
	if handler == nil {
		if ctx.Download != nil {
			fmt.Printf("[%s] No handler matched for %s\n", sr.Name, pi)
		}
		return nil, fmt.Errorf("recipe not applicable: %s", sr.Name)
	}

	return sr.executeHandler(&ctxWrapper, regexKey, handler, pkgs)
}

// ExecuteRegex runs the handler registered for a specific regex.
func (sr *StarlarkRecipe) ExecuteRegex(ctx *DiscoveryContext, regexKey string) ([]PackageDefinition, error) {
	if ctx == nil {
		return nil, fmt.Errorf("missing discovery context")
	}

	var pkgs []PackageDefinition
	ctxWrapper := *ctx
	ctxWrapper.AddVersion = func(p PackageDefinition) {
		pkgs = append(pkgs, p)
		if ctx.AddVersion != nil {
			ctx.AddVersion(p)
		}
	}
	sr.currentCtx = &ctxWrapper

	if !sr.registryLoaded {
		if err := sr.loadRegistry(); err != nil {
			return nil, err
		}
	}
	if sr.legacy {
		return nil, fmt.Errorf("legacy recipe does not support regex execution: %s", sr.Name)
	}

	handler, ok := sr.registry[regexKey]
	if !ok {
		return nil, fmt.Errorf("recipe handler not found for regex '%s' in %s", regexKey, sr.Name)
	}

	return sr.executeHandler(&ctxWrapper, regexKey, handler, pkgs)
}

func (sr *StarlarkRecipe) executeHandler(ctx *DiscoveryContext, regexKey string, handler starlark.Callable, pkgs []PackageDefinition) ([]PackageDefinition, error) {
	if ctx.Config != nil {
		if cached, ok, err := sr.loadHandlerCache(ctx, regexKey); err != nil {
			return nil, err
		} else if ok {
			return cached, nil
		}
	}

	_, err := starlark.Call(sr.thread, handler, starlark.Tuple{
		starlark.String(ctx.PkgName),
	}, nil)
	if err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			return nil, fmt.Errorf("recipe handler error in %s:\n%s", sr.Name, evalErr.Backtrace())
		}
		return nil, fmt.Errorf("recipe handler error: %s %w", sr.Name, err)
	}

	if ctx.Config != nil {
		if err := sr.storeHandlerCache(ctx, regexKey, pkgs); err != nil {
			return nil, err
		}
	}

	return pkgs, nil
}

// Registry returns registered regex patterns and whether the recipe is legacy.
func (sr *StarlarkRecipe) Registry(cfg config.Config) ([]string, bool, error) {
	ctx := &DiscoveryContext{
		Config:       cfg,
		PkgName:      "",
		VersionQuery: "",
		Download: func(url string) ([]byte, error) {
			return nil, fmt.Errorf("download disabled in index mode")
		},
	}
	sr.currentCtx = ctx
	if !sr.registryLoaded {
		if err := sr.loadRegistry(); err != nil {
			return nil, false, err
		}
		if len(sr.registry) == 0 {
			sr.legacy = true
		}
	}

	var keys []string
	for k := range sr.registry {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, sr.legacy, nil
}

// GetRegistryInfo returns the registry map with handler names.
func (sr *StarlarkRecipe) GetRegistryInfo(cfg config.Config) (map[string]string, error) {
	_, _, err := sr.Registry(cfg)
	if err != nil {
		return nil, err
	}
	info := make(map[string]string)
	for k, v := range sr.registry {
		info[k] = v.Name()
	}
	return info, nil
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

func (sr *StarlarkRecipe) loadRegistry() error {
	sr.registry = make(map[string]starlark.Callable)
	sr.regexCache = make(map[string]*regexp.Regexp)

	builtins := starlark.StringDict{
		"struct":                   starlark.NewBuiltin("struct", starlarkstruct.Make),
		"json":                     starlarkstruct.FromStringDict(starlark.String("json"), jsonBuiltins()),
		"html":                     starlarkstruct.FromStringDict(starlark.String("html"), htmlBuiltins()),
		"jq":                       starlarkstruct.FromStringDict(starlark.String("jq"), jqBuiltins()),
		"download":                 newDownloadBuiltin(sr),
		"download_github_releases": newDownloadGitHubReleasesBuiltin(sr),
		"add_version":              newAddVersionBuiltin(sr),
		"add_pkgdef":               newAddPkgdefBuiltin(sr),
	}

	globals, err := starlark.ExecFile(sr.thread, sr.Name+".star", sr.Source, builtins)
	if err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			return fmt.Errorf("failed to load recipe %s:\n%s", sr.Name, evalErr.Backtrace())
		}
		return err
	}
	sr.globals = globals
	sr.registryLoaded = true
	return nil
}

func (sr *StarlarkRecipe) matchHandler(pi string) (starlark.Callable, string, error) {
	if len(sr.registry) == 0 {
		return nil, "", nil
	}
	var keys []string
	for k := range sr.registry {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		re, ok := sr.regexCache[key]
		if !ok {
			var err error
			re, err = CompileAnchored(key)
			if err != nil {
				return nil, "", fmt.Errorf("invalid regex '%s' in recipe %s: %w", key, sr.Name, err)
			}
			sr.regexCache[key] = re
		}
		if re.MatchString(pi) {
			return sr.registry[key], key, nil
		}
	}
	return nil, "", nil
}

func (sr *StarlarkRecipe) handlerCachePath(ctx *DiscoveryContext, regexKey string) (string, error) {
	if ctx == nil || ctx.Config == nil {
		return "", fmt.Errorf("missing config for cache path")
	}
	key := fmt.Sprintf("%s|%s|%s|%s|%s|%s", sr.Name, regexKey, ctx.PkgName, ctx.VersionQuery, ctx.Config.GetOS(), ctx.Config.GetArch())
	sum := sha256.Sum256([]byte(key))
	fileName := fmt.Sprintf("handler_%x.json", sum[:])
	return filepath.Join(ctx.Config.GetDiscoveryDir(), fileName), nil
}

func (sr *StarlarkRecipe) loadHandlerCache(ctx *DiscoveryContext, regexKey string) ([]PackageDefinition, bool, error) {
	path, err := sr.handlerCachePath(ctx, regexKey)
	if err != nil {
		return nil, false, err
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if time.Since(info.ModTime()) > time.Hour {
		return nil, false, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, false, err
	}
	var pkgs []PackageDefinition
	if err := json.Unmarshal(data, &pkgs); err != nil {
		return nil, false, err
	}
	return pkgs, true, nil
}

func (sr *StarlarkRecipe) storeHandlerCache(ctx *DiscoveryContext, regexKey string, pkgs []PackageDefinition) error {
	path, err := sr.handlerCachePath(ctx, regexKey)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(pkgs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Test runs all functions in the recipe that start with "test_".
func (sr *StarlarkRecipe) Test(cfg config.Config) error {
	if !sr.registryLoaded {
		if err := sr.loadRegistry(); err != nil {
			return err
		}
	}

	var testFuncs []string
	for name, val := range sr.globals {
		if strings.HasPrefix(name, "test_") {
			if _, ok := val.(starlark.Callable); ok {
				testFuncs = append(testFuncs, name)
			}
		}
	}
	sort.Strings(testFuncs)

	if len(testFuncs) == 0 {
		return fmt.Errorf("no test functions found (starting with 'test_')")
	}

	for _, name := range testFuncs {
		fmt.Printf("  Running %s... ", name)
		_, err := starlark.Call(sr.thread, sr.globals[name], nil, nil)
		if err != nil {
			fmt.Println("FAILED")
			if evalErr, ok := err.(*starlark.EvalError); ok {
				return fmt.Errorf("test %s failed:\n%s", name, evalErr.Backtrace())
			}
			return fmt.Errorf("test %s failed: %w", name, err)
		}
		fmt.Println("OK")
	}

	return nil
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
