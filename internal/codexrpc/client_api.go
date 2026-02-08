package codexrpc

import (
	"context"
	"encoding/json"
	"fmt"
)

// Initialize performs the handshake with the app-server.
// It sends the initialize request, then the initialized notification.
func (c *Client) Initialize(ctx context.Context) (*InitializeResponse, error) {
	params := InitializeParams{
		ClientInfo: ClientInfo{
			Name:    "codex-agent-team",
			Version: "0.1.0",
		},
	}

	raw, err := c.Call(ctx, "initialize", params)
	if err != nil {
		return nil, fmt.Errorf("initialize: %w", err)
	}

	var resp InitializeResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal initialize response: %w", err)
	}

	// Send the initialized notification to complete the handshake.
	if err := c.Notify("initialized", nil); err != nil {
		return nil, fmt.Errorf("send initialized notification: %w", err)
	}

	return &resp, nil
}

// ThreadStart creates a new conversation thread.
func (c *Client) ThreadStart(ctx context.Context, params ThreadStartParams) (*ThreadStartResponse, error) {
	raw, err := c.Call(ctx, "thread/start", params)
	if err != nil {
		return nil, fmt.Errorf("thread/start: %w", err)
	}
	var resp ThreadStartResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal thread/start response: %w", err)
	}
	return &resp, nil
}

// TurnStart sends user input and begins a new turn.
func (c *Client) TurnStart(ctx context.Context, params TurnStartParams) (*TurnStartResponse, error) {
	raw, err := c.Call(ctx, "turn/start", params)
	if err != nil {
		return nil, fmt.Errorf("turn/start: %w", err)
	}
	var resp TurnStartResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal turn/start response: %w", err)
	}
	return &resp, nil
}

// TurnInterrupt stops the current turn.
func (c *Client) TurnInterrupt(ctx context.Context, params TurnInterruptParams) error {
	_, err := c.Call(ctx, "turn/interrupt", params)
	return err
}
