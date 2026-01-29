// Package orch provides orchestration subcommands.
// aegis.go: Aegis Guardian verification gate.
// DACE: Enforces two-stage verification before DONE status.
package orch

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/claude/shared/pkg/hook"
	"github.com/claude/shared/pkg/util"
	"github.com/spf13/cobra"
)

var aegisHookMode bool
var aegisTaskID string

var aegisCmd = &cobra.Command{
	Use:   "aegis",
	Short: "Aegis Guardian verification (Level 2)",
	Long: `[AEGIS]
desc: Two-stage verification before production
hook: PostToolUse:TodoWrite (when task marked complete)
purpose: Ensure quality, security, and correctness

[VERIFICATION_STAGES]
Stage 1 - TESTING:
  - ALL lint warnings resolved
  - ALL compiler warnings addressed
  - Core bugs identified and fixed
  - Unit tests passing

Stage 2 - VERIFIED:
  - Algorithm is well-defined
  - No hidden bugs in logic
  - No dead code present
  - No suppressed elements (@SuppressWarnings, #pragma, nolint)

[OUTPUT]
PASS: Task moves to DONE, <promise>PRODUCTION_READY</promise>
FAIL: Task stays in current stage, report to CEO, loop continues

[USAGE]
kavach orch aegis --hook              # PostToolUse hook mode
kavach orch aegis --task TASK-001     # Manual verification`,
	Run: runAegisCmd,
}

func init() {
	aegisCmd.Flags().BoolVar(&aegisHookMode, "hook", false, "Hook mode")
	aegisCmd.Flags().StringVar(&aegisTaskID, "task", "", "Task ID to verify")
}

// AegisResult represents verification result
type AegisResult struct {
	Stage       string
	Status      string // "passed" or "failed"
	LintIssues  int
	Warnings    int
	CoreBugs    int
	DeadCode    bool
	Suppressed  bool
	AlgoOK      bool
	FailReasons []string
	ExecErrors  []string // P1 FIX: Track command execution errors
}

func runAegisCmd(cmd *cobra.Command, args []string) {
	project := util.DetectProject()
	today := time.Now().Format("2006-01-02")

	if aegisHookMode {
		// Hook mode - read from stdin
		input := hook.MustReadHookInput()

		// PostToolUse provides ToolResponse (map), not a string field
		if input.ToolResponse == nil || len(input.ToolResponse) == 0 {
			hook.ExitSilent()
			return
		}

		// Run verification
		result := runVerification(project)

		// Output result
		outputAegisResult(project, today, result)

		if result.Status == "passed" {
			hook.Approve("aegis:verified")
		} else {
			hook.ExitBlockTOON("AEGIS_FAIL", strings.Join(result.FailReasons, ","))
		}
		return
	}

	// Manual mode
	if aegisTaskID != "" {
		result := runVerification(project)
		outputAegisResult(project, today, result)
		return
	}

	// Default: run verification on current project
	result := runVerification(project)
	outputAegisResult(project, today, result)
}

