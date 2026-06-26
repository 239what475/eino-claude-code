// Package claudecode wraps the Claude Code CLI as a first-class eino Agent and Tool.
//
// It provides [ClaudeCodeAgent] (implements [adk.Agent]) and [ClaudeCodeTool]
// (implements [tool.InvokableTool]), enabling Claude Code to participate in
// eino's multi-agent orchestration, streaming, interrupt/resume, and callback
// systems — just like any built-in eino agent.
//
// # Quick start
//
//	agent, _ := claudecode.New(claudecode.WithMaxTurns(5))
//	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
//	events := runner.Run(ctx, []adk.Message{schema.UserMessage("Hello!")})
//
// # Design
//
// Each call to [ClaudeCodeAgent.Run] spawns a one-shot claude -p process.
// Session continuity across calls is achieved via [WithSessionID] / [WithResume].
// SDK-appropriate defaults (bare mode, dontAsk permissions, prompt-cache
// optimization) are on by default — no configuration needed for basic use.
//
// # Custom tools
//
// [WithCustomTools] exposes eino [tool.InvokableTool]s to Claude Code via an
// embedded MCP HTTP server. The server starts automatically on a random
// localhost port and is passed to the CLI via --mcp-config. Claude Code
// discovers and calls these tools during task execution without any
// additional setup.
//
// # Sub-agents
//
// [WithAgents] defines custom agent types. [WithAgent] selects which agent
// to run the session as. Together they let you run Claude Code with a custom
// identity, prompt, tool set, and model — no need to pre-write agent files.
//
// [agent_tool.go]: for using ClaudeCodeAgent as a tool callable by other
// eino agents, see [NewAgentTool] and [ClaudeCodeTool].
package claudecode
