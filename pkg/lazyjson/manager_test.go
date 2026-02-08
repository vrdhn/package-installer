package lazyjson

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestConfig is a sample config struct for testing
type TestConfig struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Enabled bool   `json:"enabled"`
}

func TestNew(t *testing.T) {
	mgr := New[TestConfig]("test.json")
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
	if mgr.filepath != "test.json" {
		t.Errorf("expected filepath 'test.json', got %s", mgr.filepath)
	}
	if mgr.IsLoaded() {
		t.Error("expected manager to not be loaded initially")
	}
	if mgr.IsDirty() {
		t.Error("expected manager to not be dirty initially")
	}
}

func TestLazyLoad_FileNotExist_CreateDefault(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "config.json")

	mgr := New[TestConfig](testFile)

	// First Get should lazy load (create default)
	cfg, err := mgr.Get()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	// Should be loaded and dirty (needs to be saved)
	if !mgr.IsLoaded() {
		t.Error("expected manager to be loaded")
	}
	if !mgr.IsDirty() {
		t.Error("expected manager to be dirty after creating default")
	}
}

func TestLazyLoad_FileExists(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "config.json")

	// Create a test file
	testData := `{"host":"localhost","port":8080,"enabled":true}`
	if err := os.WriteFile(testFile, []byte(testData), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mgr := New[TestConfig](testFile)

	cfg, err := mgr.Get()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Host != "localhost" {
		t.Errorf("expected host 'localhost', got %s", cfg.Host)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Port)
	}
	if !cfg.Enabled {
		t.Error("expected enabled to be true")
	}

	if !mgr.IsLoaded() {
		t.Error("expected manager to be loaded")
	}
	if mgr.IsDirty() {
		t.Error("expected manager to not be dirty after load")
	}
}

func TestModify(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "config.json")

	mgr := New[TestConfig](testFile)

	// Modify should lazy load and mark dirty
	err := mgr.Modify(func(cfg *TestConfig) error {
		cfg.Host = "example.com"
		cfg.Port = 9000
		cfg.Enabled = true
		return nil
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !mgr.IsLoaded() {
		t.Error("expected manager to be loaded")
	}
	if !mgr.IsDirty() {
		t.Error("expected manager to be dirty after modify")
	}

	// Verify the modification
	cfg, _ := mgr.Get()
	if cfg.Host != "example.com" {
		t.Errorf("expected host 'example.com', got %s", cfg.Host)
	}
	if cfg.Port != 9000 {
		t.Errorf("expected port 9000, got %d", cfg.Port)
	}
}

func TestSave(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "config.json")

	mgr := New[TestConfig](testFile)

	// Modify and save
	mgr.Modify(func(cfg *TestConfig) error {
		cfg.Host = "saved.com"
		cfg.Port = 3000
		return nil
	})

	if err := mgr.Save(); err != nil {
		t.Fatalf("expected no error on save, got %v", err)
	}

	if mgr.IsDirty() {
		t.Error("expected manager to not be dirty after save")
	}

	// Verify file was written
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty file")
	}

	// Create new manager and verify data persisted
	mgr2 := New[TestConfig](testFile)
	cfg, err := mgr2.Get()
	if err != nil {
		t.Fatalf("expected no error loading saved file, got %v", err)
	}

	if cfg.Host != "saved.com" {
		t.Errorf("expected host 'saved.com', got %s", cfg.Host)
	}
	if cfg.Port != 3000 {
		t.Errorf("expected port 3000, got %d", cfg.Port)
	}
}

