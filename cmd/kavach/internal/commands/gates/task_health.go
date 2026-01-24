// Package gates provides hook gates for Claude Code.
// task_health.go: Runtime bug detection for Claude Code 2.1.19+ task system.
//
// Detects (based on GitHub issues):
// - #19894: Stale task count (UI shows wrong count)
// - #17542: Zombie background tasks (orphaned processes)
// - #20463: Headless mode missing Task tools
// - #20525: Silent task completion (no notification)
//
// Patterns: Go diagnostics + health check patterns (go.dev/doc/diagnostics)
package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TaskHealthState tracks task health metrics for bug detection
type TaskHealthState struct {
	// Task tracking
	ActiveTasks      map[string]*TrackedTask `json:"active_tasks"`
	CompletedTasks   map[string]*TrackedTask `json:"completed_tasks"`
	ZombieCandidates map[string]*TrackedTask `json:"zombie_candidates"`

	// Counters for stale detection
	CreatedCount    int `json:"created_count"`
	CompletedCount  int `json:"completed_count"`
	InProgressCount int `json:"in_progress_count"`

	// Health flags
	HeadlessMode     bool   `json:"headless_mode"`
	TaskToolsEnabled bool   `json:"task_tools_enabled"`
	LastHealthCheck  string `json:"last_health_check"`

	// Thresholds (configurable)
	ZombieTimeoutMinutes int `json:"zombie_timeout_minutes"`
	StaleCheckInterval   int `json:"stale_check_interval"`
}

// TrackedTask represents a task being monitored for health
type TrackedTask struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	Description  string `json:"description"`
	SessionID    string `json:"session_id"`
	IsBackground bool   `json:"is_background"`
}

// TaskHealthIssue represents a detected bug/issue
type TaskHealthIssue struct {
	IssueType   string `json:"issue_type"`
	Severity    string `json:"severity"` // critical, warning, info
	Description string `json:"description"`
	GitHubIssue string `json:"github_issue,omitempty"`
	Suggestion  string `json:"suggestion"`
	DetectedAt  string `json:"detected_at"`
	TaskID      string `json:"task_id,omitempty"`
}

// Global health state (lazy-loaded singleton)
var taskHealth *TaskHealthState

// GetTaskHealth returns the singleton task health state
func GetTaskHealth() *TaskHealthState {
	if taskHealth == nil {
		taskHealth = loadOrCreateTaskHealth()
	}
	return taskHealth
}

// loadOrCreateTaskHealth loads existing health state or creates new
func loadOrCreateTaskHealth() *TaskHealthState {
	healthPath := getHealthStatePath()

	if data, err := os.ReadFile(healthPath); err == nil {
		var state TaskHealthState
		if json.Unmarshal(data, &state) == nil {
			// Ensure maps are initialized
			if state.ActiveTasks == nil {
				state.ActiveTasks = make(map[string]*TrackedTask)
			}
			if state.CompletedTasks == nil {
				state.CompletedTasks = make(map[string]*TrackedTask)
			}
			if state.ZombieCandidates == nil {
				state.ZombieCandidates = make(map[string]*TrackedTask)
			}
			return &state
		}
	}

	// Create new with defaults
	return &TaskHealthState{
		ActiveTasks:          make(map[string]*TrackedTask),
		CompletedTasks:       make(map[string]*TrackedTask),
		ZombieCandidates:     make(map[string]*TrackedTask),
		TaskToolsEnabled:     true,
		ZombieTimeoutMinutes: 30, // 30 minutes without update = zombie candidate
		StaleCheckInterval:   60, // Check every 60 seconds
	}
}

// Save persists the health state to disk
func (h *TaskHealthState) Save() error {
	healthPath := getHealthStatePath()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(healthPath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(healthPath, data, 0644)
}

// getHealthStatePath returns the path to the health state file
func getHealthStatePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "kavach", "task_health.json")
}

// =============================================================================
// BUG #20463: Headless Mode Detection
// =============================================================================

