package codexrpc

import (
	"context"
	"encoding/json"
	"fmt"
)

// Call sends a JSON-RPC request and waits for the response.
func (c *Client) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := c.nextID.Add(1)
	idJSON, _ := json.Marshal(id)

	var paramsRaw *json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("marshal params: %w", err)
		}
		raw := json.RawMessage(b)
		paramsRaw = &raw
	}

	req := Request{
		ID:     idJSON,
		Method: method,
		Params: paramsRaw,
	}

	ch := make(chan *rpcResult, 1)
	idStr := string(idJSON)

	c.mu.Lock()
	c.pendingCalls[idStr] = ch
	c.mu.Unlock()

	if err := c.writeMessage(req); err != nil {
		c.mu.Lock()
		delete(c.pendingCalls, idStr)
		c.mu.Unlock()
		return nil, fmt.Errorf("write request: %w", err)
	}

	select {
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pendingCalls, idStr)
		c.mu.Unlock()
		return nil, ctx.Err()
	case result := <-ch:
		if result.Error != nil {
			return nil, fmt.Errorf("RPC error %d: %s", result.Error.Code, result.Error.Message)
		}
		return result.Result, nil
	case <-c.done:
		return nil, fmt.Errorf("client closed")
	}
}

// Notify sends a JSON-RPC notification (no response expected).
func (c *Client) Notify(method string, params any) error {
	var paramsRaw *json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("marshal params: %w", err)
		}
		raw := json.RawMessage(b)
		paramsRaw = &raw
	}

	notif := Notification{
		Method: method,
		Params: paramsRaw,
	}
	return c.writeMessage(notif)
}

// writeMessage serializes and writes a JSON-RPC message as a single line.
func (c *Client) writeMessage(msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_, err = c.stdin.Write(data)
	return err
}

// writeResponse sends a JSON-RPC response to a server request.
func (c *Client) writeResponse(id RequestID, result json.RawMessage, rpcErr *RPCError) {
	if rpcErr != nil {
		c.writeMessage(ErrorResponse{ID: id, Error: *rpcErr})
	} else {
		c.writeMessage(Response{ID: id, Result: result})
	}
}
