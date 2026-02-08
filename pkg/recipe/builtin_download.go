package recipe

import (
	"fmt"

	"go.starlark.net/starlark"
)

func newDownloadBuiltin(sr *StarlarkRecipe) *starlark.Builtin {
	def := CommandDef{
		Name: "download",
		Desc: "Fetches data from a URL with caching.",
		Params: []ParamDef{
			{Name: "url", Type: "string", Desc: "The URL to download"},
		},
	}

	return NewStrictBuiltin(def, func(kwargs map[string]starlark.Value) (starlark.Value, error) {
		val := sr.thread.Local(keyFetcher)
		if val == nil {
			return nil, fmt.Errorf("download called without active fetcher")
		}
		fetch := val.(Fetcher)

		url := kwargs["url"].(starlark.String).GoString()
		data, err := fetch(url)
		if err != nil {
			return nil, err
		}
		return starlark.String(string(data)), nil
	})
}
