// Package memory provides memory bank commands.
// sync.go: Memory sync command entry point.
// DACE: Micro-modular - types, scratchpad, kanban, helpers in separate files.
package memory

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/claude/shared/pkg/hook"
	"github.com/claude/shared/pkg/util"
	"github.com/spf13/cobra"
)

var syncHookMode bool
var syncTask string
var syncStatus string

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync task completion to Memory Bank",
	Long: `[SYNC]
desc: Sync tool events to Memory Bank (STM session log + scratchpad)
hook: PostToolUse:TaskCreate|TaskUpdate|Write|Edit|Bash|Task
purpose: Keep scratchpad and session log in sync with task progress

[TRIGGERS]
- PostToolUse:TodoWrite - When todo list is updated
- Manual: kavach memory sync --task "task name" --status completed

[UPDATES]
- kanban.toon: Move task between columns
- scratchpad.toon: Update current task state
- roadmap.toon: Update phase progress

[HOOK_MODE]
Reads tool_result from stdin (Claude Code hook format)
Parses TodoWrite output and syncs to Memory Bank

[USAGE]
kavach memory sync --hook < tool_result.json
kavach memory sync --task "Implement API" --status completed`,
	Run: runSyncCmd,
}

func init() {
	syncCmd.Flags().BoolVar(&syncHookMode, "hook", false, "Hook mode (reads from stdin)")
	syncCmd.Flags().StringVar(&syncTask, "task", "", "Task name to sync")
	syncCmd.Flags().StringVar(&syncStatus, "status", "", "Task status (pending, in_progress, completed)")
}

func runSyncCmd(cmd *cobra.Command, args []string) {
	// BUG FIX: Use exact matching for writes to prevent updating wrong project
	project := util.DetectProjectForWrite()
	today := time.Now().Format("2006-01-02")

	if syncHookMode {
		runSyncHookMode(project, today)
		return
	}

	// Manual mode
	if syncTask != "" && syncStatus != "" {
		updateScratchpadManual(project, today, syncTask, syncStatus)
		fmt.Printf("[SYNC] %s -> %s\n", syncTask, syncStatus)
		return
	}

	cmd.Help()
}

func runSyncHookMode(project, today string) {
	input := hook.MustReadHookInput()
	toolName := input.ToolName

	switch toolName {
	case "TaskCreate":
		syncTaskCreate(input, project, today)
	case "TaskUpdate":
		syncTaskUpdate(input, project, today)
	case "Write", "Edit":
		syncFileChange(input, project, today)
	case "Bash":
		syncBashResult(input, project, today)
	case "Task":
		syncAgentResult(input, project, today)
	default:
		// Unknown tool — silent pass
	}

	hook.ExitSilent()
}

func syncTaskCreate(input *hook.Input, project, today string) {
	subject := input.GetString("subject")
	description := input.GetString("description")
	if subject == "" {
		return
	}
	appendToSTMLog(project, today, "task_created", subject, description)
}

func syncTaskUpdate(input *hook.Input, project, today string) {
	taskID := input.GetString("taskId")
	status := input.GetString("status")
	if taskID == "" || status == "" {
		return
	}
	subject := input.GetString("subject")
	appendToSTMLog(project, today, "task_"+status, subject, taskID)

	// Update scratchpad on completion
	if status == "completed" && subject != "" {
		updateScratchpadManual(project, today, subject, "completed")
	}
	if status == "in_progress" && subject != "" {
		updateScratchpadManual(project, today, subject, "in_progress")
	}
}

func syncFileChange(input *hook.Input, project, today string) {
	filePath := input.GetString("file_path")
	if filePath == "" {
		return
	}
	appendToSTMLog(project, today, "file_"+input.ToolName, filePath, "")
}

func syncBashResult(input *hook.Input, project, today string) {
	command := input.GetString("command")
	if command == "" {
		return
	}
	// Only log significant commands (builds, tests, deploys)
	for _, sig := range []string{"build", "test", "deploy", "cargo", "go ", "bun ", "npm ", "git commit"} {
		if containsStr(command, sig) {
			appendToSTMLog(project, today, "bash_"+sig, command, "")
			return
		}
	}
}

func syncAgentResult(input *hook.Input, project, today string) {
	desc := input.GetString("description")
	agentType := input.GetString("subagent_type")
	if desc == "" {
		return
	}
	appendToSTMLog(project, today, "agent_"+agentType, desc, "")
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && strings.Contains(strings.ToLower(s), sub)
}

func appendToSTMLog(project, today, eventType, subject, detail string) {
	stmDir, err := util.EnsureScratchpadDir(project)
	if err != nil {
		return
	}
	logPath := stmDir + "/session_log.toon"

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	ts := time.Now().Format("15:04:05")
	line := fmt.Sprintf("[%s] %s: %s", ts, eventType, sanitizeTitle(subject))
	if detail != "" {
		line += " | " + sanitizeTitle(detail)
	}
	fmt.Fprintln(f, line)
}
