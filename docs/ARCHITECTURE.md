# Architecture

## Immutability Pattern

To ensure predictability and prevent accidental state mutation, `pi` employs a `ReadOnly`/`Writable` interface pattern for core configuration and state structures.

*   **ReadOnly Interface**: Provides getter methods and a `Checkout()` method to obtain a writable version.
*   **Writable Interface**: Extends the `ReadOnly` interface with setter methods and a `Freeze()` method to lock the structure.
*   **Safety**: Attempting to modify a frozen structure or checking out a writable version twice results in a panic.

## Recipe Engine (Starlark)

`pi` uses **Starlark** (a dialect of Python) for its recipe engine. Recipes are pure and declarative, meaning they cannot perform I/O themselves.

1.  **Discovery**: A recipe provides a `discover` function that returns the necessary metadata (URLs, methods) to find package versions.
2.  **Resolution**: The `pi` host performs the network I/O and passes the raw data back to the recipe's `parse` function.
3.  **Result**: The recipe returns a list of `PackageDefinition` structs which `pi` then uses to plan and execute the installation.

## Multi-processing

`pi` leverages Go's goroutines and channels for concurrent operations. Independent tasks like downloading and extracting multiple packages are performed in parallel to maximize efficiency.

## Sandboxing (Caves)

The "Cave" model provides a secure, isolated environment for running development tools.

*   **Backend**: Currently implemented using Linux `bubblewrap`.
*   **Isolation**: Restricts filesystem access to the workspace and a dedicated isolated HOME directory.
*   **Package Visibility**: Installed packages are bind-mounted into the cave as read-only directories, with symlinks provided in the cave's isolated `.local/bin`.

## Display & Monitoring

The `display` module provides a unified API for progress reporting and logging. It supports concurrent tasks, each with its own progress bar and status updates, rendered cleanly to the console.