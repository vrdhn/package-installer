package display

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Ensure interfaces are implemented
var _ Display = &consoleDisplay{}
var _ Task = &consoleTask{}

type consoleTask struct {
	name    string
	stage   string
	target  string
	percent int
	status  string
	disp    *consoleDisplay
}

type consoleDisplay struct {
	mu          sync.Mutex
	out         io.Writer
	activeTasks []*consoleTask
	lastLines   int
}

// NewConsole creates a new Display that outputs to stdout.
func NewConsole() Display {
	return &consoleDisplay{
		out: os.Stdout,
	}
}

// NewWriterDisplay creates a new Display that outputs to the given writer.
func NewWriterDisplay(w io.Writer) Display {
	return &consoleDisplay{
		out: w,
	}
}

func (d *consoleDisplay) StartTask(name string) Task {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.clearLines()
	t := &consoleTask{
		name: name,
		disp: d,
	}
	d.activeTasks = append(d.activeTasks, t)
	d.render()
	return t
}

func (d *consoleDisplay) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.clearLines()
	d.activeTasks = nil
}

func (d *consoleDisplay) log(msg string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.clearLines()
	fmt.Fprintln(d.out, msg)
	d.render()
}

func (d *consoleDisplay) update() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.clearLines()
	d.render()
}

func (d *consoleDisplay) remove(t *consoleTask) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.clearLines()

	// Filter out the task
	newTasks := make([]*consoleTask, 0, len(d.activeTasks))
	for _, task := range d.activeTasks {
		if task != t {
			newTasks = append(newTasks, task)
		}
	}
	d.activeTasks = newTasks
	d.render()
}

// Internal helper to clear previously rendered lines.
// Must be called with lock held.
func (d *consoleDisplay) clearLines() {
	for i := 0; i < d.lastLines; i++ {
		fmt.Fprint(d.out, "\033[1A\033[2K")
	}
	d.lastLines = 0
}

// Internal helper to render active tasks.
// Must be called with lock held.
func (d *consoleDisplay) render() {
	// Re-print active tasks
	for _, t := range d.activeTasks {
		bar := drawBar(t.percent)
		fullTag := t.name
		if t.stage != "" {
			fullTag += ":" + t.stage
		}
		if t.target != "" {
			fullTag += ":" + filepath.Base(t.target)
		}
		line := fmt.Sprintf("[%s] %s %s (%d%%)", fullTag, bar, t.status, t.percent)
		fmt.Fprintln(d.out, line)
	}
	d.lastLines = len(d.activeTasks)
}

func drawBar(percent int) string {
	width := 20
	completed := (percent * width) / 100
	if completed > width {
		completed = width
	}
	if completed < 0 {
		completed = 0
	}
	return fmt.Sprintf("[%s%s]",
		strings.Repeat("=", completed),
		strings.Repeat(" ", width-completed))
}

func (t *consoleTask) Log(msg string) {
	t.disp.log(fmt.Sprintf("[%s] %s", t.name, msg))
}

func (t *consoleTask) SetStage(name string, target string) {
	t.stage = name
	t.target = target
	t.disp.update()
}

func (t *consoleTask) Progress(percent int, message string) {
	t.percent = percent
	t.status = message
	t.disp.update()
}

func (t *consoleTask) Done() {
	// Log completion and remove from active list
	t.disp.log(fmt.Sprintf("[%s] Done", t.name))
	t.disp.remove(t)
}
