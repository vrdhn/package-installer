// Package lazyjson provides a thread-safe, lazy-loading manager for JSON files.
// It tracks modifications (dirty state) and ensures atomic writes when saving to disk.
package lazyjson

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Manager provides high-level control over a JSON-backed data structure.
// It handles concurrent access and ensures data is only loaded from disk when first requested.
type manager[T any] struct {
	filepath string
	data     *T
	loaded   bool
	dirty    bool
	mu       sync.RWMutex
	opts     *options[T]
}

type Manager[T any] = *manager[T]

// options holds configuration for the Manager.
type options[T any] struct {
	indent          string
	fileMode        os.FileMode
	createIfMissing bool
	defaultValue    func() *T
}

// New creates a new Manager for the given file path.
func New[T any](filepath string, opts ...Option[T]) *manager[T] {
	mgr := &manager[T]{
		filepath: filepath,
		opts: &options[T]{
			indent:          "  ",
			fileMode:        0644,
			createIfMissing: true,
			defaultValue:    nil,
		},
	}

	for _, opt := range opts {
		opt(mgr.opts)
	}

	return mgr
}

// Get returns the current data, loading it lazily if needed.
// Returns a pointer to the data for reading.
func (m *manager[T]) Get() (*T, error) {
	m.mu.RLock()
	if m.loaded {
		defer m.mu.RUnlock()
		return m.data, nil
	}
	m.mu.RUnlock()

	// Need to load, acquire write lock
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if m.loaded {
		return m.data, nil
	}

	return m.data, m.loadLocked()
}

// Modify executes a function that can modify the data.
// The data is lazily loaded if needed, and automatically marked dirty.
func (m *manager[T]) Modify(fn func(*T) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.loaded {
		if err := m.loadLocked(); err != nil {
			return err
		}
	}

	if err := fn(m.data); err != nil {
		return err
	}

	m.dirty = true
	return nil
}

// Save writes the data to disk if it's dirty.
// Does nothing if the data hasn't been modified.
func (m *manager[T]) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.dirty {
		return nil
	}

	if !m.loaded {
		return errors.New("cannot save: data not loaded")
	}

	return m.saveLocked()
}

// Reload forces a reload from disk, discarding any unsaved changes.
func (m *manager[T]) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.loaded = false
	m.dirty = false
	m.data = nil

	return m.loadLocked()
}

// IsDirty returns true if the data has been modified since the last load/save.
func (m *manager[T]) IsDirty() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.dirty
}

// IsLoaded returns true if the data has been loaded from disk.
func (m *manager[T]) IsLoaded() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.loaded
}

// MarkDirty manually marks the data as dirty.
// Use this if you've modified the data obtained from Get() directly.
func (m *manager[T]) MarkDirty() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dirty = true
}

// loadLocked loads data from the file.
// Must be called with write lock held.
func (m *manager[T]) loadLocked() error {
	data, err := os.ReadFile(m.filepath)
	if err != nil {
		if os.IsNotExist(err) {
			if m.opts.createIfMissing {
				// Use default value if provided
				if m.opts.defaultValue != nil {
					m.data = m.opts.defaultValue()
				} else {
					// Create zero value
					var zero T
					m.data = &zero
				}
				m.loaded = true
				m.dirty = true // Mark dirty so it gets saved
				return nil
			}
			return fmt.Errorf("file not found: %w", err)
		}
		return fmt.Errorf("failed to read file: %w", err)
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	m.data = &result
	m.loaded = true
	m.dirty = false

	return nil
}

// saveLocked writes data to the file atomically.
// Must be called with write lock held.
func (m *manager[T]) saveLocked() error {
	// Marshal the data
	var data []byte
	var err error

	if m.opts.indent != "" {
		data, err = json.MarshalIndent(m.data, "", m.opts.indent)
	} else {
		data, err = json.Marshal(m.data)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(m.filepath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Atomic write: write to temp file, then rename
	tempFile := m.filepath + ".tmp"
	if err := os.WriteFile(tempFile, data, m.opts.fileMode); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tempFile, m.filepath); err != nil {
		os.Remove(tempFile) // Clean up temp file on error
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	m.dirty = false
	return nil
}

// SaveIfDirty is a convenience method that saves only if dirty.
// This is equivalent to Save(), but the name makes intent clearer.
func (m *manager[T]) SaveIfDirty() error {
	return m.Save()
}

// MustSave saves the data and panics on error.
// Useful for cleanup in defer statements.
func (m *manager[T]) MustSave() {
	if err := m.Save(); err != nil {
		panic(fmt.Sprintf("failed to save: %v", err))
	}
}
