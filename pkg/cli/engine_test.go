package cli

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

type mockAction struct {
	err error
}

func TestEngineErrorPropagation(t *testing.T) {
	dsl := `
cmd parent "Parent command"
cmd parent child "Child command"
    arg required string "Required argument"
`
	engine, err := MakeEngine(dsl)
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	engine.Binder = func(inv *Invocation, global *GlobalFlags) (Action, error) {
		return func(ctx context.Context, m *Managers) (*ExecutionResult, error) {
			return nil, fmt.Errorf("action error")
		}, nil
	}

	// Invoke: pi parent child
	// Should fail with "argument required is missing" during Parse
	pr := engine.Parse([]string{"parent", "child"})
	if pr.Error == nil {
		t.Fatal("Expected error during Parse, got nil")
	}

	if !strings.Contains(pr.Error.Error(), "argument required is missing") {
		t.Errorf("Expected error 'argument required is missing', got: %v", pr.Error)
	}
}

func TestEngineUnknownCommand(t *testing.T) {
	dsl := `
cmd parent "Parent command"
cmd parent child "Child command"
`
	engine, err := MakeEngine(dsl)
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	// Invoke: pi parent unknown
	pr := engine.Parse([]string{"parent", "unknown"})
	if pr.Error == nil {
		t.Fatal("Expected error for unknown subcommand, got nil")
	}
	if !strings.Contains(pr.Error.Error(), "unknown command") {
		t.Errorf("Expected unknown command error, got: %v", pr.Error)
	}
}
