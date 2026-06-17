package claudecode

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

// Client manages a long-lived connection to the Claude Code CLI process using
// the bidirectional JSON protocol (--input-format stream-json).
//
// Unlike the simple one-shot mode (claude -p), Client keeps the CLI process
// running across multiple Send() calls, enabling multi-turn conversations
// without the cold-start overhead of launching a new process each time.
//
// Usage:
//
//	client, _ := claudecode.NewClient(claudecode.WithMaxTurns(10))
//	client.Connect(ctx)
//	defer client.Close()
//
//	responses, _ := client.Send(ctx, "First message")
//	responses, _ = client.Send(ctx, "Follow-up question")
type Client struct {
	opts   *Options
	bin    string
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	mu     sync.Mutex // protects stdin writes and stdout reads
	closed bool
}

// NewClient creates a Client with the given options.
// Call Connect() to start the CLI process.
func NewClient(opts ...Option) (*Client, error) {
	o := DefaultOptions()
	for _, opt := range opts {
		opt(o)
	}
	if o.Bin == "" {
		return nil, ErrCLINotFound
	}
	return &Client{
		opts: o,
		bin:  o.Bin,
	}, nil
}

// Connect starts the Claude Code CLI process in advanced (stdin JSON) mode.
// The process stays alive until Close() is called.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrNotConnected
	}
	if c.cmd != nil {
		return nil // already connected
	}

	args := c.opts.buildClientArgs()
	cmd := exec.CommandContext(ctx, c.bin, args...)

	if c.opts.CWD != "" {
		cmd.Dir = c.opts.CWD
	}
	if len(c.opts.Env) > 0 {
		cmd.Env = append(cmd.Environ(), c.opts.Env...)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("create stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start claude CLI: %w", err)
	}

	c.cmd = cmd
	c.stdin = stdin
	c.stdout = bufio.NewScanner(stdout)
	c.stdout.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	return nil
}

// Send writes a user message to the CLI via stdin JSON and reads all responses
// until a ResultMessage is received.
//
// Send is safe for concurrent use (protected by mutex), but callers should
// serialize calls to avoid interleaving stdin writes.
func (c *Client) Send(ctx context.Context, prompt string) ([]cliResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed || c.stdin == nil {
		return nil, ErrNotConnected
	}

	// Write user message as JSON line to stdin.
	userMsg := map[string]any{
		"type": "user",
		"message": map[string]any{
			"role":    "user",
			"content": prompt,
		},
	}
	if err := json.NewEncoder(c.stdin).Encode(userMsg); err != nil {
		return nil, fmt.Errorf("write user message: %w", err)
	}

	// Read responses until ResultMessage or error.
	var responses []cliResponse
	for c.stdout.Scan() {
		line := c.stdout.Bytes()
		if len(line) == 0 {
			continue
		}

		// Check for control_request messages (hooks / permissions).
		if isControlRequest(line) {
			c.handleControlRequest(line)
			continue
		}

		var resp cliResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
		}
		responses = append(responses, resp)
		if resp.Type == "result" {
			break
		}
	}
	if err := c.stdout.Err(); err != nil {
		return responses, fmt.Errorf("read stdout: %w", err)
	}

	return responses, nil
}

// Close gracefully shuts down the CLI process.
//
// It closes stdin to signal end-of-input, then waits up to 5 seconds
// for the process to exit. If the process doesn't exit in time, it is
// forcefully killed.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true

	if c.stdin != nil {
		_ = c.stdin.Close()
	}
	if c.cmd == nil {
		return nil
	}

	// Wait with timeout to avoid hanging if the CLI doesn't exit.
	done := make(chan error, 1)
	go func() { done <- c.cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("claude CLI exit: %w", err)
		}
		return nil
	case <-time.After(5 * time.Second):
		_ = c.cmd.Process.Kill()
		<-done // drain the goroutine
		return ErrCLITimedOut
	}
}

