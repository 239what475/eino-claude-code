// Example: Defining custom sub-agent types that Claude Code can delegate to.
//
// WithAgents registers agent type definitions passed to the CLI via --agents.
// Claude Code's internal Task tool can then delegate subtasks to these agents,
// each with its own role, prompt, tools, and model.
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

	agent, err := claudecode.New(
		claudecode.WithMaxTurns(10),
		claudecode.WithEmitToolEvents(),
		claudecode.WithAgents(map[string]claudecode.AgentDefinition{
			"code-reviewer": {
				Description: "Reviews code for bugs, style, and best practices.",
				Prompt:      "You are a thorough code reviewer. Check for correctness, edge cases, and Go idioms. Be specific.",
				Tools:       []string{"Read", "Glob", "Grep"},
				Model:       "haiku",
			},
			"doc-writer": {
				Description: "Writes and improves documentation and code comments.",
				Prompt:      "You are a technical writer. Write clear, concise Go doc comments.",
				Tools:       []string{"Read", "Write"},
				Model:       "haiku",
			},
		}),
	)
	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}

	fmt.Println("── Claude Code with custom sub-agents ──")
	fmt.Println("   Available: code-reviewer, doc-writer")
	fmt.Println()

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
	events := runner.Run(ctx, []adk.Message{
		schema.UserMessage(
			"Use the Task tool to delegate: " +
				"1) Have code-reviewer review /home/what/myproject/eino-claude-code/session.go. " +
				"2) Have doc-writer write an improved doc comment for it. " +
				"Combine their outputs into a brief report."),
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
			msg := evt.Output.MessageOutput.Message
			for _, tc := range msg.ToolCalls {
				fmt.Printf("  [tool] → %s(%s)\n", tc.Function.Name, tc.Function.Arguments)
			}
			if c := strings.TrimSpace(msg.Content); c != "" {
				fmt.Printf("  %s\n", c)
			}
		}
		if evt.Action != nil && evt.Action.Exit {
			break
		}
	}
	return nil
}
