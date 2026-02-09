package disk

import (
	"pi/pkg/config"
)

// manager defines the internal state for managing pi's local storage.
type manager struct {
	cfg config.Config
}

// Manager is a pointer to the internal manager implementation.
type Manager = *manager

// NewManager creates a new disk manager with the specified configuration.
func NewManager(cfg config.Config) Manager {
	return &manager{cfg: cfg}
}

// Usage represents disk usage information for a specific category of data.
type Usage struct {
	Label string
	Size  int64
	Items int
	Path  string
}
