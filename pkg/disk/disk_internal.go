package disk

import (
	"pi/pkg/config"
	"pi/pkg/display"
)

// manager defines the internal state for managing pi's local storage.
type manager struct {
	cfg  config.Config
	Disp display.Display
}

// Manager is a pointer to the internal manager implementation.
type Manager = *manager

// NewManager creates a new disk manager with the specified configuration and display.
func NewManager(cfg config.Config, disp display.Display) Manager {
	return &manager{cfg: cfg, Disp: disp}
}

// Usage represents disk usage information for a specific category of data.
type Usage struct {
	Label string
	Size  int64
	Items int
	Path  string
}
