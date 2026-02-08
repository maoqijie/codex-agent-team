package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"codex-agent-team/internal/codexrpc"
)

// Manager manages multiple Codex agent instances.
type Manager struct {
	mu       sync.RWMutex
	agents   map[string]*Instance
	codexBin string
	eventCh  chan AgentEvent
}

// Instance represents a running Codex agent instance.
type Instance struct {
	Config   AgentConfig
	Process  *codexrpc.Process
	Client   *codexrpc.Client
	ThreadID string
	mu       sync.Mutex // protects State
	State    AgentState
	doneCh   chan error // task completion signal
}

// NewManager creates a new Agent Manager.
func NewManager(codexBin string) *Manager {
	return &Manager{
		agents:   make(map[string]*Instance),
		codexBin: codexBin,
		eventCh:  make(chan AgentEvent, 100),
	}
}

// SpawnAgent starts a new Codex agent instance.
func (m *Manager) SpawnAgent(ctx context.Context, cfg AgentConfig) (*Instance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.agents[cfg.ID]; exists {
		return nil, fmt.Errorf("agent %s already exists", cfg.ID)
	}

	// Determine sandbox mode based on role
	sandbox := cfg.SandboxMode
	if sandbox == "" {
		switch cfg.Role {
		case RoleOrchestrator:
			sandbox = codexrpc.SandboxReadOnly
		case RoleWorker, RoleMerger:
			sandbox = codexrpc.SandboxWorkspaceWrite
		default:
			sandbox = codexrpc.SandboxReadOnly
		}
	}

	// Spawn the app-server process
	process, err := codexrpc.Spawn(ctx, codexrpc.SpawnOptions{
		BinaryPath: m.codexBin,
		ListenAddr: "stdio://",
	})
	if err != nil {
		return nil, fmt.Errorf("spawn process: %w", err)
	}

	client := process.Client()

	// Perform handshake
	if _, err := client.Initialize(ctx); err != nil {
		process.Close()
		return nil, fmt.Errorf("initialize: %w", err)
	}

	// Create thread
	threadResp, err := client.ThreadStart(ctx, codexrpc.ThreadStartParams{
		Cwd:                   &cfg.Cwd,
		Sandbox:               &sandbox,
		BaseInstructions:      &cfg.BaseInstructions,
		DeveloperInstructions: &cfg.DeveloperInstructions,
	})
	if err != nil {
		process.Close()
		return nil, fmt.Errorf("thread start: %w", err)
	}

	// Set up auto-approve handler for command/file approvals
	client.SetServerRequestHandler(m.createApprovalHandler(cfg.ID))

	// Set up notification handler for events
	client.SetNotificationHandler(m.createNotificationHandler(cfg.ID))

	instance := &Instance{
		Config:   cfg,
		Process:  process,
		Client:   client,
		ThreadID: threadResp.Thread.ID,
		State:    StateIdle,
		doneCh:   make(chan error, 1),
	}

	m.agents[cfg.ID] = instance

	// Emit agent spawned event
	m.eventCh <- AgentEvent{
		AgentID:   cfg.ID,
		EventType: "spawned",
		Data:      nil,
	}

	return instance, nil
}

// SendTask sends a task message to an agent.
func (m *Manager) SendTask(ctx context.Context, agentID string, message string) error {
	m.mu.Lock()
	instance, exists := m.agents[agentID]
	m.mu.Unlock()

	if !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}

	// Update state to running
	instance.mu.Lock()
	instance.State = StateRunning
	instance.mu.Unlock()

	// Send the task via TurnStart
	_, err := instance.Client.TurnStart(ctx, codexrpc.TurnStartParams{
		ThreadID: instance.ThreadID,
		Input: []codexrpc.UserInput{
			{
				Type: "text",
				Text: message,
			},
		},
	})
	if err != nil {
		instance.mu.Lock()
		instance.State = StateFailed
		instance.mu.Unlock()
		return fmt.Errorf("turn start: %w", err)
	}

	return nil
}

// StopAgent stops an agent instance.
func (m *Manager) StopAgent(agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance, exists := m.agents[agentID]
	if !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}

	if err := instance.Process.Close(); err != nil {
		return fmt.Errorf("close process: %w", err)
	}

	delete(m.agents, agentID)

	// Emit agent stopped event
	m.eventCh <- AgentEvent{
		AgentID:   agentID,
		EventType: "stopped",
		Data:      nil,
	}

	return nil
}

// Events returns the event channel for receiving agent events.
func (m *Manager) Events() <-chan AgentEvent {
	return m.eventCh
}

// createApprovalHandler creates a handler that auto-approves all requests.
func (m *Manager) createApprovalHandler(agentID string) codexrpc.ServerRequestHandler {
	return func(id codexrpc.RequestID, method string, params json.RawMessage) (json.RawMessage, error) {
		var decision string

		switch method {
		case "command/approval":
			decision = codexrpc.DecisionAccept
			resp := codexrpc.CommandApprovalResponse{Decision: decision}
			return json.Marshal(resp)
		case "fileChange/approval":
			decision = codexrpc.DecisionAccept
			resp := codexrpc.FileChangeApprovalResponse{Decision: decision}
			return json.Marshal(resp)
		default:
			return nil, fmt.Errorf("unknown request method: %s", method)
		}
	}
}

// createNotificationHandler creates a handler for server notifications.
func (m *Manager) createNotificationHandler(agentID string) codexrpc.NotificationHandler {
	return func(method string, params json.RawMessage) {
		m.mu.RLock()
		instance, exists := m.agents[agentID]
		m.mu.RUnlock()

		if !exists {
			return
		}

		switch method {
		case "turn/started":
			instance.mu.Lock()
			instance.State = StateRunning
			instance.mu.Unlock()
		case "turn/completed":
			// Parse the notification to check if it failed
			var notif codexrpc.TurnCompletedNotification
			if err := json.Unmarshal(params, &notif); err == nil {
				instance.mu.Lock()
				if notif.Turn.Status == "failed" {
					instance.State = StateFailed
					instance.mu.Unlock()
					select {
					case instance.doneCh <- fmt.Errorf("agent task failed"):
					default:
					}
				} else if notif.Turn.Status == "completed" {
					instance.State = StateCompleted
					instance.mu.Unlock()
					select {
					case instance.doneCh <- nil:
					default:
					}
				} else {
					instance.mu.Unlock()
				}
			}
		}

		// Forward the notification as an event
		m.eventCh <- AgentEvent{
			AgentID:   agentID,
			EventType: method,
			Data:      params,
		}
	}
}

// WaitForCompletion blocks until the agent's current task completes or the context is cancelled.
func (m *Manager) WaitForCompletion(ctx context.Context, agentID string) error {
	m.mu.RLock()
	instance, exists := m.agents[agentID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}

	select {
	case err := <-instance.doneCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
