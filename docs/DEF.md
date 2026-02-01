# CLI Definition Language (.def)

`pi` uses a simple, flat DSL to define its command-line interface. This file is parsed at runtime to build the command tree, handle flag parsing, and generate help documentation.

## General Syntax

- **Comments**: Lines starting with `#` are ignored.
- **Hierarchy-Independent**: Indentation and braces are not used. Hierarchy is defined by command paths or contextual attachment.
- **Keywords**: Each statement starts with a keyword (`flag`, `cmd`, `arg`, `example`, `topic`, `text`).
- **Strings**: Single-line strings are enclosed in double quotes `"`. Multi-line strings use triple double-quotes `"""`.

---

## Global Elements

`flag` keywords appearing before any `cmd` are considered global.

### `flag <name> <type> "<description>" [<short>]`
Defines a flag.
- **type**: `bool` or `string`.
- **short**: (Optional) Single character alias (e.g., `v` for `verbose`).

Example:
```
flag verbose bool "Enable verbose output" v
```

---

## Command Hierarchy

### `cmd <path> "<description>"`
Defines a command. The `<path>` can be a single word for top-level commands or multiple words for subcommands.

Example:
```
cmd remote "Manage repositories"
cmd remote list "List all"
```

### `arg <name> <type> "<description>"`
Defines a positional argument for the *immediately preceding* command.

### `flag <name> <type> "<description>" [<short>]`
When following a `cmd`, defines a command-specific flag.

### `example "<text>"`
Adds an example usage string to the *immediately preceding* command.

Example:
```
cmd install "Install package"
arg package string "Name of pkg"
flag force bool "Overwrite" f
example "pi install nodejs@20"
```

---

## Documentation Topics

Guide books and conceptual documentation are defined using `topic` and `text`.

### `topic <name> "<description>"`
Defines a help topic.

### `text <content>`
Adds documentation text to the *immediately preceding* topic.

### Multiline Strings (`"""`)
Used for long documentation.
**Formatting Rules:**
- Leading spaces on each line are replaced by a single space.
- Empty lines in the source are preserved as newlines.
- Leading and trailing empty lines are stripped.

Example:
```
topic cave "The Sandbox"
text """
    A 'Cave' is an isolated environment powered by Linux bubblewrap.
    It ensures that your project has exactly the tools it needs.
    """
```

---

## Behavior Features

1.  **Unique Prefix Matching**: Users can type `pi rem` instead of `pi remote` if `rem` is unambiguous.
2.  **Parent Omission**: Subcommands like `pi list` will resolve to `pi remote list` if `list` is unique across all subcommands.
3.  **Automatic Help**: The `--help` flag and `help` command are built-in and use the definitions in the `.def` file to render documentation.
