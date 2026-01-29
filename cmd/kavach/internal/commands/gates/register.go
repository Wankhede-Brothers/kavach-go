// Package gates provides gate enforcement subcommands.
package gates

import "github.com/spf13/cobra"

// Register adds all gate commands to the parent gates command.
func Register(gatesCmd *cobra.Command) {
	gatesCmd.AddCommand(ceoCmd)
	gatesCmd.AddCommand(astCmd)
	gatesCmd.AddCommand(bashCmd)
	gatesCmd.AddCommand(readCmd)
	gatesCmd.AddCommand(intentCmd)
	gatesCmd.AddCommand(skillCmd)
	gatesCmd.AddCommand(lintCmd)
	gatesCmd.AddCommand(researchCmd)
	gatesCmd.AddCommand(contentCmd)
	gatesCmd.AddCommand(qualityCmd)
	gatesCmd.AddCommand(enforcerCmd)
	gatesCmd.AddCommand(contextCmd)   // P3 FIX #16: Hot-context tracking
	gatesCmd.AddCommand(dagCmd)       // Phase 2: DAG cycle detection
	gatesCmd.AddCommand(taskCmd)      // Claude Code 2.1.19+: Persistent task system
	gatesCmd.AddCommand(codeGuardCmd) // 2026: Prevent premature code removal
	gatesCmd.AddCommand(chainCmd)     // 2026: Multi-agent verification chain
	gatesCmd.AddCommand(subagentCmd)  // SubagentStart/SubagentStop tracking
	gatesCmd.AddCommand(failureCmd)   // PostToolUseFailure handling
}
