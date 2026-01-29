package session

import (
	"fmt"
	"os"

	"github.com/claude/shared/events"
	"github.com/claude/shared/pkg/enforce"
	"github.com/claude/shared/pkg/toon"
	"github.com/claude/shared/pkg/util"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize session with date injection",
	Long: `[INIT]
desc: Initialize session with enforcement context
hook: SessionStart, UserPromptSubmit
priority: CRITICAL - must run at session start

[DACE:COMPLIANT]
- Pointers, not data dumps
- ~100 tokens normal, ~150 tokens post-compact
- Lazy load: query commands provided

[USAGE]
kavach session init`,
	Run: runInitCmd,
}

func runInitCmd(cmd *cobra.Command, args []string) {
	ctx := enforce.NewContext()
	session := enforce.GetOrCreateSession()
	bank := toon.NewMemoryBank()

	// P0 FIX: Ensure Memory Bank directory structure exists (including STM)
	// P2 FIX: Don't silently ignore error - it's not fatal but should be noted
	if err := util.EnsureMemoryBankDirs(session.Project); err != nil {
		// Non-fatal: Memory Bank dirs may already exist or have permission issues
		// Continue with session init regardless
	}

	// DACE: Publish EventSessionStart for subscribers (telemetry, hooks)
	eventBus := events.GetEventBus()
	eventBus.Publish(events.EventSessionStart, "kavach", map[string]interface{}{
		"session_id": session.ID,
		"project":    session.Project,
		"date":       ctx.Today,
		"type":       getSessionType(session),
	})

	// Detect post-compact mode and handle specially
	if session.IsPostCompact() {
		runPostCompactInit(ctx, session, bank)
		return
	}

	// DACE: Ultra-minimal output (~100 tokens)
	fmt.Println("[META]")
	fmt.Printf("protocol: SP/1.0\ndate: %s\nsession: %s\ntype: %s\n\n",
		ctx.Today, session.ID, getSessionType(session))

	fmt.Println("[TABULA_RASA]")
	fmt.Printf("cutoff: %s\ntoday: %s\nrule: WebSearch_BEFORE_code\n",
		session.TrainingCutoff, ctx.Today)
	fmt.Println("blocked: I_think,I_believe,I_recall,Based_on_my_knowledge")
	fmt.Println()

	fmt.Println("[NO_AMNESIA]")
	fmt.Println("memory: ~/.local/shared/shared-ai/memory/")
	fmt.Println("forbidden: I_have_no_memory,I_dont_have_access")
	fmt.Println()

	fmt.Println("[SESSION]")
	fmt.Printf("id: %s\nproject: %s\nresearch_mode: always\nresearch_done: %s\nmemory: %s\n\n",
		session.ID, session.Project,
		boolStr(session.ResearchDone), boolStr(session.MemoryQueried))

	// DACE: Pointer to command, NOT data dump
	total := countMemoryDocs(bank)
	fmt.Printf("[MEMORY] total: %d | query: kavach memory bank\n\n", total)

	fmt.Println("[DACE] mode: lazy_load,skill_first,on_demand")

	session.MarkMemoryQueried()

	// Write env vars to CLAUDE_ENV_FILE if set
	writeClaudeEnvFile(session)
}

// runPostCompactInit handles session init after a compact event.
// DACE: Ultra-lean ~50 tokens. Summary already has context.
func runPostCompactInit(ctx *enforce.Context, session *enforce.SessionState, bank *toon.MemoryBank) {
	session.ClearPostCompact()

	// DACE: Ultra-minimal post-compact (~50 tokens)
	// The compact summary already contains TABULA_RASA, NO_AMNESIA, etc.
	// Only inject: date, session ID, task pointer, memory count
	fmt.Println("[META]")
	fmt.Printf("protocol: SP/1.0\ndate: %s\nsession: %s\ntype: post_compact_recovery\ncompact_count: %d\n\n",
		ctx.Today, session.ID, session.CompactCount)

	fmt.Println("[SESSION]")
	fmt.Printf("id: %s\nproject: %s\nresearch_mode: always\nresearch_done: %s\nmemory: %s\n\n",
		session.ID, session.Project,
		boolStr(session.ResearchDone), boolStr(session.MemoryQueried))

	// DACE: Task pointer only if exists
	if session.CurrentTask != "" {
		fmt.Printf("[TASK:RESTORED] %s | status: %s\n\n", session.CurrentTask, session.TaskStatus)
	} else {
		scratchpad := loadScratchpad(session.Project)
		if scratchpad != nil {
			if taskBlock := scratchpad.Get("TASK"); taskBlock != nil {
				if intent := taskBlock.Get("intent"); intent != "" && intent != "null" {
					fmt.Printf("[TASK:RESTORED] %s | status: %s\n\n", intent, taskBlock.Get("status"))
				}
			}
		}
	}

	// DACE: Pointer only
	total := countMemoryDocs(bank)
	fmt.Printf("[MEMORY] total: %d | query: kavach memory bank\n\n", total)

	fmt.Println("[DACE] mode: lazy_load,skill_first,on_demand | CONTEXT_RESTORED")

	session.MarkMemoryQueried()
}

// countMemoryDocs returns total document count without dumping details.
func countMemoryDocs(bank *toon.MemoryBank) int {
	stats := bank.GetCategoryStats()
	total := 0
	for _, count := range stats {
		total += count
	}
	return total
}

// loadScratchpad loads the project scratchpad.toon file.
func loadScratchpad(project string) *toon.Document {
	if project == "" {
		return nil
	}

	scratchpadPath := util.ScratchpadPath(project)
	if !util.FileExists(scratchpadPath) {
		return nil
	}

	f, err := os.Open(scratchpadPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	parser := toon.NewParser()
	doc, _ := parser.Parse(f)
	return doc
}

func getSessionType(s *enforce.SessionState) string {
	if s.PostCompact {
		return "post_compact_recovery"
	}
	if s.ResearchDone || s.MemoryQueried {
		return "resumed_session"
	}
	return "fresh_session"
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
