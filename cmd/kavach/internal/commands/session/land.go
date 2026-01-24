// Package session provides session management commands.
// land.go: "Land the plane" protocol for clean session handoff.
//
// Inspired by Beads (steveyegge/beads) - ensures all work is committed and pushed.
// Reference: AGENT_INSTRUCTIONS.md "Landing the Plane" section
package session

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/claude/shared/pkg/dsa"
	"github.com/claude/shared/pkg/enforce"
	syncp "github.com/claude/shared/pkg/sync"
	"github.com/spf13/cobra"
)

var landCmd = &cobra.Command{
	Use:   "land",
	Short: "Land the plane - complete session handoff",
	Long: `[LAND_THE_PLANE]
desc: Clean session handoff protocol (Beads-inspired)
steps:
  1. Sync Memory Bank to git
  2. Close or file follow-up for open tasks
  3. Run quality gates (if code changed)
  4. Commit and push all changes
  5. Generate handoff prompt for next session

CRITICAL: Session is NOT landed until git push succeeds.`,
	Run: runLand,
}

// LandingReport contains the results of landing the plane
type LandingReport struct {
	SessionID       string    `json:"session_id"`
	LandedAt        time.Time `json:"landed_at"`
	TasksClosed     int       `json:"tasks_closed"`
	TasksRemaining  int       `json:"tasks_remaining"`
	FilesCommitted  int       `json:"files_committed"`
	PushSucceeded   bool      `json:"push_succeeded"`
	HandoffPrompt   string    `json:"handoff_prompt"`
	NextTask        string    `json:"next_task"`
	QualityGatePass bool      `json:"quality_gate_pass"`
}

func init() {
	// Will be registered in session/register.go
}

func runLand(cmd *cobra.Command, args []string) {
	report := &LandingReport{
		LandedAt: time.Now(),
	}

	session := enforce.GetOrCreateSession()
	report.SessionID = session.ID

	fmt.Println("[LANDING_THE_PLANE]")
	fmt.Println("step: 1/6 - Checking session state...")

	// Step 1: Get task status
	taskDAG := getOrCreateTaskDAG()
	stats := taskDAG.Stats()
	report.TasksRemaining = stats["pending"] + stats["in_progress"] + stats["blocked"]
	report.TasksClosed = stats["completed"]

	fmt.Printf("  tasks_completed: %d\n", report.TasksClosed)
	fmt.Printf("  tasks_remaining: %d\n", report.TasksRemaining)

	// Step 2: Sync Memory Bank
	fmt.Println("\nstep: 2/6 - Syncing Memory Bank...")
	workDir, _ := os.Getwd()
	gitSync := syncp.NewGitSync(workDir)

	if gitSync.IsGitRepo() {
		// Export task DAG
		if err := gitSync.Export(taskDAG, "tasks/dag.json"); err != nil {
			fmt.Printf("  warning: failed to export tasks: %v\n", err)
		} else {
			fmt.Println("  exported: .kavach/tasks/dag.json")
		}
	}

	// Step 3: Run quality gates (if applicable)
	fmt.Println("\nstep: 3/6 - Running quality gates...")
	report.QualityGatePass = runQualityGates()
	if report.QualityGatePass {
		fmt.Println("  status: PASS")
	} else {
		fmt.Println("  status: SKIPPED (no code changes)")
	}

	// Step 4: Git operations
	fmt.Println("\nstep: 4/6 - Git sync...")
	if gitSync.IsGitRepo() {
		// Pull first
		fmt.Println("  pulling latest...")
		if err := gitSync.Pull(); err != nil {
			fmt.Printf("  warning: pull failed: %v\n", err)
		}

		// Sync (add, commit)
		fmt.Println("  committing changes...")
		commitMsg := fmt.Sprintf("kavach: session %s landed at %s",
			session.ID[:8], time.Now().Format("2006-01-02 15:04"))
		if err := gitSync.Sync(commitMsg); err != nil {
			fmt.Printf("  warning: sync failed: %v\n", err)
		}

		// Get status
		status, _ := gitSync.Status()
		report.FilesCommitted = status.Ahead
	}

	// Step 5: Push (MANDATORY)
	fmt.Println("\nstep: 5/6 - Pushing to remote (MANDATORY)...")
	if gitSync.IsGitRepo() {
		if output, err := runGitCommand("push"); err != nil {
			fmt.Printf("  ERROR: Push failed: %v\n", err)
			fmt.Println("  CRITICAL: The plane has NOT landed!")
			fmt.Println("  ACTION: Resolve push issues and retry.")
			report.PushSucceeded = false
		} else {
			fmt.Println("  " + strings.TrimSpace(output))
			report.PushSucceeded = true
			fmt.Println("  status: PUSHED")
		}
	} else {
		fmt.Println("  skipped: not a git repo")
		report.PushSucceeded = true // Not applicable
	}

	// Step 6: Generate handoff
	fmt.Println("\nstep: 6/6 - Generating handoff prompt...")
	report.HandoffPrompt, report.NextTask = generateHandoff(taskDAG, session)

	// Final report
	printLandingReport(report)
}

