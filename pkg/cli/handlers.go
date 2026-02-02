package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"pi/pkg/cave"
	"pi/pkg/cave_bwrap"
	"pi/pkg/config"
	"pi/pkg/display"
	"pi/pkg/pkgs"
	"pi/pkg/repository"
)

type DefaultHandler struct {
	Repo    *repository.Manager
	Disp    display.Display
	CaveMgr *cave.Manager
	PkgsMgr *pkgs.Manager
	SysCfg  *config.Config
}

func (h *DefaultHandler) Execute(ctx context.Context, inv *Invocation) (*ExecutionResult, error) {
	if v, ok := inv.Global["verbose"].(bool); ok {
		h.Disp.SetVerbose(v)
	}

	path := getCmdPath(inv.Command)
	switch path {
	case "install":
		return h.runInstall(ctx, inv)
	case "cave/info":
		return h.runInfo(ctx, inv)
	case "cave/run":
		return h.runCaveCommand(ctx, inv)
	case "cave/sync":
		fmt.Println("Syncing workspace...")
	case "cave/init":
		return h.runInit(ctx, inv)
	case "cave/addpkg":
		return h.runAddPkg(ctx, inv)
	case "enter":
		return h.runCaveCommand(ctx, inv)
	case "remote/list":
		fmt.Println("Listing remotes...")
	case "remote/add":
		fmt.Printf("Adding remote %s: %s\n", inv.Args["name"], inv.Args["url"])
	default:
		return nil, fmt.Errorf("command not implemented: %s", path)
	}
	return &ExecutionResult{ExitCode: 0}, nil
}

func (h *DefaultHandler) runCaveCommand(ctx context.Context, inv *Invocation) (*ExecutionResult, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	c, err := h.CaveMgr.Find(cwd)
	if err != nil {
		return nil, err
	}

	variant, _ := inv.Flags["variant"].(string)
	settings, err := c.Config.Resolve(variant)
	if err != nil {
		return nil, err
	}

	// Ensure packages are installed and get symlinks
	prep, err := h.PkgsMgr.Prepare(ctx, settings.Packages)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare packages: %w", err)
	}

	// For 'run', we take the 'command' arg. For 'enter', it's empty.
	var command []string
	if cmd, ok := inv.Args["command"]; ok && cmd != "" {
		// This is a bit simplistic as it doesn't handle multiple args well if the CLI engine
		// only gives us one 'command' string.
		// In a real scenario, we'd want all remaining args.
		command = strings.Fields(cmd)
	}

	backend := cave_bwrap.Create()
	cmd, err := backend.ResolveLaunch(ctx, c, settings, prep, command)
	if err != nil {
		return nil, err
	}

	return &ExecutionResult{
		IsCave: true,
		Exe:    cmd.Path,
		Args:   cmd.Args,
		Env:    cmd.Env,
	}, nil
}

func (h *DefaultHandler) runInfo(ctx context.Context, inv *Invocation) (*ExecutionResult, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	c, err := h.CaveMgr.Find(cwd)
	if err != nil {
		fmt.Printf("Current directory is not in a pi workspace.\n")
		return &ExecutionResult{ExitCode: 0}, nil
	}

	fmt.Printf("Cave ID:    %s\n", c.ID)
	fmt.Printf("Workspace:  %s\n", c.Workspace)
	fmt.Printf("Home Path:  %s\n", c.HomePath)

	settings, _ := c.Config.Resolve("")
	if len(settings.Packages) > 0 {
		fmt.Printf("Packages:   %s\n", strings.Join(settings.Packages, ", "))
	}

	return &ExecutionResult{ExitCode: 0}, nil
}

func (h *DefaultHandler) runInit(ctx context.Context, inv *Invocation) (*ExecutionResult, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	if err := h.CaveMgr.CreateInitConfig(cwd); err != nil {
		return nil, err
	}
	fmt.Println("Initialized new workspace in", cwd)
	return &ExecutionResult{ExitCode: 0}, nil
}

func (h *DefaultHandler) runAddPkg(ctx context.Context, inv *Invocation) (*ExecutionResult, error) {
	pkgStr := inv.Args["package"]
	if pkgStr == "" {
		return nil, fmt.Errorf("package string required")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	c, err := h.CaveMgr.Find(cwd)
	if err != nil {
		return nil, err
	}

	// Add package to config
	found := false
	for _, p := range c.Config.Cave.Packages {
		if p == pkgStr {
			found = true
			break
		}
	}

	if !found {
		c.Config.Cave.Packages = append(c.Config.Cave.Packages, pkgStr)
		cfgPath := filepath.Join(c.Workspace, "pi.cave.json")
		if err := c.Config.Save(cfgPath); err != nil {
			return nil, fmt.Errorf("failed to save cave config: %w", err)
		}
		fmt.Printf("Added package %s to %s\n", pkgStr, cfgPath)
	} else {
		fmt.Printf("Package %s already exists in configuration\n", pkgStr)
	}

	return &ExecutionResult{ExitCode: 0}, nil
}

func (h *DefaultHandler) runInstall(ctx context.Context, inv *Invocation) (*ExecutionResult, error) {
	pkgQuery := inv.Args["package"]
	if pkgQuery == "" {
		return nil, fmt.Errorf("package name required")
	}

	_, err := h.PkgsMgr.Prepare(ctx, []string{pkgQuery})
	if err != nil {
		return nil, err
	}

	return &ExecutionResult{ExitCode: 0}, nil
}
