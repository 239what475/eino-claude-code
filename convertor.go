package claudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// convertMessagesToPrompt flattens a slice of eino messages into a text prompt
// suitable for passing to the Claude Code CLI.
func convertMessagesToPrompt(messages []*schema.Message) string {
	if len(messages) == 0 {
		return ""
	}

	parts := make([]string, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case schema.System:
			if msg.Content != "" {
				parts = append(parts, "[System]\n"+msg.Content)
			}
		case schema.User:
			if msg.Content != "" {
				parts = append(parts, msg.Content)
			}
		case schema.Assistant:
			if msg.Content != "" {
				parts = append(parts, "[Assistant]\n"+msg.Content)
			}
			if len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					parts = append(parts, fmt.Sprintf("[Tool Call: %s(%s)]", tc.Function.Name, tc.Function.Arguments))
				}
			}
		case schema.Tool:
			if msg.Content != "" {
				parts = append(parts, fmt.Sprintf("[Tool Result (id=%s)]\n%s", msg.ToolCallID, msg.Content))
			}
		}
	}

	return strings.Join(parts, "\n\n")
}

// convertOptions holds the behavioral flags for message conversion.
type convertOptions struct {
	emitToolEvents bool
}

// convertCLIToAgentEvents converts a batch of CLI JSON responses into AgentEvents.
func convertCLIToAgentEvents(responses []cliResponse, agentName string, gen *adk.AsyncGenerator[*adk.AgentEvent], opts convertOptions) (finalResult string, cliSessionID string, err error) {
	var hadContent bool

	for _, resp := range responses {
		switch resp.Type {
		case "assistant":
			if resp.Message == nil {
				continue
			}
			msg, ok := buildAssistantMessage(resp.Message, opts)
			if ok {
				hadContent = true
				gen.Send(assistantEvent(agentName, msg))
			}

		case "result":
			finalResult = resp.Result
			var exitMessage *schema.Message
			if !hadContent && finalResult != "" {
				exitMessage = &schema.Message{Role: schema.Assistant, Content: finalResult}
			}
			gen.Send(exitEvent(agentName, exitMessage))

		case "system":
			if resp.IsError || resp.Error != "" {
				gen.Send(errorEvent(agentName, fmt.Sprintf("CLI system error: %s", resp.Error)))
			}
			if resp.Subtype == "init" && resp.SessionID != "" {
				cliSessionID = resp.SessionID
			}
		}
	}

	return finalResult, cliSessionID, nil
}

// convertCLIStreamToAgentEvents processes CLI responses as they arrive from a channel,
// emitting AgentEvents with MessageStream for real-time text streaming.
//
// Returns finalResult, cliSessionID, and error.
func convertCLIStreamToAgentEvents(
	ctx context.Context,
	streamCh <-chan streamEvent,
	agentName string,
	gen *adk.AsyncGenerator[*adk.AgentEvent],
	opts convertOptions,
) (finalResult string, cliSessionID string, err error) {
	// Set up a Pipe for streaming text chunks.
	sr, sw := schema.Pipe[*schema.Message](1)
	var streamStarted bool
	var hadContent bool

	// Process CLI events as they arrive.
	for evt := range streamCh {
		// Check for context cancellation (interrupt).
		select {
		case <-ctx.Done():
			if sw != nil {
				sw.Close()
			}
			return "", cliSessionID, ctx.Err()
		default:
		}

		if evt.Err != nil {
			if sw != nil {
				sw.Close()
			}
			gen.Send(errorEvent(agentName, evt.Err.Error()))
			return "", cliSessionID, evt.Err
		}

		resp := evt.Response
		switch resp.Type {
		case "assistant":
			if resp.Message == nil {
				continue
			}
			msg, ok := buildAssistantMessage(resp.Message, opts)
			if !ok {
				continue
			}

			if msg.Content != "" {
				hadContent = true
				if !streamStarted {
					// Emit the first text event with MessageStream.
					streamStarted = true
					gen.Send(&adk.AgentEvent{
						AgentName: agentName,
						Output: &adk.AgentOutput{
							MessageOutput: &adk.MessageVariant{
								IsStreaming:   true,
								MessageStream: sr,
								Role:          schema.Assistant,
							},
						},
					})
				}
				// Write text chunk to the stream.
				sw.Send(msg, nil)
			} else if msg.ReasoningContent != "" || len(msg.ToolCalls) > 0 {
				// Non-text content: emit as a separate event.
				gen.Send(assistantEvent(agentName, msg))
			}

		case "result":
			finalResult = resp.Result

			// Close the text stream.
			if streamStarted {
				sw.Close()
			} else if finalResult != "" {
				// No text was streamed; emit the result directly.
				sr2, sw2 := schema.Pipe[*schema.Message](1)
				sw2.Send(&schema.Message{Role: schema.Assistant, Content: finalResult}, nil)
				sw2.Close()
				gen.Send(&adk.AgentEvent{
					AgentName: agentName,
					Output: &adk.AgentOutput{
						MessageOutput: &adk.MessageVariant{
							IsStreaming:   true,
							MessageStream: sr2,
							Role:          schema.Assistant,
						},
					},
				})
				hadContent = true
			}

			var exitMessage *schema.Message
			if !hadContent && finalResult != "" {
				exitMessage = &schema.Message{Role: schema.Assistant, Content: finalResult}
			}
			gen.Send(exitEvent(agentName, exitMessage))

		case "system":
			if resp.IsError || resp.Error != "" {
				gen.Send(errorEvent(agentName, fmt.Sprintf("CLI system error: %s", resp.Error)))
			}
			if resp.Subtype == "init" && resp.SessionID != "" {
				cliSessionID = resp.SessionID
			}
		}
	}

	return finalResult, cliSessionID, nil
}

