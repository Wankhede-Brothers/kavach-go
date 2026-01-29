// Package commands provides the root command and subcommand registration.
package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/claude/cmd/kavach/internal/commands/agents"
	"github.com/claude/cmd/kavach/internal/commands/gates"
	"github.com/claude/cmd/kavach/internal/commands/lint"
	"github.com/claude/cmd/kavach/internal/commands/memory"
	"github.com/claude/cmd/kavach/internal/commands/orch"
	"github.com/claude/cmd/kavach/internal/commands/quality"
	"github.com/claude/cmd/kavach/internal/commands/session"
	"github.com/claude/cmd/kavach/internal/commands/skills"
	"github.com/claude/shared/pkg/enforce"
	"github.com/spf13/cobra"
)

// symlinkMap maps legacy binary names to subcommand arguments.
var symlinkMap = map[string][]string{
	"ceo-gate":         {"gates", "ceo", "--hook"},
	"ast-gate":         {"gates", "ast", "--hook"},
	"bash-sanitizer":   {"gates", "bash", "--hook"},
	"read-blocker":     {"gates", "read", "--hook"},
	"intent-gate":      {"gates", "intent", "--hook"},
	"skill-gate":       {"gates", "skill", "--hook"},
	"lint-gate":        {"gates", "lint", "--hook"},
	"research-gate":    {"gates", "research", "--hook"},
	"content-gate":     {"gates", "content", "--hook"},
	"quality-gate":     {"gates", "quality", "--hook"},
	"enforcer":         {"gates", "enforcer", "--hook"},
	"memory-bank":      {"memory", "bank"},
	"memory-write":     {"memory", "write"},
	"memory-rpc":       {"memory", "rpc"},
	"stm-updater":      {"memory", "stm"},
	"rpc-inject":       {"memory", "inject"},
	"spec-inject":      {"memory", "spec"},
	"session-init":     {"session", "init"},
	"session-validate": {"session", "validate"},
	"session-end":      {"session", "end"},
	"session-resume":   {"session", "resume"},
	"pre-compact":      {"session", "compact"},
	"aegis-auto":       {"orch", "aegis", "--hook"},
	"autonomous-orch":  {"orch", "auto"},
	"post-verify":      {"orch", "post"},
	"context-monitor":  {"tracking", "context"},
	"hot-context":      {"tracking", "hot"},
	"mistake-tracker":  {"tracking", "mistake"},
	"threshold-gate":   {"tracking", "threshold"},
}

var rootCmd = &cobra.Command{
	Use:   "kavach",
	Short: "Brahmastra Stack - Universal AI CLI Enforcement",
	Long: `[KAVACH]
desc: Universal enforcement binary for AI coding assistants
stack: Brahmastra Stack
protocol: SP/1.0 (Sutra Protocol)
compatible: Claude Code, OpenCode, any SP/1.0 CLI

[COMMANDS]
gates:    Hook enforcement (PreToolUse, PostToolUse)
memory:   Query/write memory bank, context injection
session:  Lifecycle management (init, validate, end)
tracking: Context monitoring, hot paths, mistakes
orch:     Multi-agent orchestration, verification
status:   System health check
agents:   List available agents with models
skills:   List available skills

[HOOKS_MAPPING]
SessionStart:        session init
SessionEnd:          session end-hook
UserPromptSubmit:    gates intent --hook
PreToolUse:          gates enforcer --hook
PreToolUse:Task:     gates ceo --hook
PreToolUse:Bash:     gates bash --hook
PreToolUse:Read:     gates read --hook
PostToolUse:         memory sync --hook
PostToolUseFailure:  gates failure --hook
SubagentStart:       gates subagent --hook
SubagentStop:        gates subagent --hook
PermissionRequest:   gates read --hook
Stop:                session end
PreCompact:          session compact

[EXAMPLES]
kavach status                              # System health
kavach gates enforcer --hook < input.json  # Full pipeline
kavach memory bank                         # Query memory bank
kavach session init                        # Start session`,
}

// Execute runs the root command with symlink dispatch.
func Execute(version string) error {
	rootCmd.Version = version
	handleSymlinkDispatch()
	registerSubcommands()
	return rootCmd.Execute()
}

// handleSymlinkDispatch rewrites args if invoked via symlink.
func handleSymlinkDispatch() {
	baseName := filepath.Base(os.Args[0])
	if args, ok := symlinkMap[baseName]; ok {
		os.Args = append([]string{os.Args[0]}, append(args, os.Args[1:]...)...)
	}
}

// registerSubcommands adds all command groups to root.
func registerSubcommands() {
	// Register subcommands from packages
	gates.Register(gatesCmd)
	memory.Register(memoryCmd)
	session.Register(sessionCmd)
	orch.Register(orchCmd)

	rootCmd.AddCommand(gatesCmd)
	rootCmd.AddCommand(memoryCmd)
	rootCmd.AddCommand(sessionCmd)
	rootCmd.AddCommand(trackingCmd)
	rootCmd.AddCommand(orchCmd)
	rootCmd.AddCommand(statusCmd)

	// DACE: Dynamic agents and skills (micro-modular)
	rootCmd.AddCommand(agents.Cmd())
	rootCmd.AddCommand(skills.Cmd())

	// P3 FIX: Standalone lint and quality commands
	rootCmd.AddCommand(lint.LintCmd)
	rootCmd.AddCommand(quality.QualityCmd)

	// Top-level aliases for backward compatibility
	rootCmd.AddCommand(intentCmd)
}

