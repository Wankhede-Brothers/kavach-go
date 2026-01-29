// Package gates provides hook gates for Claude Code.
// enforcer.go: Main enforcer gate - DACE silent-pass mode.
// DACE: Uses shared/pkg/patterns for dynamic patterns.
// DACE: Uses shared/pkg/agentic for research gate enforcement.
package gates

import (
	"strings"

	"github.com/claude/shared/pkg/agentic"
	"github.com/claude/shared/pkg/config"
	"github.com/claude/shared/pkg/enforce"
	"github.com/claude/shared/pkg/hook"
	"github.com/claude/shared/pkg/patterns"
	"github.com/spf13/cobra"
)

// Global research gate instance (DACE: lazy-loaded once)
var researchGate *agentic.ResearchGate

var enforcerHookMode bool

var enforcerCmd = &cobra.Command{
	Use:   "enforcer",
	Short: "Full pipeline enforcer gate - DACE silent-pass",
	Run:   runEnforcerGate,
}

func init() {
	enforcerCmd.Flags().BoolVar(&enforcerHookMode, "hook", false, "Hook mode")
}

func runEnforcerGate(cmd *cobra.Command, args []string) {
	if !enforcerHookMode {
		cmd.Help()
		return
	}

	// DACE: Lazy-load research gate once
	if researchGate == nil {
		researchGate = agentic.NewResearchGate()
	}

	input := hook.MustReadHookInput()
	session := enforce.GetOrCreateSession()

	switch input.ToolName {
	case "Task":
		handleTask(input, session)
	case "TaskCreate", "TaskUpdate", "TaskGet", "TaskList", "TaskOutput":
		// Route to task gate for persistent task system (Claude Code 2.1.19+)
		handleTaskManagement(input, session)
	case "WebSearch":
		session.MarkResearchDone()
		hook.ExitSilent()
	case "Write", "Edit":
		handleWrite(input, session)
	case "Bash":
		handleBash(input)
	case "Read":
		handleRead(input)
	default:
		hook.ExitSilent()
	}
}

func handleTask(input *hook.Input, session *enforce.SessionState) {
	agent := input.GetString("subagent_type")
	if agent == "" {
		hook.ExitBlockTOON("ENFORCER", "Task:no_subagent_type")
	}
	if !patterns.IsValidAgent(agent) {
		hook.ExitBlockTOON("ENFORCER", "Task:invalid_agent:"+agent)
	}

	// DACE: Check if task prompt needs research first
	prompt := input.GetString("prompt")
	if prompt != "" && researchGate != nil {
		// P1 FIX: Require research for ALL engineer agent delegations
		// Not just when frameworks are detected - research is the DEFAULT
		if isEngineerAgent(agent) && !session.ResearchDone {
			// Build helpful search query
			frameworks := agentic.ExtractFrameworkFromTask(prompt)
			var query string
			if len(frameworks) > 0 {
				query = frameworks[0] + " " + researchGate.Year() + " best practices"
			} else {
				query = researchGate.BuildSearchQuery("implementation patterns")
			}
			hook.ExitBlockTOON("RESEARCH_GATE",
				"engineer_delegation_requires_research:agent:"+agent+":suggest:"+query)
		}

		// Check for forbidden phrases in prompt
		violations := researchGate.CheckForForbiddenPhrases(prompt)
		if len(violations) > 0 {
			hook.ExitBlockTOON("RESEARCH_GATE",
				"forbidden_phrase:"+violations[0])
		}
	}

	session.MarkCEOInvoked()
	hook.ExitSilent()
}

// isEngineerAgent returns true for agents that implement code.
// P1 FIX: These agents ALWAYS require research before delegation.
func isEngineerAgent(agent string) bool {
	engineers := []string{
		"backend-engineer", "frontend-engineer", "database-engineer",
		"devops-engineer", "security-engineer",
	}
	for _, e := range engineers {
		if agent == e {
			return true
		}
	}
	return false
}

func handleWrite(input *hook.Input, session *enforce.SessionState) {
	filePath := input.GetString("file_path")

	// DACE: Use research gate for code file detection
	if patterns.IsCodeFile(filePath) && !session.ResearchDone {
		// Build helpful search query suggestion
		query := ""
		if researchGate != nil {
			query = researchGate.BuildSearchQuery("implementation patterns")
		}
		hook.ExitBlockTOON("TABULA_RASA",
			"WebSearch_required_before_code:suggest:"+query)
	}

	// Check content for forbidden phrases
	content := input.GetString("content")
	if content != "" && researchGate != nil {
		violations := researchGate.CheckForForbiddenPhrases(content)
		if len(violations) > 0 {
			hook.ExitBlockTOON("RESEARCH_GATE",
				"forbidden_phrase_in_code:"+violations[0])
		}
	}

	// CODE GUARD: Check for premature code removal (Edit tool)
	if input.ToolName == "Edit" {
		oldString := input.GetString("old_string")
		newString := input.GetString("new_string")

		// Block removal of TODO/stub functions
		if checkCodeRemoval(oldString, newString, filePath) {
			// Already handled in checkCodeRemoval
			return
		}
	}

	// Check write blocked paths from config
	if config.IsBlockedWritePath(filePath) {
		hook.ExitBlockTOON("ENFORCER", "Write:blocked_path:"+filePath)
	}

	hook.ExitSilent()
}

