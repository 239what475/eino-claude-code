package claudecode

import (
	"context"
	"encoding/json"
)

// isControlRequest checks if a JSON line is a control request from the CLI.
func isControlRequest(line []byte) bool {
	return bytesContains(line, []byte(`"type":"control_request"`))
}

// handleControlRequest processes a control_request from the CLI.
func (c *Client) handleControlRequest(line []byte) {
	var req struct {
		RequestID string `json:"request_id"`
		Request   struct {
			Subtype  string         `json:"subtype"`
			ToolName string         `json:"tool_name"`
			Input    map[string]any `json:"input"`
		} `json:"request"`
	}
	if err := json.Unmarshal(line, &req); err != nil {
		return
	}

	switch req.Request.Subtype {
	case "can_use_tool":
		c.handleCanUseTool(req.RequestID, req.Request.ToolName, req.Request.Input)
	case "hook_callback":
		c.handleHookCallback(req.RequestID, line)
	default:
		c.writeControlResponse(req.RequestID, "success", map[string]any{})
	}
}

// handleCanUseTool processes a can_use_tool control request.
func (c *Client) handleCanUseTool(requestID, toolName string, input map[string]any) {
	if c.opts.OnToolUse != nil {
		result := c.opts.OnToolUse(context.Background(), toolName, input)
		switch r := result.(type) {
		case *PermissionAllow:
			resp := map[string]any{"behavior": "allow"}
			if r.UpdatedInput != nil {
				resp["updatedInput"] = r.UpdatedInput
			}
			c.writeControlResponse(requestID, "success", resp)
			return
		case *PermissionDeny:
			resp := map[string]any{"behavior": "deny", "message": r.Message}
			if r.Interrupt {
				resp["interrupt"] = true
			}
			c.writeControlResponse(requestID, "success", resp)
			return
		}
	}
	c.writeControlResponse(requestID, "success", map[string]any{"behavior": "allow"})
}

// handleHookCallback processes a hook_callback control request.
func (c *Client) handleHookCallback(requestID string, rawLine []byte) {
	var req struct {
		Request struct {
			HookEventName string `json:"hook_event_name"`
			CallbackID    string `json:"callback_id"`
		} `json:"request"`
	}
	_ = json.Unmarshal(rawLine, &req)

	event := HookEvent(req.Request.HookEventName)
	if matchers, ok := c.opts.Hooks[event]; ok && len(matchers) > 0 && len(matchers[0].Hooks) > 0 {
		hook := matchers[0].Hooks[0]
		input := HookInput{HookEventName: string(event)}
		output, err := hook(context.Background(), input)
		if err != nil {
			c.writeControlResponse(requestID, "error", map[string]any{"error": err.Error()})
			return
		}
		resp := map[string]any{}
		if output.Continue != nil {
			resp["continue"] = *output.Continue
		}
		if output.SuppressOutput {
			resp["suppressOutput"] = true
		}
		if output.StopReason != "" {
			resp["stopReason"] = output.StopReason
		}
		if output.Decision != "" {
			resp["decision"] = output.Decision
		}
		if output.SystemMessage != "" {
			resp["systemMessage"] = output.SystemMessage
		}
		c.writeControlResponse(requestID, "success", resp)
		return
	}
	c.writeControlResponse(requestID, "success", map[string]any{})
}

// writeControlResponse sends a control_response to the CLI via stdin.
func (c *Client) writeControlResponse(requestID, subtype string, response map[string]any) {
	msg := map[string]any{
		"type":       "control_response",
		"request_id": requestID,
		"response": map[string]any{
			"subtype":  subtype,
			"response": response,
		},
	}
	_ = json.NewEncoder(c.stdin).Encode(msg)
}

// bytesContains is a simple []byte contains check.
func bytesContains(b, sub []byte) bool {
	for i := 0; i <= len(b)-len(sub); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			if b[i+j] != sub[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
