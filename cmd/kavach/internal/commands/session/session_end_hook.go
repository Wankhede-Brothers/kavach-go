// session_end_hook.go: SessionEnd lifecycle hook (distinct from Stop).
// SessionEnd fires when session terminates. Cannot block termination.
// Use for: final memory sync, cleanup, analytics.
package session

import (
	"fmt"

	"github.com/claude/shared/pkg/enforce"
	"github.com/claude/shared/pkg/hook"
	"github.com/spf13/cobra"
)

var sessionEndHookCmd = &cobra.Command{
	Use:   "end-hook",
	Short: "SessionEnd lifecycle hook (memory sync + cleanup)",
	Long: `[SESSION_END_HOOK]
desc: Runs on SessionEnd event for final memory persistence
hook: SessionEnd
note: Cannot block session termination

[USAGE]
kavach session end-hook`,
	Run: runSessionEndHook,
}

func runSessionEndHook(cmd *cobra.Command, args []string) {
	input := hook.MustReadHookInput()
	ctx := enforce.NewContext()
	session := enforce.GetOrCreateSession()

	reason := input.Reason
	if reason == "" {
		reason = "unknown"
	}

	// Persist final session state
	session.Save()

	// Output cleanup summary
	fmt.Println("[SESSION_END]")
	fmt.Printf("date: %s\nsession: %s\nproject: %s\nreason: %s\n\n",
		ctx.Today, session.ID, session.Project, reason)

	fmt.Println("[FINAL_STATE]")
	fmt.Printf("research_done: %s\nmemory: %s\nceo: %s\naegis: %s\n",
		boolStr(session.ResearchDone), boolStr(session.MemoryQueried),
		boolStr(session.CEOInvoked), boolStr(session.AegisVerified))
	fmt.Printf("tasks_created: %d\ntasks_completed: %d\n",
		session.TasksCreated, session.TasksCompleted)
}
