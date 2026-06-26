# Claude Code CLI Coverage Gap Analysis

**Date**: 2026-06-24
**CLI Version**: 2.1.186 (Claude Code)
**Go Module**: `github.com/239what475/eino-claude-code`

## Philosophy

Claude Code is treated as a **black-box executor** — give it a task, it completes
it, you get the result. We don't need to observe or control its internal
behavior. Features that enable introspection (hooks, hook events in stream) or
human-in-the-loop interaction (AskUserQuestion) are **audit/debugging
infrastructure** — valuable to have eventually, but not core to the mission.

This document records what Claude Code CLI offers and what we expose, organized
by relevance to the black-box-executor use case.

---

## P0 — Must Have for SDK Use

### `--bare` (Minimal Mode)

```
--bare  Minimal mode: skip hooks, LSP, plugin sync, attribution, auto-memory,
        background prefetches, keychain reads, and CLAUDE.md auto-discovery.
        Sets CLAUDE_CODE_SIMPLE=1.
```

**Why**: Every `claude -p` invocation we make today goes through the full
interactive initialization path — CLAUDE.md discovery, plugin sync, keychain
reads, auto-memory, etc. For an SDK that provides its own system prompt, tools,
and model config, this is entirely wasted work every single `Run()` call.

**Decision**: `WithBare()` defaults to **true**. Provide `WithBare(false)` to opt out.

**Status**: ✅ Implemented.

### `--exclude-dynamic-system-prompt-sections`

```
--exclude-dynamic-system-prompt-sections
    Move per-machine sections (cwd, env info, memory paths, git status) from
    the system prompt into the first user message. Improves cross-user
    prompt-cache reuse.
```

**Why**: Without this, every `Run()` call produces a slightly different system
prompt (different cwd, git status, etc.), breaking Anthropic's prompt cache.
For SDK usage this directly increases cost and latency. Only applies when using
the default system prompt (ignored with `--system-prompt`).

**Decision**: `WithExcludeDynamicSystemPromptSections()` defaults to **true**.
Provide `WithExcludeDynamicSystemPromptSections(false)` to opt out.

**Status**: ✅ Implemented.

### `--no-session-persistence`

```
--no-session-persistence  Disable session persistence - sessions will not be
                          saved to disk and cannot be resumed.
```

**Why**: Session persistence is what enables cross-agent context sharing
(`WithSessionID` → `WithResume`) and interrupt/resume (`ResumableAgent`).
For stateless one-shot tasks, disabling it avoids unnecessary disk I/O. But
it should NOT be the default — session sharing between agents is a key use case.

**Decision**: `WithNoSessionPersistence()` defaults to **false**. Users opt in
when they want stateless execution.

**Status**: ✅ Implemented.

---

## P1 — Black-Box Features Worth Having

These enhance the executor's capabilities without requiring us to observe its
internals.

### `--agents` (Inline Custom Agent Definitions)

```
--agents <json>  JSON object defining custom agents (e.g.
    '{"reviewer": {"description": "Reviews code", "prompt": "You are a code reviewer"}}')
```

Entry point to Claude Code's sub-agent system. Without it, eino users must write
agent files to `.claude/agents/` on disk. With it, sub-agent types can be
defined inline at agent creation time.

**Proposed**: `WithAgents(map[string]AgentDefinition{...})` — typed Go struct,
marshaled to JSON internally.

**Status**: ✅ Implemented.

### `--plugin-dir` / `--plugin-url` (Plugin System)

```
--plugin-dir <path>   Load a plugin from a directory or .zip
--plugin-url <url>    Fetch a plugin .zip from a URL
```

Plugins add custom tools, slash commands, etc. — they **enhance the executor's
toolkit**, not our visibility into it.

**Proposed**: `WithPluginDir(path)` and `WithPluginURL(url)`, both repeatable.

**Status**: ✅ Implemented.

### `--file` (File Resources at Startup)

```
--file <specs...>  File resources to download at startup.
                   Format: file_id:relative_path
```

**Decision**: Skipped. Designed for IDE/frontend file upload integration — not
relevant for SDK/black-box-executor use. Files should be in the working directory
and accessed via the Read tool.

### `-n, --name` (Session Display Name)

```
-n, --name <name>  Set a display name for this session
```

**Decision**: Skipped. Only visible in interactive TUI (`/resume` picker, prompt
box). eino already has `WithName()` for agent naming. Sessions are identified
by UUID (`--session-id`).

### `--output-format json` (Single JSON Result)

We hardcode `--output-format stream-json`. The CLI also supports `json` (single
result object at end) and `text`.

**Decision**: Skipped. `stream-json` already covers both eino streaming and
non-streaming modes — we collect all JSON lines in batch mode, or stream them
in streaming mode. Adding `json` format would require a second parsing path
with no real benefit.

### Permission Modes: `auto` and `dontAsk`

We support `default`, `acceptEdits`, `plan`, `bypassPermissions`. The CLI also
has `auto` and `dontAsk`.

**Status**: Low priority. We pass permission mode through as a string — adding
docs for `auto`/`dontAsk` is sufficient. No code change needed.

### Effort Levels: `xhigh` and `max`

We support `low`, `medium`, `high`. The CLI also has `xhigh` and `max`.

**Status**: Low priority. We pass effort through as a string — adding docs for
`xhigh`/`max` is sufficient. No code change needed.

### `--agent` (Session Agent Override)

```
--agent <agent>  Agent for the current session. Overrides the 'agent' setting.
```

Selects which custom agent type (defined via `--agents`) to use as the default.

**Status**: ✅ Implemented. `WithAgent(name)` passes `--agent` to CLI.

### `--brief` (SendUserMessage Tool)

```
--brief  Enable SendUserMessage tool for agent-to-user communication
```

