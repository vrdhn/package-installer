// Package display defines the interfaces for user feedback and progress tracking.
// It supports both simple logging and multi-threaded task progress visualization.
package display

// Task represents a unit of work that can be monitored for progress and status.
// Tasks are typically used for long-running operations like downloads or extractions.
type Task interface {
	// Log adds a log message associated with this specific task.
	Log(msg string)
	// SetStage updates the current stage of the task (e.g., "Download", "Extract")
	// and optionally identifies the target file or component being worked on.
	SetStage(name string, target string)
	// Progress updates the completion percentage (0-100) and provides a status message.
	Progress(percent int, message string)
	// Done marks the task as completed, allowing the display to clean up its resources.
	Done()
}

// Display handles the visualization of global logs, command output, and tracked tasks.
// It serves as the primary output coordinator for the application.
type Display interface {
	// StartTask creates and returns a new tracked Task for monitoring progress.
	StartTask(name string) Task
	// Log adds a direct log message to the display (typically used for debug or info levels).
	Log(msg string)
	// Print adds a primary output message, such as search results or tables, to the display.
	Print(msg string)
	// SetVerbose enables or disables high-verbosity output modes.
	SetVerbose(v bool)
	// Close cleans up display resources and ensures all final output is flushed to the user.
	Close()
}
