package task

import (
	"errors"
	"sync"
)

// DAG represents a directed acyclic graph of tasks.
type DAG struct {
	mu    sync.RWMutex
	tasks map[string]*Task
}

// NewDAG creates a new empty DAG.
func NewDAG() *DAG {
	return &DAG{
		tasks: make(map[string]*Task),
	}
}

// AddTask adds a task to the DAG.
func (d *DAG) AddTask(t *Task) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.tasks[t.ID]; exists {
		return errors.New("task already exists")
	}

	d.tasks[t.ID] = t
	return nil
}

// Get retrieves a task by ID.
func (d *DAG) Get(id string) (*Task, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	t, ok := d.tasks[id]
	return t, ok
}

// ReadyTasks returns all tasks whose dependencies have been satisfied.
func (d *DAG) ReadyTasks() []*Task {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var ready []*Task

	for _, t := range d.tasks {
		if t.Status != StatusPending {
			continue
		}

		// Check if all dependencies are completed
		allDepsCompleted := true
		for _, depID := range t.DependsOn {
			if depTask, ok := d.tasks[depID]; !ok || depTask.Status != StatusCompleted {
				allDepsCompleted = false
				break
			}
		}

		if allDepsCompleted && len(t.DependsOn) > 0 {
			// Has dependencies and all are completed
			ready = append(ready, t)
		} else if len(t.DependsOn) == 0 && t.Status == StatusPending {
			// No dependencies, ready to run
			ready = append(ready, t)
		}
	}

	return ready
}

// UpdateStatus updates the status of a task.
func (d *DAG) UpdateStatus(id string, status TaskStatus) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if t, ok := d.tasks[id]; ok {
		t.Status = status
	}
}

// HasCycle detects if there's a cycle in the DAG using DFS with three-color marking.
func (d *DAG) HasCycle() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	const (
		colorWhite = iota // Unvisited
		colorGray         // Visiting (in current path)
		colorBlack        // Visited
	)

	colors := make(map[string]int)
	var hasCycle bool

	var dfs func(string)
	dfs = func(nodeID string) {
		if hasCycle {
			return
		}

		if colors[nodeID] == colorGray {
			// Back edge found, cycle exists
			hasCycle = true
			return
		}

		if colors[nodeID] == colorBlack {
			// Already processed
			return
		}

		// Mark as visiting
		colors[nodeID] = colorGray

		// Visit all dependent tasks (reverse edges)
		if task, ok := d.tasks[nodeID]; ok {
			for _, depID := range task.DependsOn {
				dfs(depID)
			}
		}

		// Mark as visited
		colors[nodeID] = colorBlack
	}

	// Start DFS from all nodes
	for id := range d.tasks {
		if colors[id] == colorWhite {
			dfs(id)
		}
	}

	return hasCycle
}

// AllCompleted checks if all tasks have completed (successfully or failed).
func (d *DAG) AllCompleted() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, t := range d.tasks {
		if t.Status != StatusCompleted && t.Status != StatusFailed && t.Status != StatusCancelled {
			return false
		}
	}
	return true
}

// HasFailed checks if any task has failed.
func (d *DAG) HasFailed() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, t := range d.tasks {
		if t.Status == StatusFailed {
			return true
		}
	}
	return false
}

// TopologicalOrder returns tasks in topological order using Kahn's algorithm.
func (d *DAG) TopologicalOrder() ([]*Task, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.HasCycle() {
		return nil, errors.New("cycle detected in DAG")
	}

	// Calculate in-degree for each task
	inDegree := make(map[string]int)
	for _, t := range d.tasks {
		inDegree[t.ID] = 0
	}

	// Count incoming edges
	for _, t := range d.tasks {
		for range t.DependsOn {
			// Edge from depID to t.ID means t.ID has an incoming edge
			inDegree[t.ID]++
		}
	}

	// Initialize queue with nodes having zero in-degree
	queue := make([]*Task, 0)
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, d.tasks[id])
		}
	}

	// Process nodes
	result := make([]*Task, 0, len(d.tasks))
	for len(queue) > 0 {
		// Dequeue
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// Reduce in-degree for dependent tasks
		for _, t := range d.tasks {
			for _, depID := range t.DependsOn {
				if depID == current.ID {
					inDegree[t.ID]--
					if inDegree[t.ID] == 0 {
						queue = append(queue, t)
					}
					break
				}
			}
		}
	}

	return result, nil
}
