package cli

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pi/pkg/config"
	"pi/pkg/recipe"
)

func TestSelectRegex(t *testing.T) {
	patterns := []string{"nodejs", "npm:.*", "^go$"}

	regex, err := selectRegex(patterns, "nodejs", "")
	if err != nil {
		t.Fatalf("selectRegex error: %v", err)
	}
	if regex != "nodejs" {
		t.Fatalf("expected nodejs, got %s", regex)
	}

	regex, err = selectRegex(patterns, "npm:typescript", "")
	if err != nil {
		t.Fatalf("selectRegex error: %v", err)
	}
	if regex != "npm:.*" {
		t.Fatalf("expected npm:.*, got %s", regex)
	}

	regex, err = selectRegex(patterns, "go", "")
	if err != nil {
		t.Fatalf("selectRegex error: %v", err)
	}
	if regex != "^go$" {
		t.Fatalf("expected ^go$, got %s", regex)
	}

	_, err = selectRegex([]string{"go", "g.*"}, "go", "")
	if err == nil {
		t.Fatalf("expected error for multiple matches")
	}
}

func TestSummarizeRecipeMatches(t *testing.T) {
	pkgs := []recipe.PackageDefinition{
		{Name: "demo", Version: "1.0.0"},
		{Name: "demo", Version: "1.1.0"},
		{Name: "other", Version: "2.0.0"},
	}
	stats := summarizeRecipeMatches("demo", "demo", "demo", pkgs)
	if stats.Total != 3 {
		t.Fatalf("expected total 3, got %d", stats.Total)
	}
	if stats.NameMatches != 2 {
		t.Fatalf("expected name matches 2, got %d", stats.NameMatches)
	}
	if stats.RegexMatchesName != 2 {
		t.Fatalf("expected regex matches 2, got %d", stats.RegexMatchesName)
	}
	if !stats.RegexMatchesQuery {
		t.Fatalf("expected regex matches query")
	}
}

func TestRecipeReplSmoke(t *testing.T) {
	tmpDir := t.TempDir()
	recipePath := filepath.Join(tmpDir, "demo.star")
	source := strings.TrimSpace(`
def handler(pkg_name):
    add_version(
        name = "demo",
        version = "1.0.0",
        release_status = "stable",
        release_date = "",
        os = "linux",
        arch = "x64",
        url = "https://example.com/demo.tar.gz",
        filename = "demo.tar.gz",
        checksum = "",
        env = {},
        symlinks = {}
    )

add_pkgdef("demo", handler)
`)
	if err := os.WriteFile(recipePath, []byte(source), 0644); err != nil {
		t.Fatalf("write recipe: %v", err)
	}

	cfg, err := config.Init()
	if err != nil {
		t.Fatalf("config init: %v", err)
	}

	input := bytes.NewBufferString("run demo\nexit\n")
	var output bytes.Buffer

	repl := &recipeRepl{
		in:   bufio.NewReader(input),
		out:  &output,
		err:  &output,
		cfg:  cfg,
		path: recipePath,
	}

	if err := repl.Run(context.Background()); err != nil {
		t.Fatalf("repl run: %v", err)
	}

	if !strings.Contains(output.String(), "Versions returned: 1") {
		t.Fatalf("expected version output, got: %s", output.String())
	}
}