// DetectHeadlessMode checks if running in headless/pipe mode
func (h *TaskHealthState) DetectHeadlessMode() *TaskHealthIssue {
	// Check for headless indicators
	headlessEnvVars := []string{
		"CLAUDE_CODE_HEADLESS",
		"CI",
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"JENKINS_URL",
	}

	for _, envVar := range headlessEnvVars {
		if val := os.Getenv(envVar); val != "" && val != "0" && val != "false" {
			h.HeadlessMode = true
			break
		}
	}

	// Check if stdin is a pipe (not TTY)
	if stat, err := os.Stdin.Stat(); err == nil {
		isPipe := (stat.Mode() & os.ModeCharDevice) == 0
		if isPipe {
			h.HeadlessMode = true
		}
	}

	// Check CLAUDE_CODE_ENABLE_TASKS
	if enableTasks := os.Getenv("CLAUDE_CODE_ENABLE_TASKS"); enableTasks == "0" || enableTasks == "false" {
		h.TaskToolsEnabled = false
	}

	if h.HeadlessMode && !h.TaskToolsEnabled {
		return &TaskHealthIssue{
			IssueType:   "HEADLESS_MODE_TASK_TOOLS",
			Severity:    "warning",
			Description: "Running in headless/pipe mode. Task tools (TaskCreate, TaskUpdate, etc.) may not be available.",
			GitHubIssue: "#20463",
			Suggestion:  "Set CLAUDE_CODE_ENABLE_TASKS=1 for headless task support, or use interactive mode.",
			DetectedAt:  time.Now().Format(time.RFC3339),
		}
	}

	return nil
}

// =============================================================================
// BUG #19894: Stale Task Count Detection
// =============================================================================

// TrackTaskCreation records a new task for stale detection
func (h *TaskHealthState) TrackTaskCreation(taskID, description, sessionID string, isBackground bool) {
	now := time.Now().Format(time.RFC3339)

	h.ActiveTasks[taskID] = &TrackedTask{
		ID:           taskID,
		Status:       "pending",
		CreatedAt:    now,
		UpdatedAt:    now,
		Description:  truncateString(description, 100),
		SessionID:    sessionID,
		IsBackground: isBackground,
	}

	h.CreatedCount++
	h.Save()
}

// TrackTaskUpdate records a task status change
func (h *TaskHealthState) TrackTaskUpdate(taskID, newStatus string) {
	now := time.Now().Format(time.RFC3339)

	if task, exists := h.ActiveTasks[taskID]; exists {
		oldStatus := task.Status
		task.Status = newStatus
		task.UpdatedAt = now

		if newStatus == "completed" {
			// Move to completed
			h.CompletedTasks[taskID] = task
			delete(h.ActiveTasks, taskID)
			h.CompletedCount++

			// Decrement in_progress if was in_progress
			if oldStatus == "in_progress" && h.InProgressCount > 0 {
				h.InProgressCount--
			}

			// Remove from zombie candidates if present
			delete(h.ZombieCandidates, taskID)
		} else if newStatus == "in_progress" && oldStatus != "in_progress" {
			h.InProgressCount++
		}

		h.Save()
	}
}

// DetectStaleTasks checks for mismatch between tracked and reported counts
func (h *TaskHealthState) DetectStaleTasks(reportedActiveCount int) *TaskHealthIssue {
	actualActive := len(h.ActiveTasks)

	// Check for stale active count (UI showing more than actually active)
	if reportedActiveCount > 0 && actualActive == 0 {
		return &TaskHealthIssue{
			IssueType:   "STALE_TASK_COUNT",
			Severity:    "warning",
			Description: fmt.Sprintf("UI reports %d active tasks but kavach tracks %d. UI counter may be stale.", reportedActiveCount, actualActive),
			GitHubIssue: "#19894",
			Suggestion:  "Run /tasks to refresh the task list. If persists, restart the session.",
			DetectedAt:  time.Now().Format(time.RFC3339),
		}
	}

	// Check for significant mismatch
	if reportedActiveCount > 0 && actualActive > 0 {
		diff := reportedActiveCount - actualActive
		if diff > 2 { // Allow small discrepancy
			return &TaskHealthIssue{
				IssueType:   "TASK_COUNT_MISMATCH",
				Severity:    "info",
				Description: fmt.Sprintf("UI shows %d tasks, kavach tracks %d. Difference: %d", reportedActiveCount, actualActive, diff),
				GitHubIssue: "#19894",
				Suggestion:  "Some tasks may have completed without proper tracking. Check /tasks.",
				DetectedAt:  time.Now().Format(time.RFC3339),
			}
		}
	}

	return nil
}

