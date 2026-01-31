package recipe

import (
	"testing"
)

func TestParseNodejs(t *testing.T) {
	data := []byte(`[
		{"version":"v20.11.1","date":"2024-02-14","files":["linux-x64","linux-arm64","win-x64-zip"]},
		{"version":"v21.6.2","date":"2024-02-07","files":["osx-arm64-tar"]}
	]`)

	pkgs, err := parseNodejs(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(pkgs) != 4 {
		t.Errorf("Expected 4 packages, got %d", len(pkgs))
	}

	found := false
	for _, p := range pkgs {
		if p.Version == "20.11.1" && p.OS == "linux" && p.Arch == "x64" {
			found = true
			expectedURL := "https://nodejs.org/dist/v20.11.1/node-v20.11.1-linux-x64.tar.gz"
			if p.URL != expectedURL {
				t.Errorf("URL mismatch: expected %s, got %s", expectedURL, p.URL)
			}
			if p.Filename != "node-v20.11.1-linux-x64.tar.gz" {
				t.Errorf("Filename mismatch: got %s", p.Filename)
			}
		}
	}
	if !found {
		t.Error("Did not find expected linux-x64 package for nodejs")
	}
}

func TestParseJava(t *testing.T) {
	data := []byte(`{
		"result": [
			{
				"filename": "zulu17.48.15-ca-jdk17.0.10-linux_x64.tar.gz",
				"java_version": "17.0.10",
				"distribution": "zulu",
				"operating_system": "linux",
				"architecture": "x64",
				"links": {
					"pkg_download_redirect": "https://api.foojay.io/disco/v3.0/ids/some-id/redirect"
				}
			}
		]
	}`)

	pkgs, err := parseFoojay(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(pkgs) != 1 {
		t.Errorf("Expected 1 package, got %d", len(pkgs))
	}

	p := pkgs[0]
	if p.Name != "java-zulu" || p.Version != "17.0.10" || p.OS != "linux" || p.Arch != "x64" {
		t.Errorf("Package data mismatch: %+v", p)
	}
	if p.Filename != "zulu17.48.15-ca-jdk17.0.10-linux_x64.tar.gz" {
		t.Errorf("Filename mismatch: got %s", p.Filename)
	}
}
