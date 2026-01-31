package cache

import (
	"os"
)

// Ensure ensures that the target path exists by running fn if it doesn't.
// It uses locking to prevent multiple processes from running fn for the same target.
func Ensure(target string, fn func() error) error {
	// 1. Quick check if already exists
	if _, err := os.Stat(target); err == nil {
		return nil
	}

	// 2. Acquire lock. This will wait if another process is working on it.
	unlock, err := Lock(target)
	if err != nil {
		return err
	}
	defer unlock()

	// 3. Re-check if exists (it might have been created while we waited for the lock)
	if _, err := os.Stat(target); err == nil {
		return nil
	}

	// 4. Run the function to create the target
	return fn()
}