// =============================================================================
// BUG #17542: Zombie Task Detection
// =============================================================================

// CheckForZombieTasks identifies tasks that haven't been updated in a long time
func (h *TaskHealthState) CheckForZombieTasks() []*TaskHealthIssue {
	var issues []*TaskHealthIssue
	now := time.Now()
	zombieThreshold := time.Duration(h.ZombieTimeoutMinutes) * time.Minute

	for taskID, task := range h.ActiveTasks {
		// Only check in_progress or background tasks
		if task.Status != "in_progress" && !task.IsBackground {
			continue
		}

		updatedAt, err := time.Parse(time.RFC3339, task.UpdatedAt)
		if err != nil {
			continue
		}

		timeSinceUpdate := now.Sub(updatedAt)

		if timeSinceUpdate > zombieThreshold {
			// Mark as zombie candidate
			h.ZombieCandidates[taskID] = task

			issues = append(issues, &TaskHealthIssue{
				IssueType:   "ZOMBIE_TASK",
				Severity:    "warning",
				Description: fmt.Sprintf("Task '%s' in_progress for %v without updates. May be orphaned.", truncateString(task.Description, 50), timeSinceUpdate.Round(time.Minute)),
				GitHubIssue: "#17542",
				Suggestion:  fmt.Sprintf("Check with TaskOutput(task_id='%s'). If unresponsive, use TaskStop.", taskID),
				DetectedAt:  time.Now().Format(time.RFC3339),
				TaskID:      taskID,
			})
		}
	}

	if len(issues) > 0 {
		h.Save()
	}

	return issues
}

// RecordTaskOutput tracks TaskOutput calls to detect zombie responses
func (h *TaskHealthState) RecordTaskOutput(taskID, status string) *TaskHealthIssue {
	now := time.Now().Format(time.RFC3339)

	// Update last seen time for active task
	if task, exists := h.ActiveTasks[taskID]; exists {
		task.UpdatedAt = now

		// If completed, move out of active
		if status == "completed" {
			h.TrackTaskUpdate(taskID, "completed")
		}
	}

	// Check if this was a zombie candidate that's now responding
	if zombie, wasZombie := h.ZombieCandidates[taskID]; wasZombie {
		delete(h.ZombieCandidates, taskID)
		h.Save()

		return &TaskHealthIssue{
			IssueType:   "ZOMBIE_RECOVERED",
			Severity:    "info",
			Description: fmt.Sprintf("Previously unresponsive task '%s' is now responding.", truncateString(zombie.Description, 50)),
			Suggestion:  "Task recovered. Continue monitoring.",
			DetectedAt:  now,
			TaskID:      taskID,
		}
	}

	return nil
}

// =============================================================================
// BUG #20525: Silent Completion Detection
// =============================================================================

// DetectSilentCompletion checks if background tasks completed without notification
func (h *TaskHealthState) DetectSilentCompletion(taskID string) *TaskHealthIssue {
	task, exists := h.CompletedTasks[taskID]
	if !exists || !task.IsBackground {
		return nil
	}

	// Check if completion was recent (within last 5 minutes)
	completedAt, err := time.Parse(time.RFC3339, task.UpdatedAt)
	if err != nil {
		return nil
	}

	timeSinceComplete := time.Since(completedAt)

	// If completed recently and was background, flag for notification check
	if timeSinceComplete < 5*time.Minute {
		return &TaskHealthIssue{
			IssueType:   "SILENT_COMPLETION",
			Severity:    "info",
			Description: fmt.Sprintf("Background task completed %v ago. Verify notification was shown.", timeSinceComplete.Round(time.Second)),
			GitHubIssue: "#20525",
			Suggestion:  "If no notification appeared, this may be bug #20525. Check /tasks for status.",
			DetectedAt:  time.Now().Format(time.RFC3339),
			TaskID:      taskID,
		}
	}

	return nil
}

// =============================================================================
// Health Report Generation
// =============================================================================

