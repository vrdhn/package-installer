package display

// Task represents a unit of work that can be monitored.
type Task interface {
	// Log adds a log message associated with this task.
	Log(msg string)
	// SetStage updates the current stage of the task (e.g. "Download", "Extract")
	// and the target file/folder being worked on.
	SetStage(name string, target string)
	// Progress updates the completion percentage (0-100) and status message.
	Progress(percent int, message string)
	// Done marks the task as completed and removes it from the display.
	// It is the responsibility of the caller who created the task via StartTask.
	Done()
}

// Display handles the visualization of tasks and logs.
type Display interface {
	// StartTask creates and returns a new tracked Task.
	StartTask(name string) Task
	// Log adds a direct log message to the display.
	Log(msg string)
	// Print adds a primary output message (e.g. table, info) to the display.
	Print(msg string)
	// SetVerbose enables or disables verbose logging.
	SetVerbose(v bool)
	// Close cleans up any resources and ensures final output is rendered.
	Close()
}
