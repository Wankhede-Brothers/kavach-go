// Package orch provides orchestration subcommands.
// dag.go: CLI for DAG scheduler inspection and management.
package orch

import (
	"fmt"
	"os"
	"strings"

	"github.com/claude/shared/pkg/dag"
	"github.com/claude/shared/pkg/enforce"
	"github.com/spf13/cobra"
)

var dagStatusFlag bool
var dagResetFlag bool
var dagVisualizeFlag bool

var dagOrcCmd = &cobra.Command{
	Use:   "dag",
	Short: "DAG scheduler management",
	Long: `[DAG_SCHEDULER]
desc: Inspect and manage parallel task DAG state
usage:
  kavach orch dag --status     Show current DAG state
  kavach orch dag --reset      Clear DAG for session
  kavach orch dag --visualize  ASCII visualization`,
	Run: runDAGOrch,
}

func init() {
	dagOrcCmd.Flags().BoolVar(&dagStatusFlag, "status", false, "Show current DAG state")
	dagOrcCmd.Flags().BoolVar(&dagResetFlag, "reset", false, "Clear DAG for session")
	dagOrcCmd.Flags().BoolVar(&dagVisualizeFlag, "visualize", false, "ASCII DAG visualization")
}

func runDAGOrch(cmd *cobra.Command, args []string) {
	session := enforce.GetOrCreateSession()
	sid := session.SessionID

	if dagResetFlag {
		if err := dag.Delete(sid); err != nil {
			fmt.Fprintf(os.Stderr, "[DAG] No active DAG to reset: %v\n", err)
			return
		}
		fmt.Println("[DAG] Reset complete")
		return
	}

	state, err := dag.Load(sid)
	if err != nil {
		fmt.Println("[DAG] No active DAG for this session")
		return
	}

	if dagVisualizeFlag {
		visualize(state)
		return
	}

	// Default: --status
	fmt.Printf("[DAG_STATE]\nid: %s\nsession: %s\nstatus: %s\nlevels: %d\nnodes: %d\n\n",
		state.ID, state.SessionID, state.Status, state.MaxLevel+1, len(state.Nodes))
	for _, n := range state.Nodes {
		deps := "none"
		if len(n.DependsOn) > 0 {
			deps = strings.Join(n.DependsOn, ",")
		}
		fmt.Printf("  [%s] %s (L%d) status=%s deps=%s\n", n.ID, n.Subject, n.Level, n.Status, deps)
	}
}

func visualize(state *dag.DAGState) {
	levels := make(map[int][]*dag.Node)
	for _, n := range state.Nodes {
		levels[n.Level] = append(levels[n.Level], n)
	}
	for l := 0; l <= state.MaxLevel; l++ {
		fmt.Printf("=== Level %d ===\n", l)
		for _, n := range levels[l] {
			icon := " "
			switch n.Status {
			case dag.StatusDone:
				icon = "✓"
			case dag.StatusFailed:
				icon = "✗"
			case dag.StatusSkipped:
				icon = "⊘"
			case dag.StatusRunning:
				icon = "►"
			case dag.StatusDispatched:
				icon = "→"
			case dag.StatusReady:
				icon = "○"
			}
			fmt.Printf("  [%s] %s %s\n", icon, n.ID, n.Subject)
		}
	}
}
