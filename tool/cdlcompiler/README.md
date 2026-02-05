# Command Line Interface

This CLI engine supports unique prefix matching and parent command omission.

## Syntax
```bash
<program> [global flags] <command> [command flags] <subcommand> [subcommand flags] [args]
```

*   **Prefix Matching**: `<program> rep` matches `<program> repo` if unambiguous.
*   **Command Omission**: `<program> list` resolves to `<program> repo list` if `list` is unique among all subcommands.
*   **Position Independence**: Command flags can be interleaved with args after the command is resolved.
*   **Global Flags**: Global flags are parsed anywhere in the argument list.
*   **Config Flag**: `--config/-c` is parsed but not yet wired to config loading.

## CLI Definition Language (.cdl)
The CLI structure is defined in a `.cdl` file using a simple DSL:

| Keyword | Description |
| :--- | :--- |
| `global` | Marks the global flags section. |
| `flag` | Defines a global or command-specific flag (`bool` or `string`). |
| `cmd` | Defines a command or subcommand path. |
| `arg` | Defines a positional argument for the preceding command. |
| `safe` | Marks a command as safe to execute in restricted mode. |
| `example` | Adds a usage example. |
| `topic` | Defines a help topic. |
| `text` | Adds documentation to a topic (supports `"""` for multiline). |
| `name` | Sets the binary name and punch line for the CLI help. |

## Example
`examples/example.cdl`:
```text
global
    name "example" "Minimal CLI sample"
    flag verbose bool "Enable verbose output"

cmd user "Manage users"
flag all bool "Show all users"
cmd user add "Add a user"
arg name string "User name"
example "app user add alice"

cmd project "Manage projects"
cmd project init "Initialize a project"
arg path string "Project path"
example "app project init ./demo"
```

This example defines 2 top-level commands (`user`, `project`) and 2 subcommands
(`user add`, `project init`).

## Generation
`cdlcompiler` embeds the fixed Go support template under `examples/support.tmpl`.
When you run `cligen <file>.cdl <package>`, the generated CLI is written back to the
same directory as `<file>_support.go`, and the support template is reused for every
definition.