**Status**: Deferred. For black-box executor use, agent-to-user communication
is not needed. `WithBrief()` can be added when human-in-the-loop workflows are
required.

---

## P2 — Nice to Have

| CLI Flag | Description | Proposed Option |
|----------|-------------|-----------------|
| `--ide` | Auto-connect to IDE | `WithIDE()` |
| `--from-pr` | Resume session from PR | `WithFromPR(prNum)` |
| `--debug` / `--debug-file` | Debug logging | `WithDebug(filter)` / `WithDebugFile(path)` | ✅ |
| `--worktree` + `--tmux` | Git worktree isolation | `WithWorktree(name)` / `WithTmux()` |
| `--chrome` / `--no-chrome` | Chrome integration toggle | `WithChrome()` / `WithNoChrome()` |
| `--safe-mode` | Troubleshooting mode | `WithSafeMode()` |
| `--strict-mcp-config` | Strict MCP, ignore other configs | `WithStrictMCPConfig()` | ✅ |
| `--disable-slash-commands` | Disable all skills | `WithDisableSlashCommands()` |
| `--ax-screen-reader` | Accessibility | `WithScreenReader()` |
| `--remote-control` | Remote control | `WithRemoteControl(name)` |
| `--allow-dangerously-skip-permissions` | Permission bypass opt-in | `WithAllowDangerouslySkipPermissions()` |
| `--prompt-suggestions` | Emit prompt_suggestion after each turn | `WithPromptSuggestions(enabled)` |
| `--replay-user-messages` | Echo user messages in Client mode | `WithReplayUserMessages()` |

---

## Audit / Debugging Infrastructure

These features expose Claude Code's **internal behavior** for observation or
control. They're valuable for debugging, auditing, and human-in-the-loop
workflows — but are NOT needed for the core black-box-executor use case.

### Hooks System (PreToolUse, PostToolUse, etc.)

We have the types defined (`hooks.go`) and a partial `handleHookCallback` in
`protocol.go`. Two problems:

1. **Client mode**: `handleHookCallback` only runs the first matcher's first
   callback. Needs proper iteration + tool name pattern matching. More
   fundamentally, we never send the `initialize` control request that registers
   our Go callbacks with the CLI — so the CLI never knows to invoke them.

2. **One-shot mode**: No stdin control channel exists. Need `--include-hook-events`
   flag to get hook events in the stdout JSON stream. We'd need to parse the
   `hook_event` output type and convert to AgentEvents.

**When we need it**: auditing what tools Claude Code used and why, debugging
unexpected behavior, implementing fine-grained permission control beyond
`--allowedTools`.

### AskUserQuestion

The CLI can ask the user questions during execution. It arrives as a
`can_use_tool` control request with `tool_name: "AskUserQuestion"`. We need to
parse questions from the input, call a user-provided callback for answers, and
return them as `updatedInput`.

**When we need it**: human-in-the-loop approval workflows where a real person
needs to answer questions mid-execution. For a black-box executor, if the CLI
tries to ask a question and we have no callback, it's an error — we should fail
loudly rather than silently drop it.

### `--include-hook-events`

Makes hook lifecycle events appear in the stdout JSON stream (one-shot mode).
Without stdin control protocol, this is the only way to observe hooks in
one-shot mode. It's read-only observation — no intervention possible.

---

## CLI Subcommands — Unwrapped

| Command | Purpose | Relevance |
|---------|---------|-----------|
| `claude ultrareview` | Cloud multi-agent code review | Could be a separate eino Tool |
| `claude doctor` | Health check | CI/ops health checks |
| `claude mcp` | MCP server management | `mcp list` could be useful |
| `claude plugin` | Plugin management | Plugin lifecycle |
| `claude project` | Project state management | `project info` |
| `claude agents` | Background agent management | Interactive only |
| `claude auth` | Authentication management | Interactive only |
| `claude setup-token` | Long-lived auth token | Interactive setup |

---

## stdin JSON Protocol Coverage

Our `protocol.go` handles:

```
✅ user message → stdin JSON
✅ control_request: can_use_tool
⚠️  control_request: hook_callback  (incomplete — see Audit section)
❌ control_request: ask_user_question
❌ initialize control request (needed to register hooks with CLI)
```

Missing output message types in `convertor.go`:

| Type | Triggered By | Priority |
|------|-------------|----------|
| `prompt_suggestion` | `--prompt-suggestions` | P2 |
| `hook_event` | `--include-hook-events` | Audit |
| `user` (replayed) | `--replay-user-messages` | P2 |

---

## Summary Matrix

| Category | Covered | Missing | Coverage |
|----------|---------|---------|----------|
| CLI Flags | 23 | 26 | 47% |
| Permission Modes | 4 of 6 | 2 | 67% |
| Effort Levels | 3 of 5 | 2 | 60% |
| Output Formats | 1 of 3 | 2 | 33% |
| stdin Protocol | 2 of 5+ | 3+ | ~40% |
| CLI Subcommands | 0 of 8 | 8 | 0% |

**Overall: ~40-50% of CLI surface exposed. The missing half is mostly
interactive/debugging features that don't block the black-box-executor use case.**

---

## Implementation Order

1. **P0 (done)** — `--bare`, `--exclude-dynamic-system-prompt-sections`,
   `--no-session-persistence`. Three flags, straightforward.
2. **P1 (done)** — `--agents`, `--plugin-dir/url`. Sub-agent definitions and
   plugin loading.
3. **P1 remaining** — Permission modes and effort levels (docs only).
   `--agent`, `--brief` (deferred until needed).
4. **P2 (as needed)** — everything else.
5. **Audit infrastructure (when needed)** — hooks fix, AskUserQuestion,
   `--include-hook-events`, initialize protocol.