// checkCodeRemoval detects and blocks premature code removal.
// Returns true if blocked (already exited).
func checkCodeRemoval(old, new, filePath string) bool {
	// Skip if not code file
	if !patterns.IsCodeFile(filePath) {
		return false
	}

	// Check for function removal
	oldHasFunc := containsFunctionDef(old)
	newHasFunc := containsFunctionDef(new)

	if oldHasFunc && !newHasFunc && len(new) < len(old)/2 {
		// Significant code reduction with function removal
		if containsStubMarkers(old) {
			hook.ExitBlockTOON("CODE_GUARD",
				"BLOCK:removing_unimplemented_function:file:"+filePath+
					":REASON:Never remove TODO/stub functions. Implement them first or verify with user.")
			return true
		}
	}

	// Check for TODO removal without implementation
	if containsStubMarkers(old) && !containsStubMarkers(new) {
		if len(new) <= len(old) {
			hook.ExitBlockTOON("CODE_GUARD",
				"BLOCK:stub_removed_without_implementation:file:"+filePath+
					":REASON:TODO/FIXME removed but code not expanded. Implement before removing.")
			return true
		}
	}

	// Check for complete deletion
	if len(strings.TrimSpace(new)) == 0 && len(old) > 100 {
		hook.ExitBlockTOON("CODE_GUARD",
			"BLOCK:complete_deletion:file:"+filePath+
				":REASON:Cannot delete significant code block. Verify intent first.")
		return true
	}

	return false
}

// containsFunctionDef checks if content contains function definitions.
func containsFunctionDef(content string) bool {
	funcPatterns := []string{
		"func ", "fn ", "def ", "function ", "pub fn ", "async fn ",
		"const ", "class ", "impl ", "trait ",
	}
	for _, p := range funcPatterns {
		if strings.Contains(content, p) {
			return true
		}
	}
	return false
}

// containsStubMarkers checks if content has TODO/stub markers.
func containsStubMarkers(content string) bool {
	markers := []string{
		"TODO", "FIXME", "XXX", "HACK",
		"not implemented", "placeholder", "stub",
		"unimplemented!", "todo!",
		"NotImplementedError", "pass\n",
	}
	contentLower := strings.ToLower(content)
	for _, m := range markers {
		if strings.Contains(contentLower, strings.ToLower(m)) {
			return true
		}
	}
	return false
}

func handleBash(input *hook.Input) {
	cmd := input.GetString("command")
	if cmd == "" {
		hook.ExitBlockTOON("ENFORCER", "Bash:empty_command")
	}
	// Check config.json blocked commands first
	if config.IsBlockedCommand(cmd) {
		hook.ExitBlockTOON("ENFORCER", "Bash:blocked_command")
	}
	// Fallback to patterns.toon
	if patterns.IsBlocked(cmd) {
		hook.ExitBlockTOON("ENFORCER", "Bash:blocked_command")
	}
	hook.ExitSilent()
}

func handleRead(input *hook.Input) {
	path := input.GetString("file_path")
	// Check config.json blocked paths first
	if config.IsBlockedPath(path) {
		hook.ExitBlockTOON("ENFORCER", "Read:blocked_path")
	}
	if config.IsBlockedExtension(path) {
		hook.ExitBlockTOON("ENFORCER", "Read:blocked_extension")
	}
	// Fallback to patterns.toon
	if patterns.IsSensitive(path) {
		hook.ExitBlockTOON("ENFORCER", "Read:sensitive_file")
	}
	hook.ExitSilent()
}

// handleTaskManagement routes Claude Code 2.1.19+ task management tools.
// These tools interact with the persistent task system in ~/.claude/tasks/
func handleTaskManagement(input *hook.Input, session *enforce.SessionState) {
	switch input.ToolName {
	case "TaskCreate":
		subject := input.GetString("subject")
		if subject == "" {
			hook.ExitBlockTOON("TASK_GATE", "TaskCreate:missing_subject")
		}
		description := input.GetString("description")
		if description == "" {
			hook.ExitBlockTOON("TASK_GATE", "TaskCreate:missing_description")
		}
		// Track task creation
		session.TasksCreated++
		session.Save()
	case "TaskUpdate":
		taskID := input.GetString("taskId")
		if taskID == "" {
			hook.ExitBlockTOON("TASK_GATE", "TaskUpdate:missing_taskId")
		}
		status := input.GetString("status")
		if status == "completed" {
			session.TasksCompleted++
			session.Save()
		}
	case "TaskGet":
		taskID := input.GetString("taskId")
		if taskID == "" {
			hook.ExitBlockTOON("TASK_GATE", "TaskGet:missing_taskId")
		}
	case "TaskOutput":
		taskID := input.GetString("task_id")
		if taskID == "" {
			hook.ExitBlockTOON("TASK_GATE", "TaskOutput:missing_task_id")
		}
	}
	hook.ExitSilent()
}
