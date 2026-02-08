package engine

import (
	"pi/pkg/cdl"
)

func (h *Handlers) RunPkgSync(params *cdl.PkgSyncParams) (ExecutionResult, error) {
	res, err := h.PkgsMgr.SyncPkgs(h.Ctx, params.Query)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *Handlers) RunPkgList(params *cdl.PkgListParams) (ExecutionResult, error) {
	res, err := h.PkgsMgr.ListPkgs(h.Ctx, params.Query, params.All)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}
