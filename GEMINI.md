# pi - Universal Package Installer

## Project Overview

`pi` is a **workspace-based package manager and sandbox environment**. It ensures that every project has exactly the tools it needs, isolated from the host system and other projects.

**The User Journey:**
1.  **Define**: Create a `pi.cave.json` in your project root listing required languages (Node.js, Java, Rust, etc.).
2.  **Sync**: Run `pi sync` to automatically download and extract all required toolchains into a shared, immutable cache.
3.  **Enter**: Run `pi enter` to drop into a secure sandbox (a "Cave") where only your project files and specified tools are visible.

## Core Concepts

### The Cave (Sandbox)
A "Cave" is an isolated environment powered by Linux `bubblewrap`. 
- **Isolated HOME**: Mapped to `~/.local/state/pi/homes/<project-id>`.
- **Symlink Forest**: `pi` automatically symlinks active tools into the Cave's `.local/bin`, providing a clean, unified `$PATH`.
- **Shared Cache**: Packages are stored once in `~/.cache/pi/pkgs` and reused across all Caves.

## Usage

### Workspace Management
```bash
# Initialize a new workspace
pi init

# Add a package and sync
pi install nodejs@20

# Ensure all dependencies in pi.cave.json are ready
pi sync

# Enter the sandbox shell
pi enter
```


## Development Conventions

**Language:** Go 1.25+

**Coding Style (from `Prompts/CODING.md`):**
*   **Structure:** Keep files small (~250 lines) and functions concise (~15 lines).
*   **Typing:** Avoid `any` where possible; use strict static typing.
*   **Interfaces:** Prefer defining interfaces over structs at package boundaries.
*   **Immutability:** Use read-only interfaces with explicit write-checkout mechanisms.
*   **Error Handling:** Exhaustive case handling (use `default: panic` if no default exists).

**Testing & Quality:**
*   Run `go fmt ./...` before committing.
*   Run `go vet ./...` to catch issues.
*   Run `go test ./...` to verify logic.

## Key Documentation Files
*   `Prompts/SPEC.md`: Detailed functional specification.
*   `Prompts/ARCHITECTURE.md`: Core design and architecture notes.
*   `Prompts/CODING.md`: Detailed coding standards and style guide.
*   `Prompts/MODULES.md`: Documentation for internal Go packages.
