package common

// ExecutionResult is returned from operations.
type ExecutionResult struct {
	IsCave   bool
	ExitCode int

	// Cave Launch details
	Cwd  string
	Exe  string
	Args []string
	Env  []string
}
