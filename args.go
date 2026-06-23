package claudecode

import (
	"fmt"
	"os/exec"
)

// FindCLI locates the claude CLI binary. If the given name is an absolute or
// relative path, it is returned as-is. Otherwise PATH is searched.
// Returns the original name if nothing is found (the exec layer will report
// a clear error when it can't start the process).
func FindCLI(name string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	return name
}

// BuildArgs constructs the CLI argument list for one-shot mode (claude -p).
func (o *Options) BuildArgs(prompt string) []string {
	return o.buildArgs(true, prompt)
}

// buildClientArgs constructs args for Client mode (stdin JSON protocol, no -p).
func (o *Options) buildClientArgs() []string {
	return o.buildArgs(false, "")
}

// buildArgs constructs CLI args. When oneShot is true, adds -p and a positional prompt.
func (o *Options) buildArgs(oneShot bool, prompt string) []string {
	var args []string
	if oneShot {
		args = []string{"-p", "--verbose", "--output-format", "stream-json"}
	} else {
		args = []string{"--verbose", "--output-format", "stream-json", "--input-format", "stream-json"}
	}

	if o.Bare {
		args = append(args, "--bare")
	}

	if o.AppendSystemPrompt != "" {
		args = append(args, "--append-system-prompt", o.AppendSystemPrompt)
	} else if o.SystemPrompt != "" {
		args = append(args, "--system-prompt", o.SystemPrompt)
	}
	if o.ExcludeDynamicSystemPromptSections {
		args = append(args, "--exclude-dynamic-system-prompt-sections")
	}

	if o.Tools != nil {
		if len(o.Tools) == 0 {
			args = append(args, "--tools", "")
		} else {
			args = append(args, "--tools", join(o.Tools, ","))
		}
	}
	if len(o.AllowedTools) > 0 {
		args = append(args, "--allowedTools", join(o.AllowedTools, ","))
	}
	if len(o.DisallowedTools) > 0 {
		args = append(args, "--disallowedTools", join(o.DisallowedTools, ","))
	}
	if o.Model != "" {
		args = append(args, "--model", o.Model)
	}
	if o.FallbackModel != "" {
		args = append(args, "--fallback-model", o.FallbackModel)
	}
	if o.MaxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", o.MaxTurns))
	}
	if o.MaxBudgetUSD > 0 {
		args = append(args, "--max-budget-usd", fmt.Sprintf("%f", o.MaxBudgetUSD))
	}
	if o.PermissionMode != "" {
		args = append(args, "--permission-mode", o.PermissionMode)
	}
	if o.ContinueConversation {
		args = append(args, "--continue")
	}
	if o.Resume != "" {
		args = append(args, "--resume", o.Resume)
	}
	// --session-id does not work with --input-format stream-json (Client mode).
	// In Client mode, the session is maintained by keeping the CLI process alive.
	if oneShot && o.SessionID != "" {
		args = append(args, "--session-id", o.SessionID)
	}
	if o.ForkSession {
		args = append(args, "--fork-session")
	}
	// --no-session-persistence only works with --print (one-shot mode).
	if oneShot && o.NoSessionPersistence {
		args = append(args, "--no-session-persistence")
	}
	if o.MCPConfig != "" {
		args = append(args, "--mcp-config", o.MCPConfig)
	}
	if o.MCPConfigPath != "" {
		args = append(args, "--mcp-config", o.MCPConfigPath)
	}
	if o.StrictMCPConfig {
		args = append(args, "--strict-mcp-config")
	}
	if o.Agents != "" {
		args = append(args, "--agents", o.Agents)
	}
	if o.SettingSources != nil {
		if len(o.SettingSources) == 0 {
			args = append(args, "--setting-sources", "")
		} else {
			args = append(args, "--setting-sources", join(o.SettingSources, ","))
		}
	}
	if o.Settings != "" {
		args = append(args, "--settings", o.Settings)
	}
	for _, d := range o.AddDirs {
		args = append(args, "--add-dir", d)
	}
	for _, d := range o.PluginDirs {
		args = append(args, "--plugin-dir", d)
	}
	for _, u := range o.PluginURLs {
		args = append(args, "--plugin-url", u)
	}
	for _, b := range o.Betas {
		args = append(args, "--betas", b)
	}
	if o.IncludePartialMessages {
		args = append(args, "--include-partial-messages")
	}
	if o.Debug {
		if o.DebugFilter != "" {
			args = append(args, "--debug", o.DebugFilter)
		} else {
			args = append(args, "--debug")
		}
	}
	if o.DebugFile != "" {
		args = append(args, "--debug-file", o.DebugFile)
	}

	if o.Effort != "" {
		args = append(args, "--effort", o.Effort)
	}
	if o.StructuredOutput != "" {
		args = append(args, "--json-schema", o.StructuredOutput)
	}
	if len(o.ExtraArgs) > 0 {
		for _, k := range sortedKeys(o.ExtraArgs) {
			v := o.ExtraArgs[k]
			if v == "" {
				args = append(args, "-"+k)
			} else {
				args = append(args, "-"+k, v)
			}
		}
	}

	args = append(args, prompt)
	return args
}

func join(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	s := ss[0]
	for i := 1; i < len(ss); i++ {
		s += sep + ss[i]
	}
	return s
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys)-1; i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}
