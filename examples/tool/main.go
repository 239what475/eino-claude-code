// Example: ChatModelAgent delegates tasks to ClaudeCodeTool.
//
// This is the recommended pattern: an eino ChatModel agent acts as the
// "brain" (deciding when to call tools), and ClaudeCodeTool acts as the
// "hands" (executing complex multi-step tasks with full CLI capabilities).
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
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
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

	ccTool, err := claudecode.NewTool(
		claudecode.WithName("claude_code"),
		claudecode.WithDescription(
			"Delegate complex multi-step tasks to Claude Code. "+
				"Use for: file operations, shell commands, code analysis, git, web search."),
		claudecode.WithMaxTurns(10),
		claudecode.WithPermissionMode("acceptEdits"),
		claudecode.WithAllowedTools("Read", "Write", "Edit", "Bash", "Glob", "Grep"),
	)
	if err != nil {
		return fmt.Errorf("create tool: %w", err)
	}

	// In production, replace &demoModel{} with a real ChatModel (OpenAI, Claude API).
	supervisor, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "supervisor",
		Description: "Orchestrates complex tasks by delegating to Claude Code.",
		Instruction: "You are a supervisor. For complex tasks, call claude_code.",
		Model:       &demoModel{},
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{ccTool},
			},
		},
		MaxIterations: 5,
	})
	if err != nil {
		return fmt.Errorf("create supervisor: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: supervisor})

	fmt.Println("── Supervisor delegating to ClaudeCodeTool ──")
	events := runner.Run(ctx, []adk.Message{
		schema.UserMessage("Run 'echo delegated-task-success' and report the result. Keep it brief."),
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
			if len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					fmt.Printf("  [tool call] → %s\n", tc.Function.Name)
				}
			}
			if c := strings.TrimSpace(msg.Content); c != "" && len(msg.ToolCalls) == 0 {
				fmt.Printf("  %s\n", c)
			}
		}
		if evt.Action != nil && evt.Action.Exit {
			break
		}
	}
	return nil
}

// demoModel is a mock that delegates the first message to claude_code,
// then responds with plain text to end the conversation.
// Replace with a real ChatModel (OpenAI, Claude API) in production.
type demoModel struct{ called bool }

func (m *demoModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	if !m.called {
		m.called = true
		last := input[len(input)-1]
		return &schema.Message{
			Role: schema.Assistant,
			ToolCalls: []schema.ToolCall{{
				ID:   "call-1", Type: "function",
				Function: schema.FunctionCall{
					Name:      "claude_code",
					Arguments: fmt.Sprintf(`{"task":%q}`, last.Content),
				},
			}},
		}, nil
	}
	return &schema.Message{Role: schema.Assistant, Content: "Done."}, nil
}

func (m *demoModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	msg, _ := m.Generate(ctx, input, opts...)
	return schema.StreamReaderFromArray([]*schema.Message{msg}), nil
}
