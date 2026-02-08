package lazyjson

import "os"

// Option is a functional option for configuring a Manager.
type Option[T any] func(*options[T])

// WithIndent sets the indentation string for JSON output.
// Use "" for compact JSON (no indentation).
// Default is "  " (two spaces).
func WithIndent[T any](indent string) Option[T] {
	return func(o *options[T]) {
		o.indent = indent
	}
}

// WithFileMode sets the file permissions for the JSON file.
// Default is 0644.
func WithFileMode[T any](mode os.FileMode) Option[T] {
	return func(o *options[T]) {
		o.fileMode = mode
	}
}

// WithCreateIfMissing controls whether to create a new file with zero value
// if the file doesn't exist on first load.
// Default is true.
func WithCreateIfMissing[T any](create bool) Option[T] {
	return func(o *options[T]) {
		o.createIfMissing = create
	}
}

// WithDefaultValue provides a function that returns a default value
// to use when the file doesn't exist and createIfMissing is true.
// If not provided, the zero value of type T is used.
func WithDefaultValue[T any](fn func() *T) Option[T] {
	return func(o *options[T]) {
		o.defaultValue = fn
	}
}

// WithCompactJSON disables indentation for compact JSON output.
// This is equivalent to WithIndent("").
func WithCompactJSON[T any]() Option[T] {
	return WithIndent[T]("")
}
