// Package gates provides hook gates for Claude Code.
// read.go: Read file blocker gate.
// DACE: Uses shared/pkg/config for JSON-based dynamic patterns.
package gates

import (
	"github.com/claude/shared/pkg/config"
	"github.com/claude/shared/pkg/hook"
	"github.com/claude/shared/pkg/patterns"
	"github.com/spf13/cobra"
)

var readHookMode bool

var readCmd = &cobra.Command{
	Use:   "read",
	Short: "Read file blocker gate",
	Run:   runReadGate,
}

func init() {
	readCmd.Flags().BoolVar(&readHookMode, "hook", false, "Hook mode")
}

func runReadGate(cmd *cobra.Command, args []string) {
	if !readHookMode {
		cmd.Help()
		return
	}

	input := hook.MustReadHookInput()

	// Support Read, Glob, and Grep (all hooked to this gate in settings.json)
	var filePath string
	switch input.ToolName {
	case "Read":
		filePath = input.GetString("file_path")
	case "Glob":
		filePath = input.GetString("path") // Glob uses "path" not "file_path"
	case "Grep":
		filePath = input.GetString("path")
	default:
		hook.ExitSilent()
	}

	// Glob/Grep path is optional (defaults to cwd), so only block Read with no path
	if filePath == "" && input.ToolName == "Read" {
		hook.ExitBlockTOON("READ", "no_file_path")
	}
	if filePath == "" {
		hook.ExitSilent() // Glob/Grep with no path = cwd, always allowed
	}

	// Check blocked paths from gates/config.json (priority)
	if config.IsBlockedPath(filePath) {
		hook.ExitBlockTOON("READ", "blocked_path")
	}

	// Check blocked extensions (private keys, etc.)
	if config.IsBlockedExtension(filePath) {
		hook.ExitBlockTOON("READ", "blocked_extension")
	}

	// Legacy: Check sensitive files using shared patterns
	if patterns.IsSensitive(filePath) {
		hook.ExitBlockTOON("READ", "sensitive_file")
	}

	// Warn for files that may contain secrets
	if config.IsWarnPath(filePath) {
		hook.ExitModifyTOON("READ", map[string]string{
			"warn": "may_contain_secrets",
		})
	}

	// Warn for large files
	if patterns.IsLargeFile(filePath) {
		hook.ExitModifyTOON("READ", map[string]string{
			"warn": "large_file",
		})
	}

	hook.ExitSilent()
}
