// Package codexrpc provides a Go client for the Codex App Server JSON-RPC protocol.
//
// The protocol uses line-delimited JSON (JSONL) over stdio.
// IMPORTANT: The "jsonrpc":"2.0" header is OMITTED on the wire.
package codexrpc

import "encoding/json"

// --- JSON-RPC 2.0 base types (without jsonrpc field) ---

// RequestID can be a string or integer.
type RequestID = json.RawMessage

// Request is a JSON-RPC request from client to server.
type Request struct {
	ID     RequestID        `json:"id"`
	Method string           `json:"method"`
	Params *json.RawMessage `json:"params,omitempty"`
}

// Response is a successful JSON-RPC response.
type Response struct {
	ID     RequestID       `json:"id"`
	Result json.RawMessage `json:"result"`
}

// ErrorResponse is a JSON-RPC error response.
type ErrorResponse struct {
	ID    RequestID `json:"id"`
	Error RPCError  `json:"error"`
}

// RPCError is the error body in a JSON-RPC error response.
type RPCError struct {
	Code    int64            `json:"code"`
	Message string           `json:"message"`
	Data    *json.RawMessage `json:"data,omitempty"`
}

// Notification is a JSON-RPC notification (no id field).
type Notification struct {
	Method string           `json:"method"`
	Params *json.RawMessage `json:"params,omitempty"`
}

// ServerRequest is a JSON-RPC request from server to client (e.g., approval).
// It has both id and method fields.
type ServerRequest struct {
	ID     RequestID        `json:"id"`
	Method string           `json:"method"`
	Params *json.RawMessage `json:"params,omitempty"`
}
