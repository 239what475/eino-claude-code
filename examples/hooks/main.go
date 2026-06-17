// Example: Observing and controlling Claude Code with hooks and permissions.
//
// Usage:
//
//	go run .

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

	toolCalls := make(map[string]int)

	agent, err := claudecode.New(
		claudecode.WithMaxTurns(3),
		claudecode.WithPermissionMode("default"),
		claudecode.WithAllowedTools("Bash", "Read"),

		// WithOnToolUse: permission callback before every tool execution.
		// Return PermissionAllow to permit, PermissionDeny to block.
		claudecode.WithOnToolUse(func(ctx context.Context, toolName string, input map[string]any) claudecode.PermissionResult {
			toolCalls[toolName]++
			fmt.Printf("  [permission] tool=%q allowed\n", toolName)
			return &claudecode.PermissionAllow{}
		}),

		// WithHooks: lifecycle callbacks for specific events.
		claudecode.WithHooks(claudecode.HookPreToolUse,
			claudecode.HookMatcher{
				Matcher: "Bash",
				Hooks: []claudecode.HookCallback{
					func(ctx context.Context, input claudecode.HookInput) (claudecode.HookOutput, error) {
						fmt.Printf("  [hook:PreToolUse] about to run: %s\n", input.ToolName)
						return claudecode.HookOutput{}, nil
					},
				},
			},
		),
	)
	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})

	fmt.Println("── Running with hooks ──")
	events := runner.Run(ctx, []adk.Message{
		schema.UserMessage("Run 'echo hook-demo-ok' and tell me the result. Keep it brief."),
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

	fmt.Print("\n── Tool usage summary ──\n")
	for name, count := range toolCalls {
		fmt.Printf("  %s: %d call(s)\n", name, count)
	}
	return nil
}
