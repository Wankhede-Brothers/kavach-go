// Package session provides session state management.
// load.go: Session loading from TOON file.
// DACE: Single responsibility - loading logic only.
package session

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/claude/shared/pkg/util"
)

// LoadSessionState loads existing session state from TOON file.
// Returns nil if no valid session exists or session expired.
func LoadSessionState() (*SessionState, error) {
	statePath := StatePath()
	if !util.FileExists(statePath) {
		return nil, nil
	}

	f, err := os.Open(statePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	state := &SessionState{FilesModified: []string{}}
	scanner := bufio.NewScanner(f)
	var inFiles bool

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			inFiles = false
			continue
		}

		if inFiles && strings.HasPrefix(line, "- ") {
			state.FilesModified = append(state.FilesModified, strings.TrimPrefix(line, "- "))
			continue
		}

		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			parseField(state, key, value, &inFiles)
		}
	}

	if !isValidToday(state.Today) {
		return nil, nil
	}

	return state, scanner.Err()
}

// parseField parses a single TOON field into state.
func parseField(state *SessionState, key, value string, inFiles *bool) {
	switch key {
	case "id":
		state.ID = value
	case "today":
		state.Today = value
	case "project":
		state.Project = value
	case "workdir":
		state.WorkDir = value
	case "research", "research_done": // Support both old and new key names
		state.ResearchDone = value == "true"
	case "memory":
		state.MemoryQueried = value == "true"
	case "ceo":
		state.CEOInvoked = value == "true"
	case "nlu":
		state.NLUParsed = value == "true"
	case "aegis":
		state.AegisVerified = value == "true"
	case "cutoff":
		state.TrainingCutoff = value
	case "post_compact":
		state.PostCompact = value == "true"
	case "compacted_at":
		state.CompactedAt = value
	case "compact_count":
		state.CompactCount, _ = strconv.Atoi(value)
	case "turn_count":
		state.TurnCount, _ = strconv.Atoi(value)
	case "last_reinforce_turn":
		state.LastReinforceTurn, _ = strconv.Atoi(value)
	case "reinforce_every_n":
		state.ReinforceEveryN, _ = strconv.Atoi(value)
	case "tasks_created":
		state.TasksCreated, _ = strconv.Atoi(value)
	case "tasks_completed":
		state.TasksCompleted, _ = strconv.Atoi(value)
	case "session_id":
		state.SessionID = value
	case "task":
		state.CurrentTask = value
	case "task_status":
		state.TaskStatus = value
	case "files[]":
		*inFiles = true
		if value != "" {
			state.FilesModified = append(state.FilesModified, value)
		}
	// Intent bridge fields (written by intent gate, read by CEO gate)
	case "type":
		if state.IntentType == "" { // Only set if not already set by [SESSION] block
			state.IntentType = value
		}
	case "domain":
		if state.IntentDomain == "" {
			state.IntentDomain = value
		}
	case "subagents":
		if value != "" {
			state.IntentSubAgents = splitCSV(value)
		}
	case "skills":
		if value != "" && len(state.IntentSkills) == 0 {
			state.IntentSkills = splitCSV(value)
		}
	}
}

// isValidToday checks if session date matches today.
func isValidToday(sessionDate string) bool {
	return sessionDate == time.Now().Format("2006-01-02")
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			result = append(result, t)
		}
	}
	return result
}
