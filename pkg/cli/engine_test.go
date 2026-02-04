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
	pr := engine.Parse([]string{"parent", "child"})
	if pr.Error == nil {
		t.Fatal("Expected error, got nil")
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
	engine, err := NewEngine(dsl)
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
