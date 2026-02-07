# Recipes & Packages

`pi` uses a decentralized recipe model to support various platforms and languages. Recipes are Starlark files that emit package versions via `add_version(...)`.

## Package Identifiers
Packages follow the format: `[[repo/]prefix:]name[=version]`

*   `nodejs=20`: Latest 20.x version of Node.js.
*   `go=stable`: Latest stable Go.
*   `pip:numpy`: Python package.
*   `official/nodejs`: Package from a specific repository.

## Starlark Recipes
Recipes are written in Starlark and register package handlers via `add_pkgdef(...)`.

Handlers use this signature:
`def handler(pkg_name):`
`pkg_name` is the identifier (e.g., `npm:express`). If a repository was specified in the query, it is removed before being passed to the handler.

### Execution Model
1.  Register handlers with `add_pkgdef(regex, handler)` at module load time.
2.  When invoked, handlers call `download(url=...)` to fetch discovery data (host-side cached).
3.  Handlers parse the data with `json`, `html`, or `jq`, then emit versions using `add_version(...)`.

**Note**: Recipes should emit *all* available versions, architectures, and operating systems. The host filters by OS/arch/version query.

### Required `add_version(...)` fields
All fields are required keyword args (use empty strings if unknown):
- `name`, `version`, `release_status` (stable, lts, current, rc, ea)
- `release_date` (e.g., 2024-01-12)
- `os`, `arch`, `url`, `filename`, `checksum`
- `env` (dict, can be empty)
- `symlinks` (dict, can be empty)

### Built-ins
- `download(url=...)`
- `download_github_releases(owner=..., repo=...)`
- `json.decode(data)` / `json.encode(val)`
- `html.parse(data)` / `html.to_json(data)`
- `jq.query(filter, val)`
- `add_version(...)`
- `add_pkgdef(regex, handler)`

### Context
Recipes do not receive a context object. They should emit as many versions as possible.
Filtering is performed by `pi` after discovery.

If multiple recipe regexes match a requested package, `pi` prints the matching repository/regex list and exits. You can disambiguate by specifying the repository name (e.g., `repo/pkg`).

## Recipe REPL
Use the REPL to iterate on Starlark recipes locally:

```sh
pi recipe repl ./recipes/nodejs.star
```

Commands:
- `show` / `patterns`: list registered regex patterns
- `run <pkg>`: execute the matching handler and summarize results
- `run-regex <regex> <pkg>`: run a specific regex when multiple match
- `reload`: reload the recipe file
- `exit` / `quit`: leave the REPL

## Repositories
Recipes are loaded from built-in Starlark files embedded in the binary or from local filesystem repositories added via `pi repo add <path>`.

The package index is managed via `pi repo sync` which populates `packages.csv` for fast resolution.
