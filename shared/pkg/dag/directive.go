// Package dag provides a parallel DAG scheduler for Kavach orchestration.
// directive.go: Builds TOON directives that instruct Claude to create tasks in parallel.
package dag

import "fmt"

// BuildParallelDispatch generates a TOON directive for one parallel level.
func BuildParallelDispatch(dagID string, level ParallelLevel, maxLevel int) string {
	out := fmt.Sprintf("[DAG_SCHEDULER]\ndag_id: %s\nstatus: active\nlevel: %d/%d\n\n", dagID, level.Level, maxLevel)
	out += fmt.Sprintf("[PARALLEL_DISPATCH]\ninstruction: Create ALL tasks below in a SINGLE message using parallel TaskCreate calls\ncount: %d\n\n", len(level.Nodes))

	for _, n := range level.Nodes {
		out += fmt.Sprintf("[TASK:%s]\nsubject: %s\ndescription: %s\nagent: %s\n", n.ID, n.Subject, n.Description, n.Agent)
		if n.Skill != "" {
			out += fmt.Sprintf("skill: %s\n", n.Skill)
		}
		out += fmt.Sprintf("metadata: {\"dag_node_id\": \"%s\"}\n\n", n.ID)
	}

	if level.Level < maxLevel {
		out += "[AFTER_LEVEL]\nWhen all tasks above complete, next level will be dispatched automatically.\n"
	}
	return out
}

// BuildCompletionDirective generates the "all done, run Aegis" directive.
func BuildCompletionDirective(dagID string) string {
	return fmt.Sprintf("[DAG_COMPLETE]\ndag_id: %s\nstatus: complete\naction: Run kavach orch aegis for final verification\n", dagID)
}
