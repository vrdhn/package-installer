# Roadmap

## Phase 1: Foundation (Complete)
*   [x] Core pipeline: Resolve, Download, Install.
*   [x] Basic Linux `bubblewrap` integration.
*   [x] Starlark recipe engine with JSON/HTML/JQ support.
*   [x] Workspace management with `pi.cave.json`.
*   [x] Custom CLI engine with prefix matching.

## Phase 2: Ecosystems & Repositories (Current)
*   [~] `pi repo` commands exist but are placeholders (no actual repo support yet).
*   [x] Built-in recipes for Node.js, Go, OpenJDK, and Foojay distributions.
*   [ ] Expand standard library of recipes (Rust, Python, Haskell).
*   [ ] Add support for "bootstrapping" managers (e.g., `rustup`).
*   [x] Starlark discovery caching (1-hour TTL).

## Phase 3: Advanced Sandboxing & UI (Upcoming)
*   [ ] Networking control (HTTP/DBus proxies) inside Caves.
*   [ ] Cross-platform sandboxing (macOS `sandbox-exec`).
*   [ ] Interactive TUI for progress monitoring.
*   [ ] Parallel dependency resolution.

## Phase 4: Enterprise & Polish
*   [ ] Content-addressable storage for shared package areas.
*   [ ] IDE integrations (VS Code, JetBrains).
*   [ ] Public Recipe Registry.
