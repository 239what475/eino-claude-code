package claudecode

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// streamEvent is a single event from the streaming CLI reader.
type streamEvent struct {
	Response cliResponse
	Err      error
}

// runner abstracts CLI process execution for testability.
type runner interface {
	// run executes the CLI and returns all responses (batch mode).
	run(ctx context.Context, args []string) ([]cliResponse, error)
	// runStreaming reads CLI responses as they arrive on stdout.
	// The channel is closed when the CLI process exits or ctx is cancelled.
	runStreaming(ctx context.Context, args []string) <-chan streamEvent
}

// execRunner is the production implementation that spawns the claude CLI.
type execRunner struct {
	bin    string
	cwd    string
	env    []string
	stderr func(string)
}

func (r *execRunner) run(ctx context.Context, args []string) ([]cliResponse, error) {
	//nolint:gosec
	cmd := exec.CommandContext(ctx, r.bin, args...)

	// Set working directory
	if r.cwd != "" {
		cmd.Dir = r.cwd
	}

	// Set environment
	if len(r.env) > 0 {
		cmd.Env = append(cmd.Environ(), r.env...)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start claude CLI: %w", err)
	}

	// Read stderr in background if callback is set
	if r.stderr != nil {
		go readStderr(stderrPipe, r.stderr)
	}

	// Read JSON lines from stdout
	var responses []cliResponse
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var resp cliResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
		}
		responses = append(responses, resp)
	}

	// Collect stderr for error reporting (if no callback was set)
	var stderrBytes []byte
	if r.stderr == nil {
		stderrBytes, _ = readAll(stderrPipe)
	}

	if scanErr := scanner.Err(); scanErr != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait() //nolint:errcheck
		return responses, fmt.Errorf("read stdout: %w", scanErr)
	}

	if err := cmd.Wait(); err != nil {
		if len(responses) > 0 {
			return responses, &CLIError{
				Message: fmt.Sprintf("claude CLI exited with error: %v", err),
				Stderr:  strings.TrimSpace(string(stderrBytes)),
			}
		}
		return nil, &CLIError{
			Message: fmt.Sprintf("claude CLI failed: %v", err),
			Stderr:  strings.TrimSpace(string(stderrBytes)),
		}
	}

	return responses, nil
}

// RunStreaming reads CLI responses as they arrive on stdout.
func (r *execRunner) runStreaming(ctx context.Context, args []string) <-chan streamEvent {
	ch := make(chan streamEvent, 10)

	go func() {
		defer close(ch)

		//nolint:gosec // CLI wrapper -- subprocess execution is intentional
		cmd := exec.CommandContext(ctx, r.bin, args...)
		if r.cwd != "" {
			cmd.Dir = r.cwd
		}
		if len(r.env) > 0 {
			cmd.Env = append(cmd.Environ(), r.env...)
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			ch <- streamEvent{Err: fmt.Errorf("create stdout pipe: %w", err)}
			return
		}
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			ch <- streamEvent{Err: fmt.Errorf("create stderr pipe: %w", err)}
			return
		}

		if err := cmd.Start(); err != nil {
			ch <- streamEvent{Err: fmt.Errorf("start claude CLI: %w", err)}
			return
		}

		if r.stderr != nil {
			go readStderr(stderrPipe, r.stderr)
		}

		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			var resp cliResponse
			if err := json.Unmarshal(line, &resp); err != nil {
				continue
			}
			select {
			case ch <- streamEvent{Response: resp}:
			case <-ctx.Done():
				_ = cmd.Process.Kill()
				return
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- streamEvent{Err: fmt.Errorf("read stdout: %w", err)}
		}

		if err := cmd.Wait(); err != nil {
			ch <- streamEvent{Err: fmt.Errorf("claude CLI exited with error: %w", err)}
		}
	}()

	return ch
}

// readStderr reads lines from stderr and passes them to the callback.
func readStderr(r io.Reader, fn func(string)) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fn(scanner.Text())
	}
	_ = scanner.Err() // ignore scan errors on stderr (best-effort)
}
func readAll(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	var buf []byte
	tmp := make([]byte, 4096)
	for {
		n, err := r.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err == io.EOF {
			return buf, nil
		}
		if err != nil {
			return buf, err
		}
	}
}
