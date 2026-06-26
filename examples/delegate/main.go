// Example: Sub-agent delegation in Bare mode via MCP tools.
//
// Bare mode strips the Task tool, so the CLI's built-in delegation is
// unavailable. This example shows how to build your own delegation layer:
// each agent (Explore, Plan, code-reviewer) is wrapped as an MCP tool.
// When the model calls the tool, we spawn a new claude -p --agent <name>
// process and return the result.
//
// Usage:
//
//	go run .
//
// Prerequisites: claude CLI installed and authenticated.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
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
		claudecode.WithCustomTools(
			newAgentTool("explore", "Explore the codebase: search files, find patterns, map structure. Use for 'find all X' or 'how does Y work' questions. task: what to explore."),
			newAgentTool("plan", "Design an implementation plan. Use before complex changes. task: what to plan for."),
			newAgentTool("code-reviewer", "Review code for bugs, style, and best practices. task: what file or change to review.",
				claudecode.WithAgents(map[string]claudecode.AgentDefinition{
					"code-reviewer": {
						Description: "Reviews code for bugs, style, and best practices.",
						Prompt:      "You are a thorough code reviewer. Be specific, reference line numbers, and find real issues.",
						Tools:       []string{"Read", "Glob", "Grep"},
						Model:       "haiku",
					},
				}),
			),
		),
	)
	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}

	fmt.Println("── Claude Code with delegate tools ──")
	fmt.Println("   Available: explore, plan, code-reviewer")
	fmt.Println()

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
	events := runner.Run(ctx, []adk.Message{
		schema.UserMessage(
			"Use explore to find all files under /home/what/myproject/eino-claude-code/ that define or call 'NewSessionID'. " +
				"Then use code-reviewer to review those files. " +
				"Combine the findings into a brief report."),
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
				fmt.Printf("  [delegate] → %s(%s)\n", tc.Function.Name, tc.Function.Arguments)
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

// ── Agent-as-MCP-Tool ──

// agentTool wraps a ClaudeCodeAgent as an eino InvokableTool.
// When called, it runs the agent in one-shot mode and returns the result.
type agentTool struct {
	name string
	desc string
	opts []claudecode.Option
}

func newAgentTool(name, desc string, opts ...claudecode.Option) tool.InvokableTool {
	return &agentTool{name: name, desc: desc, opts: opts}
}

func (t *agentTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.name,
		Desc: t.desc,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"task": {Type: schema.String, Desc: "The task to delegate", Required: true},
		}),
	}, nil
}

func (t *agentTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		Task string `json:"task"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}
	if args.Task == "" {
		return "", fmt.Errorf("task is required")
	}

	agent, err := claudecode.New(
		append([]claudecode.Option{
			claudecode.WithMaxTurns(10),
			claudecode.WithModel("claude-haiku-4-5"),
			claudecode.WithAgent(t.name),
		}, t.opts...)...,
	)
	if err != nil {
		return "", fmt.Errorf("create delegate agent: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
	events := runner.Run(ctx, []adk.Message{schema.UserMessage(args.Task)})

	var lastResult string
	for {
		evt, ok := events.Next()
		if !ok {
			break
		}
		if evt.Err != nil {
			return "", fmt.Errorf("agent %s failed: %w", t.name, evt.Err)
		}
		if evt.Output != nil && evt.Output.MessageOutput != nil && evt.Output.MessageOutput.Message != nil {
			if c := strings.TrimSpace(evt.Output.MessageOutput.Message.Content); c != "" {
				lastResult = c
			}
		}
		if evt.Action != nil && evt.Action.Exit {
			break
		}
	}
	return lastResult, nil
}
