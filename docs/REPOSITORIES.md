# Decentralized Recipe Repositories

`pi` uses a decentralized model for recipe distribution, allowing individuals to maintain and share their own recipe collections without a central authority.

## Repository Model

A "Repository" is a collection of recipes apwdnd discovery logic.

### Types of Repositories
1.  **Local**: A directory on the host machine.
2.  **Git**: A remote Git repository (cloned and synced by `pi`).
3.  **Archive**: A remote `.tar.gz` or `.zip` file.
4.  **Single File**: A standalone recipe file for quick sharing.

### The `pi repo` Command
Users manage their recipe sources via the `pi repo` subcommands:
- `pi repo add <name> <url>`: Adds a new source.
- `pi repo remove <name>`: Removes a source.
- `pi repo sync`: Updates all remote sources.
- `pi repo list`: Shows configured repositories and their status.

## Recipe Discovery Logic

To avoid frequent updates, recipes should favor **discovery** over **hardcoding**.

### Stable Recipes
A stable recipe defines *how* to find a package.
- **The Request**: The recipe returns a `DiscoveryRequest` struct describing a target (e.g., `https://nodejs.org/dist/index.json`).
- **The Execution**: `pi` fetches the content, respecting the user's sandbox/network rules.
- **The Response**: `pi` calls the recipe's `parse(data)` function with the fetched content. The recipe then returns the final package metadata.

## Two-Tier Installation

Recipes must cover two distinct stages of an ecosystem's lifecycle:

1.  **Bootstrapping the Manager**:
    - Installing the tools that manage other things (e.g., `rustup`, `ghcup`, `opam`).
    - These are often "singletons" within a Cave.
2.  **Individual Packages**:
    - Installing specific versions of libraries or standalone tools.
    - These may be managed by the bootstrapped manager or directly by `pi`.

## Configuration Path
Repositories are indexed in `XDG_CONFIG_DIR/pi/repos.json`.
Actual recipe files are cached in `XDG_CACHE_DIR/pi/recipes/<repo-name>/`.
