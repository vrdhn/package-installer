package display

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
)

type consoleDisplay struct {
	mu      sync.Mutex
	out     io.Writer
	verbose bool
}

func NewConsole() Display {
	return &consoleDisplay{
		out: os.Stderr,
	}
}

func NewWriterDisplay(w io.Writer) Display {
	return &consoleDisplay{
		out: w,
	}
}

func (d *consoleDisplay) StartTask(name string) Task {
	return &consoleTask{
		name: name,
		disp: d,
	}
}

func (d *consoleDisplay) Log(msg string) {
	slog.Debug(msg)
}

func (d *consoleDisplay) Print(msg string) {
	d.mu.Lock()
	out := d.out
	d.mu.Unlock()
	fmt.Fprint(out, msg)
}

func (d *consoleDisplay) SetVerbose(v bool) {
	d.mu.Lock()
	d.verbose = v
	d.mu.Unlock()
	// We don't strictly need to do anything here if slog is already configured,
	// but keeping it for compatibility with the interface if needed elsewhere.
}

func (d *consoleDisplay) Close() {
	// no-op
}

type consoleTask struct {
	name   string
	disp   *consoleDisplay
	stage  string
	target string
}

func (t *consoleTask) Log(msg string) {
	slog.Debug(msg, "task", t.name)
}

func (t *consoleTask) SetStage(name string, target string) {
	t.stage = name
	t.target = target
	slog.Debug("task stage", "task", t.name, "stage", name, "target", target)
}

func (t *consoleTask) Progress(percent int, message string) {
	slog.Debug("task progress", "task", t.name, "percent", percent, "message", message)
}

func (t *consoleTask) Done() {
	slog.Debug("task done", "task", t.name)
}
