# pi — Universal Package Installer

## About

`pi` is a **non-interactive, workspace-based package installer** that manages
dependencies across **multiple language ecosystems** (Python, JavaScript, Java,
native/source, etc.).

`pi` acts as a meta-installer:
- Bootstraps ecosystem-specific package managers when needed
- Can build packages from source
- Can reuse system-installed packages when allowed

---

## Core Properties

### Universal
- Supports packages from multiple ecosystems
- Ecosystem-specific logic is implemented via **recipes**
- Package managers themselves may be installed and managed by `pi`

### Sandboxing
- All external commands run inside a sandbox (“cave”)
- Multiple backends are supported by design
- **Current backend:** Linux `bubblewrap` only
- Sandboxing restricts visibility of files in HOME.
- An un-sandboxed process has full access to all files.

### Cave model
- Cave is composed of following elements
  - a HOME, mapped to XDG_STATE_DIR/pi/homes/<dirname>
  - a WORKSPACE, mapped to user defined path
	- defaults to invocation dir or upper path with `pi.cave.json`
  - environment variables for the cave
  - several read-only binds from the host, including XDG_CACHE_DIR/pi/pkgs
  - A workspace may contain multiple projects and languages
  - All projects in a Cave share the same tools versions
  - Filtering proxies  ( dbus, http/s etc ), and network control
  - configuration is stored in XDG_CONFIG_DIR/pi/caves/
  - CAVEDHOME/.local will have symlinkes to XDG_CACHE_DIR/pi/pkgs as needed

### File system usage
Host view:
  - `XDG_STATE_DIR/pi/homes/{ID}` is shared or exclusive HOME
  - `XDG_CACHE_DIR/pi/pkgs/` is shared installed package area
  - <User-Specified> is shared writable as workspace
  - system FHS dirs ( /usr,/etc, /lib etc) are shared read-only,
  - user freedesktop dirs ( Downloads, Public, Video etc) are shared rw
  - virtual dirs ( /sys, /proc, /dev ) are complicated
  - /var/tmp and /tmp are created empty

---

## Recipes

### Purpose
Recipes describe how packages are:
- Discovered
- Resolved
- Installed

### Language
- Recipes are written in **Starlark**
- `pi` provides built-in helpers (JSON parsing, version logic, etc.)

### Constraints
Recipes are **pure and declarative**:
- No direct filesystem or network I/O.
- **Discovery Pattern**: To find versions, a recipe returns a `DiscoveryRequest` (URLs, regex, etc.). The `pi` host executes the I/O and passes the resulting data back into a secondary "parser" function within the recipe.
- No process execution or global state mutation.

### Structure
- One recipe file may define multiple packages or an entire ecosystem
- Recipes transform input data into `pi` usable structs

---

## Recipe Repositories

- A repository is a collection of recipe files
- Distribution formats:
  - Git repository
  - Archive file
  - Single recipe file
- `pi` downloads, caches, and indexes repositories for lookup

---

## Design Intent

- security first, in all sense of security.
- Deterministic and reproducible
- Scriptable and non-interactive
- Workspace-local effects only
- Ecosystem-agnostic core
- Pluggable sandbox backends
