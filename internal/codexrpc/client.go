package codexrpc

import (
	"bufio"
	"encoding/json"
	"io"
	"sync"
	"sync/atomic"
)

// NotificationHandler is called for each server notification.
type NotificationHandler func(method string, params json.RawMessage)

// ServerRequestHandler handles server-initiated requests (e.g., approval).
// It must return a JSON-encodable result or an error.
type ServerRequestHandler func(id RequestID, method string, params json.RawMessage) (json.RawMessage, error)

// Client is a JSON-RPC 2.0 client for the Codex App Server.
// It communicates via line-delimited JSON over stdin/stdout of a child process.
type Client struct {
	stdin  io.Writer
	stdout *bufio.Reader

	mu           sync.Mutex
	writeMu      sync.Mutex
	nextID       atomic.Int64
	pendingCalls map[string]chan *rpcResult

	notifyHandler  NotificationHandler
	requestHandler ServerRequestHandler

	done chan struct{}
	err  error
}

type rpcResult struct {
	Result json.RawMessage
	Error  *RPCError
}

// NewClient creates a Client from existing reader/writer streams.
func NewClient(stdin io.Writer, stdout io.Reader) *Client {
	return &Client{
		stdin:        stdin,
		stdout:       bufio.NewReaderSize(stdout, 256*1024),
		pendingCalls: make(map[string]chan *rpcResult),
		done:         make(chan struct{}),
	}
}
