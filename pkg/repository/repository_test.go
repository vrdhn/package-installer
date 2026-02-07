package repository

import (
	"os"
	"path/filepath"
	"pi/pkg/config"
	"pi/pkg/display"
	"testing"
)

type mockConfig struct {
	config.Config
	tempDir string
}

func (m *mockConfig) GetConfigDir() string {
	return filepath.Join(m.tempDir, ".config", "pi")
}

func (m *mockConfig) GetCacheDir() string {
	return filepath.Join(m.tempDir, ".cache", "pi")
}

func TestManager_AutoSyncBuiltins(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pi-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	cfg, err := config.Init()
	if err != nil {
		t.Fatal(err)
	}

	mock := &mockConfig{Config: cfg, tempDir: tempDir}
	disp := display.NewConsole()

	m, err := NewManager(disp, mock)
	if err != nil {
		t.Fatal(err)
	}

	// Index should be empty initially as we haven't synced and file doesn't exist
	if len(m.index) != 0 {
		t.Errorf("Expected empty index, got %d", len(m.index))
	}

	// GetFullRegistryInfo should trigger auto-sync
	entries, err := m.GetFullRegistryInfo(false)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) == 0 {
		t.Errorf("Expected non-empty entries after auto-sync")
	}

	foundBuiltin := false
	for _, e := range entries {
		if e.RepoName == "builtin" {
			foundBuiltin = true
			break
		}
	}

	if !foundBuiltin {
		t.Errorf("Builtin repo not found in registry info")
	}

	// Check if packages.csv was created
	csvPath := filepath.Join(mock.GetConfigDir(), "packages.csv")
	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		t.Errorf("packages.csv should have been created at %s", csvPath)
	}
}
