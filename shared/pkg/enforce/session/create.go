// Package session provides session state management.
// create.go: Session creation and ID generation.
// DACE: Single responsibility - creation logic only.
package session

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/claude/shared/pkg/util"
)

// NewSessionState creates a new session with date injection.
// Uses util.DetectProject() for proper project detection (index.toon, git, markers).
func NewSessionState(workDir string) *SessionState {
	id := generateSessionID(workDir)
	return &SessionState{
		ID:             id,
		SessionID:      id, // Same as ID — used by DAG scheduler and task gate
		Today:          time.Now().Format("2006-01-02"),
		WorkDir:        workDir,
		Project:        util.DetectProject(),
		TrainingCutoff: "2025-01",
		FilesModified:  []string{},
	}
}

// generateSessionID creates deterministic session ID from workdir and date.
func generateSessionID(workDir string) string {
	h := sha256.New()
	h.Write([]byte(workDir))
	h.Write([]byte(time.Now().Format("20060102")))
	return "sess_" + hex.EncodeToString(h.Sum(nil))[:32]
}