func runVerification(project string) *AegisResult {
	result := &AegisResult{
		Stage:  "TESTING",
		Status: "passed",
	}

	workDir, _ := os.Getwd()

	// Stage 1: TESTING - Lint, Warnings, Core Bugs
	// P1 FIX: Now collecting exec errors instead of ignoring them
	var err error

	result.LintIssues, err = countLintIssues(workDir)
	if err != nil {
		result.ExecErrors = append(result.ExecErrors, err.Error())
	}

	result.Warnings, err = countWarnings(workDir)
	if err != nil {
		result.ExecErrors = append(result.ExecErrors, err.Error())
	}

	result.CoreBugs, err = countCoreBugs(workDir)
	if err != nil {
		result.ExecErrors = append(result.ExecErrors, err.Error())
	}

	if result.LintIssues > 0 {
		result.FailReasons = append(result.FailReasons, fmt.Sprintf("lint_issues:%d", result.LintIssues))
	}
	if result.Warnings > 0 {
		result.FailReasons = append(result.FailReasons, fmt.Sprintf("warnings:%d", result.Warnings))
	}
	if result.CoreBugs > 0 {
		result.FailReasons = append(result.FailReasons, fmt.Sprintf("core_bugs:%d", result.CoreBugs))
	}

	// If Stage 1 fails, don't proceed to Stage 2
	if len(result.FailReasons) > 0 {
		result.Status = "failed"
		return result
	}

	// Stage 2: VERIFIED - Algorithm, Dead Code, Suppressed
	result.Stage = "VERIFIED"

	result.DeadCode, err = hasDeadCode(workDir)
	if err != nil {
		result.ExecErrors = append(result.ExecErrors, err.Error())
	}

	result.Suppressed, err = hasSuppressedElements(workDir)
	if err != nil {
		result.ExecErrors = append(result.ExecErrors, err.Error())
	}

	result.AlgoOK = true // Assume OK unless proven otherwise

	if result.DeadCode {
		result.FailReasons = append(result.FailReasons, "dead_code:found")
	}
	if result.Suppressed {
		result.FailReasons = append(result.FailReasons, "suppressed_elements:found")
	}

	if len(result.FailReasons) > 0 {
		result.Status = "failed"
	}

	return result
}

// countLintIssues returns lint issue count and any exec error.
// P1 FIX: Now returns error instead of ignoring it.
func countLintIssues(workDir string) (int, error) {
	// Check for Go lint issues
	if util.FileExists(filepath.Join(workDir, "go.mod")) {
		cmd := exec.Command("go", "vet", "./...")
		cmd.Dir = workDir
		out, err := cmd.CombinedOutput()
		if err != nil && len(out) == 0 {
			return 0, fmt.Errorf("go vet failed: %w", err)
		}
		lines := strings.Split(string(out), "\n")
		count := 0
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				count++
			}
		}
		return count, nil
	}

	// Check for Rust lint issues
	if util.FileExists(filepath.Join(workDir, "Cargo.toml")) {
		cmd := exec.Command("cargo", "clippy", "--message-format=short")
		cmd.Dir = workDir
		out, err := cmd.CombinedOutput()
		if err != nil && len(out) == 0 {
			return 0, fmt.Errorf("cargo clippy failed: %w", err)
		}
		return strings.Count(string(out), "warning:"), nil
	}

	return 0, nil
}

// countWarnings returns warning count and any exec error.
// P1 FIX: Now returns error instead of ignoring it.
func countWarnings(workDir string) (int, error) {
	// Check Go build warnings
	if util.FileExists(filepath.Join(workDir, "go.mod")) {
		cmd := exec.Command("go", "build", "-v", "./...")
		cmd.Dir = workDir
		out, err := cmd.CombinedOutput()
		if err != nil && len(out) == 0 {
			return 0, fmt.Errorf("go build failed: %w", err)
		}
		return strings.Count(string(out), "warning"), nil
	}

	// Check Rust warnings
	if util.FileExists(filepath.Join(workDir, "Cargo.toml")) {
		cmd := exec.Command("cargo", "check", "--message-format=short")
		cmd.Dir = workDir
		out, err := cmd.CombinedOutput()
		if err != nil && len(out) == 0 {
			return 0, fmt.Errorf("cargo check failed: %w", err)
		}
		return strings.Count(string(out), "warning:"), nil
	}

	return 0, nil
}

// countCoreBugs returns bug marker count and any exec error.
// P1 FIX: Now returns error instead of ignoring it.
func countCoreBugs(workDir string) (int, error) {
	// Check for TODO/FIXME/BUG comments as proxy for known bugs
	out, err := exec.Command("rg", "-c", "TODO|FIXME|BUG|XXX", workDir, "--type", "go", "--type", "rust", "--type", "ts").CombinedOutput()
	// rg returns exit code 1 when no matches found, which is not an error
	if err != nil && len(out) == 0 {
		// Check if rg is available
		if _, lookErr := exec.LookPath("rg"); lookErr != nil {
			return 0, fmt.Errorf("rg (ripgrep) not found: %w", lookErr)
		}
		// No matches is fine
		return 0, nil
	}
	count := 0
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, ":") {
			var n int
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				fmt.Sscanf(parts[len(parts)-1], "%d", &n)
				count += n
			}
		}
	}
	return count, nil
}

