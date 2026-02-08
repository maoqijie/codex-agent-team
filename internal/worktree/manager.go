package worktree

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Manager 管理 Git worktree
type Manager struct {
	repoPath string // 仓库根目录
}

// Worktree 表示一个 Git worktree
type Worktree struct {
	Path   string // worktree 路径
	Branch string // 分支名
	Commit string // 提交 SHA（可选）
}

// NewManager 创建一个新的 worktree 管理器
func NewManager(repoPath string) *Manager {
	return &Manager{
		repoPath: repoPath,
	}
}

// Create 创建一个新的 worktree
// branchName: 分支名称
// commitHash: 基于哪个提交创建（可选，默认为当前 HEAD）
func (m *Manager) Create(ctx context.Context, branchName string, commitHash string) (*Worktree, error) {
	// 如果没有指定 commit，使用 HEAD
	commit := commitHash
	if commit == "" {
		commit = "HEAD"
	}

	// worktree 路径
	worktreePath := m.GetPath(branchName)

	// 构建 git worktree add -b <branch> <path> <commit> 命令
	// 使用 -b 创建命名分支，以便后续任务可以通过分支名 merge
	cmd := exec.CommandContext(ctx, "git", "worktree", "add", "-b", branchName, worktreePath, commit)
	cmd.Dir = m.repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create worktree: %w: %s", err, string(output))
	}

	// Resolve actual commit SHA
	headCmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	headCmd.Dir = worktreePath
	commitSha, err := headCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("rev-parse HEAD in new worktree: %w", err)
	}

	return &Worktree{
		Path:   worktreePath,
		Branch: branchName,
		Commit: strings.TrimSpace(string(commitSha)),
	}, nil
}

// List 列出所有 worktree
func (m *Manager) List(ctx context.Context) ([]Worktree, error) {
	cmd := exec.CommandContext(ctx, "git", "worktree", "list", "--porcelain")
	cmd.Dir = m.repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	// 解析 porcelain 格式的输出
	return parseWorktreeList(string(output))
}

// Remove 删除指定的 worktree
func (m *Manager) Remove(ctx context.Context, path string) error {
	cmd := exec.CommandContext(ctx, "git", "worktree", "remove", path)
	cmd.Dir = m.repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove worktree %s: %w: %s", path, err, string(output))
	}

	return nil
}

// GetPath 获取 worktree 的完整路径
func (m *Manager) GetPath(branchName string) string {
	return filepath.Join(m.repoPath, ".worktrees", branchName)
}

// parseWorktreeList 解析 git worktree list --porcelain 的输出
// 输出格式示例：
// worktree /path/to/worktree
// HEAD abc123
// branch refs/heads/branch-name
// ...
func parseWorktreeList(output string) ([]Worktree, error) {
	var worktrees []Worktree
	scanner := bufio.NewScanner(strings.NewReader(output))

	var currentPath string
	var currentBranch string
	var currentCommit string

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			// 空行表示一个 worktree 块的结束
			if currentPath != "" {
				worktrees = append(worktrees, Worktree{
					Path:   currentPath,
					Branch: currentBranch,
					Commit: currentCommit,
				})
			}
			currentPath = ""
			currentBranch = ""
			currentCommit = ""
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]
		switch key {
		case "worktree":
			currentPath = value
		case "branch":
			// 从 refs/heads/branch-name 提取分支名
			currentBranch = strings.TrimPrefix(value, "refs/heads/")
		case "HEAD":
			currentCommit = value
		}
	}

	// 添加最后一个 worktree（如果输出不以空行结尾）
	if currentPath != "" {
		worktrees = append(worktrees, Worktree{
			Path:   currentPath,
			Branch: currentBranch,
			Commit: currentCommit,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse worktree list: %w", err)
	}

	return worktrees, nil
}

// Merge 将指定分支合并到当前 worktree
func (m *Manager) Merge(ctx context.Context, worktreePath string, branchName string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "merge", "--no-ff",
		"-m", fmt.Sprintf("Merge %s", branchName), branchName)
	cmd.Dir = worktreePath

	output, err := cmd.CombinedOutput()
	if err != nil {
		// 清理冲突状态
		abortCmd := exec.CommandContext(ctx, "git", "merge", "--abort")
		abortCmd.Dir = worktreePath
		_ = abortCmd.Run()
		return "", fmt.Errorf("failed to merge branch %s: %w: %s", branchName, err, string(output))
	}

	headCmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	headCmd.Dir = worktreePath
	commitSha, err := headCmd.Output()
	if err != nil {
		return "", fmt.Errorf("rev-parse HEAD after merge: %w", err)
	}

	return strings.TrimSpace(string(commitSha)), nil
}

// CommitChanges 提交 worktree 中的所有修改
func (m *Manager) CommitChanges(ctx context.Context, worktreePath string, message string) (string, error) {
	// git add -A（改用 CombinedOutput 获取详细错误）
	addCmd := exec.CommandContext(ctx, "git", "add", "-A")
	addCmd.Dir = worktreePath
	addOutput, err := addCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git add failed: %w: %s", err, string(addOutput))
	}

	// git commit
	commitCmd := exec.CommandContext(ctx, "git", "commit", "-m", message)
	commitCmd.Dir = worktreePath
	output, err := commitCmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "nothing to commit") {
			return "", nil
		}
		return "", fmt.Errorf("git commit failed: %w: %s", err, string(output))
	}

	// 获取 commit SHA
	headCmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	headCmd.Dir = worktreePath
	commitSha, err := headCmd.Output()
	if err != nil {
		return "", fmt.Errorf("rev-parse HEAD after commit: %w", err)
	}

	return strings.TrimSpace(string(commitSha)), nil
}
