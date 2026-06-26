// Example: Extending Claude Code with custom eino tools via embedded MCP server.
//
// This demonstrates how to define a Go tool implementing eino's InvokableTool
// interface, and have Claude Code automatically discover and call it.
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
		claudecode.WithMaxTurns(3),
		claudecode.WithEmitToolEvents(), // surface tool calls as events
		claudecode.WithCustomTools(&calculatorTool{}, &weatherTool{}),
	)
	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}

	fmt.Println("── Claude Code with custom eino tools ──")
	fmt.Println("   Available: calculator, weather")
	fmt.Println()

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
	events := runner.Run(ctx, []adk.Message{
		schema.UserMessage("Use the calculator tool to compute 123 * 456, then the weather tool for Tokyo."),
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
			// Print tool calls to show the MCP tools were actually invoked.
			for _, tc := range msg.ToolCalls {
				fmt.Printf("  [tool call] → %s(%s)\n", tc.Function.Name, tc.Function.Arguments)
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

// ── Custom Tools ──

// calculatorTool implements tool.InvokableTool — a simple calculator.
type calculatorTool struct{}

func (t *calculatorTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "calculator",
		Desc: "Performs arithmetic. expression: a math expression like '2 + 3' or '10 * 5'.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"expression": {
				Type:     schema.String,
				Desc:     "The arithmetic expression to evaluate. Supports +, -, *, /.",
				Required: true,
			},
		}),
	}, nil
}

func (t *calculatorTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		Expression string `json:"expression"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}

	var a, b int
	var op string
	n, _ := fmt.Sscanf(args.Expression, "%d %s %d", &a, &op, &b)
	if n != 3 {
		return "", fmt.Errorf("could not parse expression: %q (use format: '2 + 3')", args.Expression)
	}

	var result int
	switch op {
	case "+":
		result = a + b
	case "-":
		result = a - b
	case "*":
		result = a * b
	case "/":
		if b == 0 {
			return "", fmt.Errorf("division by zero")
		}
		result = a / b
	default:
		return "", fmt.Errorf("unknown operator: %q", op)
	}

	return fmt.Sprintf("%d %s %d = %d", a, op, b, result), nil
}

// weatherTool implements tool.InvokableTool — returns mock weather data.
type weatherTool struct{}

func (t *weatherTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "weather",
		Desc: "Gets the current weather for a city. city: name of the city.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"city": {
				Type:     schema.String,
				Desc:     "The city name to get weather for.",
				Required: true,
			},
		}),
	}, nil
}

func (t *weatherTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		City string `json:"city"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}
	return fmt.Sprintf("Weather in %s: sunny, 22°C, humidity 45%%, wind 12 km/h", args.City), nil
}
