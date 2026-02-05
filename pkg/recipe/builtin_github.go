package recipe

import (
	"encoding/json"
	"fmt"
	"net/url"

	"go.starlark.net/starlark"
)

func newDownloadGitHubReleasesBuiltin(sr *StarlarkRecipe) *starlark.Builtin {
	def := CommandDef{
		Name: "download_github_releases",
		Desc: "Fetches GitHub releases JSON for a repository.",
		Params: []ParamDef{
			{Name: "owner", Type: "string", Desc: "GitHub org/user name"},
			{Name: "repo", Type: "string", Desc: "GitHub repository name"},
		},
	}

	return NewStrictBuiltin(def, func(kwargs map[string]starlark.Value) (starlark.Value, error) {
		ctx := sr.currentCtx
		if ctx == nil || ctx.Download == nil {
			return nil, fmt.Errorf("download_github_releases called without active context")
		}

		owner := kwargs["owner"].(starlark.String).GoString()
		repo := kwargs["repo"].(starlark.String).GoString()
		urlStr := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", url.PathEscape(owner), url.PathEscape(repo))

		data, err := ctx.Download(urlStr)
		if err != nil {
			return nil, err
		}

		var payload any
		if err := json.Unmarshal(data, &payload); err != nil {
			return nil, err
		}
		return toStarlark(payload), nil
	})
}
