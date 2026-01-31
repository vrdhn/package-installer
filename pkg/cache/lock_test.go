package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestLockSimple(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "myfile")

	unlock, err := Lock(target)
	if err != nil {
		t.Fatalf("Failed to lock: %v", err)
	}

	// Verify lock file exists
	if _, err := os.Stat(target + ".lock"); os.IsNotExist(err) {
		t.Errorf("Lock file not created")
	}

	// Unlock
	if err := unlock(); err != nil {
		t.Errorf("Failed to unlock: %v", err)
	}

	// Verify lock file gone
	if _, err := os.Stat(target + ".lock"); !os.IsNotExist(err) {
		t.Errorf("Lock file should be gone")
	}
}

func TestLockStale(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "stale")
	lockFile := target + ".lock"

	// Find a dead PID
	var stalePid int
	for i := 32000; i < 60000; i++ {
		proc, _ := os.FindProcess(i)
		err := proc.Signal(syscall.Signal(0))
		if err == syscall.ESRCH {
			stalePid = i
			break
		}
	}
	if stalePid == 0 {
		// Fallback or skip if we can't find one?
		// Try a very large one
		stalePid = 9999999
	}

	content := fmt.Sprintf("%s %d", time.Now().Format(time.RFC3339), stalePid)
	if err := os.WriteFile(lockFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Try to lock. It should detect stale and overwrite.
	// Use a channel to detect timeout/hanging in test
	done := make(chan struct{})
	go func() {
		unlock, err := Lock(target)
		if err != nil {
			t.Errorf("Failed to acquire lock over stale one: %v", err)
			close(done)
			return
		}
		unlock()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatalf("Timed out waiting for lock acquisition - isPidAlive returned true for %d?", stalePid)
	}

	// Verify we are the owner now (implicit by success above, but check file?)
	// Not strictly needed if unlock() worked.
}

func TestLockConcurrent(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "concurrent")

	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine 1 grabs lock, holds it for a bit
	go func() {
		defer wg.Done()
		unlock, err := Lock(target)
		if err != nil {
			t.Errorf("G1 failed to lock: %v", err)
			return
		}
		time.Sleep(500 * time.Millisecond)
		unlock()
	}()

	// Goroutine 2 tries to grab lock, should wait
	go func() {
		defer wg.Done()
		time.Sleep(100 * time.Millisecond) // Ensure G1 starts first
		start := time.Now()
		unlock, err := Lock(target)
		if err != nil {
			t.Errorf("G2 failed to lock: %v", err)
			return
		}
		duration := time.Since(start)
		if duration < 300*time.Millisecond {
			t.Errorf("G2 acquired lock too fast (%v), expected waiting for G1", duration)
		}
		unlock()
	}()

	wg.Wait()
}

func TestEnsure(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "ensure_target")

	callCount := 0
	fn := func() error {
		callCount++
		time.Sleep(100 * time.Millisecond)
		return os.WriteFile(target, []byte("done"), 0644)
	}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := Ensure(target, fn); err != nil {
				t.Errorf("Ensure failed: %v", err)
			}
		}()
	}

	wg.Wait()

	if callCount != 1 {
		t.Errorf("Expected fn to be called once, got %d", callCount)
	}

	content, _ := os.ReadFile(target)
	if string(content) != "done" {
		t.Errorf("Expected content 'done', got %q", string(content))
	}
}
