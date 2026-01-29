// Package gates provides hook gates for Claude Code.
// intent_output.go: Output formatting for intent classification.
// DACE: Micro-modular split from intent.go
package gates

import (
	"strconv"
	"strings"
	"time"

	"github.com/claude/shared/pkg/enforce"
)

func formatIntentDirective(intent *IntentClassification, today string) string {
	var sb strings.Builder
	session := enforce.GetOrCreateSession()

	// Compact header (3 lines)
	sb.WriteString("[INTENT] type=" + intent.Type)
	if intent.Domain != "" {
		sb.WriteString(" domain=" + intent.Domain)
	}
	sb.WriteString(" confidence=" + intent.Confidence + " date=" + today + "\n")

	// Research block (only if needed and not done)
	if intent.ResearchReq && !session.ResearchDone {
		sb.WriteString("[BLOCK:RESEARCH] WebSearch BEFORE code (cutoff:2025-01 today:" + today + ")\n")
	}

	// Skill auto-invoke (only if skills detected)
	if len(intent.Skills) > 0 {
		sb.WriteString("[SKILL:AUTO_INVOKE] MANDATORY:")
		for _, skill := range intent.Skills {
			sb.WriteString(" Skill(skill:\"" + strings.TrimPrefix(skill, "/") + "\")")
		}
		sb.WriteString("\n")
	}

	// Agent routing (1 line)
	if intent.Type == "research" {
		sb.WriteString("[AGENT] MUST: Task(subagent_type='research-director')\n")
	} else {
		sb.WriteString("[AGENT] primary=" + intent.Agent + "\n")
	}

	// DACE enforcement (always, compact)
	sb.WriteString("[DACE] max:100lines depth:5-7levels split:concern no:duplicates no:monoliths\n")

	return sb.String()
}

func statusDirective() string {
	return `[BINARY_FIRST]
action: IMMEDIATE
command: kavach status && kavach memory bank
FORBIDDEN: Task(Explore), Read(docs/*.md)
reason: Memory Bank is SINGLE SOURCE OF TRUTH`
}

func postCompactRecovery(session *enforce.SessionState) string {
	today := time.Now().Format("2006-01-02")
	return "[RECOVERY] turn=" + strconv.Itoa(session.TurnCount) +
		" memory=kavach_memory_bank research=WebSearch_" + today +
		" binary=kavach_FIRST dace=100lines_5depth"
}

func periodicReinforcement(session *enforce.SessionState) string {
	today := time.Now().Format("2006-01-02")
	return "[REINFORCE] turn=" + strconv.Itoa(session.TurnCount) +
		" research=" + today + " dace=100lines_5depth fix=root_cause"
}