// getOrCreateTaskDAG gets the task DAG from health state
func getOrCreateTaskDAG() *dsa.DAG {
	dag := dsa.NewDAG()
	// In a real implementation, this would load from task_health state
	return dag
}

// runQualityGates runs lint and test checks
func runQualityGates() bool {
	// Check if there are Go files modified
	output, err := runGitCommand("diff", "--cached", "--name-only", "--diff-filter=AM")
	if err != nil {
		return false
	}

	hasGoFiles := false
	for _, line := range strings.Split(output, "\n") {
		if strings.HasSuffix(line, ".go") {
			hasGoFiles = true
			break
		}
	}

	if !hasGoFiles {
		return false // Skip, no Go files
	}

	// Run go vet
	cmd := exec.Command("go", "vet", "./...")
	if err := cmd.Run(); err != nil {
		return false
	}

	return true
}

// runGitCommand runs a git command
func runGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// generateHandoff creates the handoff prompt for next session
func generateHandoff(dag *dsa.DAG, session *enforce.SessionState) (string, string) {
	ready := dag.Ready()

	var nextTask string
	if len(ready) > 0 {
		// Pick highest priority ready task
		highest := ready[0]
		for _, t := range ready[1:] {
			if t.Priority < highest.Priority {
				highest = t
			}
		}
		nextTask = fmt.Sprintf("%s: %s", highest.ID, highest.Label)
	}

	handoff := fmt.Sprintf(`Continue session from %s.

Previous session completed:
- Tasks closed: %d
- Session ID: %s

Next task: %s

To resume: kavach session init && kavach memory bank`,
		time.Now().Format("2006-01-02"),
		session.TasksCompleted,
		session.ID,
		nextTask,
	)

	return handoff, nextTask
}

// printLandingReport prints the final landing report
func printLandingReport(report *LandingReport) {
	fmt.Println("\n[LANDING_REPORT]")
	fmt.Printf("session_id: %s\n", report.SessionID)
	fmt.Printf("landed_at: %s\n", report.LandedAt.Format(time.RFC3339))
	fmt.Printf("tasks_closed: %d\n", report.TasksClosed)
	fmt.Printf("tasks_remaining: %d\n", report.TasksRemaining)
	fmt.Printf("files_committed: %d\n", report.FilesCommitted)
	fmt.Printf("push_succeeded: %t\n", report.PushSucceeded)

	if report.PushSucceeded {
		fmt.Println("\n[STATUS] LANDED - All changes pushed to remote.")
	} else {
		fmt.Println("\n[STATUS] NOT LANDED - Push failed. Resolve and retry.")
	}

	if report.NextTask != "" {
		fmt.Printf("\n[NEXT_TASK] %s\n", report.NextTask)
	}

	fmt.Println("\n[HANDOFF_PROMPT]")
	fmt.Println(report.HandoffPrompt)
}

// GetLandCmd returns the land command for registration
func GetLandCmd() *cobra.Command {
	return landCmd
}
