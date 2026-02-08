package agent

import (
	"context"
	"fmt"
	"strings"

	"codex-agent-team/internal/codexrpc"
	"codex-agent-team/internal/worktree"
)

// Merger handles merging worktree branches back to main.
type Merger struct {
	agentMgr    *Manager
	worktreeMgr *worktree.Manager
}

// NewMerger creates a new Merger.
func NewMerger(agentMgr *Manager, wtMgr *worktree.Manager) *Merger {
	return &Merger{
		agentMgr:    agentMgr,
		worktreeMgr: wtMgr,
	}
}

// MergeResult represents the result of a merge operation.
type MergeResult struct {
	Success         bool     `json:"success"`
	MergedCount     int      `json:"mergedCount"`
	FailedBranches  []string `json:"failedBranches,omitempty"`
	Conflicts       []string `json:"conflicts,omitempty"`
	ResolvedByAgent []string `json:"resolvedByAgent,omitempty"`
	MergeCommit     string   `json:"mergeCommit,omitempty"`
}

// MergePlan defines the order and strategy for merging.
type MergePlan struct {
	Branches     []string `json:"branches"`     // Branch names in merge order
	Strategy     string   `json:"strategy"`     // "sequential", "octopus", "auto"
	TargetBranch string   `json:"targetBranch"` // Usually "main" or current branch
}

// Merge executes the merge plan using a Codex agent for conflict resolution.
func (m *Merger) Merge(ctx context.Context, repoPath string, plan *MergePlan) (*MergeResult, error) {
	switch plan.Strategy {
	case "sequential", "auto":
		// Sequential merge with agent-assisted conflict resolution
		return m.mergeSequentialWithAgent(ctx, repoPath, plan)
	case "octopus":
		// Octopus merge (attempt all at once, fall back to sequential if conflicts)
		return m.mergeOctopusWithFallback(ctx, repoPath, plan)
	default:
		return nil, fmt.Errorf("unknown merge strategy: %s", plan.Strategy)
	}
}

// mergeSequentialWithAgent merges branches one by one with agent conflict resolution.
func (m *Merger) mergeSequentialWithAgent(ctx context.Context, repoPath string, plan *MergePlan) (*MergeResult, error) {
	result := &MergeResult{Success: true}

	// Switch to target branch first
	if err := m.checkoutBranch(ctx, repoPath, plan.TargetBranch); err != nil {
		return nil, fmt.Errorf("checkout target branch: %w", err)
	}

	// Spawn a single Merger agent for the entire process
	agentCfg := AgentConfig{
		ID:            "merger-" + GenerateID(),
		Role:          RoleMerger,
		Cwd:           repoPath,
		SandboxMode:   codexrpc.SandboxWorkspaceWrite,
		BaseInstructions: m.getMergeInstructions(plan),
	}

	instance, err := m.agentMgr.SpawnAgent(ctx, agentCfg)
	if err != nil {
		return nil, fmt.Errorf("spawn merger agent: %w", err)
	}
	defer m.agentMgr.StopAgent(instance.Config.ID)

	for _, branch := range plan.Branches {
		// Attempt merge
		commitSHA, err := m.worktreeMgr.Merge(ctx, repoPath, branch)
		if err == nil {
			// Success
			result.MergedCount++
			if result.MergeCommit == "" {
				result.MergeCommit = commitSHA
			}
			continue
		}

		// Check if there are conflicts
		hasConflicts, conflictFiles, err := m.worktreeMgr.HasConflicts(ctx, repoPath)
		if err != nil {
			result.FailedBranches = append(result.FailedBranches, branch)
			result.Success = false
			m.worktreeMgr.AbortMerge(ctx, repoPath)
			continue
		}

		if hasConflicts {
			// Try to resolve conflicts with the agent
			resolved, err := m.resolveConflictsWithAgent(ctx, instance.Config.ID, conflictFiles)
			if err != nil {
				result.FailedBranches = append(result.FailedBranches, branch)
				result.Conflicts = append(result.Conflicts, conflictFiles...)
				result.Success = false
				m.worktreeMgr.AbortMerge(ctx, repoPath)
				continue
			}

			if resolved {
				// Commit the resolved merge
				commitMsg := fmt.Sprintf("Merge %s (conflicts resolved by agent)", branch)
				commitSHA, err := m.worktreeMgr.CommitChanges(ctx, repoPath, commitMsg)
				if err != nil {
					result.FailedBranches = append(result.FailedBranches, branch)
					result.Success = false
					m.worktreeMgr.AbortMerge(ctx, repoPath)
					continue
				}
				result.MergedCount++
				result.ResolvedByAgent = append(result.ResolvedByAgent, branch)
				if result.MergeCommit == "" {
					result.MergeCommit = commitSHA
				}
			} else {
				result.FailedBranches = append(result.FailedBranches, branch)
				result.Conflicts = append(result.Conflicts, conflictFiles...)
				result.Success = false
				m.worktreeMgr.AbortMerge(ctx, repoPath)
			}
		} else {
			// Other error, not conflicts
			result.FailedBranches = append(result.FailedBranches, branch)
			result.Success = false
		}
	}

	return result, nil
}

