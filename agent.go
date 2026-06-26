package claudecode

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// ClaudeCodeAgent is an eino Agent that invokes a locally installed Claude Code CLI
// in one-shot mode (claude -p). It implements adk.Agent and can be used with
// eino's runner, composed into multi-agent topologies, or wrapped as a tool.
//
// Basic usage:
//
//	agent, _ := claudecode.New(
//	    claudecode.WithSystemPrompt("You are a helpful assistant."),
//	    claudecode.WithTools("Read", "Write", "Bash"),
//	)
//	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
//	events := runner.run(ctx, []adk.Message{schema.UserMessage("Hello!")})
type ClaudeCodeAgent struct {
	name        string
	description string
	opts        *Options
}

// compile-time interface check
var _ adk.Agent = (*ClaudeCodeAgent)(nil)

// New creates a ClaudeCodeAgent with the given options.
func New(opts ...Option) (*ClaudeCodeAgent, error) {
	o := DefaultOptions()
	for _, opt := range opts {
		opt(o)
	}
	if o.Bin == "" {
		return nil, ErrCLINotFound
	}
	if o.Runner == nil {
		o.Runner = &execRunner{
			bin:    o.Bin,
			cwd:    o.CWD,
			env:    o.Env,
			stderr: o.Stderr,
		}
	}
	return &ClaudeCodeAgent{
		name:        o.Name,
		description: o.Description,
		opts:        o,
	}, nil
}

// Name returns the agent name.
func (a *ClaudeCodeAgent) Name(ctx context.Context) string {
	return a.name
}

// Description returns the agent description.
func (a *ClaudeCodeAgent) Description(ctx context.Context) string {
	return a.description
}

// Run executes the Claude Code CLI and returns a stream of Agent events.
func (a *ClaudeCodeAgent) Run(ctx context.Context, input *adk.AgentInput, opts ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				gen.Send(&adk.AgentEvent{
					AgentName: a.name,
					Err:       fmt.Errorf("panic: %v\n\n%s", r, debug.Stack()),
				})
			}
			gen.Close()
		}()

		a.run(ctx, input, gen)
	}()

	return iter
}

// run is the internal execution logic (runs in a goroutine).
func (a *ClaudeCodeAgent) run(ctx context.Context, input *adk.AgentInput, gen *adk.AsyncGenerator[*adk.AgentEvent]) {
	prompt := convertMessagesToPrompt(input.Messages)
	if prompt == "" {
		gen.Send(&adk.AgentEvent{
			AgentName: a.name,
			Err:       ErrEmptyPrompt,
		})
		return
	}

	convOpts := convertOptions{emitToolEvents: a.opts.EmitToolEvents}

	// Start embedded MCP server if custom tools are configured.
	mcpSrv, err := newMCPServer(a.opts.CustomTools)
	if err != nil {
		gen.Send(&adk.AgentEvent{
			AgentName: a.name,
			Err:       &AgentError{Message: "failed to start MCP server", Cause: err},
		})
		return
	}
	if mcpSrv != nil {
		defer mcpSrv.close()
	}

	// Build CLI args.
	args := a.opts.BuildArgs(prompt)

	// Inject MCP config before the prompt. --mcp-config is variadic, so
	// we insert "--" to stop option parsing and protect the prompt.
	// Also auto-allow all tools from our embedded MCP server.
	if mcpSrv != nil {
		args = append(args[:len(args)-1],
			"--mcp-config", mcpSrv.mcpConfigJSON(),
			"--allowedTools", "mcp__eino-tools__*",
			"--",
			args[len(args)-1])
	}

	if input.EnableStreaming {
		// Streaming mode: emit MessageStream chunks as CLI outputs text.
		streamCh := a.opts.Runner.runStreaming(ctx, args)
		_, cliSessionID, streamErr := convertCLIStreamToAgentEvents(ctx, streamCh, a.name, gen, convOpts)

		if streamErr != nil {
			// If context was cancelled, emit an interrupt event with saved state.
			if ctx.Err() != nil && cliSessionID != "" {
				state := &agentState{CLISessionID: cliSessionID}
				interruptEvt := adk.TypedStatefulInterrupt[*schema.Message](ctx,
					fmt.Sprintf("claude CLI interrupted (session: %s)", cliSessionID),
					state,
				)
				if interruptEvt != nil {
					gen.Send(interruptEvt)
					return
				}
			}
			gen.Send(&adk.AgentEvent{
				AgentName: a.name,
				Err:       &AgentError{Message: "claude CLI streaming failed", Cause: streamErr},
			})
		}
		return
	}

	// Batch mode: read all responses, then emit events.
	responses, err := a.opts.Runner.run(ctx, args)
	if err != nil {
		if len(responses) > 0 {
			if _, _, convErr := convertCLIToAgentEvents(responses, a.name, gen, convOpts); convErr != nil {
				gen.Send(errorEvent(a.name, convErr.Error()))
			}
		}
		gen.Send(&adk.AgentEvent{
			AgentName: a.name,
			Err:       &AgentError{Message: "claude CLI execution failed", Cause: err},
		})
		return
	}

	if _, _, err := convertCLIToAgentEvents(responses, a.name, gen, convOpts); err != nil {
		gen.Send(&adk.AgentEvent{
			AgentName: a.name,
			Err:       &AgentError{Message: "failed to convert CLI output", Cause: err},
		})
	}
}
