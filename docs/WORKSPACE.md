# Workspace & Caves

`pi` manages development environments through **Caves**. A Cave is a combination of a workspace directory and an isolated home directory.

## The Manifest (`pi.cave.json`)
The manifest defines the requirements for a workspace.

```json
{
  "name": "myproject",
  "workspace": "/home/user/Projects/myproject",
  "home": "myproject",
  "variants": {
    "": {
      "pkgs": ["nodejs=20", "go=1.22"],
      "env": { "DEBUG": "1" }
    },
    "legacy": {
      "pkgs": ["nodejs=18"]
    }
  }
}
```
If `home` is omitted, `pi` falls back to a hashed ID under `~/.local/state/pi/homes/`.

## Registry
`pi` maintains a global registry at `$XDG_CONFIG_HOME/pi/caves.json` that maps cave names to workspaces.

## Environment Variables
`pi` uses environment variables to manage and discover active caves:

*   `PI_WORKSPACE`: Overrides workspace root discovery.
*   `PI_CAVENAME`: Inside a cave, set to `name:variant`. On the host, it restricts execution to `safe` commands.

## The Symlink Forest
To avoid PATH pollution, `pi` populates a symlink forest inside the Cave Home (by default `~/.local/state/pi/homes/<name>`):

*   `.local/bin/`: Symlinks to binaries discovered from installed packages.

## Sandbox Isolation (Linux)
When running inside a Cave (e.g., via `pi cave enter`), `pi` uses `bubblewrap` to:
1.  **Mount System Dirs**: `/usr`, `/lib`, `/bin`, `/sbin`, `/opt`, `/etc` are mounted read-only.
2.  **Mount Cache**: `~/.cache/pi` is bound read-only inside the cave.
3.  **Mount Workspace**: The workspace is mounted read-write at its real path.
4.  **Redirect HOME**: The cave home is bind-mounted onto the user's real `HOME` path inside the sandbox.
5.  **Environment**: `PI_WORKSPACE` and `PI_CAVENAME` are set; package env vars are applied.
