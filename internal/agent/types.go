package agent

// Role represents the agent's role in the orchestration system.
type Role string

const (
	RoleOrchestrator Role = "orchestrator"
	RoleWorker       Role = "worker"
	RoleMerger       Role = "merger"
)

// AgentState represents the current state of an agent instance.
type AgentState string

const (
	StateIdle      AgentState = "idle"
	StateRunning   AgentState = "running"
	StateCompleted AgentState = "completed"
	StateFailed    AgentState = "failed"
)

// AgentConfig holds the configuration for spawning a new agent instance.
type AgentConfig struct {
	ID                    string
	Role                  Role
	Cwd                   string // 工作目录
	SandboxMode           string // "read-only" | "workspace-write"
	BaseInstructions      string
	DeveloperInstructions string
}

// AgentEvent represents an event emitted by an agent instance.
type AgentEvent struct {
	AgentID   string
	EventType string
	Data      []byte
}
