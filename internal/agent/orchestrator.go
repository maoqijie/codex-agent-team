package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"codex-agent-team/internal/codexrpc"
)

// Orchestrator handles task decomposition using Codex.
type Orchestrator struct {
	agentMgr *Manager
}

// NewOrchestrator creates a new Orchestrator.
func NewOrchestrator(mgr *Manager) *Orchestrator {
	return &Orchestrator{
		agentMgr: mgr,
	}
}

// TaskDecomposition represents the result of task decomposition.
type TaskDecomposition struct {
	Tasks              []TaskSuggestion `json:"tasks"`
	Description        string           `json:"description"`
	TotalEstimatedTime string           `json:"totalEstimatedTime"`
}

// TaskSuggestion represents a single suggested task.
type TaskSuggestion struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	DependsOn      []string `json:"dependsOn"`
	Files          []string `json:"files,omitempty"`
	EstimatedTime  string   `json:"estimatedTime,omitempty"`
}

// Decompose analyzes the user's task and codebase, then returns a suggested task decomposition.
func (o *Orchestrator) Decompose(ctx context.Context, repoPath, userTask string) (*TaskDecomposition, error) {
	// 1. Spawn a read-only Codex instance
	agentCfg := AgentConfig{
		ID:             "orchestrator-" + GenerateID(),
		Role:           RoleOrchestrator,
		Cwd:            repoPath,
		SandboxMode:    codexrpc.SandboxReadOnly,
		BaseInstructions: o.getAnalysisPrompt(),
	}

	instance, err := o.agentMgr.SpawnAgent(ctx, agentCfg)
	if err != nil {
		return nil, fmt.Errorf("spawn orchestrator agent: %w", err)
	}
	defer o.agentMgr.StopAgent(instance.Config.ID)

	// 2. Send analysis prompt to Codex
	prompt := o.buildDecompositionPrompt(userTask)
	err = o.agentMgr.SendTask(ctx, instance.Config.ID, prompt)
	if err != nil {
		return nil, fmt.Errorf("send task: %w", err)
	}

	// 3. Wait for completion
	err = o.agentMgr.WaitForCompletion(ctx, instance.Config.ID)
	if err != nil {
		return nil, fmt.Errorf("wait for completion: %w", err)
	}

	// 4. Parse the response as TaskDecomposition
	output := o.agentMgr.GetOutput(instance.Config.ID)
	decomp, err := o.parseDecomposition(output)
	if err != nil {
		return nil, fmt.Errorf("parse decomposition: %w", err)
	}

	return decomp, nil
}

// parseDecomposition extracts JSON from the agent's output.
func (o *Orchestrator) parseDecomposition(output string) (*TaskDecomposition, error) {
	// Try to extract JSON from markdown code blocks or plain JSON
	jsonStr := output

	// Remove markdown code blocks if present
	if strings.Contains(output, "```json") {
		start := strings.Index(output, "```json")
		start += 7 // len("```json")
		end := strings.Index(output[start:], "```")
		if end > 0 {
			jsonStr = strings.TrimSpace(output[start : start+end])
		}
	} else if strings.Contains(output, "```") {
		start := strings.Index(output, "```")
		start += 3
		end := strings.Index(output[start:], "```")
		if end > 0 {
			jsonStr = strings.TrimSpace(output[start : start+end])
		}
	}

	var decomp TaskDecomposition
	if err := json.Unmarshal([]byte(jsonStr), &decomp); err != nil {
		return nil, fmt.Errorf("unmarshal JSON: %w", err)
	}

	return &decomp, nil
}

// getAnalysisPrompt returns the base instructions for the orchestrator agent.
func (o *Orchestrator) getAnalysisPrompt() string {
	return `You are a task orchestrator. Your job is to:
1. Analyze the codebase structure
2. Understand the user's requirements
3. Break down complex tasks into smaller, parallelizable sub-tasks
4. Output results as JSON in the specified format

Always respond with valid JSON, no markdown formatting.`
}

// buildDecompositionPrompt builds the prompt for task decomposition.
func (o *Orchestrator) buildDecompositionPrompt(userTask string) string {
	return fmt.Sprintf(`Analyze this codebase and decompose the following task into sub-tasks.

User Task: %s

Please analyze:
1. The current codebase structure
2. Which parts can be done in parallel
3. Which parts have dependencies

Output your analysis as a JSON object with this format:
{
  "description": "Overall approach description",
  "tasks": [
    {
      "id": "task-1",
      "title": "Brief title",
      "description": "What to do",
      "dependsOn": [],
      "files": ["path/to/file1.go", "path/to/file2.go"],
      "estimatedTime": "5-10 min"
    }
  ],
  "totalEstimatedTime": "20-30 min"
}

Respond ONLY with valid JSON, no markdown, no explanation.`, userTask)
}