// intentCmd is a top-level alias for "gates intent" for backward compatibility.
var intentCmd = &cobra.Command{
	Use:    "intent",
	Short:  "Intent classification (alias for 'gates intent')",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		// Forward to gates intent
		newArgs := []string{os.Args[0], "gates", "intent"}
		if hookMode, _ := cmd.Flags().GetBool("hook"); hookMode {
			newArgs = append(newArgs, "--hook")
		}
		os.Args = newArgs
		rootCmd.Execute()
	},
}

func init() {
	intentCmd.Flags().Bool("hook", false, "Run in hook mode")
}

// Placeholder commands - will be populated by register.go files.
var gatesCmd = &cobra.Command{
	Use:   "gates",
	Short: "Enforcement gates (ceo, ast, bash, read, etc.)",
	Long: `[GATES]
desc: Hook-based enforcement for PreToolUse/PostToolUse
input: JSON via stdin (HookInput format)
output: JSON decision (approve/block)

[AVAILABLE_GATES]
enforcer:  Full pipeline (intent→ceo→quality→aegis) - USE THIS FIRST
ceo:       Task orchestration, subagent validation
ast:       AST validation for code changes
bash:      Command sanitization, dangerous command blocking
read:      File access control, sensitive file blocking
intent:    Intent classification from user prompts
skill:     Skill invocation validation
lint:      Code style and lint checking
research:  TABULA_RASA enforcement (WebSearch before code)
content:   Content validation for writes
quality:   Code quality checks (AST + lint chain)

[WHEN_TO_USE]
PreToolUse:      gates enforcer --hook (recommended)
PreToolUse:Task: gates ceo --hook
PreToolUse:Bash: gates bash --hook
PreToolUse:Edit: gates ast --hook`,
}

var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Memory bank operations",
	Long: `[MEMORY]
desc: Memory bank operations at ~/.local/shared/shared-ai/memory/
categories: decisions, graph, kanban, patterns, proposals, research, roadmaps, STM

[AVAILABLE_COMMANDS]
bank:    Query memory bank, list categories, health check
write:   Write entries to memory bank
rpc:     JSON-RPC style memory operations
stm:     Short-term memory (scratchpad) updates
inject:  RPC context injection
spec:    Spec file injection (TOON/MD)

[WHEN_TO_USE]
SessionStart:    memory bank (load context)
PostToolUse:     memory write (persist learnings)
ContextQuery:    memory bank --status (health check)`,
}

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Session management",
	Long: `[SESSION]
desc: Session lifecycle management with date injection
enforces: TABULA_RASA, DATE_INJECTION, NO_AMNESIA

[AVAILABLE_COMMANDS]
init:     Initialize session (injects today's date, loads context)
validate: Validate session state
end:      End session, persist state
compact:  Pre-compact save (before context compaction)

[WHEN_TO_USE]
SessionStart:     session init
UserPromptSubmit: session init (refresh context)
Stop:             session end
PreCompact:       session compact`,
}

var trackingCmd = &cobra.Command{
	Use:   "tracking",
	Short: "Context tracking",
	Long: `[TRACKING]
desc: Context monitoring and hot path tracking
purpose: Optimize context usage, track patterns

[AVAILABLE_COMMANDS]
context:   Monitor context usage and patterns
hot:       Track hot (frequently accessed) files/functions
mistake:   Track and learn from mistakes
threshold: Context threshold gate (warn at 95%, block at 98%)

[WHEN_TO_USE]
PostToolUse: tracking context, tracking hot
OnError:     tracking mistake`,
}

var orchCmd = &cobra.Command{
	Use:   "orch",
	Short: "Orchestration commands",
	Long: `[ORCH]
desc: Multi-agent orchestration and verification
purpose: Coordinate agent hierarchy, verify completions

[AVAILABLE_COMMANDS]
aegis: Aegis guardian verification (Level 2)
auto:  Autonomous orchestrator
post:  Post-completion verification

[WHEN_TO_USE]
PostToolUse:Write: orch aegis --hook (verify code quality)
TaskComplete:      orch post (final verification)`,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show system status (SP/1.0 TOON format)",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := enforce.NewContext()
		sess := enforce.GetOrCreateSession()

		fmt.Println("[STATUS]")
		fmt.Println("today: " + ctx.Today)
		fmt.Println("cutoff: " + sess.TrainingCutoff)
		fmt.Println("session: " + sess.ID)
		fmt.Println("project: " + sess.Project)
		fmt.Println()
		fmt.Println("[ENFORCE]")
		fmt.Println("TABULA_RASA: active")
		fmt.Println("DATE_INJECTION: " + ctx.Today)
		fmt.Println("NO_AMNESIA: ~/.local/shared/shared-ai/memory/")
		fmt.Println("NO_ASSUMPTION: verify_before_act")
		fmt.Println("DACE: lazy_load,skill_first")
		fmt.Println()
		fmt.Println("[STATE]")
		fmt.Printf("research_done: %s\n", boolState(sess.ResearchDone))
		fmt.Printf("memory: %s\n", boolState(sess.MemoryQueried))
		fmt.Printf("ceo: %s\n", boolState(sess.CEOInvoked))
	},
}

func boolState(b bool) string {
	if b {
		return "done"
	}
	return "pending"
}

// agentsCmd and skillsCmd are now provided by the agents and skills packages
// using DACE micro-modular architecture with dynamic context loading
