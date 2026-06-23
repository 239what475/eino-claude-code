package claudecode

import "encoding/json"

// Option is a functional option for configuring a ClaudeCodeAgent.
type Option func(*Options)

// WithName sets the agent name.
func WithName(name string) Option {
	return func(o *Options) {
		o.Name = name
	}
}

// WithDescription sets the agent description.
func WithDescription(desc string) Option {
	return func(o *Options) {
		o.Description = desc
	}
}

// WithBin sets the path to the claude CLI executable.
func WithBin(bin string) Option {
	return func(o *Options) {
		o.Bin = bin
	}
}

// WithSystemPrompt sets the system prompt.
func WithSystemPrompt(prompt string) Option {
	return func(o *Options) {
		o.SystemPrompt = prompt
	}
}

// WithTools sets the list of tools the agent can use.
// nil = default CLI tools. Empty slice = no tools.
func WithTools(tools ...string) Option {
	return func(o *Options) {
		o.Tools = tools
	}
}

// WithAllowedTools sets tools that are auto-approved without prompting.
func WithAllowedTools(tools ...string) Option {
	return func(o *Options) {
		o.AllowedTools = tools
	}
}

// WithDisallowedTools sets tools that are blocked.
func WithDisallowedTools(tools ...string) Option {
	return func(o *Options) {
		o.DisallowedTools = tools
	}
}

// WithModel specifies the model to use.
func WithModel(model string) Option {
	return func(o *Options) {
		o.Model = model
	}
}

// WithMaxTurns sets the maximum number of agent turns.
func WithMaxTurns(n int) Option {
	return func(o *Options) {
		o.MaxTurns = n
	}
}

// WithMaxBudgetUSD sets a spending cap in USD.
func WithMaxBudgetUSD(budget float64) Option {
	return func(o *Options) {
		o.MaxBudgetUSD = budget
	}
}

// WithPermissionMode sets the permission mode.
// Valid values: "default", "acceptEdits", "plan", "bypassPermissions", "auto", "dontAsk".
// Default: "dontAsk".
func WithPermissionMode(mode string) Option {
	return func(o *Options) {
		o.PermissionMode = mode
	}
}

// WithContinueConversation resumes the most recent conversation.
func WithContinueConversation() Option {
	return func(o *Options) {
		o.ContinueConversation = true
	}
}

// WithResume resumes a specific session by ID.
func WithResume(sessionID string) Option {
	return func(o *Options) {
		o.Resume = sessionID
	}
}

// WithSessionID creates a new session with the given session ID.
func WithSessionID(id string) Option {
	return func(o *Options) {
		o.SessionID = id
	}
}

// WithMCPConfig sets inline MCP server configuration (JSON format).
// Example: `{"github":{"type":"http","url":"..."}}`.
func WithMCPConfig(configJSON string) Option {
	return func(o *Options) {
		o.MCPConfig = configJSON
	}
}

// WithSettingSources controls which settings files to load.
// nil = CLI default. Empty slice = no settings.
func WithSettingSources(sources ...string) Option {
	return func(o *Options) {
		o.SettingSources = sources
	}
}

// WithExtraArgs passes additional CLI flags through.
// Key is the flag name (without leading dash), value is the flag value
// (empty string for boolean flags).
func WithExtraArgs(args map[string]string) Option {
	return func(o *Options) {
		o.ExtraArgs = args
	}
}

// WithIncludePartialMessages enables partial streaming events from the CLI.
func WithIncludePartialMessages() Option {
	return func(o *Options) {
		o.IncludePartialMessages = true
	}
}

// WithEffort sets the thinking effort level.
// Valid values: "low", "medium", "high", "xhigh", "max".
func WithEffort(effort string) Option {
	return func(o *Options) {
		o.Effort = effort
	}
}

// WithBare controls minimal mode (bare). Default: true.
// Bare mode skips hooks, LSP, plugin sync, attribution, auto-memory, keychain
// reads, and CLAUDE.md auto-discovery — appropriate for SDK/programmatic use.
// Set WithBare(false) if you need interactive initialization features.
func WithBare(v bool) Option {
	return func(o *Options) {
		o.Bare = v
	}
}

// WithExcludeDynamicSystemPromptSections controls whether per-machine sections
// (cwd, env info, git status) are moved from the system prompt into the first
// user message. Default: true. Improves Anthropic prompt-cache reuse.
// Ignored when a custom SystemPrompt is set.
func WithExcludeDynamicSystemPromptSections(v bool) Option {
	return func(o *Options) {
		o.ExcludeDynamicSystemPromptSections = v
	}
}

// WithNoSessionPersistence disables writing sessions to disk.
// Use for stateless one-shot tasks where sessions don't need to be resumed.
// Default: false. Only works with one-shot mode (--print).
func WithNoSessionPersistence(v bool) Option {
	return func(o *Options) {
		o.NoSessionPersistence = v
	}
}

