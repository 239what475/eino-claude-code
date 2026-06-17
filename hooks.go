package claudecode

import "context"

// HookEvent identifies a lifecycle event in Claude Code execution.
type HookEvent string

const (
	HookPreToolUse         HookEvent = "PreToolUse"
	HookPostToolUse        HookEvent = "PostToolUse"
	HookPostToolUseFailure HookEvent = "PostToolUseFailure"
	HookUserPromptSubmit   HookEvent = "UserPromptSubmit"
	HookStop               HookEvent = "Stop"
	HookSubagentStop       HookEvent = "SubagentStop"
	HookPreCompact         HookEvent = "PreCompact"
	HookNotification       HookEvent = "Notification"
	HookSubagentStart      HookEvent = "SubagentStart"
	HookPermissionRequest  HookEvent = "PermissionRequest"
)

// HookMatcher pairs a tool name pattern with a list of callbacks.
// The Matcher field is a tool name pattern (e.g., "Bash", "Write|Edit").
type HookMatcher struct {
	Matcher string
	Hooks   []HookCallback
	Timeout float64 // seconds, 0 = no timeout
}

// HookCallback is invoked when a hook event fires.
type HookCallback func(ctx context.Context, input HookInput) (HookOutput, error)

// HookInput carries the event-specific fields passed from the CLI.
type HookInput struct {
	HookEventName  string         `json:"hook_event_name,omitempty"`
	SessionID      string         `json:"session_id,omitempty"`
	TranscriptPath string         `json:"transcript_path,omitempty"`
	CWD            string         `json:"cwd,omitempty"`
	PermissionMode string         `json:"permission_mode,omitempty"`
	ToolName       string         `json:"tool_name,omitempty"`
	ToolUseID      string         `json:"tool_use_id,omitempty"`
	ToolInput      map[string]any `json:"tool_input,omitempty"`
	ToolResponse   any            `json:"tool_response,omitempty"`
	Error          string         `json:"error,omitempty"`
	IsInterrupt    bool           `json:"is_interrupt,omitempty"`
	Prompt         string         `json:"prompt,omitempty"`
	StopHookActive bool           `json:"stop_hook_active,omitempty"`
}

// HookOutput is returned by a HookCallback to control behavior.
type HookOutput struct {
	Continue           *bool          `json:"continue,omitempty"`       // false stops processing
	SuppressOutput     bool           `json:"suppressOutput,omitempty"` // hide tool output
	StopReason         string         `json:"stopReason,omitempty"`     // reason for stopping
	Decision           string         `json:"decision,omitempty"`       // PermissionRequest: "allow"/"deny"/"ask"
	SystemMessage      string         `json:"systemMessage,omitempty"`  // inject system message
	Reason             string         `json:"reason,omitempty"`         // explanation
	HookSpecificOutput map[string]any `json:"hookSpecificOutput,omitempty"`
}

// OnToolUseFunc is called before each tool execution to decide permission.
// Return PermissionAllow to permit, PermissionDeny to block.
type OnToolUseFunc func(ctx context.Context, toolName string, input map[string]any) PermissionResult

// PermissionResult is the interface for tool permission decisions.
type PermissionResult interface {
	permissionResult()
}

// PermissionAllow permits a tool to execute, optionally with modified input.
type PermissionAllow struct {
	UpdatedInput map[string]any
}

func (*PermissionAllow) permissionResult() {}

// PermissionDeny blocks a tool from executing.
type PermissionDeny struct {
	Message   string
	Interrupt bool // also interrupt the agent
}

func (*PermissionDeny) permissionResult() {}
