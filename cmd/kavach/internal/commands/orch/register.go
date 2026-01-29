// Package orch provides orchestration subcommands.
package orch

import "github.com/spf13/cobra"

// Register adds all orch commands to the parent orch command.
func Register(orchCmd *cobra.Command) {
	orchCmd.AddCommand(aegisCmd)
	orchCmd.AddCommand(verifyCmd)
	orchCmd.AddCommand(taskHealthCmd) // Claude Code 2.1.19+: Task health monitoring
	orchCmd.AddCommand(dagOrcCmd)     // Parallel DAG scheduler
}
