// Package gates provides hook gates for Claude Code.
// task.go: Task management gate for Claude Code 2.1.19+ persistent task system.
// Handles: TaskCreate, TaskUpdate, TaskGet, TaskList, TaskOutput
// Multi-session coordination via CLAUDE_CODE_TASK_LIST_ID
package gates

import (
	"os"
	"strings"
	"time"

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
		// Don't block, just inject warning
		metadata := map[string]string{
			"task_list_id":   taskListID,
			"created_date":   today,
			"session_id":     session.SessionID,
			"health_warning": issue.Description,
		}
		session.TasksCreated++
		session.Save()
		hook.ExitModifyTOON("TASK_CREATE", metadata)
	}

	metadata := map[string]string{
		"task_list_id": taskListID,
		"created_date": today,
		"session_id":   session.SessionID,
	}

	// Track task creation in session state
	session.TasksCreated++
	session.Save()

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
	validStatuses := []string{"pending", "in_progress", "completed", ""}
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

// generateTaskID creates a unique task ID from subject and timestamp
func generateTaskID(subject string) string {
	// Use first 8 chars of subject + timestamp hash
	prefix := sanitizeID(subject)
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}
	timestamp := time.Now().UnixNano()
	return prefix + "_" + string(rune(timestamp%1000000))
}
