// Package gates provides hook gates for Claude Code.
// code_guard.go: Prevent premature code removal and hallucination-based changes.
// CRITICAL: Blocks removal of functions without understanding use case.
package gates

import (
	"regexp"
	"strings"

	"github.com/claude/shared/pkg/hook"
	"github.com/spf13/cobra"
)

var codeGuardHookMode bool

var codeGuardCmd = &cobra.Command{
	Use:   "code-guard",
	Short: "Prevent premature code removal and hallucination-based changes",
	Run:   runCodeGuardGate,
}

func init() {
	codeGuardCmd.Flags().BoolVar(&codeGuardHookMode, "hook", false, "Hook mode")
}

// Patterns that indicate function/method definitions
var functionPatterns = []*regexp.Regexp{
	// Go
	regexp.MustCompile(`func\s+(\w+)\s*\(`),
	regexp.MustCompile(`func\s+\([^)]+\)\s+(\w+)\s*\(`),
	// Rust
	regexp.MustCompile(`fn\s+(\w+)\s*[<(]`),
	regexp.MustCompile(`pub\s+fn\s+(\w+)\s*[<(]`),
	regexp.MustCompile(`impl\s+\w+`),
	// TypeScript/JavaScript
	regexp.MustCompile(`function\s+(\w+)\s*\(`),
	regexp.MustCompile(`const\s+(\w+)\s*=\s*(?:async\s*)?\(`),
	regexp.MustCompile(`(\w+)\s*:\s*(?:async\s*)?\(`),
	// Python
	regexp.MustCompile(`def\s+(\w+)\s*\(`),
	regexp.MustCompile(`class\s+(\w+)`),
}

// Patterns indicating stub/placeholder code
var stubPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)todo|fixme|xxx|hack`),
	regexp.MustCompile(`(?i)not\s+implemented`),
	regexp.MustCompile(`(?i)placeholder`),
	regexp.MustCompile(`(?i)stub`),
	regexp.MustCompile(`unimplemented!`),                         // Rust
	regexp.MustCompile(`todo!`),                                  // Rust
	regexp.MustCompile(`pass\s*$`),                               // Python
	regexp.MustCompile(`raise\s+NotImplementedError`),            // Python
	regexp.MustCompile(`throw\s+new\s+Error.*not\s+implemented`), // JS/TS
}

// runCodeGuardGate checks Edit operations for premature code removal.
func runCodeGuardGate(cmd *cobra.Command, args []string) {
	if !codeGuardHookMode {
		cmd.Help()
		return
	}

	input := hook.MustReadHookInput()

	// Only check Edit tool
	if input.ToolName != "Edit" {
		hook.ExitSilent()
	}

	oldString := input.GetString("old_string")
	newString := input.GetString("new_string")
	filePath := input.GetString("file_path")

	// Check 1: Detect function removal (old has function, new doesn't or is much shorter)
	removedFunctions := detectFunctionRemoval(oldString, newString)
	if len(removedFunctions) > 0 {
		// Check if the removed functions had stubs/placeholders
		hasStubs := containsStubPatterns(oldString)
		if hasStubs {
			// Blocking removal of unimplemented code
			hook.ExitBlockTOON("CODE_GUARD",
				"BLOCK_REMOVAL:unimplemented_code:functions:"+strings.Join(removedFunctions, ",")+
					":REASON:Never remove TODO/stub functions without implementing them first")
		}

		// Warn about function removal even if not stubs
		if len(newString) < len(oldString)/2 {
			hook.ExitBlockTOON("CODE_GUARD",
				"BLOCK_REMOVAL:significant_code_reduction:functions:"+strings.Join(removedFunctions, ",")+
					":REASON:Verify use case before removing functions. Read file first if not done.")
		}
	}

	// Check 2: Detect removal of TODO/FIXME comments without implementation
	if containsStubPatterns(oldString) && !containsStubPatterns(newString) {
		// Check if new code is actually implementing (longer, more complex)
		if len(newString) <= len(oldString) {
			hook.ExitBlockTOON("CODE_GUARD",
				"BLOCK_REMOVAL:stub_removed_without_implementation:file:"+filePath+
					":REASON:TODO/FIXME removed but code not expanded. Implement before removing stubs.")
		}
	}

	// Check 3: Detect empty replacement (complete deletion)
	if strings.TrimSpace(newString) == "" && len(oldString) > 50 {
		hook.ExitBlockTOON("CODE_GUARD",
			"BLOCK_REMOVAL:complete_deletion:file:"+filePath+
				":REASON:Cannot delete significant code block. Verify intent first.")
	}

	// Check 4: Detect impl block removal (Rust)
	if strings.Contains(oldString, "impl ") && !strings.Contains(newString, "impl ") {
		hook.ExitBlockTOON("CODE_GUARD",
			"BLOCK_REMOVAL:impl_block:file:"+filePath+
				":REASON:Cannot remove impl block without understanding trait implementation.")
	}

	hook.ExitSilent()
}

// detectFunctionRemoval finds functions in old that are missing in new.
func detectFunctionRemoval(old, new string) []string {
	var removed []string

	for _, pattern := range functionPatterns {
		oldMatches := pattern.FindAllStringSubmatch(old, -1)
		newContent := new

		for _, match := range oldMatches {
			if len(match) > 1 {
				funcName := match[1]
				// Check if function name exists in new content
				if !strings.Contains(newContent, funcName) {
					removed = append(removed, funcName)
				}
			}
		}
	}

	return removed
}

// containsStubPatterns checks if content has TODO/stub markers.
func containsStubPatterns(content string) bool {
	for _, pattern := range stubPatterns {
		if pattern.MatchString(content) {
			return true
		}
	}
	return false
}
