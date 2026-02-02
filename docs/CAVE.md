# Cave

`pi` always works in the context of a Cave.
Cave may be ad-hoc, unnamed, or named and bound to location.

Cave has reference to
 - path to HOME to use
 - path to Workspace
 - environment variables
 - command to execute

Later this will grow to have proxy/filtering for network, dbus etc.

Cave can also have 'variants', which can have a few things different
from the base, like a different executable etc.
They can also have different versions of packages, which is the most
important use case. Variants will have same workspace

The cave is named like an identifier, and variant is a suffix with :name
For example proj-a:studio , proj-a:go25  etc

cave maintains the `pi.cave.json` file in the root of workspace.
and maintains a copy in it's CONFIG directory.

The _current_ cave is automatically determined by the current working directory,
or any --chdir global option on the command line.

if the cwd can't be mapped to any workspace of any cave, then a temporary cave
is generated, this will not be saved, and next garbage collection can delete the
packages that are installed.

Note that two cave or variants can share a home, however the ~/.local will be mounted to
different folder, so they can have different packages.

`pi.cave.json` has the list of  packages, and sub packages.

# command line

## `pi cave` commands and sub commands

* info : prints information about the cave ( named or ad-hoc)
* run  : launches sandbox on the cave or it's variant
* sync : syncs all the packages required by cave or the variant

# go level modules

- `pkg/cave`: Core logic for Cave management and workspace discovery.
- `pkg/cave/config`: Types and logic for parsing `pi.cave.json`.
- `pkg/cave/manager`: Handles discovery of workspace roots and management of cave instances.
- `pkg/cave/backend`: Interface for sandbox execution (e.g., Bubblewrap).
- `pkg/cave_bwrap`: Implementation of the sandbox backend using Linux Bubblewrap.

# Plan

## 1. Configuration Structures (pkg/cave/config)
- [ ] Define `CaveConfig` struct mirroring `pi.cave.json`
- [ ] Add support for `Packages` list (with version constraints)
- [ ] Add support for `Env` map for environment variable injection
- [ ] Implement variant support (merging logic for variants)

## 2. Cave Module (pkg/cave)
- [ ] Create `Cave` struct representing an active context (ID, Workspace, HomePath, Variant)
- [ ] Implement `Manager` for discovery logic (walking up from `cwd` to find `pi.cave.json`)
- [ ] Implement `Home` management (mapping project-id to state directory)
- [ ] Define `Backend` interface for sandbox abstraction

## 3. Sandbox Integration (pkg/cave_bwrap)
- [x] Basic Bubblewrap wrapper implemented
- [ ] Implement `Backend` interface using `Bubblewrap`
- [ ] Implement "Symlink Forest" logic:
    - [ ] Create `.local/bin`, `.local/lib`, etc. in Cave HOME
    - [ ] Populate with symlinks to active package contents

## 4. CLI Commands (pkg/cli)
- [ ] `cave info`: Display current workspace root, active packages, and environment
- [ ] `cave run`: Execute a command inside the sandbox
- [ ] `cave sync`: Reify the Cave (ensure packages are installed and symlinks are current)
- [ ] `cave init`: Create a default `pi.cave.json` in the current directory
- [ ] `enter`: Interactive shortcut for `cave run $SHELL`
