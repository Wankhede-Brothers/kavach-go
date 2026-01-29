// Package dag provides a parallel DAG scheduler for Kavach orchestration.
// types.go: Core type definitions for DAG scheduler state.
package dag

// NodeStatus represents the lifecycle state of a DAG node.
type NodeStatus string

const (
	StatusPending    NodeStatus = "pending"
	StatusReady      NodeStatus = "ready"
	StatusDispatched NodeStatus = "dispatched"
	StatusRunning    NodeStatus = "running"
	StatusDone       NodeStatus = "done"
	StatusFailed     NodeStatus = "failed"
	StatusSkipped    NodeStatus = "skipped"
)

// IsTerminal returns true if the status is a final state.
func (s NodeStatus) IsTerminal() bool {
	return s == StatusDone || s == StatusFailed || s == StatusSkipped
}

// Node represents a single task in the DAG.
type Node struct {
	ID          string            `json:"id"`
	Subject     string            `json:"subject"`
	Description string            `json:"description"`
	Agent       string            `json:"agent"`
	Skill       string            `json:"skill,omitempty"`
	Status      NodeStatus        `json:"status"`
	DependsOn   []string          `json:"depends_on,omitempty"`
	Blocks      []string          `json:"blocks,omitempty"`
	Level       int               `json:"level"`
	TaskID      string            `json:"task_id,omitempty"` // Claude task ID once created
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// DAGStatus represents the overall state of the DAG.
type DAGStatus string

const (
	DAGActive   DAGStatus = "active"
	DAGComplete DAGStatus = "complete"
	DAGFailed   DAGStatus = "failed"
)

// DAGState holds the full scheduler state for a session.
type DAGState struct {
	ID         string           `json:"id"`
	SessionID  string           `json:"session_id"`
	RootPrompt string           `json:"root_prompt"`
	Nodes      map[string]*Node `json:"nodes"`
	MaxLevel   int              `json:"max_level"`
	Status     DAGStatus        `json:"status"`
}

// ParallelLevel groups nodes that can execute concurrently.
type ParallelLevel struct {
	Level int     `json:"level"`
	Nodes []*Node `json:"nodes"`
}