// mergeOctopusWithFallback attempts octopus merge, falls back to sequential.
func (m *Merger) mergeOctopusWithFallback(ctx context.Context, repoPath string, plan *MergePlan) (*MergeResult, error) {
	// Switch to target branch first
	if err := m.checkoutBranch(ctx, repoPath, plan.TargetBranch); err != nil {
		return nil, fmt.Errorf("checkout target branch: %w", err)
	}

	// Try octopus merge
	commitSHA, err := m.worktreeMgr.OctopusMerge(ctx, repoPath, plan.Branches)
	if err == nil {
		return &MergeResult{
			Success:     true,
			MergedCount: len(plan.Branches),
			MergeCommit: commitSHA,
		}, nil
	}

	// Octopus failed, check for conflicts and fall back to sequential
	hasConflicts, _, err := m.worktreeMgr.HasConflicts(ctx, repoPath)
	if err == nil && hasConflicts {
		// Abort and fall back to sequential with agent
		_ = m.worktreeMgr.AbortMerge(ctx, repoPath)
		return m.mergeSequentialWithAgent(ctx, repoPath, plan)
	}

	// No conflicts but octopus failed (e.g., unrelated histories)
	// Fall back to sequential
	_ = m.worktreeMgr.AbortMerge(ctx, repoPath)
	return m.mergeSequentialWithAgent(ctx, repoPath, plan)
}

// resolveConflictsWithAgent asks the Merger agent to resolve conflicts.
func (m *Merger) resolveConflictsWithAgent(ctx context.Context, agentID string, conflictFiles []string) (bool, error) {
	if len(conflictFiles) == 0 {
		return false, fmt.Errorf("no conflict files to resolve")
	}

	prompt := fmt.Sprintf(`Please resolve the merge conflicts in the following files:
%s

For each conflict:
1. Open the file and examine both sides
2. Understand the intent of both changes
3. Create a merged version that preserves functionality from both sides
4. Use git add to mark each file as resolved

After resolving all conflicts, report "DONE". If you cannot resolve a conflict, report "FAILED: <reason>".`,
		strings.Join(conflictFiles, "\n"))

	err := m.agentMgr.SendTask(ctx, agentID, prompt)
	if err != nil {
		return false, fmt.Errorf("send task: %w", err)
	}

	err = m.agentMgr.WaitForCompletion(ctx, agentID)
	if err != nil {
		return false, fmt.Errorf("wait for completion: %w", err)
	}

	// Check if agent reported success
	output := m.agentMgr.GetOutput(agentID)
	if strings.Contains(output, "DONE") {
		return true, nil
	}

	return false, nil
}

// checkoutBranch switches to the specified branch.
func (m *Merger) checkoutBranch(ctx context.Context, repoPath, branch string) error {
	// For now, assume we're always on main in the repo root
	// In production, you'd run: git checkout <branch>
	return nil
}

// getMergeInstructions returns instructions for the merge agent.
func (m *Merger) getMergeInstructions(plan *MergePlan) string {
	return fmt.Sprintf(`You are a merge assistant. Your job is to help merge branches into %s.

When conflicts occur:
1. Analyze both sides carefully
2. Prefer the version that preserves functionality
3. If both changes are valid but incompatible, keep both with conditional logic
4. Never delete code without clear reason
5. Add comments explaining merge decisions

Merge strategy: %s

You will be asked to resolve conflicts as they arise. Focus on creating a clean, functional merge.`, plan.TargetBranch, plan.Strategy)
}

// CreateMergePlan creates a merge plan from completed tasks.
func (m *Merger) CreateMergePlan(taskIDs []string, branchMap map[string]string) *MergePlan {
	branches := make([]string, 0, len(taskIDs))
	for _, id := range taskIDs {
		if branch, ok := branchMap[id]; ok {
			branches = append(branches, branch)
		}
	}

	strategy := "sequential"
	if len(branches) > 3 {
		// Use octopus for many branches
		strategy = "octopus"
	}

	return &MergePlan{
		Branches:     branches,
		Strategy:     strategy,
		TargetBranch: "main",
	}
}
