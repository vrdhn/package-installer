# Pi: Gemini Package Installer

Pi is a modern, sandbox-first package manager and workspace orchestrator. It allows you to define hermetic environments (Caves) where packages are installed and run in isolation using Linux namespaces (Bubblewrap).

## Features

- **Sandboxed Execution**: All commands run inside a `bubblewrap` container with restricted access to your host system.
- **Hermetic Workspaces (Caves)**: Define project-specific environments with their own set of packages and environment variables.
- **Starlark Recipes**: Package definitions are written in Starlark (a dialect of Python), making them flexible and easy to read.
- **Multi-Manager Support**: Seamlessly integrate with language-specific managers like Go, Node.js (npm), and Rust (cargo).
- **Intelligent Caching**: (Available in `better-cache` branch) Persistent state management and journaling to skip redundant downloads and extractions.
- **Zero-Config Portability**: Shared `.pilocal` directories make it easy to move workspaces between machines.

## Requirements

- **Linux**: Requires Linux namespaces for sandboxing.
- **Bubblewrap**: The `bwrap` executable must be in your `PATH`.
- **Rust**: To build from source (2024 edition).

## Quick Start

### 1. Add a Repository
Pi uses repositories of Starlark recipes to know how to install packages.
```bash
pi repo add official https://github.com/example/pi-recipes
# Or add a local directory
pi repo add local ./my-recipes
```

### 2. Initialize a Cave
Create a new workspace (a "Cave") in your current directory.
```bash
pi cave init
```
This creates a `pi.cave.json` file in the current directory.

### 3. Add Packages
Add packages to your cave. You can specify versions or use the default (stable).
```bash
pi cave add nodejs go rust
# Add a specific version
pi cave add python=3.11
```

### 4. Run Commands
Run any command inside the cave's sandbox. The first run will automatically build/sync the environment.
```bash
pi cave run node --version
pi cave run bash # Enter an interactive shell inside the sandbox
```

## Useful Commands

| Command | Description |
|---------|-------------|
| `pi repo sync` | Update package lists from all repositories. |
| `pi package list` | Search for available packages. |
| `pi cave info` | Show details about the current cave and its mounts. |
| `pi disk clean` | Clear the global download and package cache. |

## Security & Safety

Pi is built with a "zero-trust" approach to third-party packages:

- **Sandbox-First**: No binary, script, or manager command downloaded from the internet is ever executed directly on your host system. Everything runs inside a restricted `bubblewrap` container.
- **Restricted Recipes**: Package recipes use **Starlark**, a deterministic language that prevents recipes from performing arbitrary system calls, file access, or network operations.
- **Protected Home Directory**: Pi maps a private, cave-specific directory to the container's `~`. This ensures that even if a package is malicious, it cannot access your SSH keys, browser data, or personal documents.
- **Sandboxed Build Hooks**: Post-install scripts and package manager commands (like `npm install`) are automatically executed within the sandbox with only the necessary paths mounted as writable.

## How it Works

When you run a command in a Cave, Pi:
1. Resolves the requested packages from your repositories.
2. Downloads and extracts them to a global shared cache.
3. Symlinks necessary binaries and files into a local `.pilocal` directory.
4. Spawns `bubblewrap` to mount the Cave's workspace and `.pilocal` while hiding the rest of the host system.

---
*Pi: Simple, Safe, and Swift.*
