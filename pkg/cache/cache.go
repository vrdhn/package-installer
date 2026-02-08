// Package cache provides a simple filesystem cache with locking and TTL support.
// It ensures that concurrent processes do not perform redundant work for the same resource.
package cache

import (
	"os"
	"time"
)

// IsFresh checks if the target path exists and was modified within the specified duration (TTL).
// A TTL of 0 means the file is considered fresh as long as it exists.
func IsFresh(target string, ttl time.Duration) bool {
	info, err := os.Stat(target)
	if err != nil {
		return false
	}
	if ttl == 0 {
		return true
	}
	return time.Since(info.ModTime()) < ttl
}

// Ensure ensures that the target path exists by running fn if it doesn't.
// It uses locking to prevent multiple processes from running fn for the same target.
func Ensure(target string, fn func() error) error {
	return EnsureWithTTL(target, 0, fn)
}

// EnsureWithTTL ensures that the target path exists and is not older than ttl by running fn if it doesn't.
// A ttl of 0 means the file never expires.
func EnsureWithTTL(target string, ttl time.Duration, fn func() error) error {
	// 1. Quick check if already exists and is fresh
	if IsFresh(target, ttl) {
		return nil
	}

	// 2. Acquire lock. This will wait if another process is working on it.
	unlock, err := Lock(target)
	if err != nil {
		return err
	}
	defer unlock()

	// 3. Re-check if exists and is fresh (it might have been updated while we waited for the lock)
	if IsFresh(target, ttl) {
		return nil
	}

	// 4. Run the function to create the target
	return fn()
}
