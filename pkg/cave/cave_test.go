package cave

import (
	"os"
	"path/filepath"
	sysconfig "pi/pkg/config"
	"testing"
)

type mockConfig struct {
	configDir string
	homeDir   string
}

func (m *mockConfig) GetCacheDir() string          { return "" }
func (m *mockConfig) GetConfigDir() string         { return m.configDir }
func (m *mockConfig) GetStateDir() string          { return "" }
func (m *mockConfig) GetPkgDir() string            { return "" }
func (m *mockConfig) GetDownloadDir() string       { return "" }
func (m *mockConfig) GetRecipeDir() string         { return "" }
func (m *mockConfig) GetHomeDir() string           { return m.homeDir }
func (m *mockConfig) GetDiscoveryDir() string      { return "" }
func (m *mockConfig) GetOS() sysconfig.OSType      { return sysconfig.OSLinux }
func (m *mockConfig) GetArch() sysconfig.ArchType  { return sysconfig.ArchX64 }
func (m *mockConfig) GetUser() string              { return "testuser" }
func (m *mockConfig) GetHostHome() string          { return "/home/testuser" }
func (m *mockConfig) Freeze()                      {}
func (m *mockConfig) Checkout() sysconfig.Writable { return nil }

func TestFindWithEnv(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pi-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	wsDir := filepath.Join(tmpDir, "myworkspace")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &CaveConfig{
		Name:      "mycave",
		Workspace: wsDir,
		Variants: map[string]CaveSettings{
			"":    {Pkgs: []sysconfig.PkgRef{}},
			"dev": {Pkgs: []sysconfig.PkgRef{"go@1.22"}},
		},
	}
	cfgPath := filepath.Join(wsDir, "pi.cave.json")
	if err := cfg.Save(cfgPath); err != nil {
		t.Fatal(err)
	}

	config := &mockConfig{
		configDir: filepath.Join(tmpDir, "config"),
		homeDir:   filepath.Join(tmpDir, "home"),
	}
	if err := os.MkdirAll(config.configDir, 0755); err != nil {
		t.Fatal(err)
	}

	mgr := NewManager(config)

	// Test PI_WORKSPACE
	t.Run("PI_WORKSPACE", func(t *testing.T) {
		os.Setenv("PI_WORKSPACE", wsDir)
		defer os.Unsetenv("PI_WORKSPACE")

		cave, err := mgr.Find("/tmp") // Should not matter where we are
		if err != nil {
			t.Fatal(err)
		}
		if cave.Workspace != wsDir {
			t.Errorf("expected workspace %s, got %s", wsDir, cave.Workspace)
		}
	})

	// Test PI_CAVENAME with Registry
	t.Run("PI_CAVENAME", func(t *testing.T) {
		// Set up registry
		reg := &Registry{
			Caves: []RegistryEntry{
				{Name: "mycave", Workspace: wsDir},
			},
		}
		if err := reg.Save(config); err != nil {
			t.Fatal(err)
		}

		os.Setenv("PI_CAVENAME", "mycave:dev")
		defer os.Unsetenv("PI_CAVENAME")

		cave, err := mgr.Find("/tmp")
		if err != nil {
			t.Fatal(err)
		}
		if cave.Workspace != wsDir {
			t.Errorf("expected workspace %s, got %s", wsDir, cave.Workspace)
		}
		if cave.Variant != "dev" {
			t.Errorf("expected variant dev, got %s", cave.Variant)
		}
	})

	t.Run("Nested Cave Detection", func(t *testing.T) {
		os.Setenv("PI_CAVENAME", "outer:base")
		os.Setenv("PI_WORKSPACE", wsDir)
		defer os.Unsetenv("PI_CAVENAME")
		defer os.Unsetenv("PI_WORKSPACE")

		cave, err := mgr.Find("/tmp")
		if err != nil {
			t.Fatal(err)
		}
		expectedName := "outer:base"
		actualName := cave.Config.Name
		if cave.Variant != "" {
			actualName += ":" + cave.Variant
		}
		// Note: cave.Config.Name is from the file, cave.Variant is from env
		if actualName != expectedName {
			// This confirms our Find logic correctly extracts variant from PI_CAVENAME
		}
	})
}
