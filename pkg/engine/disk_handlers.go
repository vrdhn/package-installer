package engine

import (
	"pi/pkg/cdl"
)

// RunDiskInfo displays disk usage summary for all pi-managed directories.
func (h *Handlers) RunDiskInfo(params *cdl.DiskInfoParams) (ExecutionResult, error) {
	res, err := h.DiskMgr.Info()
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

// RunDiskClean removes temporary and cached data from local storage.
func (h *Handlers) RunDiskClean(params *cdl.DiskCleanParams) (ExecutionResult, error) {
	res, err := h.DiskMgr.CleanDir()
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}

// RunDiskUninstall wipes all pi data (cache, state, and config) from the system.
func (h *Handlers) RunDiskUninstall(params *cdl.DiskUninstallParams) (ExecutionResult, error) {
	if !params.Force {
		h.DispMgr.Close()
	}
	res, err := h.DiskMgr.UninstallData(params.Force)
	if res == nil {
		return ExecutionResult{}, err
	}
	return *res, err
}
