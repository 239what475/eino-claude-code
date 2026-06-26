// Example: Chaining two ClaudeCodeAgents sequentially.
//
// Agent 1 ("writer"): writes Go code implementing a specified function.
// Agent 2 ("reviewer"): reviews the code and suggests improvements.
//
// This demonstrates how ClaudeCodeAgent, as a standard eino adk.Agent,
// can be composed sequentially with any other agent via adk.Runner.
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

	// ── Agent 1: Writer ──
	// Writes Go code implementing a specified function.
	writer, err := claudecode.New(
		claudecode.WithName("writer"),
		claudecode.WithDescription("Writes Go code to implement requested functionality."),
		claudecode.WithMaxTurns(10),
		claudecode.WithTools("Write", "Read"),
	)
	if err != nil {
		return fmt.Errorf("create writer: %w", err)
	}

	// ── Agent 2: Reviewer ──
	// Reviews the code and provides feedback.
	reviewer, err := claudecode.New(
		claudecode.WithName("reviewer"),
		claudecode.WithDescription("Reviews Go code for correctness, style, and best practices."),
		claudecode.WithMaxTurns(10),
		claudecode.WithTools("Read", "Grep"),
	)
	if err != nil {
		return fmt.Errorf("create reviewer: %w", err)
	}

	// ── Step 1: Writer ──
	fmt.Println("═══ Step 1: Writer ───")
	fmt.Println()

	writerRunner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: writer})
	writerOutput, err := runAndCollect(ctx, writerRunner, []adk.Message{
		schema.UserMessage("Write a Go function called Fib(n int) int that returns the nth Fibonacci number. Write it to /tmp/fib.go. Keep it simple."),
	})
	if err != nil {
		return fmt.Errorf("writer failed: %w", err)
	}
	fmt.Printf("  Writer output:\n  %s\n\n", indent(writerOutput, "  "))

	// ── Step 2: Reviewer ──
	// Feed the writer's output as context for the reviewer.
	fmt.Println("═══ Step 2: Reviewer ───")
	fmt.Println()

	reviewerRunner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: reviewer})
	reviewerOutput, err := runAndCollect(ctx, reviewerRunner, []adk.Message{
		schema.UserMessage(fmt.Sprintf(
			"The writer agent was asked to write Fib(n int). Here is what the writer produced:\n\n%s\n\n"+
				"Review /tmp/fib.go for correctness, style, and edge cases. Be specific about any issues.",
			writerOutput,
		)),
	})
	if err != nil {
		return fmt.Errorf("reviewer failed: %w", err)
	}
	fmt.Printf("  Reviewer output:\n  %s\n", indent(reviewerOutput, "  "))

	return nil
}

// runAndCollect runs an agent and returns its final text output.
func runAndCollect(ctx context.Context, runner *adk.Runner, messages []adk.Message) (string, error) {
	events := runner.Run(ctx, messages)
	var lastText string
	for {
		evt, ok := events.Next()
		if !ok {
			break
		}
		if evt.Err != nil {
			return "", evt.Err
		}
		if evt.Output != nil && evt.Output.MessageOutput != nil && evt.Output.MessageOutput.Message != nil {
			if c := strings.TrimSpace(evt.Output.MessageOutput.Message.Content); c != "" {
				lastText = c
			}
		}
		if evt.Action != nil && evt.Action.Exit {
			break
		}
	}
	return lastText, nil
}

func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}
