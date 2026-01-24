// Package orch provides orchestration subcommands.
// task_health.go: Task health monitoring and bug detection command.
//
// Detects runtime bugs in Claude Code 2.1.19+ task system:
// - #19894: Stale task count
// - #17542: Zombie tasks
// - #20463: Headless mode issues
// - #20525: Silent completion
package orch

import (
	"fmt"

	"github.com/claude/cmd/kavach/internal/commands/gates"
	"github.com/spf13/cobra"
)

var taskHealthCleanup bool
var taskHealthCleanupDays int

var taskHealthCmd = &cobra.Command{
	Use:   "task-health",
	Short: "Check task system health and detect runtime bugs",
	Long: `[TASK_HEALTH]
desc: Runtime bug detection for Claude Code 2.1.19+ task system
detects:
  - Stale task counts (GitHub #19894)
  - Zombie background tasks (GitHub #17542)
  - Headless mode issues (GitHub #20463)
  - Silent completion (GitHub #20525)

usage:
  kavach orch task-health           # Run full health check
  kavach orch task-health --cleanup # Clean old completed tasks`,
	Run: runTaskHealth,
}

func init() {
	taskHealthCmd.Flags().BoolVar(&taskHealthCleanup, "cleanup", false, "Clean up old completed tasks")
	taskHealthCmd.Flags().IntVar(&taskHealthCleanupDays, "days", 7, "Days to keep completed tasks (with --cleanup)")
}

func runTaskHealth(cmd *cobra.Command, args []string) {
	health := gates.GetTaskHealth()

	if taskHealthCleanup {
		removed := health.CleanupOldTasks(taskHealthCleanupDays)
		fmt.Printf("[CLEANUP] Removed %d tasks older than %d days\n\n", removed, taskHealthCleanupDays)
	}

	report := health.GenerateHealthReport()
	fmt.Print(report)

	// Print summary
	summary := health.GetIssuesSummary()
	if summary != "" {
		fmt.Printf("\n%s\n", summary)
	}
}
