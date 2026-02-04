# Architecture

`pi` is designed for safety, speed, and universality. It manages package lifecycles within isolated environments called Caves.

## Core Patterns

### Immutability
The `config` package uses a `ReadOnly`/`Writable` interface pattern to protect base paths and system info.
*   **ReadOnly**: Getters + `Checkout()`.
*   **Writable**: Setters + `Freeze()`.
*   **Safety**: Mutating a frozen config or checking out twice panics.

### Starlark Recipe Engine
`pi` executes **Starlark** recipes with a small set of strict built-ins. Recipes are declarative and do not perform arbitrary I/O.
1.  **Register**: Recipes call `add_pkgdef(regex, handler)` at load time to register lazy handlers.
2.  **Download**: Handlers call `download(url=...)` which uses host-side caching and HTTP.
3.  **Parse**: Handlers parse data using `json`, `html`, or `jq` helpers.
4.  **Emit**: Handlers register versions with `add_version(...)` (OS, arch, URL, filename, etc.).

### Sandboxing (Caves)
Caves provide isolation using Linux `bubblewrap`.
*   **Filesystem**: System directories are mounted read-only (`/usr`, `/lib`, `/etc`, etc.). The workspace is mounted read-write at its real path.
*   **Home**: The cave home is bind-mounted onto the user's real `HOME` path inside the sandbox.
*   **Symlink Forest**: Package binaries are exposed via `.local/bin` in the cave home.
*   **Devices/Agents**: `XDG_RUNTIME_DIR`, Wayland/DBus, and SSH agent are optionally passed through.

### CLI Execution Flow
The CLI follows a strict multi-stage initialization:
1.  **Parse DSL**: Load command definitions from `cli.def`.
2.  **Parse Args**: Parse global flags, then resolve commands and parameters.
3.  **Restriction Check**: If `PI_CAVENAME` is set, only `safe` commands may run.
4.  **Display Init**: Setup Bubble Tea console and verbosity.
5.  **Help/Error Handling**: Render help or return parse errors.
6.  **Execute**: Run the bound command handler.

### Build Information
`pi` embeds build metadata using linker flags:
*   `BuildVersion`: Git tag or commit hash.
*   `BuildTimestamp`: UTC build time.

## Pipeline
1.  **Select**: Match the package identifier against recipe regexes. If multiple match, print matches and exit.
2.  **Resolve**: Run the selected Starlark handler and filter by OS/arch/version/extension.
3.  **Download**: Fetch the artifact to `~/.cache/pi/downloads` with cache locking.
4.  **Install**: Extract into `~/.cache/pi/pkgs/<name-version-os-arch>` (atomic tmp dir).
5.  **Prepare**: Build symlink map and env for the cave.
6.  **Run**: For `cave run/enter`, launch bubblewrap with binds and env.
