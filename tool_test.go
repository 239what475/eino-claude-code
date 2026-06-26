package claudecode

import (
	"context"
	"testing"
)

func TestClaudeCodeTool_Info(t *testing.T) {
	ct, err := NewTool(
		WithName("claude_code_test"),
		WithDescription("A test tool"),
	)
	if err != nil {
		t.Fatalf("NewTool() error = %v", err)
	}

	info, err := ct.Info(context.Background())
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	if info.Name != "claude_code_test" {
		t.Errorf("expected Name 'claude_code_test', got %q", info.Name)
	}
	if info.Desc != "A test tool" {
		t.Errorf("expected Desc 'A test tool', got %q", info.Desc)
	}
	if info.ParamsOneOf == nil {
		t.Error("expected non-nil ParamsOneOf")
	}
}

func TestClaudeCodeTool_InvokableRun_MissingTask(t *testing.T) {
	ct, err := NewTool(WithName("test"))
	if err != nil {
		t.Fatalf("NewTool() error = %v", err)
	}

	_, err = ct.InvokableRun(context.Background(), `{}`)
	if err == nil {
		t.Error("expected error for missing task argument")
	}
}

func TestClaudeCodeTool_InvokableRun_InvalidJSON(t *testing.T) {
	ct, err := NewTool(WithName("test"))
	if err != nil {
		t.Fatalf("NewTool() error = %v", err)
	}

	_, err = ct.InvokableRun(context.Background(), `not valid json`)
	if err == nil {
		t.Error("expected error for invalid JSON arguments")
	}
}

func TestClaudeCodeTool_InvokableRun_WithTask(t *testing.T) {
	mock := &mockRunner{
		responses: []cliResponse{
			{Type: "system", Subtype: "init", SessionID: "s1"},
			{
				Type: "assistant",
				Message: &cliMessage{
					Role: "assistant",
					Content: []cliContentBlock{
						{Type: "text", Text: "Task completed."},
					},
				},
			},
			{Type: "result", Subtype: "success", Result: "Task completed.", StopReason: "end_turn"},
		},
	}

	ct, err := NewTool(
		WithName("executor"),
		withRunner(mock),
	)
	if err != nil {
		t.Fatalf("NewTool() error = %v", err)
	}

	result, err := ct.InvokableRun(context.Background(), `{"task":"run echo hello"}`)
	if err != nil {
		t.Fatalf("InvokableRun() error = %v", err)
	}
	if result != "Task completed." {
		t.Errorf("expected 'Task completed.', got %q", result)
	}
}

func TestClaudeCodeTool_InvokableRun_AgentError(t *testing.T) {
	mock := &mockRunner{
		err: &CLIError{Message: "CLI crashed"},
	}

	ct, err := NewTool(
		WithName("failing"),
		withRunner(mock),
	)
	if err != nil {
		t.Fatalf("NewTool() error = %v", err)
	}

	_, err = ct.InvokableRun(context.Background(), `{"task":"do something"}`)
	if err == nil {
		t.Error("expected error when CLI fails")
	}
}
