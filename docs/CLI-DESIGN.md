# CLI Design

`pi` follows the general syntax

    pi <global flags> command <command flags> subcommand <subcommand flags> <subcommand arguments ...>


The command and subcommands can be abbriviated to shortest possible uambigious name.
The command can be omitted if subcommand can be used to unambigously find command.

global, command, and subcommand flags never overlap, so that the user doesn't have
to remember the order .. e.g. verbosity flag, though global, can be given last also.


# Requirement
Ideally i need to declare all commands, subcommands, flags, and choices in a
text file, and a tiny compiler which would read this and generate the required
go files with all the required structures, and code to do parsing.



# Plan

1.  **Define Specification Language (`cli.def`)**:
    *   Design a simple, keyword-based DSL to declare global flags, commands, subcommands, and their respective flags/arguments.
    *   Indentation and braces are ignored; hierarchy is defined via command paths (e.g., `cmd remote list`) or contextual order.
    *   Support types: `bool`, `string`.
    *   Include descriptions for automatic help generation.
    *   Support multi-line strings with specific formatting rules (leading spaces replaced by single space, empty lines preserved).

2.  **CLI Engine**:
    *   The `pkg/cli` package includes a custom lexer and parser that reads `cli.def` at runtime.
    *   **Lexer**: A fast, character-by-character scanner.
    *   **Parser**: A stateful parser that builds the command tree using path-based resolution and contextual attachment.

3.  **Parsing Logic**:
    *   **Registry**: A generated data structure containing all possible command paths and flag definitions.
    *   **Custom Parser**:
        *   **Prefix Matching**: Logic to resolve `i` to `install` or `e` to `enter` by checking for the shortest unambiguous match.
        *   **Command Omission**: If a word doesn't match a top-level command but uniquely matches a subcommand (e.g., `pi list` instead of `pi remote list`), the parser will automatically infer the parent command.
        *   **Position Independence**: Since flags are guaranteed not to overlap, the parser will collect them from any position in the argument list.
    *   **Interfaces**: A `Runner` interface with an `Execute(ctx context.Context) error` method for each command/subcommand.

4.  **Integration & Usage**:
    *   The `cli.def` file is embedded into the binary.
    *   `main.go` initializes the CLI engine with the embedded definition.
    *   Command handlers are registered to the engine based on their command path.

5.  **Evolution**:
    *   Add validation (e.g., choice lists, range checks) to the DSL.
    *   Support shell completion generation based on the registry.
