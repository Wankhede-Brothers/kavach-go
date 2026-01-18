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

	sb.WriteString("[NLU:INTENT_CLASSIFIED]\n")
	sb.WriteString("type: " + intent.Type + "\n")
	if intent.Domain != "" {
		sb.WriteString("domain: " + intent.Domain + "\n")
	}
	sb.WriteString("confidence: " + intent.Confidence + "\n")
	sb.WriteString("date: " + today + "\n\n")

	// CRITICAL: HARD BLOCK for implementation intents without research
	if intent.ResearchReq && isImplementationIntent(intent.Type) && !session.ResearchDone {
		sb.WriteString("[HARD_BLOCK:TABULA_RASA]\n")
		sb.WriteString("status: TRAINING_WEIGHTS_STALE\n")
		sb.WriteString("cutoff: 2025-01\n")
		sb.WriteString("today: " + today + "\n")
		sb.WriteString("BLOCKING: You MUST WebSearch BEFORE giving implementation advice\n")
		sb.WriteString("FORBIDDEN: Suggesting versions, configs, env vars from memory\n")
		sb.WriteString("FIRST_ACTION: WebSearch \"[topic] " + today + " documentation\"\n\n")
	}

	if len(intent.Skills) > 0 {
		sb.WriteString("[SKILL:INJECT]\n")
		for _, skill := range intent.Skills {
			sb.WriteString("invoke: " + skill + "\n")
		}
		sb.WriteString("\n")
	}

	if intent.Type == "research" {
		sb.WriteString("[AGENT:MANDATORY]\n")
		sb.WriteString("MUST_INVOKE: Task tool with subagent_type='research-director'\n")
		sb.WriteString("REASON: Research intent requires research-director agent\n")
		sb.WriteString("FORBIDDEN: Direct implementation without research delegation\n\n")
	} else {
		sb.WriteString("[AGENT:RECOMMEND]\n")
		sb.WriteString("primary: " + intent.Agent + "\n")
		if len(intent.SubAgents) > 0 {
			sb.WriteString("sub_agents: " + strings.Join(intent.SubAgents, ", ") + "\n")
		}
		sb.WriteString("\n")
	}

	if intent.ResearchReq {
		sb.WriteString("[TABULA_RASA:ENFORCE]\n")
		sb.WriteString("cutoff: 2025-01\n")
		sb.WriteString("today: " + today + "\n")
		sb.WriteString("research_done: " + strconv.FormatBool(session.ResearchDone) + "\n")
		sb.WriteString("action: WebSearch BEFORE implementation\n")
		sb.WriteString("FORBIDDEN: Assuming from stale training weights\n\n")
	}

	sb.WriteString("[BEFORE:MEMORY_BANK]\n")
	sb.WriteString("action: kavach memory bank (load context FIRST)\n\n")

	sb.WriteString("[WORKFLOW]\n")
	sb.WriteString("1. [MEMORY] kavach memory bank\n")
	sb.WriteString("2. [RESEARCH] WebSearch with date: " + today + "\n")
	sb.WriteString("3. [DELEGATE] CEO -> Engineer with skill\n")
	sb.WriteString("4. [VERIFY] Aegis before DONE\n")
	sb.WriteString("5. [SYNC] kavach memory sync\n\n")

	sb.WriteString("[CRITICAL:RULES]\n")
	sb.WriteString("NO_AMNESIA: Memory Bank at ~/.local/shared/shared-ai/memory/\n")
	sb.WriteString("TABULA_RASA: WebSearch BEFORE code\n")
	sb.WriteString("DATE: " + today + "\n")

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
	return `[CONTEXT:RECOVERY]
trigger: post_compact
turn: ` + strconv.Itoa(session.TurnCount) + `

[NO_AMNESIA]
memory_bank: ~/.local/shared/shared-ai/memory/
query: kavach memory bank

[TABULA_RASA]
cutoff: 2025-01
today: ` + today + `
RULE: WebSearch BEFORE code

[BINARY_FIRST]
binary: kavach
RULE: kavach commands BEFORE Read/Explore`
}

func periodicReinforcement(session *enforce.SessionState) string {
	today := time.Now().Format("2006-01-02")
	return `[CONTEXT:REINFORCE]
turn: ` + strconv.Itoa(session.TurnCount) + `

CRITICAL:BINARY_FIRST - kavach BEFORE Read/Explore
CRITICAL:TABULA_RASA - WebSearch BEFORE code (cutoff: 2025-01, today: ` + today + `)
CRITICAL:NO_AMNESIA - Memory Bank EXISTS`
}
