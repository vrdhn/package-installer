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
    *   Design a simple, line-oriented DSL to declare global flags, commands, subcommands, and their respective flags/arguments.
    *   Support types: `bool`, `string`, `int`.
    *   Include descriptions for automatic help generation.
    *   Support positional arguments with optionality and varargs.

2.  **Develop CLI Generator (`tool/cligen`)**:
    *   A small Go utility that reads `cli.def`.
    *   Parses the DSL into an internal representation.
    *   Uses `text/template` to generate `pkg/cli/generated.go`.
    *   **Decision**: We will generate a **custom parser** instead of using a library like `cobra`. This allows us to natively support the "shortest unique prefix" and "omitted command" requirements without the overhead and rigidity of an external dependency.

3.  **Generated Code Structure & Parsing Logic**:
    *   **Registry**: A generated data structure containing all possible command paths and flag definitions.
    *   **Custom Parser**:
        *   **Prefix Matching**: Logic to resolve `i` to `install` or `e` to `enter` by checking for the shortest unambiguous match.
        *   **Command Omission**: If a word doesn't match a top-level command but uniquely matches a subcommand (e.g., `pi list` instead of `pi remote list`), the parser will automatically infer the parent command.
        *   **Position Independence**: Since flags are guaranteed not to overlap, the parser will collect them from any position in the argument list.
    *   **Interfaces**: A `Runner` interface with an `Execute(ctx context.Context) error` method for each command/subcommand.

4.  **Integration & Usage**:
    *   Add `//go:generate go run ./tool/cligen` to `main.go`.
    *   Implement command handlers in `pkg/cli/handlers/` (manual code) that satisfy the generated interfaces.
    *   Update `main.go` to be a thin wrapper around the generated parser.

5.  **Evolution**:
    *   Add validation (e.g., choice lists, range checks) to the DSL.
    *   Support shell completion generation based on the registry.
