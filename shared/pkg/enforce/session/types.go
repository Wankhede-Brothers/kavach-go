// Package session provides session state management.
// types.go: SessionState struct definition.
// DACE: Single responsibility - type definitions only.
package session

// SessionState tracks enforcement state across a session.
// Fields are organized by category for clarity.
type SessionState struct {
	// Identity
	ID      string
	Today   string
	Project string
	WorkDir string

	// Enforcement flags
	ResearchDone   bool
	MemoryQueried  bool
	CEOInvoked     bool
	NLUParsed      bool
	AegisVerified  bool
	TrainingCutoff string

	// Compact tracking
	PostCompact  bool
	CompactedAt  string
	CompactCount int

	// Lost-in-Middle mitigation (attention decay)
	TurnCount         int // Tracks conversation turns
	LastReinforceTurn int // Turn when last reinforcement was injected
	ReinforceEveryN   int // Reinforce every N turns (default: 15)

	// Task state
	CurrentTask   string
	TaskStatus    string
	FilesModified []string

	// Task management (Claude Code 2.1.19+)
	SessionID      string // Session identifier for multi-session coordination
	TasksCreated   int    // Count of tasks created this session
	TasksCompleted int    // Count of tasks completed this session
	TaskListID     string // CLAUDE_CODE_TASK_LIST_ID for shared task lists
}
