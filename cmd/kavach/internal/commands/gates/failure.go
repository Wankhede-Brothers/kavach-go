// failure.go: PostToolUseFailure gate.
// Reacts to tool failures: logs patterns, suggests fixes.
package gates

import (
	"strings"

	"github.com/claude/shared/pkg/hook"
	"github.com/spf13/cobra"
)

var failureHookMode bool

var failureCmd = &cobra.Command{
	Use:   "failure",
	Short: "PostToolUseFailure gate - react to tool failures",
	Long: `[FAILURE_GATE]
desc: Handle tool execution failures
hook: PostToolUseFailure

[USAGE]
kavach gates failure --hook`,
	Run: runFailureGate,
}

func init() {
	failureCmd.Flags().BoolVar(&failureHookMode, "hook", false, "Hook mode")
}

func runFailureGate(cmd *cobra.Command, args []string) {
	if !failureHookMode {
		cmd.Help()
		return
	}

	input := hook.MustReadHookInput()
	toolName := input.ToolName

	// Extract error from tool_response
	errMsg := extractErrorMessage(input.ToolResponse)

	// Detect common failure patterns and suggest fixes
	suggestion := detectFailurePattern(toolName, errMsg)
	if suggestion != "" {
		hook.ExitModifyTOON("FAILURE_GATE", map[string]string{
			"tool":       toolName,
			"error":      truncate(errMsg, 200),
			"suggestion": suggestion,
		})
	}

	hook.ExitSilent()
}

func extractErrorMessage(resp map[string]interface{}) string {
	if resp == nil {
		return ""
	}
	if err, ok := resp["error"].(string); ok {
		return err
	}
	if stderr, ok := resp["stderr"].(string); ok {
		return stderr
	}
	return ""
}

func detectFailurePattern(tool, err string) string {
	if err == "" {
		return ""
	}
	lower := strings.ToLower(err)

	switch tool {
	case "Bash":
		if strings.Contains(lower, "command not found") {
			return "Binary not installed or not in PATH"
		}
		if strings.Contains(lower, "permission denied") {
			return "Check file permissions or use appropriate user"
		}
	case "Write", "Edit":
		if strings.Contains(lower, "no such file") {
			return "Parent directory may not exist - create it first"
		}
		if strings.Contains(lower, "not unique") {
			return "Edit old_string not unique - add more surrounding context"
		}
	case "Read":
		if strings.Contains(lower, "no such file") {
			return "File does not exist - verify path with Glob first"
		}
	}
	return ""
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
