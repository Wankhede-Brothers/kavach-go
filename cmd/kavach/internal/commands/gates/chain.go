// Package gates provides hook gates for Claude Code.
// chain.go: Multi-agent verification chain gate.
// Pattern: Intent → CEO → Aegis → Research
// Reference: Microsoft Defender webhook pattern for AI agents
package gates

import (
	"os"

	"github.com/claude/shared/pkg/chain"
	"github.com/claude/shared/pkg/enforce"
	"github.com/claude/shared/pkg/hook"
	"github.com/claude/shared/pkg/types"
	"github.com/spf13/cobra"
)

var chainHookMode bool
var chainDebugMode bool

var chainCmd = &cobra.Command{
	Use:   "chain",
	Short: "Multi-agent verification chain (Intent → CEO → Aegis → Research)",
	Long: `Runs the complete verification chain for high-risk operations.

The chain validates:
1. INTENT: Classifies user intent and risk level
2. CEO: Validates delegation strategy and agent assignment
3. AEGIS: Security verification and threat detection
4. RESEARCH: TABULA_RASA compliance (research before code)

Use this gate for Write, Edit, Task, and other high-risk tools.`,
	Run: runChainGate,
}

func init() {
	chainCmd.Flags().BoolVar(&chainHookMode, "hook", false, "Hook mode")
	chainCmd.Flags().BoolVar(&chainDebugMode, "debug", false, "Debug mode")
}

func runChainGate(cmd *cobra.Command, args []string) {
	if !chainHookMode {
		cmd.Help()
		return
	}

	// Enable debug if flag set
	if chainDebugMode {
		os.Setenv("KAVACH_DEBUG", "1")
	}

	input := hook.MustReadHookInput()
	session := enforce.GetOrCreateSession()

	// Get prompt from various sources
	prompt := getPromptFromInput(input)

	// Create and run the chain
	runner := chain.NewRunner(session.ID)
	state := runner.RunFull(prompt, input.ToolName, input.ToolInput, session.ResearchDone)

	// Handle result based on chain status
	if state.IsBlocked() {
		blockReason := state.GetBlockReason()
		context := runner.ToTOON()

		// Use new Claude Code 2026 format
		hook.Output(&types.HookResponse{
			HookSpecificOutput: &types.HookSpecificOutput{
				HookEventName:            "PreToolUse",
				PermissionDecision:       "deny",
				PermissionDecisionReason: blockReason,
				AdditionalContext:        context,
			},
		})
		os.Exit(0)
	}

	// Chain passed - add context if there are warnings
	hasWarnings := false
	for _, r := range state.Results {
		if r.Status == "warn" {
			hasWarnings = true
			break
		}
	}

	if hasWarnings {
		context := runner.ToTOON()
		hook.Output(&types.HookResponse{
			HookSpecificOutput: &types.HookSpecificOutput{
				HookEventName:            "PreToolUse",
				PermissionDecision:       "allow",
				PermissionDecisionReason: "Chain passed with warnings",
				AdditionalContext:        context,
			},
		})
		os.Exit(0)
	}

	// Silent pass
	hook.ExitSilent()
}

// getPromptFromInput extracts the prompt from various input sources.
func getPromptFromInput(input *hook.Input) string {
	// Direct prompt (UserPromptSubmit)
	if input.Prompt != "" {
		return input.Prompt
	}

	// Task prompt
	if p := input.GetString("prompt"); p != "" {
		return p
	}

	// Write/Edit content as context
	if content := input.GetString("content"); content != "" {
		return content
	}

	// Bash command as context
	if cmd := input.GetString("command"); cmd != "" {
		return cmd
	}

	// Description
	if desc := input.GetString("description"); desc != "" {
		return desc
	}

	return ""
}
