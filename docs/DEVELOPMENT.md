# Development Guide

## Coding Standards
*   **Safety**: Prefer explicit types; use `any` sparingly. Use `default: panic` in truly exhaustive switches.
*   **Structure**: Keep functions small where practical, but no hard limits are enforced.
*   **Immutability**: The ReadOnly/Writable pattern is used in `config`; other packages are conventional Go structs.
*   **Packages**: The codebase favors small, focused packages.

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
