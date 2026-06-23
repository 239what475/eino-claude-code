package claudecode

// Options holds all configuration for a ClaudeCodeAgent.
type Options struct {
	// Name is the agent name (default: "claude-code").
	Name string
	// Description is a short description of the agent.
	Description string
	// Bin is the path to the claude CLI executable (default: auto-discovered).
	Bin string

	// --- Prompt options ---
	// SystemPrompt sets the system prompt (replaces default).
	SystemPrompt string
	// AppendSystemPrompt appends to the default system prompt.
	AppendSystemPrompt string

	// --- Tool options ---
	// Tools is the list of tool names the agent can use.
	// nil means default CLI tools; empty slice means no tools.
	Tools []string
	// AllowedTools is the list of tools auto-approved without prompting.
	AllowedTools []string
	// DisallowedTools is the list of tools blocked from the agent.
	DisallowedTools []string

	// --- Model options ---
	// Model specifies the model to use (e.g., "claude-sonnet-4-6").
	Model string
	// FallbackModel specifies a fallback model if the primary is unavailable.
	FallbackModel string

	// --- Budget options ---
	// MaxTurns limits the number of agent turns (0 = default).
	MaxTurns int
	// MaxBudgetUSD sets a spending cap in USD (0 = no limit).
	MaxBudgetUSD float64

	// --- Permission options ---
	// PermissionMode sets the permission mode.
	// Valid: "default", "acceptEdits", "plan", "bypassPermissions", "auto", "dontAsk".
	// Default: "dontAsk" (no interactive prompts — appropriate for SDK use).
	PermissionMode string
	// PermissionPromptToolName sets the tool used for permission prompts (default: auto when hooks are set).
	PermissionPromptToolName string

	// --- Session options ---
	// ContinueConversation resumes the most recent conversation.
	ContinueConversation bool
	// Resume resumes a specific session by ID.
	Resume string
	// SessionID creates a new session with the given ID.
	SessionID string
	// ForkSession forks the resumed session to a new session ID.
	ForkSession bool
	// NoSessionPersistence disables writing sessions to disk.
	// When true, sessions cannot be resumed. Only works with --print (one-shot mode).
	// Default: false (sessions are persisted for cross-agent context sharing).
	NoSessionPersistence bool

	// --- MCP options ---
	// MCPConfig is inline MCP server configuration in JSON format.
	MCPConfig string
	// MCPConfigPath is the path to an MCP configuration file.
	MCPConfigPath string

	// --- Config options ---
	// Bare enables minimal mode: skip hooks, LSP, plugin sync, attribution,
	// auto-memory, keychain reads, and CLAUDE.md auto-discovery.
	// Default: true (SDK/programmatic use doesn't need interactive initialization).
	Bare bool
	// ExcludeDynamicSystemPromptSections moves per-machine sections (cwd, env,
	// git status) from the system prompt into the first user message, improving
	// Anthropic prompt-cache reuse. Default: true. Ignored when SystemPrompt is set.
	ExcludeDynamicSystemPromptSections bool
	// SettingSources controls which settings files to load (nil = default, empty = none).
	SettingSources []string
	// Settings is JSON settings to pass inline.
	Settings string
	// AddDirs adds directories to the CLI's workspace.
	AddDirs []string
	// Betas enables beta features.
	Betas []string
	// Agents is a JSON string defining custom agent types for sub-agent delegation.
	// Use WithAgents(map[string]AgentDefinition{...}) to build it.
	Agents string
	// PluginDirs are paths to plugin directories or .zip files to load.
	PluginDirs []string
	// PluginURLs are URLs to fetch plugin .zip files from.
	PluginURLs []string

	// --- Environment options ---
	// CWD sets the working directory for the CLI process.
	CWD string
	// Env adds environment variables (KEY=VALUE) for the CLI process.
	Env []string

	// --- Advanced options ---
	// IncludePartialMessages requests partial message streaming events.
	IncludePartialMessages bool
	// Effort sets the thinking effort level.
	// Valid: "low", "medium", "high", "xhigh", "max".
	Effort string
	// StructuredOutput sets a JSON Schema for structured output.
	StructuredOutput string

	// --- eino integration options ---
	// EmitToolEvents controls whether Claude Code's internal tool calls are surfaced
	// as AgentEvents with ToolCalls.
	EmitToolEvents bool
	// Client, if set, enables multi-turn mode. The agent will use this
	// long-lived connection instead of spawning a new process per Run() call.
	Client *Client

	// --- Callbacks ---
	// Stderr receives each line of stderr output from the CLI process.
	Stderr func(string)

	// ExtraArgs are additional CLI flags to pass through.
	// Key is the flag name (without leading dash), value is the flag value
	// (empty string for boolean flags).
	ExtraArgs map[string]string

	// --- Hooks (see hooks.go) ---
	// Hooks maps hook events to matchers with callbacks.
	Hooks map[HookEvent][]HookMatcher
	// OnToolUse is called before each tool execution to decide allow/deny.
	OnToolUse OnToolUseFunc

	// --- Internal / testing ---
	// Runner abstracts CLI execution for testing.
	Runner runner
}

// AgentDefinition defines a custom agent type for Claude Code's sub-agent system.
// It is serialized to JSON and passed via --agents.
type AgentDefinition struct {
	Description string   `json:"description"`
	Prompt      string   `json:"prompt"`
	Tools       []string `json:"tools,omitempty"`
	Model       string   `json:"model,omitempty"` // "sonnet", "opus", "haiku", "inherit"
}

// DefaultOptions returns the recommended default configuration.
func DefaultOptions() *Options {
	return &Options{
		Name:                              "claude-code",
		Description:                       "Invokes the locally installed Claude Code CLI to handle complex, multi-step tasks with file operations, bash commands, and MCP tools.",
		Bin:                               FindCLI("claude"),
		MaxTurns:                          0,
		Bare:                              true,
		ExcludeDynamicSystemPromptSections: true,
		PermissionMode:                    "dontAsk",
	}
}
