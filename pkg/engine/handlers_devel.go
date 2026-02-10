package engine

import (
	"pi/pkg/cdl"
	"pi/pkg/devel"
)

// RunDevelTest executes a recipe test for a specific package in a temporary work directory.
func (h *Handlers) RunDevelTest(params *cdl.DevelTestParams) (ExecutionResult, error) {
	out, err := devel.Test(h.Ctx, params.File, params.Package)
	if err != nil {
		return ExecutionResult{}, err
	}

	return ExecutionResult{
		ExitCode: 0,
		Output:   out,
	}, nil
}
