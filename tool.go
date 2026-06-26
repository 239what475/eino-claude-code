package claudecode

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// compile-time interface check
var _ tool.BaseTool = (*ClaudeCodeTool)(nil)
var _ tool.InvokableTool = (*ClaudeCodeTool)(nil)

// ClaudeCodeTool wraps a ClaudeCodeAgent as an eino Tool so it can be used by
// any eino ChatModelAgent as a "super tool" for delegating complex, multi-step tasks.
//
// When invoked, it runs the Claude Code CLI with the given task and returns the
// result text. This integrates naturally with eino's ReAct loop — the parent agent
// decides when to delegate a task to Claude Code and receives the result.
type ClaudeCodeTool struct {
	name        string
	description string
	agent       *ClaudeCodeAgent
}

// NewTool creates a ClaudeCodeTool backed by the given agent configuration.
func NewTool(opts ...Option) (*ClaudeCodeTool, error) {
	agent, err := New(opts...)
	if err != nil {
		return nil, fmt.Errorf("claudecode: create tool: %w", err)
	}
	return NewToolFromAgent(agent), nil
}

// NewToolFromAgent wraps an existing [ClaudeCodeAgent] as a [ClaudeCodeTool].
// The agent's name and description become the tool's metadata.
// This is useful when the same agent configuration needs to serve both as a
// standalone agent and as a callable tool in a multi-agent setup.
func NewToolFromAgent(agent *ClaudeCodeAgent) *ClaudeCodeTool {
	return &ClaudeCodeTool{
		name:        agent.name,
		description: agent.description,
		agent:       agent,
	}
}

// Info returns the tool metadata for registration with eino.
func (t *ClaudeCodeTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.name,
		Desc: t.description,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"task": {
				Type:     schema.String,
				Desc:     "The task description to delegate to Claude Code.",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun executes the tool. Expects {"task": "..."}.
func (t *ClaudeCodeTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		Task string `json:"task"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("claudecode: parse tool arguments: %w", err)
	}
	if args.Task == "" {
		return "", fmt.Errorf("claudecode: task is required")
	}

	iter := t.agent.Run(ctx, &adk.AgentInput{
		Messages: []*schema.Message{{Role: schema.User, Content: args.Task}},
	})

	var lastResult string
	for {
		evt, ok := iter.Next()
		if !ok {
			break
		}
		if evt.Err != nil {
			// Drain remaining events so the goroutine exits cleanly.
			for {
				_, ok := iter.Next()
				if !ok {
					break
				}
			}
			return "", fmt.Errorf("claudecode: tool execution error: %w", evt.Err)
		}
		if evt.Output != nil && evt.Output.MessageOutput != nil && evt.Output.MessageOutput.Message != nil {
			lastResult = evt.Output.MessageOutput.Message.Content
		}
	}

	return lastResult, nil
}
