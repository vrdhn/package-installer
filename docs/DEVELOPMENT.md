# Development Guide

## Coding Standards
*   **Safety**: Prefer explicit types; use `any` sparingly. Use `default: panic` in truly exhaustive switches.
*   **Functional Style**: Favor immutable structs and data-driven design. Logic should be separated from data definitions where possible. Commands return structured results (`ExecutionResult`) instead of printing directly.
*   **Encapsulation**: If a struct is not supposed to be copied, export a pointer type using the pattern `type T = *t`.
*   **Interfaces**: Only use interfaces when multiple implementations are truly expected or required for testing. Avoid unneeded abstractions.
*   **Logging**: Use `log/slog` for all internal progress reporting and status updates. Avoid custom "Task" or "Progress" abstractions in the core logic.
*   **Structure**: Keep functions small and focused on a single purpose. Break down complex logic into meaningful helper methods.
*   **Packages**: The codebase favors small, focused packages with single responsibilities. Avoid abstraction leakage by keeping internal details private.

## Module Overview
| Package | Description |
| :--- | :--- |
| `pkg/archive` | Archive extraction (tar.gz, zip). |
| `pkg/bubblewrap` | Linux sandboxing implementation. |
| `pkg/cache` | File locking and idempotency. |
| `pkg/cave` | Workspace and Cave management. |
| `pkg/cli` | Custom CLI parser and DSL engine. |
| `pkg/display` | Progress bars and console UI. |
| `pkg/downloader` | Pluggable artifact downloading. |
| `pkg/recipe` | Starlark-based package discovery. |
| `pkg/resolver` | Version and platform matching. |
| `pkg/installer` | Extraction and plan execution. |

## Verification
Before committing:
```bash
go fmt ./...
go vet ./...
go test ./...
```
