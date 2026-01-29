// Package session provides session state management.
// save.go: Session persistence to TOON file.
// DACE: Single responsibility - saving logic only.
package session

import (
	"fmt"
	"os"

	"github.com/claude/shared/pkg/util"
)

// Save persists session state to TOON file.
func (s *SessionState) Save() error {
	statePath := StatePath()
	if err := util.EnsureParentDir(statePath); err != nil {
		return err
	}

	f, err := os.Create(statePath)
	if err != nil {
		return err
	}
	defer f.Close()

	writeHeader(f)
	writeSessionBlock(f, s)
	writeStateBlock(f, s)
	writeCompactBlock(f, s)
	writeTaskBlock(f, s)

	return nil
}

func writeHeader(f *os.File) {
	fmt.Fprintln(f, "# Session State - SP/1.0")
	fmt.Fprintln(f, "# Auto-generated, do not edit")
	fmt.Fprintln(f)
}

func writeSessionBlock(f *os.File, s *SessionState) {
	fmt.Fprintln(f, "[SESSION]")
	fmt.Fprintf(f, "id: %s\n", s.ID)
	fmt.Fprintf(f, "today: %s\n", s.Today)
	fmt.Fprintf(f, "project: %s\n", s.Project)
	fmt.Fprintf(f, "workdir: %s\n", s.WorkDir)
	fmt.Fprintf(f, "cutoff: %s\n", s.TrainingCutoff)
	fmt.Fprintln(f)
}

func writeStateBlock(f *os.File, s *SessionState) {
	fmt.Fprintln(f, "[STATE]")
	fmt.Fprintf(f, "research_done: %s\n", boolStr(s.ResearchDone))
	fmt.Fprintf(f, "memory: %s\n", boolStr(s.MemoryQueried))
	fmt.Fprintf(f, "ceo: %s\n", boolStr(s.CEOInvoked))
	fmt.Fprintf(f, "nlu: %s\n", boolStr(s.NLUParsed))
	fmt.Fprintf(f, "aegis: %s\n", boolStr(s.AegisVerified))
	fmt.Fprintf(f, "turn_count: %d\n", s.TurnCount)
	fmt.Fprintf(f, "last_reinforce_turn: %d\n", s.LastReinforceTurn)
	fmt.Fprintf(f, "reinforce_every_n: %d\n", s.ReinforceEveryN)
	fmt.Fprintf(f, "tasks_created: %d\n", s.TasksCreated)
	fmt.Fprintf(f, "tasks_completed: %d\n", s.TasksCompleted)
	fmt.Fprintf(f, "session_id: %s\n", s.SessionID)
	fmt.Fprintln(f)
}

func writeCompactBlock(f *os.File, s *SessionState) {
	fmt.Fprintln(f, "[COMPACT]")
	fmt.Fprintf(f, "post_compact: %s\n", boolStr(s.PostCompact))
	fmt.Fprintf(f, "compacted_at: %s\n", s.CompactedAt)
	fmt.Fprintf(f, "compact_count: %d\n", s.CompactCount)
	fmt.Fprintln(f)
}

func writeTaskBlock(f *os.File, s *SessionState) {
	fmt.Fprintln(f, "[TASK]")
	fmt.Fprintf(f, "task: %s\n", s.CurrentTask)
	fmt.Fprintf(f, "task_status: %s\n", s.TaskStatus)
	writeFilesArray(f, s.FilesModified)
	fmt.Fprintln(f)
	writeIntentBlock(f, s)
}

func writeIntentBlock(f *os.File, s *SessionState) {
	if s.IntentType == "" {
		return
	}
	fmt.Fprintln(f, "[INTENT_BRIDGE]")
	fmt.Fprintf(f, "type: %s\n", s.IntentType)
	fmt.Fprintf(f, "domain: %s\n", s.IntentDomain)
	if len(s.IntentSubAgents) > 0 {
		fmt.Fprintf(f, "subagents: %s\n", joinCSV(s.IntentSubAgents))
	}
	if len(s.IntentSkills) > 0 {
		fmt.Fprintf(f, "skills: %s\n", joinCSV(s.IntentSkills))
	}
}

func joinCSV(items []string) string {
	result := ""
	for i, s := range items {
		if i > 0 {
			result += ","
		}
		result += s
	}
	return result
}

func writeFilesArray(f *os.File, files []string) {
	if len(files) == 0 {
		fmt.Fprintln(f, "files[]:")
	} else if len(files) == 1 {
		fmt.Fprintf(f, "files[]: %s\n", files[0])
	} else {
		fmt.Fprintln(f, "files[]:")
		for _, file := range files {
			fmt.Fprintf(f, "  - %s\n", file)
		}
	}
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
