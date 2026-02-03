# Modules Documentation

## pkg/archive
Logic to unarchive various formats.

- **Functions**:
  - `SupportedExtensions() []string`: Returns supported archive extensions.
  - `IsSupported(filename string) bool`: Checks if a filename has a supported extension.
  - `Extract(src string, dest string) error`: Extracts an archive to a destination.

## pkg/cache
Manages file system locking and idempotency.

- **Functions**:
  - `Lock(target string) (func() error, error)`: Acquires a file-based lock.
  - `Ensure(target string, fn func() error) error`: Runs a function only if target doesn't exist, with locking.

## pkg/config
System configuration and XDG path derivation.

- **Types**:
  - `OSType`: String enumeration for Operating Systems.
  - `ArchType`: String enumeration for Architectures.
  - `Config`: Struct containing derived paths and system info.
- **Constants**:
  - `OSLinux`, `OSDarwin`, `OSWindows`, `OSUnknown`
  - `ArchX64`, `ArchArm64`, `ArchUnknown`
- **Functions**:
  - `Init() (*Config, error)`: Initializes configuration.
  - `ParseOS(os string) (OSType, error)`: Parses string to `OSType`.
  - `ParseArch(arch string) (ArchType, error)`: Parses string to `ArchType`.

## pkg/display
Visualization of tasks and progress.

- **Interfaces**:
  - `Task`: Interface for reporting progress, logs, and stages.
  - `Display`: Interface for managing tasks.
- **Functions**:
  - `NewConsole() Display`: Creates a stdout-based display.
  - `NewWriterDisplay(w io.Writer) Display`: Creates a custom writer-based display.

## pkg/downloader
URI-based artifact downloading.

- **Interfaces**:
  - `Downloader`: Main interface for downloading resources.
  - `SchemeHandler`: Interface for protocol-specific handlers.
- **Functions**:
  - `NewDefaultDownloader() Downloader`: Returns a downloader with HTTP/HTTPS support.
  - `NewHTTPHandler() SchemeHandler`: Returns an HTTP/HTTPS scheme handler.

## pkg/recipe
Ecosystem-specific package discovery.

- **Types**:
  - `PackageDefinition`: Metadata about a specific package build.
  - `Recipe`: Defines discovery URLs, parsers, and filters.
- **Functions**:
  - `GetNodejsRecipe() *Recipe`: Returns the Node.js recipe.
  - `GetJavaRecipe() *Recipe`: Returns the Java (Foojay) recipe.

## pkg/resolver
Package version and platform resolution.

- **Functions**:
  - `Resolve(ctx context.Context, cfg *config.Config, r *recipe.Recipe, version string, task display.Task) (*recipe.PackageDefinition, error)`: Resolves a recipe/version to a specific package.

## pkg/installer
Orchestrates the download and extraction process.

- **Types**:
  - `Plan`: Installation metadata and target paths.
  - `Stage`: Function type for an installation step.
- **Functions**:
  - `NewPlan(cfg *config.Config, pkg recipe.PackageDefinition) (*Plan, error)`: Creates an installation plan.
  - `Install(ctx context.Context, plan *Plan, task display.Task) error`: Runs the full installation flow.
  - `DownloadStage`, `ExtractStage`: Individual installation stages.

## pkg/bubblewrap
Bubblewrap-based sandboxing.

- **Types**:
  - `Bubblewrap`: Struct to configure a bubblewrap sandbox.
  - `BindType`: String enumeration for bind mount types.
- **Functions**:
  - `Create() *Bubblewrap`: Initializes a bubblewrap configuration.
  - `Exec(cmd *exec.Cmd) error`: Replaces current process with the command.
