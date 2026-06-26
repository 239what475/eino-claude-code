// Example: Running Claude Code as a custom agent type.
//
// WithAgents defines agent types. WithAgent selects which one to run as.
// The session executes with that agent's system prompt, tools, and model.
//
// This works in Bare mode (default) because --agent selects the session
// identity, independent of the Task tool.
//
// For delegation (Task tool dispatching to sub-agents), use WithBare(false).
//
// Usage:
//
//	go run .
//
// Prerequisites: claude CLI installed and authenticated.

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	claudecode "github.com/239what475/eino-claude-code"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	// Define a custom agent type and run AS it.
	// The code-reviewer prompt and tools become the session's identity.
	agent, err := claudecode.New(
		claudecode.WithMaxTurns(10),
		claudecode.WithAgents(map[string]claudecode.AgentDefinition{
			"code-reviewer": {
				Description: "Thorough code reviewer focused on bugs, style, and best practices.",
				Prompt:      "You are a thorough code reviewer. When given a file to review, check for: correctness bugs, edge cases, performance issues, security vulnerabilities, and Go idioms. Be specific and actionable. Always reference exact line numbers.",
				Tools:       []string{"Read", "Glob", "Grep"},
				Model:       "haiku",
			},
		}),
		claudecode.WithAgent("code-reviewer"), // run AS code-reviewer
	)
	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}

	fmt.Println("── Running as code-reviewer agent ──")
	fmt.Println("   (system prompt: thorough code review)")
	fmt.Println()

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
	events := runner.Run(ctx, []adk.Message{
		schema.UserMessage("Review /home/what/myproject/eino-claude-code/session.go. Be specific."),
	})

	for {
		evt, ok := events.Next()
		if !ok {
			break
		}
		if evt.Err != nil {
			return evt.Err
		}
		if evt.Output != nil && evt.Output.MessageOutput != nil && evt.Output.MessageOutput.Message != nil {
			if c := strings.TrimSpace(evt.Output.MessageOutput.Message.Content); c != "" {
				fmt.Printf("  %s\n", c)
			}
		}
		if evt.Action != nil && evt.Action.Exit {
			break
		}
	}
	return nil
}
