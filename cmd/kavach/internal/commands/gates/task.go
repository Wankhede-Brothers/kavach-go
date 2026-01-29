// Package gates provides hook gates for Claude Code.
// task.go: Task management gate for Claude Code 2.1.19+ persistent task system.
// Handles: TaskCreate, TaskUpdate, TaskGet, TaskList, TaskOutput
// Multi-session coordination via CLAUDE_CODE_TASK_LIST_ID
package gates

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/claude/shared/pkg/dag"
	"github.com/claude/shared/pkg/enforce"
	"github.com/claude/shared/pkg/hook"
	"github.com/spf13/cobra"
)

var taskHookMode bool

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Task management gate (Claude Code 2.1.19+)",
	Long: `[TASK_GATE]
desc: Validates task management operations for persistent task system
tools: TaskCreate, TaskUpdate, TaskGet, TaskList, TaskOutput
env: CLAUDE_CODE_TASK_LIST_ID for multi-session coordination
path: ~/.claude/tasks/{task_list_id}/`,
	Run: runTaskGate,
}

func init() {
	taskCmd.Flags().BoolVar(&taskHookMode, "hook", false, "Hook mode")
}

func runTaskGate(cmd *cobra.Command, args []string) {
	if !taskHookMode {
		cmd.Help()
		return
	}

	input := hook.MustReadHookInput()
	session := enforce.GetOrCreateSession()

	// Handle based on specific task tool
	switch input.ToolName {
	case "TaskCreate":
		handleTaskCreate(input, session)
	case "TaskUpdate":
		handleTaskUpdate(input, session)
	case "TaskGet":
		handleTaskGet(input, session)
	case "TaskList":
		handleTaskList(input, session)
	case "TaskOutput":
		handleTaskOutput(input, session)
	default:
		// Not a task tool - silent pass
		hook.ExitSilent()
	}
}

// handleTaskCreate validates new task creation
func handleTaskCreate(input *hook.Input, session *enforce.SessionState) {
	subject := input.GetString("subject")
	if subject == "" {
		hook.ExitBlockTOON("TASK_GATE", "TaskCreate:missing_subject")
	}

	description := input.GetString("description")
	if description == "" {
		hook.ExitBlockTOON("TASK_GATE", "TaskCreate:missing_description")
	}

	// Inject task list context for multi-session awareness
	taskListID := getTaskListID()
	today := time.Now().Format("2006-01-02")

	// Check if this is a background task
	isBackground := input.GetBool("run_in_background")

	// Track in health monitoring system
	health := GetTaskHealth()
	taskID := generateTaskID(subject)
	health.TrackTaskCreation(taskID, description, session.SessionID, isBackground)

	// Check for headless mode issues
	if issue := health.DetectHeadlessMode(); issue != nil {
		metadata := map[string]string{
			"task_list_id":   taskListID,
			"created_date":   today,
			"session_id":     session.SessionID,
			"health_warning": issue.Description,
		}
		hook.ExitModifyTOON("TASK_CREATE", metadata)
	}

	metadata := map[string]string{
		"task_list_id": taskListID,
		"created_date": today,
		"session_id":   session.SessionID,
	}

	// Only increment on PostToolUse (settings.json fires this gate for both Pre and Post).
	// At PreToolUse, HookEventName is "PreToolUse"; at PostToolUse it's "PostToolUse".
	if input.HookEventName == "PostToolUse" {
		session.TasksCreated++
		session.Save()
	}

	// DAG Scheduler: map Claude task ID to DAG node
	if state, err := dag.Load(session.SessionID); err == nil {
		_, _, directive := dag.HandleTaskEvent(state, "TaskCreate", input.ToolInput)
		if err := dag.Save(state); err != nil {
				fmt.Fprintf(os.Stderr, "[TASK_DAG] Save error: %v\n", err)
			}
		if directive != "" {
			metadata["dag_directive"] = "active"
			hook.ExitModifyTOONWithModule("TASK_CREATE", metadata, directive)
		}
	}

	hook.ExitModifyTOON("TASK_CREATE", metadata)
}

