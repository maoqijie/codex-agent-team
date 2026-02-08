package codexrpc

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
)

// SpawnOptions configures how the codex2 app-server process is started.
type SpawnOptions struct {
	// BinaryPath is the path to the codex2 binary.
	BinaryPath string
	// ListenAddr is the transport address (default: "stdio://").
	ListenAddr string
}

// Process wraps a running codex2 app-server subprocess and its RPC client.
type Process struct {
	cmd       *exec.Cmd
	client    *Client
	stdinPipe io.Closer
	stderr    *bytes.Buffer
}

// Spawn starts a codex2 app-server process and returns a Process with
// an attached JSON-RPC Client ready for communication.
func Spawn(ctx context.Context, opts SpawnOptions) (*Process, error) {
	listenAddr := opts.ListenAddr
	if listenAddr == "" {
		listenAddr = "stdio://"
	}

	cmd := exec.CommandContext(ctx, opts.BinaryPath, "app-server", "--listen", listenAddr)

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdin pipe: %w", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start codex2 app-server: %w", err)
	}

	client := NewClient(stdinPipe, io.Reader(stdoutPipe))
	client.Start()

	return &Process{
		cmd:       cmd,
		client:    client,
		stdinPipe: stdinPipe,
		stderr:    &stderrBuf,
	}, nil
}

// Client returns the JSON-RPC client attached to this process.
func (p *Process) Client() *Client {
	return p.client
}

// Stderr returns any captured stderr output from the subprocess.
func (p *Process) Stderr() string {
	return p.stderr.String()
}

// Close gracefully shuts down the process by closing the client's stdin
// (which signals EOF to the child) and waits for the process to exit.
func (p *Process) Close() error {
	// Close stdin to signal the child process to exit.
	if p.stdinPipe != nil {
		p.stdinPipe.Close()
	}
	// Wait for the process to finish.
	return p.cmd.Wait()
}
