# Recipes & Packages

`pi` uses a decentralized recipe model to support any ecosystem.

## Package Identifiers
Packages follow the format: `[ecosystem:]name[@version]`

*   `nodejs@20`: Latest 20.x version of Node.js.
*   `go=stable`: Latest stable Go.
*   `pip:numpy`: Python package (handled by ecosystem manager).

## Starlark Recipes
Recipes are written in Starlark and must implement `discover` and `parse`.

### `discover(pkg_name, version_query, context)`
Returns a dict with `url` and `method` for version discovery.

### `parse(pkg_name, data, version_query, context)`
Parses the discovery response and returns a list of `PackageDefinition` dicts. 

**Note**: Recipes should return *all* available versions, architectures, and operating systems found in the data. The `pi` host handles filtering based on the user's system and version query.

Each package dict should include:
- `name`, `version`, `release_status` (stable, lts, current, rc, ea)
- `os`, `arch`, `url`, `filename`
- `env` (optional map)
- `symlinks` (optional map)
*   `json.decode(data)` / `json.encode(val)`
*   `html.parse(data)` / `html.to_json(data)`
*   `jq.query(filter, val)`

## Repositories
Recipes are collected in repositories (Local, Git, or Archive).
*   `pi remote add <name> <url>`: Register a new recipe source.
*   `pi remote list`: View active sources.
