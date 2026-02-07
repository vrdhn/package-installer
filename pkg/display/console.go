package display

import (
	"fmt"
	"io"
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
	d.mu.Lock()
	verbose := d.verbose
	out := d.out
	d.mu.Unlock()

	if !verbose {
		return
	}

	fmt.Fprintf(out, "[log] %s\n", msg)
}

func (d *consoleDisplay) SetVerbose(v bool) {
	d.mu.Lock()
	d.verbose = v
	d.mu.Unlock()
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
	t.print("[%s] %s\n", t.name, msg)
}

func (t *consoleTask) SetStage(name string, target string) {
	t.stage = name
	t.target = target
	t.print("[%s] stage=%s target=%s\n", t.name, name, target)
}

func (t *consoleTask) Progress(percent int, message string) {
	t.print("[%s] progress=%d message=%s\n", t.name, percent, message)
}

func (t *consoleTask) Done() {
	t.print("[%s] done\n", t.name)
}

func (t *consoleTask) print(format string, args ...any) {
	t.disp.mu.Lock()
	defer t.disp.mu.Unlock()
	fmt.Fprintf(t.disp.out, format, args...)
}
