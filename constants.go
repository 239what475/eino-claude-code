package claudecode

// Built-in tool names. Use these with WithTools, WithAllowedTools,
// WithDisallowedTools instead of raw strings for compile-time safety.
const (
	ToolBash       = "Bash"
	ToolRead       = "Read"
	ToolWrite      = "Write"
	ToolEdit       = "Edit"
	ToolGlob       = "Glob"
	ToolGrep       = "Grep"
	ToolWebFetch   = "WebFetch"
	ToolWebSearch  = "WebSearch"
	ToolTask       = "Task"
)

// Built-in agent names. Use these with WithAgent instead of raw strings.
const (
	AgentDefault        = "claude"
	AgentExplore        = "Explore"
	AgentPlan           = "Plan"
	AgentGeneralPurpose = "general-purpose"
)