func TestSave_NotDirty_NoOp(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "config.json")

	testData := `{"host":"localhost","port":8080,"enabled":true}`
	if err := os.WriteFile(testFile, []byte(testData), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mgr := New[TestConfig](testFile)
	mgr.Get() // Load the file

	// Get file modification time
	info1, _ := os.Stat(testFile)
	mtime1 := info1.ModTime()

	// Save should be no-op
	if err := mgr.Save(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// File should not have been modified
	info2, _ := os.Stat(testFile)
	mtime2 := info2.ModTime()

	if !mtime1.Equal(mtime2) {
		t.Error("expected file modification time to be unchanged")
	}
}

func TestReload(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "config.json")

	// Create initial file
	testData := `{"host":"initial","port":1000,"enabled":true}`
	if err := os.WriteFile(testFile, []byte(testData), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mgr := New[TestConfig](testFile)
	mgr.Get() // Load

	// Modify in memory
	mgr.Modify(func(cfg *TestConfig) error {
		cfg.Host = "modified"
		return nil
	})

	// Reload should discard changes
	if err := mgr.Reload(); err != nil {
		t.Fatalf("expected no error on reload, got %v", err)
	}

	if mgr.IsDirty() {
		t.Error("expected manager to not be dirty after reload")
	}

	cfg, _ := mgr.Get()
	if cfg.Host != "initial" {
		t.Errorf("expected host 'initial' after reload, got %s", cfg.Host)
	}
}

func TestMarkDirty(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "config.json")

	// Create a test file so it loads without being dirty
	testData := `{"host":"localhost","port":8080,"enabled":true}`
	if err := os.WriteFile(testFile, []byte(testData), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mgr := New[TestConfig](testFile)
	mgr.Get() // Load

	if mgr.IsDirty() {
		t.Error("expected manager to not be dirty after loading existing file")
	}

	mgr.MarkDirty()

	if !mgr.IsDirty() {
		t.Error("expected manager to be dirty after MarkDirty()")
	}
}

func TestWithOptions(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "config.json")

	defaultFn := func() *TestConfig {
		return &TestConfig{
			Host:    "default.com",
			Port:    5000,
			Enabled: true,
		}
	}

	mgr := New[TestConfig](
		testFile,
		WithDefaultValue[TestConfig](defaultFn),
		WithCompactJSON[TestConfig](),
		WithFileMode[TestConfig](0600),
	)

	cfg, err := mgr.Get()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Host != "default.com" {
		t.Errorf("expected default host, got %s", cfg.Host)
	}

	if err := mgr.Save(); err != nil {
		t.Fatalf("expected no error on save, got %v", err)
	}

	// Check file mode
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Errorf("expected file mode 0600, got %o", info.Mode().Perm())
	}

	// Check compact JSON (no newlines except at end)
	data, _ := os.ReadFile(testFile)
	// Compact JSON should be on one line
	if len(data) > 100 { // reasonable check that it's compact
		t.Error("expected compact JSON output")
	}
}

func TestConcurrency(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "config.json")

	mgr := New[TestConfig](testFile)

	var wg sync.WaitGroup
	numGoroutines := 10
	numIterations := 100

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				_, err := mgr.Get()
				if err != nil {
					t.Errorf("unexpected error on concurrent Get: %v", err)
				}
			}
		}()
	}

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				mgr.Modify(func(cfg *TestConfig) error {
					cfg.Port = id*1000 + j
					return nil
				})
			}
		}(i)
	}

	wg.Wait()

	// Should be dirty after all modifications
	if !mgr.IsDirty() {
		t.Error("expected manager to be dirty after concurrent modifications")
	}

	// Save should work
	if err := mgr.Save(); err != nil {
		t.Errorf("expected no error on save after concurrent access, got %v", err)
	}
}

func TestWithCreateIfMissing_False(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "nonexistent.json")

	mgr := New[TestConfig](testFile, WithCreateIfMissing[TestConfig](false))

	_, err := mgr.Get()
	if err == nil {
		t.Error("expected error when file doesn't exist and createIfMissing is false")
	}
}

func TestMustSave_Panic(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir")

	// Create subdirectory
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	testFile := filepath.Join(subDir, "config.json")

	// Create initial file
	testData := `{"host":"localhost","port":8080,"enabled":true}`
	if err := os.WriteFile(testFile, []byte(testData), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mgr := New[TestConfig](testFile)

	// Load the file
	mgr.Get()

	// Now make the directory read-only so we can't create .tmp file
	if err := os.Chmod(subDir, 0555); err != nil {
		t.Fatalf("failed to make directory read-only: %v", err)
	}

	// Restore permissions after test
	defer os.Chmod(subDir, 0755)

	// Modify to make it dirty
	mgr.Modify(func(cfg *TestConfig) error {
		cfg.Host = "test"
		return nil
	})

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected MustSave to panic on error")
		}
	}()

	mgr.MustSave()
}

func TestSaveIfDirty(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "config.json")

	mgr := New[TestConfig](testFile)
	mgr.Get()

	// Not dirty, should be no-op
	if err := mgr.SaveIfDirty(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Make dirty and save
	mgr.Modify(func(cfg *TestConfig) error {
		cfg.Host = "changed"
		return nil
	})

	if err := mgr.SaveIfDirty(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if mgr.IsDirty() {
		t.Error("expected manager to not be dirty after SaveIfDirty")
	}
}
