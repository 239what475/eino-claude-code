// Example: Simplest usage of eino-claude-code.
//
// Usage:
//
//	go run .

package main

import (
	"context"
	"fmt"
	"io"
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

	// NewSessionID generates a unique session identifier. Pass it to
	// WithSessionID on the first call and WithResume on later calls
	// to persist conversation context across program restarts.
	sid := claudecode.NewSessionID()

	agent, err := claudecode.New(
		claudecode.WithSessionID(sid),
		claudecode.WithMaxTurns(3),
		claudecode.WithPermissionMode("acceptEdits"),
	)
	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}

	// EnableStreaming:true delivers text chunks in real time through MessageStream.
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:          agent,
		EnableStreaming: true,
	})

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
		if evt.Output != nil && evt.Output.MessageOutput != nil {
			mv := evt.Output.MessageOutput
			if mv.IsStreaming && mv.MessageStream != nil {
				for {
					chunk, err := mv.MessageStream.Recv()
					if err == io.EOF {
						break
					}
					if err != nil {
						return err
					}
					fmt.Print(chunk.Content)
				}
				fmt.Println()
			} else if mv.Message != nil {
				if c := strings.TrimSpace(mv.Message.Content); c != "" {
					fmt.Println(c)
				}
			}
		}
		if evt.Action != nil && evt.Action.Exit {
			break
		}
	}
	return nil
}
