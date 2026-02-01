# pi - Project Roadmap

This roadmap outlines the evolution of `pi` from a proof-of-concept meta-installer to a robust, secure, and universal package management system.

## Phase 1: Foundation (Current)
*Focus: Core architecture and initial ecosystem support.*

- [x] **Core Modules**: Implementation of `archive`, `downloader`, `installer`, and `resolver`.
- [x] **Sandboxing**: Basic Linux `bubblewrap` integration.
- [x] **Initial Recipes**: Go-based recipes for Node.js and Java.
- [x] **CLI**: Basic `install` and `help` commands.
- [x] **XDG Compliance**: Standardized path management for config, cache, and state.

## Phase 2: Ecosystem Research & Repository Model (Q1 - Q2)
*Focus: Deep dive into diverse package manager 'stories' and defining the decentralized recipe model.*

- [ ] **Ecosystem Deep Dives**: Investigate bootstrapping and resolution for Rust, Haskell, OCaml, Go, etc.
- [ ] **Package Manager Bootstrapping**: Define recipes specifically for installing the ecosystem's own tools (e.g., `rustup`, `opam`).
- [ ] **Decentralized Repository Design**: 
    - Design the `pi repo` command suite (`add`, `remove`, `sync`, `list`).
    - Define the structure for local and remote recipe repositories (Git/Archive).
- [ ] **Built-in Discovery Logic**: Research how to make recipes "stable" by using discovery APIs rather than hardcoded versions.
- [ ] **Standardized Package Schema**: Define common metadata for binaries and environments.

## Phase 3: Implementation & Starlark (Q3)
*Focus: Implementing the 'pi repo' command and the declarative engine.*

- [ ] **`pi repo` Implementation**: CLI logic for managing decentralized recipe sources.
- [ ] **Starlark Integration**: Implement the discovery-first recipe engine.
- [ ] **Caching V2**: Content-addressable storage for shared package areas.

- [ ] **Cave Management**: Implementation of `pi.cave.json` for workspace-specific configurations.
- [ ] **Networking Control**: Filtering HTTP/S and DBus proxies for processes running inside the cave.
- [ ] **Cross-Platform Sandboxing**: 
    - macOS: Investigate `sandbox-exec` or similar backends.
    - Windows: Investigate Windows Containers or Windows Sandbox.
- [ ] **Environment Injection**: Fine-grained control over environment variables and symlink trees within the cave's `HOME`.

## Phase 4: Developer Experience & Optimization
*Focus: Performance, UI, and community tools.*

- [ ] **Curses UI**: Interactive progress monitoring and log tailing via the `display` module.
- [ ] **Parallel Dependency Resolution**: Building and traversing a dynamic dependency graph for faster installs.
- [ ] **Binary Deltas**: Optimize downloads for package updates.
- [ ] **Public Recipe Registry**: A curated community repository for common development tools.

---

## Future Considerations
- Integration with IDEs (VS Code, JetBrains) to automatically configure caves based on project roots.
- Support for OCI-based build steps.
- Native build hooks for source-based packages.
