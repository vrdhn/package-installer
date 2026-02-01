# CLI Definition Language (.def)

`pi` uses a simple, indentation-aware DSL to define its command-line interface. This file is parsed at runtime to build the command tree, handle flag parsing, and generate help documentation.

## General Syntax

- **Comments**: Lines starting with `#` are ignored.
- **Indentation**: Used to define hierarchy. Commands, flags, and arguments belong to the parent keyword they are indented under.
- **Keywords**: Each line starts with a keyword (`flag`, `cmd`, `arg`, `example`, `topic`, `text`).

---

## Global Elements

Keywords at the root (no indentation) apply globally.

### `flag <name> <type> "<description>" [<short>]`
Defines a global flag.
- **type**: `bool` or `string`.
- **short**: (Optional) Single character alias (e.g., `v` for `verbose`).

Example:
```
flag verbose bool "Enable verbose output" v
```

---

## Command Hierarchy

### `cmd <name> "<description>"`
Defines a top-level command. Sub-commands are defined by nesting another `cmd` under it with indentation.

Example:
```
cmd remote "Manage repositories"
    cmd list "List all"
```

### `arg <name> <type> "<description>"`
Defines a positional argument for the parent command.
- **type**: Usually `string`.

### `flag <name> <type> "<description>" [<short>]`
When indented under a `cmd`, defines a command-specific flag.

### `example "<text>"`
Adds an example usage string to the command's help page.

Example:
```
cmd install "Install package"
    arg package string "Name of pkg"
    flag force bool "Overwrite" f
    example "pi install nodejs@20"
```

---

## Documentation Topics

Guide books and conceptual documentation are defined using `topic`.

### `topic <name> "<description>"`
Defines a help topic.

### `text <content>`
Adds documentation text to the parent topic.

### Multiline Strings (`"""`)
For long documentation, use triple double-quotes.

Example:
```
topic cave "The Sandbox"
    text """
    This is a multiline description
    of how caves work in pi.
    """
```

---

## Behavior Features

1.  **Unique Prefix Matching**: Users can type `pi rem` instead of `pi remote` if `rem` is unambiguous.
2.  **Parent Omission**: Subcommands like `pi list` will resolve to `pi remote list` if `list` is unique across all subcommands.
3.  **Automatic Help**: The `--help` flag and `help` command are built-in and use the definitions in the `.def` file to render documentation.
