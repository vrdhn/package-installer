# Command Line Interface

`pi` uses a flexible CLI engine that supports unique prefix matching and parent command omission.

## Syntax
```bash
pi [global flags] <command> [command flags] <subcommand> [subcommand flags] [args]
```

*   **Prefix Matching**: `pi rem` matches `pi remote` if unambiguous.
*   **Command Omission**: `pi list` resolves to `pi remote list` if `list` is unique.
*   **Position Independence**: Flags can be placed anywhere after the command.

## CLI Definition Language (.def)
The CLI structure is defined in `pkg/cli/cli.def` using a simple DSL:

| Keyword | Description |
| :--- | :--- |
| `flag` | Defines a global or command-specific flag (`bool` or `string`). |
| `cmd` | Defines a command or subcommand path. |
| `arg` | Defines a positional argument for the preceding command. |
| `example` | Adds a usage example. |
| `topic` | Defines a help topic. |
| `text` | Adds documentation to a topic (supports `"""` for multiline). |

## Core Commands
*   `pkg install <pkg>`: Installs a package to the global cache.
*   `cave init`: Initializes a `pi.cave.json` in the current directory.
*   `cave sync`: Ensures all packages in `pi.cave.json` are installed and symlinked.
*   `cave run <cmd>`: Executes a command inside the sandbox.
*   `cave enter`: Shortcut for `cave run $SHELL`.
