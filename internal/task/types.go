package task

import "time"

// TaskStatus represents the current status of a task.
type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusReady     TaskStatus = "ready"
	StatusRunning   TaskStatus = "running"
	StatusCompleted TaskStatus = "completed"
	StatusFailed    TaskStatus = "failed"
	StatusCancelled TaskStatus = "cancelled"
)

// Task represents a single task in the DAG.
type Task struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	Status       TaskStatus `json:"status"`
	DependsOn    []string   `json:"dependsOn"`    // 依赖的任务 ID 列表
	AgentID      string     `json:"agentId"`      // 分配的代理 ID
	WorktreePath string     `json:"worktreePath"` // Git worktree 路径
	BranchName   string     `json:"branchName"`   // Git 分支名
	CreatedAt    time.Time  `json:"createdAt"`
	StartedAt    *time.Time `json:"startedAt,omitempty"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
	Error        string     `json:"error,omitempty"`
	Output       []string   `json:"output"` // 代理输出
}
