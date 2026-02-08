// Package display implementation for terminal-based output.
package display

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
)

// consoleDisplay implements the Display interface for standard terminal output.
type consoleDisplay struct {
	mu      sync.Mutex
	out     io.Writer
	verbose bool
}

// NewConsole creates a Display that writes to standard error.
func NewConsole() Display {
	return &consoleDisplay{
		out: os.Stderr,
	}
}

// NewWriterDisplay creates a Display that writes to the provided io.Writer.
func NewWriterDisplay(w io.Writer) Display {
	return &consoleDisplay{
		out: w,
	}
}

// StartTask creates a new console-based task tracker.
func (d *consoleDisplay) StartTask(name string) Task {
	return &consoleTask{
		name: name,
		disp: d,
	}
}

// Log writes a message to slog at Debug level.
func (d *consoleDisplay) Log(msg string) {
	slog.Debug(msg)
}

// Print writes a message directly to the output writer.
func (d *consoleDisplay) Print(msg string) {
	d.mu.Lock()
	out := d.out
	d.mu.Unlock()
	fmt.Fprint(out, msg)
}

// SetVerbose toggles verbose output mode.
func (d *consoleDisplay) SetVerbose(v bool) {
	d.mu.Lock()
	d.verbose = v
	d.mu.Unlock()
}

// Close is a no-op for the console display.
func (d *consoleDisplay) Close() {
	// no-op
}

// consoleTask implements the Task interface for terminal tracking.
type consoleTask struct {
	name   string
	disp   *consoleDisplay
	stage  string
	target string
}

// Log writes a task-specific debug message.
func (t *consoleTask) Log(msg string) {
	slog.Debug(msg, "task", t.name)
}

// SetStage records and logs a new processing stage for the task.
func (t *consoleTask) SetStage(name string, target string) {
	t.stage = name
	t.target = target
	slog.Debug("task stage", "task", t.name, "stage", name, "target", target)
}

// Progress logs the numerical progress of the task.
func (t *consoleTask) Progress(percent int, message string) {
	slog.Debug("task progress", "task", t.name, "percent", percent, "message", message)
}

// Done logs task completion.
func (t *consoleTask) Done() {
	slog.Debug("task done", "task", t.name)
}
