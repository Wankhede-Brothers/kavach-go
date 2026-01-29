// Package dag provides a parallel DAG scheduler for Kavach orchestration.
// scheduler.go: Decomposes task breakdowns into parallel DAG, handles events.
package dag

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// researchKeywords detects steps that are parallelizable (no inter-deps).
var researchKeywords = []string{"research", "search", "explore", "investigate", "find", "read"}

// Decompose creates nodes from a CEO TaskBreakdown with agent assignments.
// Steps containing research keywords are treated as parallel-safe (no inter-deps).
func Decompose(breakdown []string, agents []string) []*Node {
	nodes := make([]*Node, len(breakdown))
	for i, step := range breakdown {
		agent := "general-purpose"
		if i < len(agents) && agents[i] != "" {
			agent = agents[i]
		}
		id := nodeID(step)
		nodes[i] = &Node{
			ID:          id,
			Subject:     step,
			Description: step,
			Agent:       agent,
			Status:      StatusPending,
			Metadata:    map[string]string{"dag_node_id": id},
		}
	}
	return nodes
}

func nodeID(label string) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", label, time.Now().UnixNano())))
	return "kv-" + hex.EncodeToString(hash[:])[:6]
}

func isResearch(step string) bool {
	lower := strings.ToLower(step)
	for _, kw := range researchKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// Schedule builds a DAGState from decomposed nodes, adding sequential deps
// for non-research steps while keeping research steps parallel.
func Schedule(sessionID, prompt string, nodes []*Node) (*DAGState, error) {
	state := NewDAGState(sessionID, prompt)
	for _, n := range nodes {
		if err := state.AddNode(n); err != nil {
			return nil, err
		}
	}
	// Add sequential edges: non-research step[i] depends on step[i-1]
	var lastNonResearch string
	for _, n := range nodes {
		if isResearch(n.Subject) {
			continue // research nodes have no sequential deps
		}
		if lastNonResearch != "" {
			if err := state.AddEdge(lastNonResearch, n.ID); err != nil {
				return nil, fmt.Errorf("edge %s->%s: %w", lastNonResearch, n.ID, err)
			}
		}
		lastNonResearch = n.ID
	}
	// Mark initial ready nodes
	for _, n := range state.Nodes {
		if len(n.DependsOn) == 0 {
			n.Status = StatusReady
		}
	}
	// Compute levels
	if _, err := TopoLevels(state); err != nil {
		return nil, err
	}
	return state, nil
}

// BuildDirective generates the TOON directive for the current frontier level.
func BuildDirective(state *DAGState) string {
	ready := state.ReadyNodes()
	if len(ready) == 0 {
		if state.IsComplete() {
			return BuildCompletionDirective(state.ID)
		}
		return ""
	}
	level := ParallelLevel{Level: ready[0].Level, Nodes: ready}
	return BuildParallelDispatch(state.ID, level, state.MaxLevel)
}

// HandleTaskEvent processes TaskCreate/TaskUpdate hooks and advances DAG state.
// Returns: (complete, needsAegis, nextDirective).
func HandleTaskEvent(state *DAGState, toolName string, toolInput map[string]interface{}) (bool, bool, string) {
	switch toolName {
	case "TaskCreate":
		// At PreToolUse, extract dag_node_id from metadata and mark dispatched.
		// TaskID is not yet available (assigned after creation).
		md, _ := toolInput["metadata"].(map[string]interface{})
		nodeID, _ := md["dag_node_id"].(string)
		if nodeID == "" {
			return false, false, ""
		}
		if n, ok := state.Nodes[nodeID]; ok {
			// Store subject for later matching since taskId isn't available yet
			n.Status = StatusDispatched
		}

	case "TaskUpdate":
		status, _ := toolInput["status"].(string)
		taskID, _ := toolInput["taskId"].(string)
		if taskID == "" || (status != "completed" && status != "in_progress") {
			break
		}
		// Match by taskId first, fall back to subject for nodes without taskId
		subject, _ := toolInput["subject"].(string)
		for _, n := range state.Nodes {
			matched := false
			if n.TaskID != "" && n.TaskID == taskID {
				matched = true
			} else if n.TaskID == "" && subject != "" && n.Subject == subject {
				n.TaskID = taskID // late-bind the taskId
				matched = true
			}
			if matched {
				if status == "completed" {
					state.UpdateNodeStatus(n.ID, StatusDone)
				} else {
					n.Status = StatusRunning
				}
				break
			}
		}
	}

	if state.IsComplete() {
		allDone := true
		for _, n := range state.Nodes {
			if n.Status != StatusDone {
				allDone = false
				break
			}
		}
		return true, allDone, BuildCompletionDirective(state.ID)
	}

	directive := BuildDirective(state)
	return false, false, directive
}
