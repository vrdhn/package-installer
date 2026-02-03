package display

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	subtleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	infoStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	doneStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
)

// teaTask represents a single task in the Bubble Tea model
type teaTask struct {
	id      string
	name    string
	stage   string
	target  string
	percent float64
	status  string
	prog    progress.Model
}

// model holds the state for the Bubble Tea program
type model struct {
	tasks []teaTask
	logs  []string
	mu    sync.Mutex
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case taskUpdateMsg:
		for i, t := range m.tasks {
			if t.id == msg.id {
				m.tasks[i].percent = msg.percent
				m.tasks[i].status = msg.status
				m.tasks[i].stage = msg.stage
				m.tasks[i].target = msg.target
				return m, nil
			}
		}
	case logMsg:
		m.logs = append(m.logs, string(msg))
		if len(m.logs) > 10 { // Keep only last 10 logs for display
			m.logs = m.logs[len(m.logs)-10:]
		}
	case taskDoneMsg:
		for i, t := range m.tasks {
			if t.id == string(msg) {
				m.tasks = append(m.tasks[:i], m.tasks[i+1:]...)
				break
			}
		}
	case newTaskMsg:
		m.tasks = append(m.tasks, teaTask{
			id:   msg.id,
			name: msg.name,
			prog: progress.New(progress.WithDefaultGradient()),
		})
	}
	return m, nil
}

func (m *model) View() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	var s string

	// Render logs
	for _, l := range m.logs {
		s += subtleStyle.Render(l) + "\n"
	}

	if len(m.tasks) > 0 {
		s += "\n"
	}

	// Render tasks
	for _, t := range m.tasks {
		name := infoStyle.Render(t.name)
		if t.stage != "" {
			name += subtleStyle.Render(":" + t.stage)
		}

		s += fmt.Sprintf("%-30s %s %s\n", name, t.prog.ViewAs(t.percent), t.status)
	}

	return s
}

type taskUpdateMsg struct {
	id      string
	percent float64
	status  string
	stage   string
	target  string
}
type logMsg string
type taskDoneMsg string
type newTaskMsg struct {
	id   string
	name string
}

// consoleDisplay implements Display using Bubble Tea
type consoleDisplay struct {
	p       *tea.Program
	model   *model
	verbose bool
	wg      sync.WaitGroup
}

func NewConsole() Display {
	m := &model{}
	p := tea.NewProgram(m)
	d := &consoleDisplay{
		p:     p,
		model: m,
	}

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running display: %v\n", err)
		}
	}()

	return d
}

func (d *consoleDisplay) StartTask(name string) Task {
	id := fmt.Sprintf("%p", &name) // Simple unique ID
	d.p.Send(newTaskMsg{id: id, name: name})
	return &consoleTask{id: id, name: name, disp: d}
}

func (d *consoleDisplay) Log(msg string) {
	if d.verbose {
		d.p.Send(logMsg(msg))
	}
}

func (d *consoleDisplay) SetVerbose(v bool) {
	d.verbose = v
}

func (d *consoleDisplay) Close() {
	if d.p != nil {
		d.p.Quit()
		d.wg.Wait()
		d.p = nil
	}
}

// consoleTask implements Task
type consoleTask struct {
	id      string
	name    string
	stage   string
	target  string
	percent float64
	status  string
	disp    *consoleDisplay
}

func (t *consoleTask) Log(msg string) {
	t.disp.Log(fmt.Sprintf("[%s] %s", t.name, msg))
}

func (t *consoleTask) SetStage(name string, target string) {
	t.stage = name
	t.target = target
	t.update()
}

func (t *consoleTask) Progress(percent int, message string) {
	t.percent = float64(percent) / 100.0
	t.status = message
	t.update()
}

func (t *consoleTask) update() {
	t.disp.p.Send(taskUpdateMsg{
		id:      t.id,
		percent: t.percent,
		status:  t.status,
		stage:   t.stage,
		target:  t.target,
	})
}

func (t *consoleTask) Done() {
	t.disp.Log(doneStyle.Render(fmt.Sprintf("[%s] Completed", t.name)))
	t.disp.p.Send(taskDoneMsg(t.id))
}

// NewWriterDisplay not supported well with interactive Bubble Tea,
// but we keep it for API compatibility, just using a no-op or simple logger.
func NewWriterDisplay(w io.Writer) Display {
	return NewConsole() // Fallback
}
