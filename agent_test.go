package claudecode

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// mockRunner records the args and returns pre-configured responses.
type mockRunner struct {
	responses []cliResponse
	err       error
	lastArgs  []string
}

func (m *mockRunner) run(ctx context.Context, args []string) ([]cliResponse, error) {
	m.lastArgs = append([]string(nil), args...)
	return m.responses, m.err
}

func (m *mockRunner) runStreaming(ctx context.Context, args []string) <-chan streamEvent {
	m.lastArgs = append([]string(nil), args...)
	ch := make(chan streamEvent, len(m.responses)+1)
	go func() {
		defer close(ch)
		for _, r := range m.responses {
			ch <- streamEvent{Response: r}
		}
		if m.err != nil {
			ch <- streamEvent{Err: m.err}
		}
	}()
	return ch
}

// countEvents counts events of each role: text, toolCall, exit.
func countEvents(t *testing.T, iter *adk.AsyncIterator[*adk.AgentEvent]) (textCount, toolCount, exitCount int) {
	t.Helper()
	for {
		evt, ok := iter.Next()
		if !ok {
			break
		}
		if evt.Output != nil && evt.Output.MessageOutput != nil && evt.Output.MessageOutput.Message != nil {
			msg := evt.Output.MessageOutput.Message
			if msg.Content != "" {
				textCount++
			}
			if len(msg.ToolCalls) > 0 {
				toolCount++
			}
		}
		if evt.Action != nil && evt.Action.Exit {
			exitCount++
			break
		}
	}
	return
}

