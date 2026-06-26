// Example: Running multiple ClaudeCodeAgents in parallel.
//
// Two agents analyze the same file from different perspectives simultaneously:
//   Agent 1 ("perf"): performance analysis
//   Agent 2 ("security"): security review
//
// This demonstrates how ClaudeCodeAgent, as a standard eino adk.Agent,
// can be used in parallel patterns via goroutines.
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
	"sync"

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

	// ── Agent 1: Performance analyzer ──
	perfAgent, err := claudecode.New(
		claudecode.WithName("perf-analyzer"),
		claudecode.WithDescription("Analyzes Go code for performance issues."),
		claudecode.WithMaxTurns(10),
		claudecode.WithTools("Read", "Grep", "Bash"),
	)
	if err != nil {
		return fmt.Errorf("create perf agent: %w", err)
	}

	// ── Agent 2: Security reviewer ──
	secAgent, err := claudecode.New(
		claudecode.WithName("security-reviewer"),
		claudecode.WithDescription("Reviews Go code for security vulnerabilities."),
		claudecode.WithMaxTurns(10),
		claudecode.WithTools("Read", "Grep", "Bash"),
	)
	if err != nil {
		return fmt.Errorf("create security agent: %w", err)
	}

	// ── Run both in parallel ──
	fmt.Println("═══ Running parallel analysis: performance + security ───")
	fmt.Println()

	var wg sync.WaitGroup
	results := make(chan [2]string, 2) // agent name + output

	// Performance analysis
	wg.Add(1)
	go func() {
		defer wg.Done()
		runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: perfAgent})
		out, err := runAndCollect(ctx, runner, []adk.Message{
			schema.UserMessage("Analyze the file /home/what/myproject/eino-claude-code/agent.go for performance issues. Focus on goroutine usage, memory allocation, and IO patterns. Be specific and concise."),
		})
		if err != nil {
			results <- [2]string{"[PERF]", fmt.Sprintf("Error: %v", err)}
			return
		}
		results <- [2]string{"[PERF]", out}
	}()

	// Security review
	wg.Add(1)
	go func() {
		defer wg.Done()
		runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: secAgent})
		out, err := runAndCollect(ctx, runner, []adk.Message{
			schema.UserMessage("Review /home/what/myproject/eino-claude-code/agent.go for security issues. Focus on input validation, error handling, and process execution safety. Be specific and concise."),
		})
		if err != nil {
			results <- [2]string{"[SEC]", fmt.Sprintf("Error: %v", err)}
			return
		}
		results <- [2]string{"[SEC]", out}
	}()

	// Wait for both to complete.
	go func() { wg.Wait(); close(results) }()

	for r := range results {
		fmt.Printf("── %s ──\n", r[0])
		fmt.Printf("  %s\n\n", indent(r[1], "  "))
	}

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
