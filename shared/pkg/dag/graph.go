// Package dag provides a parallel DAG scheduler for Kavach orchestration.
// graph.go: Graph construction and node state transitions.
package dag

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// NewDAGState creates a new DAG for the given session and prompt.
func NewDAGState(sessionID, prompt string) *DAGState {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", prompt, time.Now().UnixNano())))
	id := "kv-" + hex.EncodeToString(hash[:])[:6]
	return &DAGState{
		ID:         id,
		SessionID:  sessionID,
		RootPrompt: prompt,
		Nodes:      make(map[string]*Node),
		Status:     DAGActive,
	}
}

// AddNode adds a node, returning error on duplicate ID.
func (s *DAGState) AddNode(n *Node) error {
	if _, exists := s.Nodes[n.ID]; exists {
		return fmt.Errorf("duplicate node: %s", n.ID)
	}
	if n.Status == "" {
		n.Status = StatusPending
	}
	s.Nodes[n.ID] = n
	return nil
}

// AddEdge creates a dependency: depID must complete before nodeID starts.
// Includes inline cycle detection via DFS.
func (s *DAGState) AddEdge(depID, nodeID string) error {
	dep, ok := s.Nodes[depID]
	if !ok {
		return fmt.Errorf("node not found: %s", depID)
	}
	node, ok := s.Nodes[nodeID]
	if !ok {
		return fmt.Errorf("node not found: %s", nodeID)
	}
	// Cycle check: would nodeID->...->depID form a path?
	if s.hasPath(nodeID, depID, make(map[string]bool)) {
		return fmt.Errorf("cycle detected: %s -> %s", depID, nodeID)
	}
	node.DependsOn = append(node.DependsOn, depID)
	dep.Blocks = append(dep.Blocks, nodeID)
	return nil
}

func (s *DAGState) hasPath(from, to string, visited map[string]bool) bool {
	if from == to {
		return true
	}
	if visited[from] {
		return false
	}
	visited[from] = true
	node := s.Nodes[from]
	if node == nil {
		return false
	}
	for _, blocked := range node.Blocks {
		if s.hasPath(blocked, to, visited) {
			return true
		}
	}
	return false
}

// UpdateNodeStatus transitions a node and propagates ready/skipped.
func (s *DAGState) UpdateNodeStatus(id string, status NodeStatus) {
	node, ok := s.Nodes[id]
	if !ok {
		return
	}
	node.Status = status

	if status == StatusDone {
		// Check if blocked nodes become ready
		for _, blockedID := range node.Blocks {
			s.checkReady(blockedID)
		}
	}
	if status == StatusFailed {
		// Propagate skipped to all dependents
		for _, blockedID := range node.Blocks {
			s.propagateSkip(blockedID)
		}
	}
	// Update overall DAG status
	if s.IsComplete() {
		s.Status = DAGComplete
		for _, n := range s.Nodes {
			if n.Status == StatusFailed || n.Status == StatusSkipped {
				s.Status = DAGFailed
				break
			}
		}
	}
}

func (s *DAGState) checkReady(id string) {
	node := s.Nodes[id]
	if node == nil || node.Status.IsTerminal() {
		return
	}
	for _, depID := range node.DependsOn {
		dep := s.Nodes[depID]
		if dep == nil || !dep.Status.IsTerminal() {
			return
		}
		if dep.Status != StatusDone {
			return // dep failed/skipped
		}
	}
	node.Status = StatusReady
}

func (s *DAGState) propagateSkip(id string) {
	node := s.Nodes[id]
	if node == nil || node.Status.IsTerminal() {
		return
	}
	node.Status = StatusSkipped
	for _, blockedID := range node.Blocks {
		s.propagateSkip(blockedID)
	}
}

// ReadyNodes returns nodes where all dependencies are done.
func (s *DAGState) ReadyNodes() []*Node {
	var ready []*Node
	for _, n := range s.Nodes {
		if n.Status == StatusReady {
			ready = append(ready, n)
		}
	}
	return ready
}

// IsComplete returns true when all nodes are in a terminal state.
func (s *DAGState) IsComplete() bool {
	for _, n := range s.Nodes {
		if !n.Status.IsTerminal() {
			return false
		}
	}
	return len(s.Nodes) > 0
}
