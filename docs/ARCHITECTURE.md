# Architecture

`pi` is designed for safety, speed, and universality. It manages package lifecycles within isolated environments called Caves.

## Core Patterns

### Immutability
To ensure predictability, core structures use a `ReadOnly`/`Writable` interface pattern.
*   **ReadOnly**: Provides getters and a `Checkout()` method.
*   **Writable**: Extends ReadOnly with setters and a `Freeze()` method.
*   **Safety**: Attempting to modify a frozen structure or checking out a writable version twice results in a panic.

### Starlark Recipe Engine
`pi` uses **Starlark** for its recipe engine. Recipes are pure, declarative, and isolated from I/O.
1.  **Discovery**: A recipe returns a `DiscoveryRequest` (URLs, methods).
2.  **Resolution**: The `pi` host performs the network I/O.
3.  **Parsing**: The host passes raw data back to the recipe's `parse` function to produce `PackageDefinition` structs.

### Sandboxing (Caves)
Caves provide isolation using Linux `bubblewrap`.
*   **Filesystem**: Restricts access to the workspace and a dedicated isolated HOME.
*   **Symlink Forest**: Installed packages are bind-mounted read-only, with symlinks provided in the cave's `.local/bin`.

## Pipeline
1.  **Resolve**: Map package name/version to a specific artifact via Starlark recipes.
2.  **Download**: Fetch the artifact (HTTP/HTTPS) to a shared cache.
3.  **Install**: Extract the artifact to a version-specific directory in `~/.cache/pi/pkgs`.
4.  **Reify**: Update the symlink forest in the Cave Home to include the active packages.
