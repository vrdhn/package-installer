package engine

import (
	"pi/pkg/bubblewrap"
	"pi/pkg/cdl"
)

// RunCaveInfo displays information about the current active cave.
func (h *Handlers) RunCaveInfo(params *cdl.CaveInfoParams) (ExecutionResult, error) {
	res, err := h.CaveMgr.Info(h.Ctx)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

// RunCaveList lists all registered caves and their variants.
func (h *Handlers) RunCaveList(params *cdl.CaveListParams) (ExecutionResult, error) {
	res, err := h.CaveMgr.List(h.Ctx, h.DispMgr)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

// RunCaveUse starts a cave session by name.
func (h *Handlers) RunCaveUse(params *cdl.CaveUseParams) (ExecutionResult, error) {
	res, err := h.CaveMgr.Use(h.Ctx, bubblewrap.Create(), h.PkgsMgr, params.Cave)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

// RunCaveRun executes a command inside a specific cave.
func (h *Handlers) RunCaveRun(params *cdl.CaveRunParams) (ExecutionResult, error) {
	res, err := h.CaveMgr.RunCommand(h.Ctx, bubblewrap.Create(), h.PkgsMgr, params.Variant, params.Command)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

// RunCaveEnter enters the interactive sandbox shell.
func (h *Handlers) RunCaveEnter(params *cdl.CaveEnterParams) (ExecutionResult, error) {
	res, err := h.CaveMgr.RunCommand(h.Ctx, bubblewrap.Create(), h.PkgsMgr, "", "")
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

// RunCaveInit initializes a new workspace directory with a pi.cave.json.
func (h *Handlers) RunCaveInit(params *cdl.CaveInitParams) (ExecutionResult, error) {
	res, err := h.CaveMgr.Init(h.Ctx)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

// RunCaveSync synchronizes all packages defined in the workspace's cave configuration.
func (h *Handlers) RunCaveSync(params *cdl.CaveSyncParams) (ExecutionResult, error) {
	res, err := h.CaveMgr.Sync(h.Ctx, h.PkgsMgr)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

// RunCaveAddpkg adds a new package requirement to the current cave configuration.
func (h *Handlers) RunCaveAddpkg(params *cdl.CaveAddpkgParams) (ExecutionResult, error) {
	res, err := h.CaveMgr.AddPkg(h.Ctx, params.Package)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}
