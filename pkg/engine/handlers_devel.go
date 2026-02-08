package engine

import (
	"pi/pkg/cdl"
	"pi/pkg/devel/repl"
)

func (h *Handlers) RunDevelRepl(params *cdl.DevelReplParams) (ExecutionResult, error) {
	err := repl.Run(h.Ctx, h.Config, params.File)
	if err != nil {
		return ExecutionResult{}, err
	}
	return ExecutionResult{ExitCode: 0}, nil
}
