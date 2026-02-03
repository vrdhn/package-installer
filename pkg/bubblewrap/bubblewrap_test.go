package bubblewrap

import (
	"os"
	"strings"
	"testing"
)

func TestAddEnvFirst(t *testing.T) {
	// Setup env for test
	os.Setenv("TEST_VAR", "a:b:c")
	defer os.Unsetenv("TEST_VAR")

	b := Create()

	// Test 1: Add to existing env
	b.AddEnvFirst("TEST_VAR", "d")
	if b.envs["TEST_VAR"] != "d:a:b:c" {
		t.Errorf("Expected d:a:b:c, got %s", b.envs["TEST_VAR"])
	}

	// Test 2: Add existing entry
	b.AddEnvFirst("TEST_VAR", "b")
	if b.envs["TEST_VAR"] != "b:d:a:c" {
		t.Errorf("Expected b:d:a:c, got %s", b.envs["TEST_VAR"])
	}

	// Test 3: Multiple calls same entry
	b.AddEnvFirst("TEST_VAR", "b")
	if b.envs["TEST_VAR"] != "b:d:a:c" {
		t.Errorf("Expected b:d:a:c, got %s", b.envs["TEST_VAR"])
	}

	// Test 4: New variable (not in host env)
	b.AddEnvFirst("NEW_VAR", "foo")
	if b.envs["NEW_VAR"] != "foo" {
		t.Errorf("Expected foo, got %s", b.envs["NEW_VAR"])
	}
}

func TestCmdGeneration(t *testing.T) {
	b := Create()
	b.SetCommand("/bin/bash", "-c", "echo hello")
	b.AddBind(BIND_RO, "/etc")
	b.AddEnvFirst("PATH", "/custom/bin")

	cmd := b.Cmd()

	if cmd.Path != "/usr/bin/bwrap" {
		t.Errorf("Expected command /usr/bin/bwrap, got %s", cmd.Path)
	}

	// Check arguments
	args := strings.Join(cmd.Args, " ")
	// Note: cmd.Args[0] is usually the command name itself

	expectedSubstrings := []string{
		"--argv0 /bin/bash",
		"--ro-bind /etc /etc",
		"-- /bin/bash -c echo hello",
	}

	for _, sub := range expectedSubstrings {
		if !strings.Contains(args, sub) {
			t.Errorf("Expected args to contain '%s', got: %s", sub, args)
		}
	}

	// Check Env
	foundPath := false
	for _, e := range cmd.Env {
		if strings.HasPrefix(e, "PATH=") {
			if strings.HasPrefix(e, "PATH=/custom/bin") {
				foundPath = true
			}
		}
	}
	if !foundPath {
		t.Errorf("Expected PATH to start with /custom/bin in envs")
	}
}

func TestSpawn(t *testing.T) {
	if _, err := os.Stat("/usr/bin/bwrap"); os.IsNotExist(err) {
		t.Skip("bwrap not found, skipping integration test")
	}

	b := Create()
	// Minimal bwrap command
	// We need to bind /usr, /lib, /bin etc for ls to work
	b.AddBind(BIND_RO, "/usr")
	b.AddBind(BIND_RO, "/lib")
	if _, err := os.Stat("/lib64"); err == nil {
		b.AddBind(BIND_RO, "/lib64")
	}
	b.AddBind(BIND_RO, "/bin")
	b.SetCommand("/bin/ls", "/")

	err := b.Spawn()
	if err != nil {
		t.Errorf("Spawn failed: %v", err)
	}
}
