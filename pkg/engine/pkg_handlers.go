package engine

import (
	"pi/pkg/cdl"
)

// RunPkgSync synchronizes package versions from repositories based on the provided query.
func (h *Handlers) RunPkgSync(params *cdl.PkgSyncParams) (ExecutionResult, error) {
	res, err := h.PkgsMgr.SyncPkgs(h.Ctx, params.Query)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

// RunPkgList lists available versions for a package matching the query.
func (h *Handlers) RunPkgList(params *cdl.PkgListParams) (ExecutionResult, error) {
	res, err := h.PkgsMgr.ListPkgs(h.Ctx, params.Query, params.All)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}
