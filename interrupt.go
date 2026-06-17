package claudecode

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"runtime/debug"

	"github.com/cloudwego/eino/adk"
)

func init() {
	gob.Register(&agentState{})
}

// agentState is the serializable state for checkpoint/resume.
type agentState struct {
	CLISessionID string
}

// compile-time check
var _ adk.ResumableAgent = (*ClaudeCodeAgent)(nil)

// Resume restores the agent from a checkpoint and continues execution
// using the CLI --resume flag to reconnect to the saved session.
func (a *ClaudeCodeAgent) Resume(ctx context.Context, info *adk.ResumeInfo, opts ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
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

		a.resume(ctx, info, gen)
	}()

	return iter
}

func (a *ClaudeCodeAgent) resume(ctx context.Context, info *adk.ResumeInfo, gen *adk.AsyncGenerator[*adk.AgentEvent]) {
	// Restore CLI session ID from checkpoint state.
	cliSessionID := ""
	if info.InterruptState != nil {
		if state, ok := info.InterruptState.(*agentState); ok {
			cliSessionID = state.CLISessionID
		} else if b, ok := info.InterruptState.([]byte); ok {
			if state := decodeAgentState(b); state != nil {
				cliSessionID = state.CLISessionID
			}
		}
	}

	convOpts := convertOptions{emitToolEvents: a.opts.EmitToolEvents}

	// Get the follow-up prompt from resume data.
	prompt := ""
	if info.ResumeData != nil {
		if s, ok := info.ResumeData.(string); ok {
			prompt = s
		} else {
			prompt = fmt.Sprintf("%v", info.ResumeData)
		}
	}

	if prompt == "" {
		gen.Send(&adk.AgentEvent{
			AgentName: a.name,
			Err:       fmt.Errorf("claudecode: no prompt in resume data"),
		})
		return
	}

	// Use --resume to reconnect to the saved CLI session.
	origResume := a.opts.Resume
	if cliSessionID != "" {
		a.opts.Resume = cliSessionID
	}
	defer func() { a.opts.Resume = origResume }()

	args := a.opts.BuildArgs(prompt)
	responses, err := a.opts.Runner.run(ctx, args)
	if err != nil {
		if len(responses) > 0 {
			if _, _, e := convertCLIToAgentEvents(responses, a.name, gen, convOpts); e != nil {
				gen.Send(errorEvent(a.name, e.Error()))
			}
		}
		gen.Send(&adk.AgentEvent{
			AgentName: a.name,
			Err:       &AgentError{Message: "claude CLI resume failed", Cause: err},
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

// decodeAgentState deserializes agent state from bytes.
func decodeAgentState(b []byte) *agentState {
	var state agentState
	if err := gob.NewDecoder(bytes.NewReader(b)).Decode(&state); err != nil {
		return nil
	}
	return &state
}
