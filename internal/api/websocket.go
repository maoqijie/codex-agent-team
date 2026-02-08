package api

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"nhooyr.io/websocket"
)

// Event represents a server-sent event.
type Event struct {
	Type string      `json:"type"`
	Data any         `json:"data"`
}

// Client represents a WebSocket client connection.
type Client struct {
	SessionID string
	Conn      *websocket.Conn
	Send      chan Event
	hub       *Hub
	ctx       context.Context
}

// NewClient creates a new WebSocket client.
func NewClient(sessionID string, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		SessionID: sessionID,
		Conn:      conn,
		Send:      make(chan Event, 256),
		hub:       hub,
		ctx:       context.Background(),
	}
}

// ReadLoop reads messages from the WebSocket connection.
func (c *Client) ReadLoop() {
	defer func() {
		c.hub.Unregister(c)
		c.Conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		_, _, err := c.Conn.Read(c.ctx)
		if err != nil {
			break
		}
		// We don't expect client messages, just keep the connection alive
	}
}

// WriteLoop writes events to the WebSocket connection.
func (c *Client) WriteLoop() {
	defer c.Conn.Close(websocket.StatusNormalClosure, "")

	for event := range c.Send {
		data, err := json.Marshal(event)
		if err != nil {
			log.Printf("Failed to marshal event: %v", err)
			continue
		}

		err = c.Conn.Write(c.ctx, websocket.MessageText, data)
		if err != nil {
			log.Printf("Failed to write to WebSocket: %v", err)
			break
		}
	}
}

// Hub manages WebSocket clients and broadcasts events.
type Hub struct {
	// Registered clients by session ID
	clients map[string][]*Client

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast events to a session
	broadcast chan broadcastMsg

	mu sync.RWMutex
}

type broadcastMsg struct {
	SessionID string
	Event     Event
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		clients:   make(map[string][]*Client),
		register:  make(chan *Client),
		unregister: make(chan *Client),
		broadcast: make(chan broadcastMsg, 256),
	}
}

// Run starts the hub's event loop.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.SessionID] = append(h.clients[client.SessionID], client)
			h.mu.Unlock()
			log.Printf("Client registered for session: %s", client.SessionID)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.SessionID]; ok {
				// Remove this client from the list
				for i, c := range clients {
					if c == client {
						h.clients[client.SessionID] = append(clients[:i], clients[i+1:]...)
						break
					}
				}
				// Clean up empty slices
				if len(h.clients[client.SessionID]) == 0 {
					delete(h.clients, client.SessionID)
				}
			}
			h.mu.Unlock()
			close(client.Send)
			log.Printf("Client unregistered for session: %s", client.SessionID)

		case msg := <-h.broadcast:
			h.mu.RLock()
			clients := h.clients[msg.SessionID]
			h.mu.RUnlock()

			for _, client := range clients {
				select {
				case client.Send <- msg.Event:
				default:
					// Client channel is full, close it
					h.Unregister(client)
				}
			}
		}
	}
}

// Register adds a new client.
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast sends an event to all clients subscribed to a session.
func (h *Hub) Broadcast(sessionID string, event Event) {
	h.broadcast <- broadcastMsg{
		SessionID: sessionID,
		Event:     event,
	}
}
