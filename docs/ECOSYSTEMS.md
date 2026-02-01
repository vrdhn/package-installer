# Ecosystem Integration Research

To build a truly universal installer, `pi` must accommodate the varied "stories" of existing package managers. This document tracks the investigation into how each ecosystem handles toolchains, dependencies, and isolation.

## Research Dimensions
For each ecosystem, we must answer:
1.  **Bootstrapping**: How is the primary tool installed? (e.g., `rustup`, `ghcup`, `nvm`). This is the "meta-recipe" that enables the rest of the ecosystem.
2.  **Resolution**: Where is the source of truth for versions? (REST API, Git, Scrapyard?)
3.  **Layout**: Where are binaries, libraries, and headers stored?
4.  **Isolation**: How does it handle project-local dependencies? (e.g., `node_modules`, `target/`, `_build/`)
5.  **Sandbox Constraints**: Does it need specific syscalls or filesystem mounts?

---

## 1. Rust (Cargo / Rustup)
- **Toolchain**: `rustup` is the standard. It manages multiple toolchains in `~/.rustup`.
- **Packages**: `cargo` handles dependencies, usually downloading to `~/.cargo/registry`.
- **pi Story**: 
    - `pi` should likely manage the toolchain versions (mimicking `rustup` but within a Cave).
    - Cargo's cache (`CARGO_HOME`) should be mapped to the shared `pkgs` area if possible, or isolated per Cave.
- **Challenges**: Cargo relies heavily on `TMPDIR` for compilation; large target directories.

## 2. Haskell (Stack / Cabal / GHCup)
- **Toolchain**: `ghcup` is the modern way to get GHC, Stack, and Cabal.
- **Packages**: `stack` uses snapshots (LTS) which are highly deterministic.
- **pi Story**:
    - `pi` could manage GHC versions directly.
    - Integration with `stack` snapshots would align well with `pi`'s determinism goals.
- **Challenges**: Haskell builds are notoriously resource-intensive and produce many small files.

## 3. OCaml (Opam)
- **Toolchain/Packages**: `opam` manages both. It uses "switches" for isolation.
- **pi Story**:
    - `opam switch` is very similar to the "Cave" concept.
    - `pi` could potentially manage the `OPAMROOT`.
- **Challenges**: Many OCaml packages depend on system libraries (C headers); requires robust `depext` handling.

## 4. Go
- **Toolchain**: Distributed as a single archive.
- **Packages**: `go mod` with a global `GOMODCACHE`.
- **pi Story**:
    - Simple to bootstrap.
    - `GOPATH` and `GOMODCACHE` management is straightforward.
- **Challenges**: Go's toolchain is increasingly "smart" about downloading its own versions (`GOTOOLCHAIN`).

## 5. Node.js (Existing)
- **Toolchain**: `pi` currently downloads the official binary archives.
- **Packages**: Handled by `npm`/`pnpm`/`yarn`.
- **pi Story**: `pi` installs the `node` binary; user runs `npm` inside the cave.

## 6. Java (Existing)
- **Toolchain**: `pi` uses Foojay API to find JDKs.
- **Packages**: `maven`/`gradle` (usually not managed by `pi` yet).
- **pi Story**: `pi` installs the JDK; sets `JAVA_HOME`.
