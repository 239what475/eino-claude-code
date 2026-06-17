package claudecode

// cliResponse represents one JSON line from the Claude Code CLI stdout.
// This is an internal type — callers work with eino types, not raw CLI JSON.
type cliResponse struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype,omitempty"`

	// System fields
	SessionID string `json:"session_id,omitempty"`
	UUID      string `json:"uuid,omitempty"`
	CWD       string `json:"cwd,omitempty"`

	// Assistant fields
	Message *cliMessage `json:"message,omitempty"`

	// Result fields
	Result       string    `json:"result,omitempty"`
	IsError      bool      `json:"is_error,omitempty"`
	StopReason   string    `json:"stop_reason,omitempty"`
	TotalCostUSD float64   `json:"total_cost_usd,omitempty"`
	DurationMS   int       `json:"duration_ms,omitempty"`
	NumTurns     int       `json:"num_turns,omitempty"`
	Usage        *cliUsage `json:"usage,omitempty"`

	Error string `json:"error,omitempty"`
}

// cliMessage is the inner message object in an assistant response.
type cliMessage struct {
	ID      string            `json:"id,omitempty"`
	Role    string            `json:"role,omitempty"`
	Model   string            `json:"model,omitempty"`
	Content []cliContentBlock `json:"content,omitempty"`
	Usage   *cliUsage         `json:"usage,omitempty"`
}

// cliContentBlock is one block in the assistant message content array.
type cliContentBlock struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	Thinking  string `json:"thinking,omitempty"`
	Signature string `json:"signature,omitempty"`

	// Tool use
	ID    string         `json:"id,omitempty"`
	Name  string         `json:"name,omitempty"`
	Input map[string]any `json:"input,omitempty"`

	// Tool result
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   any    `json:"content,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`
}

// cliUsage holds token and cost information from the CLI.
type cliUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}
