// Package display implementation for terminal-based output.
package display

import (
	"fmt"
	"io"
	"os"
	"pi/pkg/common"
	"strings"
)

// consoleDisplay handles terminal output.
type consoleDisplay struct {
	out io.Writer
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

// Print writes a message directly to the output writer.
func (d *consoleDisplay) Print(msg string) {
	fmt.Fprint(d.out, msg)
}

// RenderOutput displays structured data from an Output struct to the console.
func (d *consoleDisplay) RenderOutput(out *common.Output) {
	if out == nil {
		return
	}

	if out.Message != "" {
		d.Print(fmt.Sprintln(out.Message))
	}

	if len(out.KV) > 0 {
		for _, kv := range out.KV {
			d.Print(fmt.Sprintf("%-12s %s\n", kv.Key+":", kv.Value))
		}
	}

	if out.Table != nil {
		d.renderTable(out.Table)
	}
}

func (d *consoleDisplay) renderTable(t *common.Table) {
	if len(t.Header) == 0 {
		return
	}

	// Simple column width calculation
	widths := make([]int, len(t.Header))
	for i, h := range t.Header {
		widths[i] = len(h)
	}
	for _, row := range t.Rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	var sb strings.Builder
	for i, h := range t.Header {
		fmt.Fprintf(&sb, "%-*s  ", widths[i], h)
	}
	d.Print(sb.String() + "\n")

	// Print separator
	totalWidth := 0
	for _, w := range widths {
		totalWidth += w + 2
	}
	d.Print(strings.Repeat("-", totalWidth) + "\n")

	// Print rows
	for _, row := range t.Rows {
		sb.Reset()
		for i, cell := range row {
			if i < len(widths) {
				fmt.Fprintf(&sb, "%-*s  ", widths[i], cell)
			}
		}
		d.Print(sb.String() + "\n")
	}
}
