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
	// Create cancellable context for cascading cancellation
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	sem := make(chan struct{}, e.maxParallel)
	var wg sync.WaitGroup

	for {
		if e.dag.AllCompleted() {
			break
		}

		if e.dag.HasFailed() {
			cancel() // Cancel all running tasks
			break
		}

		ready := e.dag.ReadyTasks()
		if len(ready) == 0 {
			// Wait for a running task to complete
			select {
			case <-runCtx.Done():
				return runCtx.Err()
			case <-time.After(100 * time.Millisecond):
				continue
			}
		}

		for _, task := range ready {
			// Update status to running via DAG (thread-safe)
			e.dag.UpdateStatus(task.ID, StatusRunning)

			// Acquire semaphore
			sem <- struct{}{}
			wg.Add(1)

			go func(t *Task) {
				defer wg.Done()
				defer func() { <-sem }()

				err := e.executeTask(runCtx, t)
				if err != nil {
					e.dag.SetTaskFailed(t.ID, err.Error())
					e.eventCh <- ExecutionEvent{
						TaskID:    t.ID,
						EventType: "failed",
						Data:      err.Error(),
					}
				} else {
					e.dag.SetTaskCompleted(t.ID)
					e.eventCh <- ExecutionEvent{
						TaskID:    t.ID,
						EventType: "completed",
					}
				}
			}(task)
		}
	}

	wg.Wait()

	if e.dag.HasFailed() {
		return fmt.Errorf("task execution failed")
	}
	return nil
}

// executeTask executes a single task using an agent.
func (e *Executor) executeTask(ctx context.Context, t *Task) error {
	agentID := "agent-" + t.ID

	e.eventCh <- ExecutionEvent{
		TaskID:    t.ID,
		EventType: "started",
	}

	// 1. Prepare branch name
	if t.BranchName == "" {
		t.BranchName = "task-" + t.ID
	}

	// 2. Create worktree (path derived from branchName inside Create)
	wt, err := e.worktreeMgr.Create(ctx, t.BranchName, "")
	if err != nil {
		return fmt.Errorf("create worktree: %w", err)
	}
	t.WorktreePath = wt.Path
	t.BaseCommit = wt.Commit

	// 3. Merge all dependency task branches
	depBranches := e.dag.GetDependencyBranches(t.ID)
	for _, depBranch := range depBranches {
		commitSHA, mergeErr := e.worktreeMgr.Merge(ctx, t.WorktreePath, depBranch)
		if mergeErr != nil {
			e.cleanupWorktree(t.WorktreePath)
			return fmt.Errorf("merge dependency branch %s: %w", depBranch, mergeErr)
		}
		if commitSHA != "" {
			t.MergedCommits = append(t.MergedCommits, commitSHA)
		}
	}

	// 4. Spawn agent for this task
	agentCfg := agent.AgentConfig{
		ID:          agentID,
		Role:        agent.RoleWorker,
		Cwd:         t.WorktreePath,
		SandboxMode: codexrpc.SandboxWorkspaceWrite,
	}

	_, err = e.agentMgr.SpawnAgent(ctx, agentCfg)
	if err != nil {
		e.cleanupWorktree(t.WorktreePath)
		return fmt.Errorf("spawn agent: %w", err)
	}
	t.AgentID = agentID

	// 5. Send task to agent
	err = e.agentMgr.SendTask(ctx, agentID, t.Description)
	if err != nil {
		e.cleanup(agentID, t.WorktreePath)
		return fmt.Errorf("send task: %w", err)
	}

	// 6. Wait for agent to complete
	err = e.agentMgr.WaitForCompletion(ctx, agentID)
	if err != nil {
		e.cleanup(agentID, t.WorktreePath)
		return fmt.Errorf("agent execution: %w", err)
	}

	// 7. Commit agent's changes
	commitMsg := fmt.Sprintf("Task %s: %s", t.ID, t.Title)
	commitSHA, err := e.worktreeMgr.CommitChanges(ctx, t.WorktreePath, commitMsg)
	if err != nil {
		e.cleanup(agentID, t.WorktreePath)
		return fmt.Errorf("commit changes: %w", err)
	}
	if commitSHA != "" {
		t.ResultCommit = commitSHA
		e.dag.UpdateTaskResult(t.ID, commitSHA)
	}

	// 8. Cleanup: stop agent (worktree kept for merge)
	_ = e.agentMgr.StopAgent(agentID)

	return nil
}

// cleanup stops the agent and removes the worktree on failure.
func (e *Executor) cleanup(agentID string, worktreePath string) {
	_ = e.agentMgr.StopAgent(agentID)
	_ = e.worktreeMgr.Remove(context.Background(), worktreePath)
}

// cleanupWorktree removes worktree only (before agent is spawned).
func (e *Executor) cleanupWorktree(worktreePath string) {
	_ = e.worktreeMgr.Remove(context.Background(), worktreePath)
}
