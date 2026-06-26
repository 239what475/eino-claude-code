// Example: Simplest usage of eino-claude-code.
//
// Bare mode, dontAsk permission, and prompt-cache optimization are on by default.
// Just create an agent, run it, and read the output.
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
		claudecode.WithMaxTurns(3),
	)
	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
	events := runner.Run(ctx, []adk.Message{
		schema.UserMessage("Say hello in exactly one sentence."),
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
				fmt.Println(c)
			}
		}
		if evt.Action != nil && evt.Action.Exit {
			break
		}
	}
	return nil
}
