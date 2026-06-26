// Example: Session persistence across separate Run() calls.
//
// In one-shot mode, each Run() starts a new CLI process. Sessions let you
// persist conversation context across these separate runs:
//
//   Turn 1: WithSessionID → the CLI remembers "blue, Max"
//   Turn 2: WithResume   → the CLI recalls it from the saved session
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

	sessionID := claudecode.NewSessionID()

	// ── Turn 1: Start a new session ──
	fmt.Println("═══ Turn 1: Starting session ───")
	fmt.Printf("   Session ID: %s\n\n", sessionID[:8]+"...")

	agent1, err := claudecode.New(
		claudecode.WithSessionID(sessionID),
		claudecode.WithMaxTurns(3),
	)
	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}

	runner1 := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent1})
	events1 := runner1.Run(ctx, []adk.Message{
		schema.UserMessage("My name is Alice and my favorite color is blue. Remember this and reply briefly."),
	})

	for {
		evt, ok := events1.Next()
		if !ok {
			break
		}
		if evt.Err != nil {
			return evt.Err
		}
		if evt.Output != nil && evt.Output.MessageOutput != nil && evt.Output.MessageOutput.Message != nil {
			if c := strings.TrimSpace(evt.Output.MessageOutput.Message.Content); c != "" {
				fmt.Printf("  %s\n\n", c)
			}
		}
		if evt.Action != nil && evt.Action.Exit {
			break
		}
	}

	// ── Turn 2: Resume the same session ──
	fmt.Println("═══ Turn 2: Resuming session ───")
	fmt.Println("   (no session ID passed — the CLI loads from disk)")

	agent2, err := claudecode.New(
		claudecode.WithResume(sessionID),
		claudecode.WithMaxTurns(3),
	)
	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}

	runner2 := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent2})
	events2 := runner2.Run(ctx, []adk.Message{
		schema.UserMessage("What is my name and favorite color? Answer in one sentence."),
	})

	for {
		evt, ok := events2.Next()
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
