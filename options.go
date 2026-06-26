package claudecode

import "github.com/cloudwego/eino/components/tool"

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

	// StrictMCPConfig ignores all MCP servers except those from --mcp-config.
	// Default: true (SDK should not load local .mcp.json).
	StrictMCPConfig bool

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
	// Agent selects which agent type to run the session as.
	// Must match a key in Agents or a built-in agent.
	// Passed to the CLI via --agent.
	Agent string
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
	// Debug enables debug mode. Default: false.
	Debug bool
	// DebugFilter is an optional category filter for debug output (e.g. "api,hooks").
	// Only used when Debug is true.
	DebugFilter string
	// DebugFile writes debug logs to a specific file path.
	// Implicitly enables debug mode.
	DebugFile string

	// --- eino integration options ---
	// EmitToolEvents controls whether Claude Code's internal tool calls are surfaced
	// as AgentEvents with ToolCalls.
	EmitToolEvents bool
	// CustomTools are eino InvokableTools exposed to Claude Code via an embedded
	// MCP HTTP server. The server is started on a random localhost port and
	// passed to the CLI via --mcp-config.
	CustomTools []tool.InvokableTool

	// --- Callbacks ---
	// Stderr receives each line of stderr output from the CLI process.
	Stderr func(string)

	// ExtraArgs are additional CLI flags to pass through.
	// Key is the flag name (without leading dash), value is the flag value
	// (empty string for boolean flags).
	ExtraArgs map[string]string

	// --- Internal / testing ---
	// Runner allows custom CLI execution (default: execRunner).
	Runner Runner
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
		Name:                               "claude-code",
		Description:                        "Invokes the locally installed Claude Code CLI to handle complex, multi-step tasks with file operations, bash commands, and MCP tools.",
		Bin:                                FindCLI("claude"),
		MaxTurns:                           0,
		Bare:                               true,
		ExcludeDynamicSystemPromptSections: true,
		PermissionMode:                     "dontAsk",
		StrictMCPConfig:                    true,
	}
}
