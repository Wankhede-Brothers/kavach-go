// Package session provides session management subcommands.
package session

import "github.com/spf13/cobra"

// Register adds all session commands to the parent session command.
func Register(sessionCmd *cobra.Command) {
	sessionCmd.AddCommand(initCmd)
	sessionCmd.AddCommand(validateCmd)
	sessionCmd.AddCommand(endCmd)
	sessionCmd.AddCommand(compactCmd)
	sessionCmd.AddCommand(resumeCmd)
	sessionCmd.AddCommand(landCmd) // Beads-inspired "land the plane" protocol
}
