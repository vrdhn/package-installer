package resolver

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"pi/pkg/config"
	"pi/pkg/recipe"
	"testing"
)

type mockTask struct{}

func (m *mockTask) Log(msg string)                      {}
func (m *mockTask) SetStage(name string, target string) {}
func (m *mockTask) Progress(percent int, message string) {}
func (m *mockTask) Done()                               {}

func TestResolve(t *testing.T) {
	cfg, _ := config.Init()
	osName := cfg.OS
	arch := cfg.Arch

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `[
			{"version":"v20.11.1","files":["%s-%s"]},
			{"version":"v18.0.0","files":["%s-%s"]}
		]`, osName, arch, osName, arch)
	}))
	defer ts.Close()

	r := recipe.GetNodejsRecipe()
	r.DiscoveryURL = ts.URL // Point to mock server

	tmpDir := t.TempDir()
	cfg.DiscoveryDir = filepath.Join(tmpDir, "discovery")

	// Test latest
	pkg, err := Resolve(context.Background(), cfg, r, "latest", &mockTask{})
	if err != nil {
		t.Fatalf("Resolve latest failed: %v", err)
	}
	if pkg.Version != "20.11.1" {
		t.Errorf("Expected version 20.11.1, got %s", pkg.Version)
	}

	// Test specific
	pkg, err = Resolve(context.Background(), cfg, r, "18.0.0", &mockTask{})
	if err != nil {
		t.Fatalf("Resolve specific failed: %v", err)
	}
	if pkg.Version != "18.0.0" {
		t.Errorf("Expected version 18.0.0, got %s", pkg.Version)
	}
}