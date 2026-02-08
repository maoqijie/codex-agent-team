package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"codex-agent-team/internal/agent"
	"codex-agent-team/internal/task"
	"codex-agent-team/internal/worktree"
)

// Session represents a complete task execution session.
type Session struct {
	ID           string
	UserTask     string
	RepoPath     string
	Status       SessionStatus
	DAG          *task.DAG
	Orchestrator *agent.Orchestrator
	Merger       *agent.Merger
	Executor     *task.Executor
	CreatedAt    time.Time
	StartedAt    *time.Time
	CompletedAt  *time.Time

	mu          sync.RWMutex
	agentMgr    *agent.Manager
	worktreeMgr *worktree.Manager
}

// SessionStatus represents the current status of a session.
type SessionStatus string

const (
	StatusCreated     SessionStatus = "created"
	StatusDecomposing SessionStatus = "decomposing"
	StatusReady       SessionStatus = "ready"
	StatusRunning     SessionStatus = "running"
	StatusCompleted   SessionStatus = "completed"
	StatusFailed      SessionStatus = "failed"
	StatusMerging     SessionStatus = "merging"
)

// Manager manages multiple sessions.
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	agentMgr *agent.Manager
	wtMgr    *worktree.Manager
}

// NewManager creates a new Session Manager.
func NewManager(codexBin, repoPath string) *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
		agentMgr: agent.NewManager(codexBin),
		wtMgr:    worktree.NewManager(repoPath),
	}
}

// Create creates a new session for a user task.
func (m *Manager) Create(ctx context.Context, userTask string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := fmt.Sprintf("session-%d", time.Now().UnixNano())

	sess := &Session{
		ID:       id,
		UserTask: userTask,
		RepoPath: m.wtMgr.GetRepoPath(),
		Status:   StatusCreated,
		DAG:      task.NewDAG(),
		CreatedAt: time.Now(),
		agentMgr:    m.agentMgr,
		worktreeMgr: m.wtMgr,
	}

	sess.Orchestrator = agent.NewOrchestrator(m.agentMgr)
	sess.Merger = agent.NewMerger(m.agentMgr, m.wtMgr)

	m.sessions[id] = sess
	return sess, nil
}

// Decompose decomposes the user task into sub-tasks.
func (s *Session) Decompose(ctx context.Context) error {
	s.mu.Lock()
	s.Status = StatusDecomposing
	now := time.Now()
	s.StartedAt = &now
	s.mu.Unlock()

	decomp, err := s.Orchestrator.Decompose(ctx, s.RepoPath, s.UserTask)
	if err != nil {
		s.mu.Lock()
		s.Status = StatusFailed
		s.mu.Unlock()
		return fmt.Errorf("decompose: %w", err)
	}

	// Convert suggestions to Tasks and add to DAG
	for _, sug := range decomp.Tasks {
		t := &task.Task{
			ID:          sug.ID,
			Title:       sug.Title,
			Description: sug.Description,
			Status:      task.StatusPending,
			DependsOn:   sug.DependsOn,
			CreatedAt:   time.Now(),
		}
		if err := s.DAG.AddTask(t); err != nil {
			return fmt.Errorf("add task: %w", err)
		}
	}

	s.mu.Lock()
	s.Status = StatusReady
	s.mu.Unlock()

	return nil
}

// Execute starts executing the task DAG.
func (s *Session) Execute(ctx context.Context) error {
	s.mu.Lock()
	s.Status = StatusRunning
	s.mu.Unlock()

	s.Executor = task.NewExecutor(s.DAG, s.agentMgr, s.worktreeMgr, 3)

	if err := s.Executor.Run(ctx); err != nil {
		s.mu.Lock()
		s.Status = StatusFailed
		s.mu.Unlock()
		return err
	}

	s.mu.Lock()
	s.Status = StatusMerging
	s.mu.Unlock()

	return nil
}

// Merge merges all worktree branches back to main.
func (s *Session) Merge(ctx context.Context) error {
	// Get all completed tasks
	tasks := s.DAG.GetTasks()

	branchMap := make(map[string]string)
	var taskIDs []string
	for _, t := range tasks {
		if t.Status == task.StatusCompleted && t.BranchName != "" {
			taskIDs = append(taskIDs, t.ID)
			branchMap[t.ID] = t.BranchName
		}
	}

	plan := s.Merger.CreateMergePlan(taskIDs, branchMap)

	result, err := s.Merger.Merge(ctx, s.RepoPath, plan)
	if err != nil {
		s.mu.Lock()
		s.Status = StatusFailed
		s.mu.Unlock()
		return fmt.Errorf("merge: %w", err)
	}

	if !result.Success {
		s.mu.Lock()
		s.Status = StatusFailed
		s.mu.Unlock()
		return fmt.Errorf("merge failed for branches: %v", result.FailedBranches)
	}

	s.mu.Lock()
	s.Status = StatusCompleted
	now := time.Now()
	s.CompletedAt = &now
	s.mu.Unlock()

	return nil
}

// Get retrieves a session by ID.
func (m *Manager) Get(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sess, ok := m.sessions[id]
	return sess, ok
}

// CreateWithPath creates a new session for a user task with a specific repo path.
func (m *Manager) CreateWithPath(ctx context.Context, userTask, repoPath string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := fmt.Sprintf("session-%d", time.Now().UnixNano())

	// Create a new worktree manager for this session's repo
	wtMgr := worktree.NewManager(repoPath)

	sess := &Session{
		ID:       id,
		UserTask: userTask,
		RepoPath: repoPath,
		Status:   StatusCreated,
		DAG:      task.NewDAG(),
		CreatedAt: time.Now(),
		agentMgr:    m.agentMgr,
		worktreeMgr: wtMgr,
	}

	sess.Orchestrator = agent.NewOrchestrator(m.agentMgr)
	sess.Merger = agent.NewMerger(m.agentMgr, wtMgr)

	m.sessions[id] = sess
	return sess, nil
}

// ListAll returns all sessions.
func (m *Manager) ListAll() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}
