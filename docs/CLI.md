# Command Line Interface

`pi` uses a flexible CLI engine that supports unique prefix matching and parent command omission.

## Syntax
```bash
pi [global flags] <command> [command flags] <subcommand> [subcommand flags] [args]
```

*   **Prefix Matching**: `pi rem` matches `pi remote` if unambiguous.
*   **Command Omission**: `pi list` resolves to `pi remote list` if `list` is unique among all subcommands.
*   **Position Independence**: Command flags can be interleaved with args after the command is resolved.
*   **Global Flags**: Global flags are parsed anywhere in the argument list.
*   **Config Flag**: `--config/-c` is parsed but not yet wired to config loading.

## CLI Definition Language (.def)
The CLI structure is defined in `pkg/cli/cli.def` using a simple DSL:

| Keyword | Description |
| :--- | :--- |
| `global` | Marks the global flags section. |
| `flag` | Defines a global or command-specific flag (`bool` or `string`). |
| `cmd` | Defines a command or subcommand path. |
| `arg` | Defines a positional argument for the preceding command. |
| `safe` | Marks a command as safe to execute inside a cave sandbox. |
| `example` | Adds a usage example. |
| `topic` | Defines a help topic. |
| `text` | Adds documentation to a topic (supports `"""` for multiline). |

## Core Commands
*   `version`: Show build version and timestamp.
*   `pkg install <pkg>`: Installs a package into the global cache.
*   `pkg list <pkg>`: Lists available versions (filters to host OS/arch unless `-a`).
*   `pkg list --index`: Lists registered recipe patterns without executing handlers.
*   `cave init`: Initializes a `pi.cave.json` in the current directory.
*   `cave list`: Lists all registered caves and their variants.
*   `cave use <name[:variant]>`: Enters a registered cave from any directory.
*   `cave sync`: Placeholder (currently prints a message).
*   `cave run <cmd>`: Executes a command inside the sandbox.
*   `cave enter`: Shortcut for `cave run` with `/bin/bash`.
*   `remote list` / `remote add`: Placeholders (currently print messages).
