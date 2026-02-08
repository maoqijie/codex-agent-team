package codexrpc

import "encoding/json"

// --- Turn ---

// TurnStartParams sends user input and begins a turn.
type TurnStartParams struct {
	ThreadID       string      `json:"threadId"`
	Input          []UserInput `json:"input"`
	ApprovalPolicy *string     `json:"approvalPolicy,omitempty"`
	Model          *string     `json:"model,omitempty"`
}

// UserInput represents a single user input item.
type UserInput struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

type TurnStartResponse struct {
	Turn Turn `json:"turn"`
}

type Turn struct {
	ID     string          `json:"id"`
	Status string          `json:"status"` // "completed"|"interrupted"|"failed"|"inProgress"
	Error  *TurnError      `json:"error,omitempty"`
	Items  json.RawMessage `json:"items,omitempty"`
}

type TurnError struct {
	Message           string  `json:"message"`
	AdditionalDetails *string `json:"additionalDetails,omitempty"`
}

// TurnInterruptParams stops the current turn.
type TurnInterruptParams struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
}

// --- Notifications (server → client) ---

// AgentMessageDelta streams agent text output.
type AgentMessageDelta struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
	ItemID   string `json:"itemId"`
	Delta    string `json:"delta"`
}

// TurnStartedNotification is emitted when a turn begins.
type TurnStartedNotification struct {
	ThreadID string `json:"threadId"`
	Turn     Turn   `json:"turn"`
}

// TurnCompletedNotification is emitted when a turn finishes.
type TurnCompletedNotification struct {
	ThreadID string `json:"threadId"`
	Turn     Turn   `json:"turn"`
}

// ItemStartedNotification is emitted when an item begins.
type ItemStartedNotification struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
	ItemID   string `json:"itemId"`
}

// ItemCompletedNotification is emitted when an item finishes.
type ItemCompletedNotification struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
	ItemID   string `json:"itemId"`
}

// --- Approval requests (server → client) ---

// CommandApprovalParams is sent when the agent wants to execute a command.
type CommandApprovalParams struct {
	ThreadID string  `json:"threadId"`
	TurnID   string  `json:"turnId"`
	ItemID   string  `json:"itemId"`
	Command  *string `json:"command,omitempty"`
	Cwd      *string `json:"cwd,omitempty"`
	Reason   *string `json:"reason,omitempty"`
}

// CommandApprovalResponse is the client's decision.
// Valid values: "accept", "acceptForSession", "decline"
type CommandApprovalResponse struct {
	Decision string `json:"decision"`
}

// FileChangeApprovalParams is sent when the agent wants to modify a file.
type FileChangeApprovalParams struct {
	ThreadID string  `json:"threadId"`
	TurnID   string  `json:"turnId"`
	ItemID   string  `json:"itemId"`
	Reason   *string `json:"reason,omitempty"`
}

// FileChangeApprovalResponse is the client's decision.
// Valid values: "accept", "acceptForSession", "decline", "cancel"
type FileChangeApprovalResponse struct {
	Decision string `json:"decision"`
}
