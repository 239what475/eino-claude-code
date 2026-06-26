package claudecode

import (
	"errors"
	"fmt"
)

// Sentinel errors for programmatic error handling.
var (
	ErrCLINotFound  = errors.New("claude CLI not found")
	ErrEmptyPrompt  = errors.New("empty prompt")
)

// CLIError is returned when the Claude Code CLI process fails.
type CLIError struct {
	Message string
	Stderr  string
}

func (e *CLIError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("%s\nstderr: %s", e.Message, e.Stderr)
	}
	return e.Message
}

// AgentError wraps an error that occurs during agent execution.
type AgentError struct {
	Message string
	Cause   error
}

func (e *AgentError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *AgentError) Unwrap() error {
	return e.Cause
}
