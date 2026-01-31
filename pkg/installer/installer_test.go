package installer

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"pi/pkg/config"
	"pi/pkg/recipe"
	"os"
	"path/filepath"
	"testing"
)

type mockTask struct{}

func (m *mockTask) Log(msg string)                       {}
func (m *mockTask) SetStage(name string, target string)  {}
func (m *mockTask) Progress(percent int, message string) {}
func (m *mockTask) Done()                                {}

func TestInstall(t *testing.T) {
	// 0. Setup Config
	cfg, _ := config.Init()
	// Override paths for testing to avoid touching system cache
	tmpDir := t.TempDir()
	cfg.DownloadDir = filepath.Join(tmpDir, "downloads")
	cfg.PkgDir = filepath.Join(tmpDir, "pkgs")

	// 1. Create a mock tar.gz file
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gw := gzip.NewWriter(w)
		tw := tar.NewWriter(gw)

		content := []byte("hello from package")
		hdr := &tar.Header{
			Name: "testpkg/hello.txt",
			Mode: 0644,
			Size: int64(len(content)),
		}
		tw.WriteHeader(hdr)
		tw.Write(content)

		tw.Close()
		gw.Close()
	}))
	defer ts.Close()

	// 2. Setup Plan
	pkg := recipe.PackageDefinition{
		Name:    "testpkg",
		Version: "1.0.0",
		OS:      config.OSLinux,
		Arch:    config.ArchX64,
		URL:     ts.URL + "/testpkg.tar.gz",
	}

	plan, err := NewPlan(cfg, pkg)
	if err != nil {
		t.Fatal(err)
	}

	// 3. Run Install
	err = Install(context.Background(), plan, &mockTask{})
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// 4. Verify
	if _, err := os.Stat(plan.DownloadPath); err != nil {
		t.Errorf("Download file missing: %v", err)
	}

	helloFile := filepath.Join(plan.InstallPath, "testpkg/hello.txt")
	content, err := os.ReadFile(helloFile)
	if err != nil {
		t.Fatalf("Extracted file missing: %v", err)
	}
	if string(content) != "hello from package" {
		t.Errorf("Content mismatch: %q", string(content))
	}
}

func TestNewPlanWithFilename(t *testing.T) {
	cfg, _ := config.Init()
	pkg := recipe.PackageDefinition{
		Name:     "testpkg",
		Version:  "1.0.0",
		OS:       config.OSLinux,
		Arch:     config.ArchX64,
		URL:      "https://example.com/redirect",
		Filename: "explicit.tar.gz",
	}

	plan, err := NewPlan(cfg, pkg)
	if err != nil {
		t.Fatal(err)
	}

	expected := filepath.Join(cfg.DownloadDir, "explicit.tar.gz")
	if plan.DownloadPath != expected {
		t.Errorf("Expected DownloadPath %s, got %s", expected, plan.DownloadPath)
	}
}
