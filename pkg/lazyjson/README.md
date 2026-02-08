# jsonstore

A generic Go package for lazy-loading JSON configuration files with automatic dirty tracking and thread-safe operations.

## Features

- **Generic Type Support**: Works with any JSON-serializable struct
- **Lazy Loading**: Data loaded only when first accessed
- **Dirty Tracking**: Automatic tracking of modifications
- **Thread-Safe**: Built-in mutex protection for concurrent access
- **Atomic Writes**: Safe file writes with temp file + rename
- **Functional Options**: Flexible configuration with sensible defaults
- **Zero Dependencies**: Uses only Go standard library

## Installation

```bash
go get github.com/yourusername/jsonstore
```

## Quick Start

```go
package main

import (
    "log"
    "jsonstore"
)

type Config struct {
    Host string `json:"host"`
    Port int    `json:"port"`
}

func main() {
    // Create manager - file doesn't need to exist yet
    mgr := jsonstore.New[Config]("config.json")

    // Lazy load and read
    cfg, err := mgr.Get()
    if err != nil {
        log.Fatal(err)
    }

    // Modify - automatically marks dirty
    err = mgr.Modify(func(c *Config) error {
        c.Port = 8080
        return nil
    })

    // Save only if modified
    if err := mgr.Save(); err != nil {
        log.Fatal(err)
    }
}
```

## Core API

### Creating a Manager

```go
// Basic creation
mgr := jsonstore.New[YourType]("path/to/file.json")

// With options
mgr := jsonstore.New[YourType](
    "config.json",
    jsonstore.WithDefaultValue[YourType](defaultFunc),
    jsonstore.WithIndent[YourType]("    "),
    jsonstore.WithFileMode[YourType](0600),
)
```

### Reading Data

```go
// Get returns a pointer to the data (lazy loads on first call)
cfg, err := mgr.Get()
if err != nil {
    // handle error
}

// Use the data
fmt.Println(cfg.Host)
```

### Modifying Data

```go
// Modify executes a function that can change the data
// Automatically marks as dirty
err := mgr.Modify(func(cfg *Config) error {
    cfg.Port = 9000
    cfg.Host = "example.com"
    return nil
})

// Multiple modifications are batched
mgr.Modify(func(cfg *Config) error {
    cfg.Setting1 = "value1"
    return nil
})
mgr.Modify(func(cfg *Config) error {
    cfg.Setting2 = "value2"
    return nil
})
// Both tracked under single dirty flag
```

### Saving Data

```go
// Save writes to disk only if dirty
err := mgr.Save()

// SaveIfDirty is an alias (more explicit)
err := mgr.SaveIfDirty()

// MustSave panics on error (useful in defer)
defer mgr.MustSave()
```

### Checking State

```go
// Check if data has been loaded
if mgr.IsLoaded() {
    // data is in memory
}

// Check if data has been modified
if mgr.IsDirty() {
    // unsaved changes exist
}
```

### Reloading

```go
// Reload from disk (discards unsaved changes)
err := mgr.Reload()
```

### Manual Dirty Tracking

```go
// If you modify the data from Get() directly
cfg, _ := mgr.Get()
cfg.Port = 8080  // Direct modification

// Manually mark as dirty
mgr.MarkDirty()
```

## Options

### WithDefaultValue

Provide a custom default value when file doesn't exist:

```go
defaultFunc := func() *Config {
    return &Config{
        Host: "localhost",
        Port: 8080,
    }
}

mgr := jsonstore.New[Config](
    "config.json",
    jsonstore.WithDefaultValue[Config](defaultFunc),
)
```

### WithIndent

Control JSON formatting:

```go
// Pretty-printed with 4 spaces
mgr := jsonstore.New[Config](
    "config.json",
    jsonstore.WithIndent[Config]("    "),
)

// Compact JSON (single line)
mgr := jsonstore.New[Config](
    "config.json",
    jsonstore.WithCompactJSON[Config](),
)
```

### WithFileMode

Set file permissions:

```go
mgr := jsonstore.New[Config](
    "config.json",
    jsonstore.WithFileMode[Config](0600), // Owner read/write only
)
```

### WithCreateIfMissing

Control behavior when file doesn't exist:

```go
// Default: true (creates file with zero value)
mgr := jsonstore.New[Config]("config.json")

// Fail if file doesn't exist
mgr := jsonstore.New[Config](
    "config.json",
    jsonstore.WithCreateIfMissing[Config](false),
)
```

## Thread Safety

All operations are thread-safe. The manager uses `sync.RWMutex` internally:

```go
// Safe concurrent reads
go func() {
    cfg, _ := mgr.Get()
    fmt.Println(cfg.Port)
}()

go func() {
    cfg, _ := mgr.Get()
    fmt.Println(cfg.Host)
}()

// Safe concurrent writes
go func() {
    mgr.Modify(func(c *Config) error {
        c.Port = 8080
        return nil
    })
}()

go func() {
    mgr.Modify(func(c *Config) error {
        c.Host = "example.com"
        return nil
    })
}()
```

## How It Works

### Lazy Loading

Data is only loaded from disk when first accessed:

```go
mgr := jsonstore.New[Config]("config.json")  // No file I/O

cfg, _ := mgr.Get()  // File loaded here (if exists)

cfg2, _ := mgr.Get()  // Returns cached data, no I/O
```

### Dirty Tracking

Changes are tracked automatically:

```go
mgr.Get()              // loaded=true, dirty=false
mgr.Modify(...)        // loaded=true, dirty=true
mgr.Save()             // loaded=true, dirty=false
mgr.Reload()           // loaded=true, dirty=false (reloaded)
```

### Atomic Writes

Files are written atomically to prevent corruption:

1. Marshal data to JSON
2. Write to temporary file (`.tmp` suffix)
3. Rename temporary file to target (atomic operation)
4. Clean up on errors

## Best Practices

### 1. Use Modify() for Changes

```go
// Good: Thread-safe, automatically marks dirty
mgr.Modify(func(cfg *Config) error {
    cfg.Port = 8080
    return nil
})

// Avoid: Requires manual dirty tracking
cfg, _ := mgr.Get()
cfg.Port = 8080
mgr.MarkDirty()
```

### 2. Save After All Modifications

```go
// Batch modifications
mgr.Modify(func(cfg *Config) error {
    cfg.Port = 8080
    return nil
})
mgr.Modify(func(cfg *Config) error {
    cfg.Host = "example.com"
    return nil
})

// Single save for all changes
mgr.Save()
```

### 3. Use Defer for Cleanup

```go
func processConfig() error {
    mgr := jsonstore.New[Config]("config.json")
    defer mgr.MustSave()  // Ensures save on function exit

    // Make changes...
    mgr.Modify(func(cfg *Config) error {
        cfg.Port = 8080
        return nil
    })

    return nil
}
```

### 4. Check Errors

```go
// Always check errors on critical operations
cfg, err := mgr.Get()
if err != nil {
    return fmt.Errorf("failed to load config: %w", err)
}

err = mgr.Save()
if err != nil {
    return fmt.Errorf("failed to save config: %w", err)
}
```

## Testing

Run tests:

```bash
cd jsonstore
go test -v
go test -race  # With race detector
```

## License

MIT
