package codexrpc

import "encoding/json"

// SetNotificationHandler sets the handler for server notifications.
func (c *Client) SetNotificationHandler(h NotificationHandler) {
	c.notifyHandler = h
}

// SetServerRequestHandler sets the handler for server-initiated requests.
func (c *Client) SetServerRequestHandler(h ServerRequestHandler) {
	c.requestHandler = h
}

// Start begins reading messages from stdout. Must be called after NewClient.
func (c *Client) Start() {
	go c.readLoop()
}

// Done returns a channel that is closed when the client stops.
func (c *Client) Done() <-chan struct{} {
	return c.done
}

// Err returns the error that caused the client to stop, if any.
func (c *Client) Err() error {
	return c.err
}

// readLoop reads JSONL messages from stdout and dispatches them.
func (c *Client) readLoop() {
	defer close(c.done)
	for {
		line, err := c.stdout.ReadBytes('\n')
		if err != nil {
			c.err = err
			// Drain all pending calls with error
			c.mu.Lock()
			errResult := &rpcResult{Error: &RPCError{Code: -1, Message: "client closed"}}
			for id, ch := range c.pendingCalls {
				// Non-blocking send to avoid deadlock
				select {
				case ch <- errResult:
				default:
				}
				delete(c.pendingCalls, id)
			}
			c.mu.Unlock()
			return
		}
		if len(line) <= 1 {
			continue // skip empty lines
		}
		c.dispatch(line)
	}
}

// dispatch routes an incoming JSON-RPC message based on its fields.
func (c *Client) dispatch(line []byte) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(line, &raw); err != nil {
		return
	}

	_, hasID := raw["id"]
	_, hasMethod := raw["method"]
	_, hasResult := raw["result"]
	_, hasError := raw["error"]

	switch {
	case hasResult && hasID:
		// Success response
		var resp Response
		if json.Unmarshal(line, &resp) != nil {
			return
		}
		c.resolveCall(resp.ID, &rpcResult{Result: resp.Result})

	case hasError && hasID:
		// Error response
		var errResp ErrorResponse
		if json.Unmarshal(line, &errResp) != nil {
			return
		}
		c.resolveCall(errResp.ID, &rpcResult{Error: &errResp.Error})

	case hasMethod && hasID:
		// Server request (needs response)
		var req ServerRequest
		if json.Unmarshal(line, &req) != nil {
			return
		}
		go c.handleServerRequest(req)

	case hasMethod && !hasID:
		// Notification
		var notif Notification
		if json.Unmarshal(line, &notif) != nil {
			return
		}
		if c.notifyHandler != nil {
			var params json.RawMessage
			if notif.Params != nil {
				params = *notif.Params
			}
			c.notifyHandler(notif.Method, params)
		}
	}
}

func (c *Client) resolveCall(id RequestID, result *rpcResult) {
	idStr := string(id)
	c.mu.Lock()
	ch, ok := c.pendingCalls[idStr]
	if ok {
		delete(c.pendingCalls, idStr)
	}
	c.mu.Unlock()
	if ok {
		// Non-blocking send to avoid deadlock if receiver has already returned.
		select {
		case ch <- result:
		default:
			// Channel full or receiver gone, drop the result.
		}
	}
}

func (c *Client) handleServerRequest(req ServerRequest) {
	if c.requestHandler == nil {
		c.writeResponse(req.ID, nil, &RPCError{
			Code:    -32601,
			Message: "no handler registered",
		})
		return
	}

	var params json.RawMessage
	if req.Params != nil {
		params = *req.Params
	}

	result, err := c.requestHandler(req.ID, req.Method, params)
	if err != nil {
		c.writeResponse(req.ID, nil, &RPCError{
			Code:    -1,
			Message: err.Error(),
		})
		return
	}
	c.writeResponse(req.ID, result, nil)
}
