package engine

import "pi/pkg/common"

// ExecutionResult is returned at top level, to figure out if to execute bwrap
// or just exit with error code. The curses library should have
// cleaned up when this is returned, so exec is safe.
type ExecutionResult = common.ExecutionResult
