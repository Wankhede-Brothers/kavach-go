// subagent.go: SubagentStart/SubagentStop gates.
// Tracks subagent DAG, validates agent types, enforces research-before-implement.
package gates

import (
	"github.com/claude/shared/pkg/enforce"
	"github.com/claude/shared/pkg/hook"
	"github.com/claude/shared/pkg/patterns"
	"github.com/spf13/cobra"
)

var subagentHookMode bool

var subagentCmd = &cobra.Command{
	Use:   "subagent",
	Short: "SubagentStart/SubagentStop gate",
	Long: `[SUBAGENT_GATE]
desc: Track subagent spawning and verify output quality
hooks: SubagentStart, SubagentStop

[USAGE]
kavach gates subagent --hook`,
	Run: runSubagentGate,
}

func init() {
	subagentCmd.Flags().BoolVar(&subagentHookMode, "hook", false, "Hook mode")
}

func runSubagentGate(cmd *cobra.Command, args []string) {
	if !subagentHookMode {
		cmd.Help()
		return
	}

	input := hook.MustReadHookInput()
	session := enforce.GetOrCreateSession()

	switch input.HookEventName {
	case "SubagentStart":
		handleSubagentStart(input, session)
	case "SubagentStop":
		handleSubagentStop(input, session)
	default:
		hook.ExitSilent()
	}
}

func handleSubagentStart(input *hook.Input, session *enforce.SessionState) {
	agentType := input.AgentType
	agentID := input.AgentID

	// Validate known agent types
	if agentType != "" && !isBuiltinAgent(agentType) && !patterns.IsValidAgent(agentType) {
		hook.ExitBlockTOON("SUBAGENT_GATE", "unknown_agent_type:"+agentType)
	}

	// Enforce research for engineer-type subagents
	if isEngineerAgent(agentType) && !session.ResearchDone {
		hook.ExitBlockTOON("SUBAGENT_GATE",
			"engineer_subagent_requires_research:agent:"+agentType+":id:"+agentID)
	}

	hook.ExitSubagentStart("[SUBAGENT:START] type:" + agentType + " id:" + agentID)
}

func handleSubagentStop(input *hook.Input, session *enforce.SessionState) {
	agentType := input.AgentType
	agentID := input.AgentID

	// Log subagent completion for DAG tracking
	hook.ExitSubagentStop("[SUBAGENT:STOP] type:" + agentType + " id:" + agentID)
}

// isBuiltinAgent checks for Claude Code built-in agent types.
func isBuiltinAgent(agent string) bool {
	builtins := []string{
		"Bash", "Explore", "Plan", "general-purpose",
		"code-simplifier", "statusline-setup",
	}
	for _, b := range builtins {
		if agent == b {
			return true
		}
	}
	return false
}
