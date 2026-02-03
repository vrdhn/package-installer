# Workspace & Caves

`pi` manages development environments through **Caves**. A Cave is a combination of a workspace directory and an isolated home directory.

## The Manifest (`pi.cave.json`)
The manifest defines the requirements for a workspace.

```json
{
  "cave": {
    "packages": ["nodejs@20", "go@1.22"],
    "env": { "DEBUG": "1" }
  },
  "variants": {
    "legacy": {
      "packages": ["nodejs@18"]
    }
  }
}
```

## The Symlink Forest
To avoid PATH pollution, `pi` populates a "Symlink Forest" inside the Cave's private HOME (`~/.local/state/pi/homes/<id>`).

*   `bin/`: Symlinks to binaries in the package cache.
*   `lib/`: Symlinks to libraries.
*   `include/`: Symlinks to headers.

## Sandbox Isolation (Linux)
When running inside a Cave (e.g., via `pi enter`), `pi` uses `bubblewrap` to:
1.  **Mount System Dirs**: `/usr`, `/lib`, `/etc` are mounted Read-Only.
2.  **Mount Pkg Cache**: `~/.cache/pi/pkgs` is mounted Read-Only.
3.  **Mount Workspace**: The current project directory is mounted Read-Write.
4.  **Redirect HOME**: `HOME` is set to the private Cave Home.
5.  **Environment**: Variables like `GOPATH` or `CARGO_HOME` are redirected into the Cave Home to ensure complete isolation from the host.
