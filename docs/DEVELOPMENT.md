# Development Guide

## Coding Standards
*   **Safety**: Use latest Go features. Avoid `any`. Always include `default: panic` in exhaustive switches.
*   **Structure**: Keep files < 250 lines. Keep functions < 15 lines. 
*   **Immutability**: Use ReadOnly/Writable interfaces for core state.
*   **Packages**: Prefer many small packages over a few large ones.

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
