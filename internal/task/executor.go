package task

import (
	"context"
	"fmt"
	"sync"
	"time"

	"codex-agent-team/internal/agent"
	"codex-agent-team/internal/codexrpc"
	"codex-agent-team/internal/worktree"
)

// Executor executes a DAG of tasks using multiple agents.
type Executor struct {
	dag         *DAG
	agentMgr    *agent.Manager
	worktreeMgr *worktree.Manager
	maxParallel int
	eventCh     chan ExecutionEvent
}

// ExecutionEvent represents an event during task execution.
type ExecutionEvent struct {
	TaskID    string
	EventType string // "started", "completed", "failed", "output"
	Data      interface{}
}

// NewExecutor creates a new Executor.
func NewExecutor(dag *DAG, agentMgr *agent.Manager, wtMgr *worktree.Manager, maxParallel int) *Executor {
	if maxParallel <= 0 {
		maxParallel = 1
	}
	return &Executor{
		dag:         dag,
		agentMgr:    agentMgr,
		worktreeMgr: wtMgr,
		maxParallel: maxParallel,
		eventCh:     make(chan ExecutionEvent, 256),
	}
}

// Events returns the event channel.
func (e *Executor) Events() <-chan ExecutionEvent {
	return e.eventCh
}

// Run executes the DAG until all tasks complete or fail.
func (e *Executor) Run(ctx context.Context) error {
	sem := make(chan struct{}, e.maxParallel)
	var wg sync.WaitGroup

	for {
		if e.dag.AllCompleted() {
			break
		}

		if e.dag.HasFailed() {
			return fmt.Errorf("task execution failed")
		}

		ready := e.dag.ReadyTasks()
		if len(ready) == 0 {
			// Wait for a running task to complete
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
				continue
			}
		}

		for _, task := range ready {
			// Update status to running
			e.dag.UpdateStatus(task.ID, StatusRunning)
			task.Status = StatusRunning
			now := time.Now()
			task.StartedAt = &now

			// Acquire semaphore
			sem <- struct{}{}
			wg.Add(1)

			go func(t *Task) {
				defer wg.Done()
				defer func() { <-sem }()

				err := e.executeTask(ctx, t)
				if err != nil {
					t.Status = StatusFailed
					t.Error = err.Error()
					e.eventCh <- ExecutionEvent{
						TaskID:    t.ID,
						EventType: "failed",
						Data:      err.Error(),
					}
				} else {
					t.Status = StatusCompleted
					now := time.Now()
					t.CompletedAt = &now
					e.eventCh <- ExecutionEvent{
						TaskID:    t.ID,
						EventType: "completed",
					}
				}
				e.dag.UpdateStatus(t.ID, t.Status)
			}(task)
		}
	}

	wg.Wait()
	return nil
}

// executeTask executes a single task using an agent.
func (e *Executor) executeTask(ctx context.Context, t *Task) error {
	e.eventCh <- ExecutionEvent{
		TaskID:    t.ID,
		EventType: "started",
	}

	// Create worktree for this task
	if t.WorktreePath == "" {
		t.WorktreePath = e.worktreeMgr.GetPath(t.ID)
	}
	if t.BranchName == "" {
		t.BranchName = "task-" + t.ID
	}

	_, err := e.worktreeMgr.Create(ctx, t.BranchName, "")
	if err != nil {
		return fmt.Errorf("create worktree: %w", err)
	}

	// Spawn agent for this task
	agentCfg := agent.AgentConfig{
		ID:      "agent-" + t.ID,
		Role:    agent.RoleWorker,
		Cwd:     t.WorktreePath,
		SandboxMode: codexrpc.SandboxWorkspaceWrite,
	}

	_, err = e.agentMgr.SpawnAgent(ctx, agentCfg)
	if err != nil {
		return fmt.Errorf("spawn agent: %w", err)
	}

	// Send task to agent
	err = e.agentMgr.SendTask(ctx, agentCfg.ID, t.Description)
	if err != nil {
		return fmt.Errorf("send task: %w", err)
	}

	return nil
}
