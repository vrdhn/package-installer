package cache

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Lock attempts to lock the given path (file or folder) by creating a .lock file.
// If the lock exists, it checks if the PID in the lock file is still alive.
// If the process is alive, it waits.
// If the process is dead, it cleans up the stale lock and acquires it.
// Returns an unlock function and an error if any (other than waiting).
func Lock(target string) (func() error, error) {
	lockFile := target + ".lock"

	// Ensure parent dir exists
	if err := os.MkdirAll(filepath.Dir(lockFile), 0755); err != nil {
		return nil, fmt.Errorf("failed to create parent dir for lock: %w", err)
	}

	for {
		// 1. Try to create the lock file exclusively
		f, err := os.OpenFile(lockFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err == nil {
			// Success! Write info.
			pid := os.Getpid()
			ts := time.Now().Format(time.RFC3339)
			content := fmt.Sprintf("%s %d", ts, pid)
			if _, err := f.WriteString(content); err != nil {
				f.Close()
				os.Remove(lockFile) // cleanup
				return nil, fmt.Errorf("failed to write to lock file: %w", err)
			}
			f.Close()

			// Return unlocker
			return func() error {
				return os.Remove(lockFile)
			}, nil
		}

		if !os.IsExist(err) {
			// Unexpected error
			return nil, fmt.Errorf("failed to acquire lock: %w", err)
		}

		// 2. Lock file exists. Check if stale.
		content, err := os.ReadFile(lockFile)
		if err != nil {
			if os.IsNotExist(err) {
				// Lock disappeared, retry immediately
				continue
			}
			// Access error? Wait a bit and retry
			time.Sleep(100 * time.Millisecond)
			continue
		}

		parts := strings.Split(strings.TrimSpace(string(content)), " ")
		if len(parts) < 2 {
			// Invalid format. Assume stale? Or corrupted.
			// Let's treat as stale to be safe/robust, or wait?
			// If we assume stale, we might race. But an empty/corrupt lock file is bad.
			// Let's remove it.
			os.Remove(lockFile)
			continue
		}

		pidStr := parts[len(parts)-1] // Last part is PID
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			// Invalid PID. Remove.
			os.Remove(lockFile)
			continue
		}

		if isPidAlive(pid) {
			// Process is alive. Wait.
			time.Sleep(200 * time.Millisecond)
			continue
		}

		// Process is dead. Remove stale lock.
		// Note: os.Remove might fail if someone else removed it. That's fine.
		os.Remove(lockFile)
		// Loop back to try creating it.
	}
}

func isPidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		// Should not happen on Unix
		return false
	}

	// Send signal 0 to check existence
	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}

	if errors.Is(err, syscall.ESRCH) || errors.Is(err, os.ErrProcessDone) {
		return false
	}

	// EPERM or other error: process exists but we can't signal it.
	// Assume alive.
	return true
}