// buildAssistantMessage converts a CLI assistant message into a schema.Message.
// Returns (msg, true) if the message has any content, (nil, false) otherwise.
func buildAssistantMessage(m *cliMessage, opts convertOptions) (*schema.Message, bool) {
	var textContent string
	var reasoningContent string
	var toolCalls []schema.ToolCall

	for _, block := range m.Content {
		switch block.Type {
		case "text":
			textContent += block.Text
		case "thinking":
			reasoningContent += block.Thinking
		case "tool_use":
			if !opts.emitToolEvents {
				continue
			}
			argsJSON := "{}"
			if block.Input != nil {
				if b, err := json.Marshal(block.Input); err == nil {
					argsJSON = string(b)
				}
			}
			toolCalls = append(toolCalls, schema.ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: schema.FunctionCall{
					Name:      block.Name,
					Arguments: argsJSON,
				},
			})
		case "tool_result":
			// Handled internally by Claude Code.
		}
	}

	if textContent == "" && reasoningContent == "" && len(toolCalls) == 0 {
		return nil, false
	}

	return &schema.Message{
		Role:             schema.Assistant,
		Content:          textContent,
		ReasoningContent: reasoningContent,
		ToolCalls:        toolCalls,
	}, true
}

func assistantEvent(agentName string, msg *schema.Message) *adk.AgentEvent {
	evt := &adk.AgentEvent{
		AgentName: agentName,
		Output: &adk.AgentOutput{
			MessageOutput: &adk.MessageVariant{
				Message: msg,
				Role:    schema.Assistant,
			},
		},
	}
	// Check for Task tool calls → emit TransferToAgent
	for _, tc := range msg.ToolCalls {
		if tc.Function.Name == "Task" {
			subAgent := parseTaskSubAgent(tc.Function.Arguments)
			if subAgent != "" {
				evt.Action = adk.NewTransferToAgentAction(subAgent)
			}
		}
	}
	return evt
}

// parseTaskSubAgent extracts the subagent_type from Task tool arguments.
func parseTaskSubAgent(argsJSON string) string {
	var args struct {
		SubagentType string `json:"subagent_type"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return ""
	}
	return args.SubagentType
}

func exitEvent(agentName string, msg *schema.Message) *adk.AgentEvent {
	return &adk.AgentEvent{
		AgentName: agentName,
		Output: &adk.AgentOutput{
			MessageOutput: &adk.MessageVariant{
				Message: msg,
				Role:    schema.Assistant,
			},
		},
		Action: adk.NewExitAction(),
	}
}

func errorEvent(agentName string, errMsg string) *adk.AgentEvent {
	return &adk.AgentEvent{
		AgentName: agentName,
		Err:       &AgentError{Message: errMsg},
	}
}


