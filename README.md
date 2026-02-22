# Pi: Gemini Package Installer

Pi is a modern, sandbox-first package manager and workspace orchestrator. It allows you to define hermetic environments (Caves) where packages are installed and run in isolation using Linux namespaces (Bubblewrap).

## Features

- **Sandboxed Execution**: All commands run inside a `bubblewrap` container with restricted access to your host system.
- **Hermetic Workspaces (Caves)**: Define project-specific environments with their own set of packages and environment variables.
- **Unified Pipeline Model**: A coherent installation sequence (Fetch -> Extract -> Run -> Export) that handles binaries, source builds, and managers identically.
- **Dynamic Build Options**: Customize package builds (e.g., toggle feature flags) directly from your `pi.cave.json`.
- **Stateful Build Caching**: Intelligent hashing of pipeline steps allows Pi to skip redundant work and resume failed builds exactly where they stopped.
- **Starlark Recipes**: Package definitions are written in Starlark (a dialect of Python), making them flexible and easy to read.
- **Multi-Manager Support**: Seamlessly integrate with language-specific managers like Go, Node.js (npm), and Rust (cargo).

## Requirements

- **Linux**: Requires Linux namespaces for sandboxing.
- **Bubblewrap**: The `bwrap` executable must be in your `PATH`.
- **Rust**: To build from source (2024 edition).

## Quick Start

### 1. Add a Repository
```bash
pi repo add official https://github.com/example/pi-recipes
```

### 2. Initialize a Cave
```bash
pi cave init
```

### 3. Add Packages
```bash
pi cave add erlang nodejs go
```

### 4. Configure Build Options (Optional)
Customize your packages by adding an `options` block to your `pi.cave.json`:
```json
{
  "packages": ["erlang", "nodejs"],
  "options": {
    "erlang": {
      "javac": true,
      "termcap": false
    }
  }
}
```

### 5. Run Commands
```bash
pi cave run erlang -version
```

## How it Works

When you run a command in a Cave, Pi:
1. Resolves the requested packages.
2. **Re-evaluates** the package recipes using the `options` specified in your `pi.cave.json`.
3. Checks the **Build Cache** using hashes of the generated pipeline steps.
4. Executes any missing steps (Fetch, Extract, Run) within a secure sandbox.
5. Applies **Exports** by symlinking results into a local `.pilocal` directory.
6. Spawns `bubblewrap` to mount the Cave's workspace and `.pilocal`.

---
*Pi: Simple, Safe, and Swift.*
