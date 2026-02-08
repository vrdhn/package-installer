package engine

import (
	"pi/pkg/bubblewrap"
	"pi/pkg/cdl"
)

func (h *Handlers) RunCaveInfo(params *cdl.CaveInfoParams) (ExecutionResult, error) {
	res, err := h.CaveMgr.Info(h.Ctx)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *Handlers) RunCaveList(params *cdl.CaveListParams) (ExecutionResult, error) {
	res, err := h.CaveMgr.List(h.Ctx, h.DispMgr)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *Handlers) RunCaveUse(params *cdl.CaveUseParams) (ExecutionResult, error) {
	res, err := h.CaveMgr.Use(h.Ctx, bubblewrap.Create(), h.PkgsMgr, params.Cave)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *Handlers) RunCaveRun(params *cdl.CaveRunParams) (ExecutionResult, error) {
	res, err := h.CaveMgr.RunCommand(h.Ctx, bubblewrap.Create(), h.PkgsMgr, params.Variant, params.Command)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *Handlers) RunCaveEnter(params *cdl.CaveEnterParams) (ExecutionResult, error) {
	res, err := h.CaveMgr.RunCommand(h.Ctx, bubblewrap.Create(), h.PkgsMgr, "", "")
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *Handlers) RunCaveInit(params *cdl.CaveInitParams) (ExecutionResult, error) {
	res, err := h.CaveMgr.Init(h.Ctx)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *Handlers) RunCaveSync(params *cdl.CaveSyncParams) (ExecutionResult, error) {
	res, err := h.CaveMgr.Sync(h.Ctx, h.PkgsMgr)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

func (h *Handlers) RunCaveAddpkg(params *cdl.CaveAddpkgParams) (ExecutionResult, error) {
	res, err := h.CaveMgr.AddPkg(h.Ctx, params.Package)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}