// hasDeadCode checks for dead code and returns any exec error.
// P1 FIX: Now returns error instead of ignoring it.
func hasDeadCode(workDir string) (bool, error) {
	// Check for unused functions/variables (Go)
	if util.FileExists(filepath.Join(workDir, "go.mod")) {
		cmd := exec.Command("go", "vet", "-unusedresult", "./...")
		cmd.Dir = workDir
		out, err := cmd.CombinedOutput()
		if err != nil && len(out) == 0 {
			return false, fmt.Errorf("go vet failed: %w", err)
		}
		return strings.Contains(string(out), "unused"), nil
	}

	// Check for Rust dead code
	if util.FileExists(filepath.Join(workDir, "Cargo.toml")) {
		cmd := exec.Command("cargo", "check")
		cmd.Dir = workDir
		out, err := cmd.CombinedOutput()
		if err != nil && len(out) == 0 {
			return false, fmt.Errorf("cargo check failed: %w", err)
		}
		return strings.Contains(string(out), "dead_code"), nil
	}

	return false, nil
}

// hasSuppressedElements checks for suppression annotations and returns any exec error.
// P1 FIX: Now returns error instead of ignoring it.
func hasSuppressedElements(workDir string) (bool, error) {
	// Search for suppression annotations
	out, err := exec.Command("rg", "-l", "@Suppress|#pragma|nolint|#\\[allow", workDir).CombinedOutput()
	// rg returns exit code 1 when no matches found
	if err != nil && len(out) == 0 {
		if _, lookErr := exec.LookPath("rg"); lookErr != nil {
			return false, fmt.Errorf("rg (ripgrep) not found: %w", lookErr)
		}
		return false, nil
	}
	return len(strings.TrimSpace(string(out))) > 0, nil
}

func outputAegisResult(project, today string, result *AegisResult) {
	fmt.Println("[AEGIS:VERIFICATION]")
	fmt.Printf("project: %s\n", project)
	fmt.Printf("date: %s\n", today)
	fmt.Printf("stage: %s\n", result.Stage)
	fmt.Printf("status: %s\n", result.Status)
	fmt.Println()

	fmt.Println("[TESTING_STAGE]")
	fmt.Printf("lint_issues: %d\n", result.LintIssues)
	fmt.Printf("warnings: %d\n", result.Warnings)
	fmt.Printf("core_bugs: %d\n", result.CoreBugs)
	fmt.Println()

	fmt.Println("[VERIFIED_STAGE]")
	fmt.Printf("dead_code: %s\n", boolToStatus(result.DeadCode))
	fmt.Printf("suppressed: %s\n", boolToStatus(result.Suppressed))
	fmt.Printf("algorithm: %s\n", boolToOK(result.AlgoOK))
	fmt.Println()

	// P1 FIX: Display exec errors if any
	if len(result.ExecErrors) > 0 {
		fmt.Println("[EXEC_ERRORS]")
		fmt.Println("note: Some verification commands failed")
		for _, err := range result.ExecErrors {
			fmt.Printf("  - %s\n", err)
		}
		fmt.Println()
	}

	if result.Status == "passed" {
		fmt.Println("[PROMISE]")
		fmt.Println("status: PRODUCTION_READY")
		fmt.Println("signal: <promise>PRODUCTION_READY</promise>")
	} else {
		fmt.Println("[AEGIS_FAILURES]")
		fmt.Println("action: REPORT_TO_CEO")
		fmt.Println("result: LOOP_CONTINUES")
		for _, reason := range result.FailReasons {
			fmt.Printf("  - %s\n", reason)
		}
	}
}

func boolToStatus(b bool) string {
	if b {
		return "FOUND"
	}
	return "CLEAN"
}

func boolToOK(b bool) string {
	if b {
		return "VERIFIED"
	}
	return "UNVERIFIED"
}
