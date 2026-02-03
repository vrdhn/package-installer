package cli

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

type mockHandler struct {
	err error
}

func (m *mockHandler) Execute(ctx context.Context, inv *Invocation) (*ExecutionResult, error) {
	return nil, m.err
}

func TestEngineErrorPropagation(t *testing.T) {
	dsl := `
cmd parent "Parent command"
cmd parent child "Child command"
    arg required string "Required argument"
`
	engine, err := NewEngine(dsl)
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	// Register a handler that returns a validation error
	engine.Register("parent/child", &mockHandler{
		err: fmt.Errorf("argument required is missing"),
	})

	// Invoke: pi parent child
	// Should fail with "argument required is missing"
	// Before fix: would show help for "parent" and return nil error (or exit code 0)

	_, err = engine.Run(context.Background(), []string{"parent", "child"})
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "argument required is missing") {
		t.Errorf("Expected error 'argument required is missing', got: %v", err)
	}
}

func TestEngineUnknownCommand(t *testing.T) {
	dsl := `
cmd parent "Parent command"
cmd parent child "Child command"
`
	engine, err := NewEngine(dsl)
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	// Invoke: pi parent unknown
	// Should fall back to showing help for parent (which returns nil error in Run logic currently)
	// Actually Run() returns {ExitCode:0}, nil if help is printed.

	res, err := engine.Run(context.Background(), []string{"parent", "unknown"})
	if err != nil {
		t.Fatalf("Expected nil error (help shown), got: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", res.ExitCode)
	}
}
