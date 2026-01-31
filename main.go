package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"pi/pkg/config"
	"pi/pkg/display"
	"pi/pkg/installer"
	"pi/pkg/recipe"
	"pi/pkg/resolver"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	command := os.Args[1]
	switch command {
	case "install":
		if len(os.Args) < 3 {
			fmt.Println("Error: package name required. E.g. nodejs@latest")
			os.Exit(1)
		}
		runInstall(os.Args[2])
	case "help":
		usage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("pi - Universal Package Installer")
	fmt.Println("\nUsage:")
	fmt.Println("  pi install <package>[@version]")
	fmt.Println("  pi help")
}

func runInstall(pkgQuery string) {
	parts := strings.Split(pkgQuery, "@")
	name := parts[0]
	version := "latest"
	if len(parts) > 1 {
		version = parts[1]
	}

	cfg, err := config.Init()
	if err != nil {
		fmt.Printf("Error initializing config: %v\n", err)
		os.Exit(1)
	}

	disp := display.NewConsole()
	defer disp.Close()

	ctx := context.Background()

	// Find recipe
	var r *recipe.Recipe
	switch name {
	case "nodejs":
		r = recipe.GetNodejsRecipe()
	case "java":
		r = recipe.GetJavaRecipe()
	default:
		fmt.Printf("Error: unknown package ecosystem: %s\n", name)
		return
	}

	task := disp.StartTask(name)
	defer task.Done()

	// Resolve
	pkgDef, err := resolver.Resolve(ctx, cfg, r, version, task)
	if err != nil {
		task.Log(fmt.Sprintf("Resolution failed: %v", err))
		return
	}

	// Plan
	plan, err := installer.NewPlan(cfg, *pkgDef)
	if err != nil {
		task.Log(fmt.Sprintf("Planning failed: %v", err))
		return
	}

	// Install
	if err := installer.Install(ctx, plan, task); err != nil {
		task.Log(fmt.Sprintf("Installation failed: %v", err))
		return
	}
}