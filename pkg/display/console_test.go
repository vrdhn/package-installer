package display

import (
	"bytes"
	"strings"
	"testing"
)

func TestConsoleDisplay(t *testing.T) {
	buf := &bytes.Buffer{}
	d := NewWriterDisplay(buf)
	d.SetVerbose(true)

	task := d.StartTask("TestTask")

	// Check initial output
	output := buf.String()
	if !strings.Contains(output, "[TestTask]") {
		t.Errorf("Expected output to contain task name, got: %q", output)
	}

	buf.Reset()
	task.SetStage("Download", "/tmp/file")
	task.Progress(50, "Working")
	// Output should contain move up + clear line + new status
	output = buf.String()
	// \033[1A\033[2K
	if !strings.Contains(output, "\x1b[1A\x1b[2K") {
		t.Errorf("Expected ANSI clear codes, got: %q", output)
	}
	if !strings.Contains(output, "Download") {
		t.Errorf("Expected Download stage, got: %q", output)
	}
	if !strings.Contains(output, "50%") {
		t.Errorf("Expected 50%%, got: %q", output)
	}

	buf.Reset()
	task.Log("Hello")
	output = buf.String()
	// Should clear lines, print log, reprint task
	if !strings.Contains(output, "Hello") {
		t.Errorf("Expected log message, got: %q", output)
	}
	if !strings.Contains(output, "50%") {
		t.Errorf("Expected task reprint, got: %q", output)
	}

	buf.Reset()
	task.Done()
	output = buf.String()
	// Should clear lines, log Done
	if !strings.Contains(output, "Done") {
		t.Errorf("Expected Done message, got: %q", output)
	}

	// Verify closing
	d.Close()
}