// RunFullHealthCheck performs all detection checks and returns issues
func (h *TaskHealthState) RunFullHealthCheck() []*TaskHealthIssue {
	var allIssues []*TaskHealthIssue

	// Check headless mode
	if issue := h.DetectHeadlessMode(); issue != nil {
		allIssues = append(allIssues, issue)
	}

	// Check for zombie tasks
	zombieIssues := h.CheckForZombieTasks()
	allIssues = append(allIssues, zombieIssues...)

	// Check for silent completions on all recently completed
	for taskID := range h.CompletedTasks {
		if issue := h.DetectSilentCompletion(taskID); issue != nil {
			allIssues = append(allIssues, issue)
		}
	}

	// Update last check time
	h.LastHealthCheck = time.Now().Format(time.RFC3339)
	h.Save()

	return allIssues
}

// GenerateHealthReport creates a TOON-formatted health report
func (h *TaskHealthState) GenerateHealthReport() string {
	issues := h.RunFullHealthCheck()

	var sb strings.Builder
	sb.WriteString("[TASK_HEALTH]\n")
	sb.WriteString(fmt.Sprintf("checked: %s\n", h.LastHealthCheck))
	sb.WriteString(fmt.Sprintf("active_tasks: %d\n", len(h.ActiveTasks)))
	sb.WriteString(fmt.Sprintf("completed_tasks: %d\n", len(h.CompletedTasks)))
	sb.WriteString(fmt.Sprintf("zombie_candidates: %d\n", len(h.ZombieCandidates)))
	sb.WriteString(fmt.Sprintf("headless_mode: %t\n", h.HeadlessMode))
	sb.WriteString(fmt.Sprintf("task_tools_enabled: %t\n", h.TaskToolsEnabled))
	sb.WriteString(fmt.Sprintf("issues_found: %d\n", len(issues)))

	if len(issues) > 0 {
		sb.WriteString("\n[ISSUES]\n")
		for i, issue := range issues {
			sb.WriteString(fmt.Sprintf("  [%d]\n", i+1))
			sb.WriteString(fmt.Sprintf("    type: %s\n", issue.IssueType))
			sb.WriteString(fmt.Sprintf("    severity: %s\n", issue.Severity))
			sb.WriteString(fmt.Sprintf("    desc: %s\n", issue.Description))
			if issue.GitHubIssue != "" {
				sb.WriteString(fmt.Sprintf("    github: anthropics/claude-code%s\n", issue.GitHubIssue))
			}
			sb.WriteString(fmt.Sprintf("    fix: %s\n", issue.Suggestion))
			if issue.TaskID != "" {
				sb.WriteString(fmt.Sprintf("    task_id: %s\n", issue.TaskID))
			}
		}
	} else {
		sb.WriteString("\n[STATUS] OK - No issues detected.\n")
	}

	return sb.String()
}

// GetIssuesSummary returns a brief summary for hook injection
func (h *TaskHealthState) GetIssuesSummary() string {
	issues := h.RunFullHealthCheck()

	if len(issues) == 0 {
		return ""
	}

	var criticalCount, warningCount, infoCount int
	for _, issue := range issues {
		switch issue.Severity {
		case "critical":
			criticalCount++
		case "warning":
			warningCount++
		case "info":
			infoCount++
		}
	}

	if criticalCount > 0 {
		return fmt.Sprintf("[TASK_HEALTH] %d critical, %d warnings. Run: kavach orch task-health", criticalCount, warningCount)
	}

	if warningCount > 0 {
		return fmt.Sprintf("[TASK_HEALTH] %d warnings detected. Run: kavach orch task-health", warningCount)
	}

	return ""
}

// CleanupOldTasks removes tasks older than specified days from tracking
func (h *TaskHealthState) CleanupOldTasks(maxAgeDays int) int {
	cutoff := time.Now().AddDate(0, 0, -maxAgeDays)
	removed := 0

	for taskID, task := range h.CompletedTasks {
		completedAt, err := time.Parse(time.RFC3339, task.UpdatedAt)
		if err != nil {
			continue
		}
		if completedAt.Before(cutoff) {
			delete(h.CompletedTasks, taskID)
			removed++
		}
	}

	if removed > 0 {
		h.Save()
	}

	return removed
}

// truncateString truncates string to max length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
