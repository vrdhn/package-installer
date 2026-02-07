# cdlcompiler

`cdlcompiler` is a tool that generates type-safe, generic Go command-line interfaces from a simple CLI Definition Language (`.cdl`). It emphasizes developer productivity by automating boilerplate while providing a powerful and flexible command structure.

## Key Features

- **Generics**: The generated parser is generic over the execution result type.
- **Unique Prefix Matching**: Users can type `app rep` instead of `app repository` if it's unambiguous.
- **Command Omission**: Deeply nested subcommands can be invoked directly if their name is unique (e.g., `app list` instead of `app user list`).
- **Type-Safe Params**: Commands receive a dedicated struct containing all parsed flags and arguments with correct Go types.
- **Automatic Help**: Generates comprehensive help screens for the root, individual commands, and custom documentation topics.
- **Attributes**: Attach custom metadata (bool, string, int) to commands for use in your implementation.

## Usage

1.  Define your CLI in a `.cdl` file.
2.  Run the compiler:
    ```bash
    go run pi/tool/cdlcompiler <file.cdl> <package_name>
    ```
3.  Implement the required `Handlers[T]` interface and your result type `T` in your Go package.

## CLI Definition Language (.cdl) Syntax

### Global Section
The `global` keyword starts the global configuration.
- `name "app" "tagline"`: Sets the binary name and a short description.
- `flag <name> <type> "description" [<short>]`: Defines a global flag available to all commands.
- `attr <name> = <default-value>`: Defines a global attribute inherited by all commands.

### Commands
- `cmd <path> "description"`: Defines a command or subcommand. Paths use spaces (e.g., `cmd user add`).
- `flag <name> <type> "description" [<short>]`: Defines a flag for the preceding command.
- `arg <name> <type> "description"`: Defines a positional argument for the preceding command.
- `example "command string"`: Adds a usage example to the help output.
- `attr <name> = <override-value>`: Sets or overrides an attribute for the preceding command.

### Documentation Topics
- `topic <name> "description"`: Defines a standalone help topic.
- `text "content"` or `text """multiline content"""`: Sets the body of the topic.

### Supported Types
- `bool`: Parsed as a boolean flag.
- `string`: Parsed as a string.
- `int`: (Attributes only) Parsed as an integer.

## Go Integration Conventions

The compiler generates two files in the specified package:
1. `<basename>.go`: Contains the CLI definitions and internal parsing logic.
2. `<basename>_support.go`: Contains helper functions for flag retrieval and resolution.

### What You Must Implement

In your application, you must define a result type (e.g., `ExecutionResult`) and implement the generated `Handlers[T]` interface.

#### 1. ExecutionResult
```go
type ExecutionResult struct {
    ExitCode int
}
```

#### 2. Handlers Interface
The compiler generates a `Handlers[T]` interface. You must implement `Help` and a `Run<Command>` method for every leaf command.

```go
import "yourpackage/cli"

type myHandlers struct {}

func (h *myHandlers) Help(args []string) (ExecutionResult, error) {
    cli.PrintHelp(args...)
    return ExecutionResult{ExitCode: 0}, nil
}

func (h *myHandlers) RunUserAdd(params *cli.UserAddParams) (ExecutionResult, error) {
    // Your logic here
    return ExecutionResult{ExitCode: 0}, nil
}
```

### The Parse Function

The generated `cli.Parse[T](h Handlers[T], args []string)` function is the entry point. It returns a `cli.Action[T]` (a function that executes the command) and the resolved `*cli.CommandDef`.

#### Entry Point Example
```go
func main() {
    h := &myHandlers{}

    // Parse os.Args[1:] with generics
    action, cmd, err := cli.Parse[ExecutionResult](h, os.Args[1:])
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }

    // Inspect resolved command and its attributes
    if cmd != nil && (cmd.Safe == false) {
        fmt.Println("Warning: This is a potentially dangerous command!")
    }

    // Execute the action
    res, err := action()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }

	os.Exit(res.ExitCode)

}
```