func TestNewAgent(t *testing.T) {
	agent, err := New(
		WithName("test-agent"),
		WithDescription("test description"),
		WithModel("claude-sonnet-4-6"),
		WithMaxTurns(10),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if agent.Name(context.Background()) != "test-agent" {
		t.Errorf("Name() = %q, want %q", agent.Name(context.Background()), "test-agent")
	}
	if agent.Description(context.Background()) != "test description" {
		t.Errorf("Description() = %q, want %q", agent.Description(context.Background()), "test description")
	}
}

func TestAgentRunSimpleText(t *testing.T) {
	mock := &mockRunner{
		responses: []cliResponse{
			{Type: "system", Subtype: "init", SessionID: "test-session"},
			{
				Type: "assistant",
				Message: &cliMessage{
					Role: "assistant",
					Content: []cliContentBlock{
						{Type: "text", Text: "Hello! How can I help you today?"},
					},
				},
			},
			{
				Type: "result", Subtype: "success",
				Result:       "Hello! How can I help you today?",
				StopReason:   "end_turn",
				NumTurns:     1,
				TotalCostUSD: 0.05,
			},
		},
	}

	agent, err := New(
		WithName("test-agent"),
		withRunner(mock),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	input := &adk.AgentInput{
		Messages: []*schema.Message{
			{Role: schema.User, Content: "Hi!"},
		},
	}

	iter := agent.Run(context.Background(), input)
	text, tool, exit := countEvents(t, iter)

	if text < 1 || tool < 0 || exit < 1 {
		t.Errorf("events: text=%d tool=%d exit=%d", text, tool, exit)
	}

	if len(mock.lastArgs) == 0 {
		t.Error("mock runner was not called")
	}
	if lastArg := mock.lastArgs[len(mock.lastArgs)-1]; lastArg != "Hi!" {
		t.Errorf("last arg = %q, want %q", lastArg, "Hi!")
	}
}

func TestConvertMessagesToPrompt(t *testing.T) {
	tests := []struct {
		name     string
		messages []*schema.Message
		want     string
	}{
		{
			name:     "empty",
			messages: nil,
			want:     "",
		},
		{
			name: "single user",
			messages: []*schema.Message{
				{Role: schema.User, Content: "Hello!"},
			},
			want: "Hello!",
		},
		{
			name: "system and user",
			messages: []*schema.Message{
				{Role: schema.System, Content: "You are helpful."},
				{Role: schema.User, Content: "Hi!"},
			},
			want: "[System]\nYou are helpful.\n\nHi!",
		},
		{
			name: "multi turn",
			messages: []*schema.Message{
				{Role: schema.User, Content: "What is 2+2?"},
				{Role: schema.Assistant, Content: "2+2 = 4"},
				{Role: schema.User, Content: "What about 3+3?"},
			},
			want: "What is 2+2?\n\n[Assistant]\n2+2 = 4\n\nWhat about 3+3?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertMessagesToPrompt(tt.messages)
			if got != tt.want {
				t.Errorf("convertMessagesToPrompt() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAgentRunWithToolEvents(t *testing.T) {
	mock := &mockRunner{
		responses: []cliResponse{
			{Type: "system", Subtype: "init", SessionID: "test"},
			{
				Type: "assistant",
				Message: &cliMessage{
					Role: "assistant",
					Content: []cliContentBlock{
						{Type: "thinking", Thinking: "I need to run a command."},
					},
				},
			},
			{
				Type: "assistant",
				Message: &cliMessage{
					Role: "assistant",
					Content: []cliContentBlock{
						{Type: "tool_use", ID: "call-1", Name: "Bash", Input: map[string]any{"command": "echo test"}},
					},
				},
			},
			{
				Type: "assistant",
				Message: &cliMessage{
					Role: "assistant",
					Content: []cliContentBlock{
						{Type: "text", Text: "Command executed successfully."},
					},
				},
			},
			{Type: "result", Subtype: "success", Result: "Command executed successfully.", StopReason: "end_turn"},
		},
	}

	t.Run("EmitToolEvents=true", func(t *testing.T) {
		agent, err := New(
			WithName("tool-test"),
			WithEmitToolEvents(),
			withRunner(mock),
		)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		input := &adk.AgentInput{
			Messages: []*schema.Message{
				{Role: schema.User, Content: "Run echo test"},
			},
		}

		iter := agent.Run(context.Background(), input)
		var toolCallFound, thinkingFound, textFound bool
		for {
			evt, ok := iter.Next()
			if !ok {
				break
			}
			if evt.Output != nil && evt.Output.MessageOutput != nil && evt.Output.MessageOutput.Message != nil {
				msg := evt.Output.MessageOutput.Message
				if msg.ReasoningContent != "" {
					thinkingFound = true
				}
				if len(msg.ToolCalls) > 0 {
					toolCallFound = true
					if msg.ToolCalls[0].Function.Name != "Bash" {
						t.Errorf("expected tool Bash, got %s", msg.ToolCalls[0].Function.Name)
					}
				}
				if msg.Content == "Command executed successfully." {
					textFound = true
				}
			}
		}
		if !thinkingFound {
			t.Error("thinking not found")
		}
		if !toolCallFound {
			t.Error("tool_use not surfaced with EmitToolEvents=true")
		}
		if !textFound {
			t.Error("text not found")
		}
	})

	t.Run("EmitToolEvents=false (default)", func(t *testing.T) {
		agent, err := New(
			WithName("no-tool-test"),
			withRunner(mock),
		)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		input := &adk.AgentInput{
			Messages: []*schema.Message{
				{Role: schema.User, Content: "Run echo test"},
			},
		}

		iter := agent.Run(context.Background(), input)
		var toolCallFound bool
		for {
			evt, ok := iter.Next()
			if !ok {
				break
			}
			if evt.Output != nil && evt.Output.MessageOutput != nil && evt.Output.MessageOutput.Message != nil {
				if len(evt.Output.MessageOutput.Message.ToolCalls) > 0 {
					toolCallFound = true
				}
			}
		}
		if toolCallFound {
			t.Error("tool_use should be hidden when EmitToolEvents=false")
		}
	})
}

func TestNewSessionID(t *testing.T) {
	id1 := NewSessionID()
	id2 := NewSessionID()
	if id1 == "" || id2 == "" {
		t.Error("NewSessionID returned empty string")
	}
	if id1 == id2 {
		t.Error("NewSessionID should return unique values")
	}
}

func TestBuildArgs(t *testing.T) {
	opts := DefaultOptions()
	opts.SystemPrompt = "You are helpful."
	opts.Model = "claude-sonnet-4-6"
	opts.MaxTurns = 5
	opts.PermissionMode = "acceptEdits"
	opts.Tools = []string{"Read", "Bash"}

	args := opts.BuildArgs("Hello!")
	argStr := strings.Join(args, " ")

	// Check key flags are present
	checks := []string{
		"-p",
		"--verbose",
		"--output-format stream-json",
		"--bare",
		"--system-prompt You are helpful.",
		"--exclude-dynamic-system-prompt-sections",
		"--model claude-sonnet-4-6",
		"--max-turns 5",
		"--permission-mode acceptEdits",
		"--tools Read,Bash",
		"Hello!",
	}
	for _, check := range checks {
		if !strings.Contains(argStr, check) {
			t.Errorf("args missing %q in: %s", check, argStr)
		}
	}
}

func TestAgentErrorPaths(t *testing.T) {
	t.Run("empty prompt returns error", func(t *testing.T) {
		agent, _ := New(withRunner(&mockRunner{}))
		iter := agent.Run(context.Background(), &adk.AgentInput{
			Messages: []*schema.Message{},
		})
		foundErr := false
		for {
			evt, ok := iter.Next()
			if !ok {
				break
			}
			if evt.Err != nil {
				foundErr = true
				break
			}
		}
		if !foundErr {
			t.Error("expected error for empty prompt")
		}
	})

	t.Run("mock runner error", func(t *testing.T) {
		mock := &mockRunner{
			responses: []cliResponse{},
			err:       &CLIError{Message: "CLI crashed"},
		}
		agent, _ := New(withRunner(mock))
		iter := agent.Run(context.Background(), &adk.AgentInput{
			Messages: []*schema.Message{{Role: schema.User, Content: "hello"}},
		})
		foundErr := false
		for {
			evt, ok := iter.Next()
			if !ok {
				break
			}
			if evt.Err != nil {
				foundErr = true
				break
			}
		}
		if !foundErr {
			t.Error("expected error from mock runner")
		}
	})

	t.Run("sentinel error wrapping", func(t *testing.T) {
		err := &AgentError{Message: "wrapped", Cause: ErrEmptyPrompt}
		if err.Unwrap() != ErrEmptyPrompt {
			t.Error("Unwrap() should return ErrEmptyPrompt")
		}
	})

	t.Run("empty bin returns error", func(t *testing.T) {
		_, err := New(WithBin(""))
		if err == nil {
			t.Error("expected error for empty bin")
		}
	})
}

func TestBuildArgsAllOptions(t *testing.T) {
	opts := DefaultOptions()
	opts.AppendSystemPrompt = "Be helpful."
	opts.FallbackModel = "claude-haiku-4-5"
	opts.MaxBudgetUSD = 10.0
	opts.PermissionPromptToolName = "custom"
	opts.ForkSession = true
	opts.MCPConfigPath = "/path/to/mcp.json"
	opts.StructuredOutput = `{"type":"object"}`
	opts.Settings = `{"theme":"dark"}`
	opts.AddDirs = []string{"/workspace"}
	opts.Betas = []string{"beta-feature"}
	opts.IncludePartialMessages = true
	opts.Effort = "high"

	args := opts.BuildArgs("test-prompt")
	argStr := strings.Join(args, " ")

	checks := []string{
		"--bare",
		"--append-system-prompt Be helpful.",
		"--exclude-dynamic-system-prompt-sections",
		"--fallback-model claude-haiku-4-5",
		"--max-budget-usd 10.000000",
		"--permission-prompt-tool custom",
		"--fork-session",
		"--mcp-config /path/to/mcp.json",
		"--json-schema {\"type\":\"object\"}",
		"--settings {\"theme\":\"dark\"}",
		"--add-dir /workspace",
		"--betas beta-feature",
		"--include-partial-messages",
		"--effort high",
		"test-prompt",
	}
	for _, check := range checks {
		if !strings.Contains(argStr, check) {
			t.Errorf("args missing %q in: %s", check, argStr)
		}
	}
}

func TestBuildArgsContinue(t *testing.T) {
	opts := DefaultOptions()
	opts.ContinueConversation = true
	opts.Resume = "session-123"

	args := opts.BuildArgs("prompt")
	argStr := strings.Join(args, " ")

	if !strings.Contains(argStr, "--continue") {
		t.Error("missing --continue flag")
	}
	if !strings.Contains(argStr, "--resume session-123") {
		t.Error("missing --resume flag")
	}
}

func TestBuildArgs_NoBare(t *testing.T) {
	opts := DefaultOptions()
	opts.Bare = false

	args := opts.BuildArgs("test")
	argStr := strings.Join(args, " ")

	if strings.Contains(argStr, "--bare") {
		t.Error("--bare should not be present when Bare=false")
	}
}

func TestBuildArgs_NoExcludeDynamic(t *testing.T) {
	opts := DefaultOptions()
	opts.ExcludeDynamicSystemPromptSections = false

	args := opts.BuildArgs("test")
	argStr := strings.Join(args, " ")

	if strings.Contains(argStr, "--exclude-dynamic-system-prompt-sections") {
		t.Error("--exclude-dynamic-system-prompt-sections should not be present when disabled")
	}
}

func TestBuildArgs_NoSessionPersistence(t *testing.T) {
	opts := DefaultOptions()
	opts.NoSessionPersistence = true

	args := opts.BuildArgs("test")
	argStr := strings.Join(args, " ")

	if !strings.Contains(argStr, "--no-session-persistence") {
		t.Error("missing --no-session-persistence flag")
	}
}

func TestBuildArgs_NoSessionPersistence_ClientMode(t *testing.T) {
	// --no-session-persistence should NOT appear in Client mode args.
	opts := DefaultOptions()
	opts.NoSessionPersistence = true

	args := opts.buildClientArgs()
	argStr := strings.Join(args, " ")

	if strings.Contains(argStr, "--no-session-persistence") {
		t.Error("--no-session-persistence should not be present in Client mode")
	}
}

func TestBuildArgs_Agents(t *testing.T) {
	opts := DefaultOptions()
	opts.Agents = `{"reviewer":{"description":"Reviews code","prompt":"You are a code reviewer"}}`

	args := opts.BuildArgs("test")
	argStr := strings.Join(args, " ")

	if !strings.Contains(argStr, "--agents") {
		t.Error("missing --agents flag")
	}
	if !strings.Contains(argStr, `{"reviewer":`) {
		t.Errorf("agents JSON not found in: %s", argStr)
	}
}

func TestBuildArgs_PluginDirs(t *testing.T) {
	opts := DefaultOptions()
	opts.PluginDirs = []string{"/path/to/plugin", "/other/plugin.zip"}

	args := opts.BuildArgs("test")
	argStr := strings.Join(args, " ")

	if !strings.Contains(argStr, "--plugin-dir /path/to/plugin") {
		t.Error("missing first --plugin-dir flag")
	}
	if !strings.Contains(argStr, "--plugin-dir /other/plugin.zip") {
		t.Error("missing second --plugin-dir flag")
	}
}

func TestBuildArgs_PluginURLs(t *testing.T) {
	opts := DefaultOptions()
	opts.PluginURLs = []string{"https://example.com/plugin.zip"}

	args := opts.BuildArgs("test")
	argStr := strings.Join(args, " ")

	if !strings.Contains(argStr, "--plugin-url https://example.com/plugin.zip") {
		t.Error("missing --plugin-url flag")
	}
}

func TestWithAgents(t *testing.T) {
	agents := map[string]AgentDefinition{
		"reviewer": {
			Description: "Reviews code",
			Prompt:      "You are a code reviewer.",
			Tools:       []string{"Read", "Glob", "Grep"},
			Model:       "sonnet",
		},
		"tester": {
			Description: "Runs tests",
			Prompt:      "You are a test engineer.",
		},
	}

	opts := DefaultOptions()
	opt := WithAgents(agents)
	opt(opts)

	if opts.Agents == "" {
		t.Fatal("Agents should not be empty")
	}

	// Verify JSON structure
	expectedKeys := []string{`"reviewer"`, `"tester"`, `"description"`, `"prompt"`, `code reviewer`, `test engineer`}
	for _, key := range expectedKeys {
		if !strings.Contains(opts.Agents, key) {
			t.Errorf("Agents JSON missing %q: %s", key, opts.Agents)
		}
	}

	// Verify Tools are included for reviewer
	if !strings.Contains(opts.Agents, `"Read","Glob","Grep"`) {
		t.Errorf("Agents JSON missing tools: %s", opts.Agents)
	}

	// Verify Model is included for reviewer
	if !strings.Contains(opts.Agents, `"model":"sonnet"`) {
		t.Errorf("Agents JSON missing model: %s", opts.Agents)
	}

	// Verify tester has no tools (omitempty)
	if strings.Contains(opts.Agents, `"tester".*"tools"`) {
		t.Errorf("tester should not have tools field (omitempty): %s", opts.Agents)
	}
}

func TestBuildArgs_StrictMCPConfig(t *testing.T) {
	// Default should include --strict-mcp-config
	opts := DefaultOptions()
	args := opts.BuildArgs("test")
	argStr := strings.Join(args, " ")

	if !strings.Contains(argStr, "--strict-mcp-config") {
		t.Error("missing --strict-mcp-config flag (default true)")
	}

	// When disabled, should not appear
	opts.StrictMCPConfig = false
	args = opts.BuildArgs("test")
	argStr = strings.Join(args, " ")
	if strings.Contains(argStr, "--strict-mcp-config") {
		t.Error("--strict-mcp-config should not be present when disabled")
	}
}

func TestBuildArgs_Debug(t *testing.T) {
	// Debug with filter
	opts := DefaultOptions()
	opts.Debug = true
	opts.DebugFilter = "api,hooks"

	args := opts.BuildArgs("test")
	argStr := strings.Join(args, " ")

	if !strings.Contains(argStr, "--debug api,hooks") {
		t.Errorf("missing --debug with filter in: %s", argStr)
	}

	// Debug without filter
	opts.DebugFilter = ""
	args = opts.BuildArgs("test")
	argStr = strings.Join(args, " ")
	if strings.Count(argStr, "--debug") != 1 {
		t.Errorf("expected exactly one --debug (no filter) in: %s", argStr)
	}
}

func TestBuildArgs_DebugFile(t *testing.T) {
	opts := DefaultOptions()
	opts.DebugFile = "/tmp/claude-debug.log"

	args := opts.BuildArgs("test")
	argStr := strings.Join(args, " ")

	if !strings.Contains(argStr, "--debug-file /tmp/claude-debug.log") {
		t.Errorf("missing --debug-file in: %s", argStr)
	}
}

func TestBuildArgs_DebugAndFile(t *testing.T) {
	opts := DefaultOptions()
	opts.Debug = true
	opts.DebugFilter = "api"
	opts.DebugFile = "/tmp/claude-debug.log"

	args := opts.BuildArgs("test")
	argStr := strings.Join(args, " ")

	if !strings.Contains(argStr, "--debug api") {
		t.Error("missing --debug with filter")
	}
	if !strings.Contains(argStr, "--debug-file /tmp/claude-debug.log") {
		t.Error("missing --debug-file")
	}
}
