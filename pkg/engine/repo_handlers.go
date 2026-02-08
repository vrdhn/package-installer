package engine

import (
	"pi/pkg/cdl"
)

// RunRepoList lists all registered repositories and their indexed patterns.
func (h *Handlers) RunRepoList(params *cdl.RepoListParams) (ExecutionResult, error) {
	res, err := h.RepoMgr.List(params.Verbose)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

// RunRepoAdd adds a new repository from a local path.
func (h *Handlers) RunRepoAdd(params *cdl.RepoAddParams) (ExecutionResult, error) {
	res, err := h.RepoMgr.Add(params.Path, params.Verbose)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

// RunRepoSync regenerates the package index from all registered repositories.
func (h *Handlers) RunRepoSync(params *cdl.RepoSyncParams) (ExecutionResult, error) {
	res, err := h.RepoMgr.SyncRepo(params.Verbose)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}
