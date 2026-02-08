package engine

import (
	"pi/pkg/cdl"
)

func (h *Handlers) RunRepoList(params *cdl.RepoListParams) (ExecutionResult, error) {
	res, err := h.RepoMgr.List(params.Verbose)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *Handlers) RunRepoAdd(params *cdl.RepoAddParams) (ExecutionResult, error) {
	res, err := h.RepoMgr.Add(params.Path, params.Verbose)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *Handlers) RunRepoSync(params *cdl.RepoSyncParams) (ExecutionResult, error) {
	res, err := h.RepoMgr.SyncRepo(params.Verbose)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}