// handleTaskUpdate validates task status updates
func handleTaskUpdate(input *hook.Input, session *enforce.SessionState) {
	taskID := input.GetString("taskId")
	if taskID == "" {
		hook.ExitBlockTOON("TASK_GATE", "TaskUpdate:missing_taskId")
	}

	status := input.GetString("status")

	// Validate status transitions
	validStatuses := []string{"pending", "in_progress", "completed", "deleted", ""}
	if status != "" && !contains(validStatuses, status) {
		hook.ExitBlockTOON("TASK_GATE", "TaskUpdate:invalid_status:"+status)
	}

	// Track in health monitoring system
	health := GetTaskHealth()
	if status != "" {
		health.TrackTaskUpdate(taskID, status)
	}

	// Track completion in session
	if status == "completed" {
		session.TasksCompleted++
		session.Save()
	}

	// DAG Scheduler: advance state on task updates
	if state, err := dag.Load(session.SessionID); err == nil {
		complete, needsAegis, directive := dag.HandleTaskEvent(state, "TaskUpdate", input.ToolInput)
		if err := dag.Save(state); err != nil {
				fmt.Fprintf(os.Stderr, "[TASK_DAG] Save error: %v\n", err)
			}
		if complete && needsAegis {
			hook.ExitModifyTOON("TASK_UPDATE_DAG_COMPLETE", map[string]string{
				"dag_status": "complete",
				"action":     "Run kavach orch aegis for final verification",
			})
		}
		if directive != "" {
			hook.ExitModifyTOONWithModule("TASK_UPDATE_DAG_ADVANCE", map[string]string{
				"dag_status": "advancing",
			}, directive)
		}
	}

	hook.ExitSilent()
}

// handleTaskGet validates task retrieval
func handleTaskGet(input *hook.Input, session *enforce.SessionState) {
	taskID := input.GetString("taskId")
	if taskID == "" {
		hook.ExitBlockTOON("TASK_GATE", "TaskGet:missing_taskId")
	}

	// Allow read operations
	hook.ExitSilent()
}

// handleTaskList validates task listing
func handleTaskList(input *hook.Input, session *enforce.SessionState) {
	// TaskList takes no required parameters
	// Inject multi-session context
	taskListID := getTaskListID()

	if taskListID != "" {
		metadata := map[string]string{
			"task_list_id":  taskListID,
			"multi_session": "true",
		}
		hook.ExitModifyTOON("TASK_LIST_CONTEXT", metadata)
	}

	hook.ExitSilent()
}

// handleTaskOutput validates background task output retrieval
func handleTaskOutput(input *hook.Input, session *enforce.SessionState) {
	taskID := input.GetString("task_id")
	if taskID == "" {
		hook.ExitBlockTOON("TASK_GATE", "TaskOutput:missing_task_id")
	}

	// Track in health monitoring for zombie detection
	health := GetTaskHealth()

	// Check if zombie recovered
	if issue := health.RecordTaskOutput(taskID, ""); issue != nil {
		// Zombie recovered - inject info
		metadata := map[string]string{
			"task_id":      taskID,
			"health_event": issue.IssueType,
			"health_desc":  issue.Description,
		}
		hook.ExitModifyTOON("TASK_OUTPUT", metadata)
	}

	// Allow read operations
	hook.ExitSilent()
}

// getTaskListID returns the task list ID for multi-session coordination
func getTaskListID() string {
	// Check environment variable first
	if id := os.Getenv("CLAUDE_CODE_TASK_LIST_ID"); id != "" {
		return id
	}

	// Fall back to project-based ID
	if id := os.Getenv("CLAUDE_PROJECT"); id != "" {
		return "project_" + sanitizeID(id)
	}

	// Default to session-based
	return ""
}

// sanitizeID removes unsafe characters from ID strings
func sanitizeID(id string) string {
	// Replace unsafe chars with underscore
	unsafe := []string{"/", "\\", " ", ":", "*", "?", "\"", "<", ">", "|"}
	result := id
	for _, char := range unsafe {
		result = strings.ReplaceAll(result, char, "_")
	}
	return result
}

// contains checks if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// generateTaskID creates a Beads-style hash ID (kv-a1b2c3)
// Reference: github.com/steveyegge/beads - hash-based IDs prevent merge conflicts
func generateTaskID(subject string) string {
	data := subject + time.Now().Format(time.RFC3339Nano)
	// Simple hash using time + subject
	hash := 0
	for _, c := range data {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return fmt.Sprintf("kv-%06x", hash%0xFFFFFF)
}