// WithEmitToolEvents enables surfacing Claude Code's internal tool calls as
// AgentEvents (with ToolCalls on the message). This gives parent agents
// visibility into what tools Claude Code is using.
func WithEmitToolEvents() Option {
	return func(o *Options) {
		o.EmitToolEvents = true
	}
}

// WithClient sets a long-lived Client for multi-turn conversations.
// When set, the agent reuses the same CLI process across Run() calls,
// avoiding cold-start overhead. The caller is responsible for calling
// client.Connect() before use and client.Close() when done.
func WithClient(client *Client) Option {
	return func(o *Options) {
		o.Client = client
	}
}

// WithCWD sets the working directory for the CLI process.
func WithCWD(dir string) Option {
	return func(o *Options) {
		o.CWD = dir
	}
}

// WithEnv adds environment variables (KEY=VALUE) for the CLI process.
func WithEnv(env ...string) Option {
	return func(o *Options) {
		o.Env = append(o.Env, env...)
	}
}

// WithFallbackModel specifies a fallback model if the primary is unavailable.
func WithFallbackModel(model string) Option {
	return func(o *Options) {
		o.FallbackModel = model
	}
}

// WithAppendSystemPrompt appends to the default system prompt instead of replacing it.
func WithAppendSystemPrompt(prompt string) Option {
	return func(o *Options) {
		o.AppendSystemPrompt = prompt
	}
}

// WithForkSession forks a resumed session to a new session ID.
func WithForkSession() Option {
	return func(o *Options) {
		o.ForkSession = true
	}
}

// WithStructuredOutput sets a JSON Schema for structured output.
func WithStructuredOutput(schema string) Option {
	return func(o *Options) {
		o.StructuredOutput = schema
	}
}

// WithBetas enables beta features on the CLI.
func WithBetas(betas ...string) Option {
	return func(o *Options) {
		o.Betas = betas
	}
}

// WithSettings sets inline JSON settings for the CLI.
func WithSettings(settingsJSON string) Option {
	return func(o *Options) {
		o.Settings = settingsJSON
	}
}

// WithAddDirs adds directories to the CLI's workspace.
func WithAddDirs(dirs ...string) Option {
	return func(o *Options) {
		o.AddDirs = append(o.AddDirs, dirs...)
	}
}

// WithMCPConfigPath sets the path to an MCP configuration file.
func WithMCPConfigPath(path string) Option {
	return func(o *Options) {
		o.MCPConfigPath = path
	}
}

// WithStderr sets a callback that receives each line of stderr output
// from the CLI process. Useful for debugging CLI issues.
func WithStderr(fn func(string)) Option {
	return func(o *Options) {
		o.Stderr = fn
	}
}

// WithPermissionPromptTool sets the tool used for permission prompts.
func WithPermissionPromptTool(toolName string) Option {
	return func(o *Options) {
		o.PermissionPromptToolName = toolName
	}
}

// WithHooks registers lifecycle hook callbacks.
// event: the hook event (e.g., HookPreToolUse).
// matchers: tool name patterns and their callbacks.
func WithHooks(event HookEvent, matchers ...HookMatcher) Option {
	return func(o *Options) {
		if o.Hooks == nil {
			o.Hooks = make(map[HookEvent][]HookMatcher)
		}
		o.Hooks[event] = append(o.Hooks[event], matchers...)
	}
}

// WithOnToolUse sets a callback that is called before each tool execution.
// Return PermissionAllow to permit, PermissionDeny to block.
func WithOnToolUse(fn OnToolUseFunc) Option {
	return func(o *Options) {
		o.OnToolUse = fn
	}
}

// withRunner overrides the CLI runner (for testing).
func withRunner(r runner) Option {
	return func(o *Options) {
		o.Runner = r
	}
}

// WithAgents registers custom agent definitions for sub-agent delegation.
// Keys are agent type names (e.g., "reviewer", "tester").
// The map is serialized to JSON and passed to the CLI via --agents.
func WithAgents(agents map[string]AgentDefinition) Option {
	return func(o *Options) {
		b, _ := json.Marshal(agents)
		o.Agents = string(b)
	}
}

// WithPluginDir adds a plugin directory or .zip file to load for this session.
// Repeatable — each call appends another path.
func WithPluginDir(path string) Option {
	return func(o *Options) {
		o.PluginDirs = append(o.PluginDirs, path)
	}
}

// WithPluginURL adds a URL to fetch a plugin .zip from for this session.
// Repeatable — each call appends another URL.
func WithPluginURL(url string) Option {
	return func(o *Options) {
		o.PluginURLs = append(o.PluginURLs, url)
	}
}
