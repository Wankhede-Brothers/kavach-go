// Package dag provides a parallel DAG scheduler for Kavach orchestration.
// state.go: JSON persistence for DAG state.
package dag

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// StatePath returns the file path for a session's DAG state.
func StatePath(sessionID string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "dag", sessionID+".json")
}

// Save persists DAG state to disk as JSON.
func Save(state *DAGState) error {
	path := StatePath(state.SessionID)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// Load reads DAG state from disk.
func Load(sessionID string) (*DAGState, error) {
	data, err := os.ReadFile(StatePath(sessionID))
	if err != nil {
		return nil, err
	}
	var state DAGState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &state, nil
}

// Delete removes DAG state for a session.
func Delete(sessionID string) error {
	return os.Remove(StatePath(sessionID))
}
