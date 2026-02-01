# Workspace UX & Technical Design

This document details the intended user experience and the technical implementation of the "Workspace Manager" flow.

## 1. Workspace Manifest (`pi.cave.json`)

Every project directory should contain a `pi.cave.json` file.

```json
{
  "name": "my-project",
  "dependencies": {
    "nodejs": "20.10.0",
    "java": "17"
  },
  "env": {
    "FOO": "bar"
  },
  "binds": [
    { "src": "~/data", "dest": "/data", "ro": true }
  ]
}
```

## 2. The Symlink Forest

To make multiple packages usable without polluting the `$PATH` with dozens of entries, `pi` uses a symlink forest inside the Cave's private `HOME`.

**Internal Layout of a Cave Home (`~/.local/state/pi/homes/<id>/`):**
- `.local/bin/`: Contains symlinks to binaries in `~/.cache/pi/pkgs/<pkg>/bin/*`
- `.local/lib/`: Symlinks to libraries.
- `.local/include/`: Symlinks to headers.

When `pi enter` starts the sandbox:
1.  It identifies all active packages.
2.  It populates/refreshes the symlink forest in the Cave Home.
3.  It sets `HOME` to the Cave Home directory inside the sandbox.
4.  It adds `~/.local/bin` to the front of `$PATH`.

## 3. Command Workflows

### `pi sync`
- Reads `pi.cave.json`.
- For each dependency:
    - Checks if it exists in `~/.cache/pi/pkgs`.
    - If not, resolves the version, downloads, and extracts it.
- Updates the symlink forest in the Cave's state directory.

### `pi enter`
- Runs `pi sync` (fast check).
- Configures `bubblewrap`:
    - RO bind system dirs (`/usr`, `/etc`, etc.).
    - RW bind the Shared Pkg Cache at its absolute host path to ensure relative-path tools work.
    - RW bind the current directory at its absolute host path.
    - RW bind the Cave Home as the user's `HOME`.
- **Manager Isolation**: Injects ecosystem-specific environment variables to redirect local state into the Cave Home:
    - `RUSTUP_HOME`, `CARGO_HOME` -> `~/.local/share/rust`
    - `OPAMROOT` -> `~/.local/share/opam`
    - `GOPATH`, `GOMODCACHE` -> `~/.local/share/go`
- Spawns the user's shell.

## 4. User Benefits
- **Zero Pollution**: Installing a tool for one project doesn't affect the host or other projects.
- **Instant Setup**: New developers just run `pi enter` and have the exact environment needed.
- **Security**: The project cannot read the user's personal files (SSH keys, browser data) by default.
- **Consistency**: The same tool version is used across all machines.
