package cli

import (
	"context"
	"fmt"
	"strings"

	"pi/pkg/config"
	"pi/pkg/display"
	"pi/pkg/installer"
	"pi/pkg/recipe"
	"pi/pkg/repository"
	"pi/pkg/resolver"
)

type DefaultHandler struct {
	Repo *repository.Manager
	Disp display.Display
}

func (h *DefaultHandler) Execute(ctx context.Context, inv *Invocation) error {
	if v, ok := inv.Global["verbose"].(bool); ok {
		h.Disp.SetVerbose(v)
	}

	path := getCmdPath(inv.Command)
	switch path {
	case "install":
		return h.runInstall(ctx, inv)
	case "sync":
		fmt.Println("Syncing workspace...")
	case "init":
		fmt.Println("Initializing workspace...")
	case "enter":
		fmt.Println("Entering sandbox...")
	case "remote/list":
		fmt.Println("Listing remotes...")
	case "remote/add":
		fmt.Printf("Adding remote %s: %s\n", inv.Args["name"], inv.Args["url"])
	default:
		return fmt.Errorf("command not implemented: %s", path)
	}
	return nil
}

func (h *DefaultHandler) runInstall(ctx context.Context, inv *Invocation) error {
	pkgQuery := inv.Args["package"]
	if pkgQuery == "" {
		return fmt.Errorf("package name required")
	}

	parts := strings.Split(pkgQuery, "@")
	name := parts[0]
	version := "latest"
	if len(parts) > 1 {
		version = parts[1]
	}

	cfg, err := config.Init()
	if err != nil {
		return fmt.Errorf("error initializing config: %v", err)
	}

	// Find recipe
	src, err := h.Repo.GetRecipe(name)
	if err != nil {
		return fmt.Errorf("error loading recipe: %v", err)
	}

	task := h.Disp.StartTask(name)
	defer task.Done()

	recipeObj, err := recipe.NewStarlarkRecipe(name, src, task.Log)
	if err != nil {
		return fmt.Errorf("error initializing recipe: %v", err)
	}

	// Resolve
	pkgDef, err := resolver.Resolve(ctx, cfg, recipeObj, version, task)
	if err != nil {
		task.Log(fmt.Sprintf("Resolution failed: %v", err))
		return err
	}

	// Plan
	plan, err := installer.NewPlan(cfg, *pkgDef)
	if err != nil {
		task.Log(fmt.Sprintf("Planning failed: %v", err))
		return err
	}

	// Install
	if err := installer.Install(ctx, plan, task); err != nil {
		task.Log(fmt.Sprintf("Installation failed: %v", err))
		return err
	}

	return nil
}
