// Example: Multi-turn conversation with session continuity.
//
// This demonstrates Client mode, which keeps one CLI process alive across
// multiple turns. Also shows CLISessionID for explicit session control.
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

	// In Client mode, the CLI process stays alive between turns, so session
	// continuity is automatic — no need for --session-id or --resume.
	client, err := claudecode.NewClient(
		claudecode.WithMaxTurns(10),
		claudecode.WithPermissionMode("acceptEdits"),
	)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer func() { _ = client.Close() }()

	agent, err := claudecode.New(claudecode.WithClient(client))
	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}

	fmt.Println("── Turn 1 ──")
	if err := runTurn(ctx, agent, "My favorite color is blue and my pet's name is Max. Remember this."); err != nil {
		return err
	}

	fmt.Println("── Turn 2 ──")
	if err := runTurn(ctx, agent, "What is my pet's name and favorite color? Keep it brief."); err != nil {
		return err
	}

	fmt.Println("── Turn 3 ──")
	if err := runTurn(ctx, agent, "Run 'echo persisted-across-turns' and tell me the output."); err != nil {
		return err
	}

	fmt.Println("\nDone. All three turns shared one CLI process.")
	return nil
}

func runTurn(ctx context.Context, agent *claudecode.ClaudeCodeAgent, prompt string) error {
	input := &adk.AgentInput{
		Messages: []*schema.Message{{Role: schema.User, Content: prompt}},
	}
	iter := agent.Run(ctx, input)

	for {
		evt, ok := iter.Next()
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
