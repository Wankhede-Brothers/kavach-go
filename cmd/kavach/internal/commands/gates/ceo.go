// Package gates provides hook gates for Claude Code.
// ceo.go: CEO orchestration gate with DACE skill injection.
// NO HARDCODING - All patterns loaded from config/*.toon at runtime.
package gates

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/claude/shared/pkg/config"
	"github.com/claude/shared/pkg/dag"
	"github.com/claude/shared/pkg/enforce"
	"github.com/claude/shared/pkg/hook"
	"github.com/spf13/cobra"
)

var ceoHookMode bool

var ceoCmd = &cobra.Command{
	Use:   "ceo",
	Short: "CEO orchestration gate",
	Long: `[CEO_GATE]
desc: Orchestration gate with DACE skill injection
flow: Task -> Skill detection -> Agent validation -> Context injection
config: Patterns loaded from config/skill-patterns.toon (NO HARDCODING)`,
	Run: runCEOGate,
}

func init() {
	ceoCmd.Flags().BoolVar(&ceoHookMode, "hook", false, "Hook mode")
}

func runCEOGate(cmd *cobra.Command, args []string) {
	if !ceoHookMode {
		cmd.Help()
		return
	}

	input := hook.MustReadHookInput()

	if input.ToolName != "Task" {
		hook.ExitSilent()
	}

	subagentType := input.GetString("subagent_type")
	if subagentType == "" {
		hook.ExitBlockTOON("CEO", "Task_requires_subagent_type")
	}

	// Validate agent using dynamic config (not hardcoded)
	// SECURITY FIX: BLOCK unknown agents instead of just warning
	if !config.IsValidAgent(subagentType) {
		hook.ExitBlockTOON("CEO", "unknown_agent:"+subagentType)
	}

	// DACE: Detect skill from task prompt using dynamic config
	prompt := input.GetString("prompt")
	skill := detectSkillFromConfig(prompt)
	today := time.Now().Format("2006-01-02")

	// P1 FIX: CEO orchestration flow directive
	if config.IsEngineer(subagentType) {
		orchDirective := map[string]string{
			"agent":  subagentType,
			"date":   today,
			"cutoff": "2025-01",
		}

		if skill != "" {
			orchDirective["skill"] = skill
			orchDirective["inject"] = "Invoke " + skill + " for domain expertise"
		}

		// P1 FIX: Add orchestration flow instructions
		orchDirective["CEO_FLOW"] = "DELEGATE->VERIFY->AEGIS"
		orchDirective["AFTER_TASK"] = "Verify result meets requirements"
		orchDirective["IF_FAIL"] = "Re-delegate with specific feedback"
		orchDirective["IF_PASS"] = "Run kavach orch aegis for final verification"

		// DAG Scheduler: If breakdown has >1 step, build parallel dispatch
		breakdown := extractBreakdown(prompt)
		if len(breakdown) > 1 {
			session := enforce.GetOrCreateSession()
			nodes := dag.Decompose(breakdown, []string{subagentType})
			state, err := dag.Schedule(session.SessionID, prompt, nodes)
			if err == nil {
				if err := dag.Save(state); err != nil {
					fmt.Fprintf(os.Stderr, "[CEO_DAG] Save error: %v\n", err)
				}
				directive := dag.BuildDirective(state)
				hook.ExitModifyTOONWithModule("CEO_DAG_DISPATCH", orchDirective, directive)
			}
		}

		hook.ExitModifyTOON("CEO_ORCHESTRATION", orchDirective)
	}

	// Non-engineer agents (ceo, research-director) - just validate
	hook.ExitSilent()
}

// detectSkillFromConfig detects skill using patterns from config file
// P1 FIX #1: NO HARDCODING - skills loaded dynamically with priority from config/skill-patterns.toon
func detectSkillFromConfig(prompt string) string {
	lower := strings.ToLower(prompt)

	// P1 FIX #1: Get skills sorted by priority from config (not hardcoded)
	skills := config.GetSkillsByPriority()

	for _, skill := range skills {
		if matchesKeywords(lower, skill.Keywords) {
			return skill.Name
		}
	}

	return ""
}

// extractBreakdown splits a multi-step prompt into individual steps.
// Looks for numbered lists (1. 2. 3.), bullet lists (- ), or (• ).
func extractBreakdown(prompt string) []string {
	lines := strings.Split(prompt, "\n")
	var steps []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Strip leading digits + separator ("1. ", "10) ", "2. ")
		i := 0
		for i < len(trimmed) && trimmed[i] >= '0' && trimmed[i] <= '9' {
			i++
		}
		if i > 0 && i < len(trimmed) && (trimmed[i] == '.' || trimmed[i] == ')') {
			steps = append(steps, strings.TrimSpace(trimmed[i+1:]))
		} else if strings.HasPrefix(trimmed, "- ") {
			steps = append(steps, strings.TrimSpace(trimmed[2:]))
		}
	}
	return steps
}

// matchesKeywords checks if text contains any keyword (word boundary)
func matchesKeywords(text string, keywords []string) bool {
	padded := " " + text + " "
	for _, kw := range keywords {
		patterns := []string{" " + kw + " ", " " + kw + ",", " " + kw + "."}
		for _, p := range patterns {
			if strings.Contains(padded, p) {
				return true
			}
		}
		if strings.Contains(kw, " ") && strings.Contains(text, kw) {
			return true
		}
	}
	return false
}
