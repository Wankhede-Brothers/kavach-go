// Package chain provides multi-agent verification chain for kavach.
// runner.go: Orchestrates the Intent → CEO → Aegis → Research pipeline.
package chain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Runner orchestrates the verification chain.
type Runner struct {
	state     *ChainState
	cacheDir  string
	debugMode bool
}

// NewRunner creates a new chain runner.
func NewRunner(sessionID string) *Runner {
	home, _ := os.UserHomeDir()
	return &Runner{
		state:     NewChainState(sessionID),
		cacheDir:  filepath.Join(home, ".claude", "chain"),
		debugMode: os.Getenv("KAVACH_DEBUG") == "1",
	}
}

// RunFull executes the complete verification chain.
// Returns the final state after all gates have run.
func (r *Runner) RunFull(prompt, toolName string, toolInput map[string]interface{}, researchDone bool) *ChainState {
	r.debug("Starting verification chain for tool: %s", toolName)

	// Gate 1: Intent Analysis
	r.runIntentGate(prompt)
	if r.state.IsBlocked() {
		return r.finalize()
	}

	// Gate 2: CEO Validation
	agentType := ""
	if at, ok := toolInput["subagent_type"].(string); ok {
		agentType = at
	}
	r.runCEOGate(toolName, agentType)
	if r.state.IsBlocked() {
		return r.finalize()
	}

	// Gate 3: Aegis Security
	r.runAegisGate(toolName, toolInput)
	if r.state.IsBlocked() {
		return r.finalize()
	}

	// Gate 4: Research Check
	r.runResearchGate(researchDone, prompt)
	if r.state.IsBlocked() {
		return r.finalize()
	}

	// All gates passed
	r.state.FinalStatus = "approved"
	return r.finalize()
}

// runIntentGate executes the Intent classification gate.
func (r *Runner) runIntentGate(prompt string) {
	r.debug("Running Intent gate")

	intent := AnalyzeIntent(prompt)
	r.state.Intent = intent

	result := VerificationResult{
		Gate:   "INTENT",
		Status: "pass",
		Reason: fmt.Sprintf("type=%s confidence=%.2f risk=%s", intent.Type, intent.Confidence, intent.RiskLevel),
		Context: map[string]string{
			"type":       intent.Type,
			"complexity": intent.Complexity,
			"risk_level": intent.RiskLevel,
		},
	}

	// Block if critical risk and low confidence
	if intent.RiskLevel == "critical" && intent.Confidence < 0.7 {
		result.Status = "block"
		result.Reason = "Critical risk with low confidence - requires explicit verification"
		result.NextAction = "Clarify user intent before proceeding"
	}

	r.state.AddResult(result)
}

// runCEOGate executes the CEO validation gate.
func (r *Runner) runCEOGate(toolName, agentType string) {
	r.debug("Running CEO gate")

	ceo := CEOValidate(r.state.Intent, toolName, agentType)
	r.state.CEO = ceo

	result := VerificationResult{
		Gate:   "CEO",
		Status: "pass",
		Reason: "Delegation strategy validated",
	}

	if !ceo.Approved {
		result.Status = "block"
		if len(ceo.Blockers) > 0 {
			result.Reason = ceo.Blockers[0]
		}
		result.NextAction = "Provide required parameters or clarify task"
	} else if len(ceo.Warnings) > 0 {
		result.Status = "warn"
		result.Reason = ceo.Warnings[0]
	}

	if ceo.DelegationPlan != "" {
		result.Context = map[string]string{
			"plan": ceo.DelegationPlan,
		}
	}

	r.state.AddResult(result)
}

// runAegisGate executes the Aegis security gate.
func (r *Runner) runAegisGate(toolName string, toolInput map[string]interface{}) {
	r.debug("Running Aegis gate")

	aegis := AegisVerify(r.state.Intent, toolName, toolInput)
	r.state.Aegis = aegis

	result := VerificationResult{
		Gate:   "AEGIS",
		Status: "pass",
		Reason: fmt.Sprintf("security_score=%.2f threat=%s", aegis.SecurityScore, aegis.ThreatLevel),
		Context: map[string]string{
			"threat_level":   aegis.ThreatLevel,
			"security_score": fmt.Sprintf("%.2f", aegis.SecurityScore),
		},
	}

	if !aegis.Passed {
		result.Status = "block"
		if len(aegis.ViolationsFound) > 0 {
			result.Reason = aegis.ViolationsFound[0]
		}
		result.NextAction = "Address security violations before proceeding"
	}

	if len(aegis.Recommendations) > 0 {
		result.Context["recommendations"] = aegis.Recommendations[0]
	}

	r.state.AddResult(result)
}

// runResearchGate executes the Research (TABULA_RASA) gate.
// STRICT: High-risk intents always require fresh research.
func (r *Runner) runResearchGate(researchDone bool, prompt string) {
	r.debug("Running Research gate")

	research := ResearchCheck(r.state.Intent, researchDone, prompt)
	r.state.Research = research

	result := VerificationResult{
		Gate:   "RESEARCH",
		Status: "pass",
		Reason: "TABULA_RASA compliance verified",
	}

	// If bypassed, just pass
	if research.Bypass {
		result.Reason = "Bypassed: " + research.BypassReason
		r.state.AddResult(result)
		return
	}

	// Block only if research is required AND not yet done
	if !research.Done && r.state.Intent != nil && r.state.Intent.RequiresResearch {
		result.Status = "block"
		result.Reason = "TABULA_RASA: Research required before " + r.state.Intent.Type
		if research.SuggestedQuery != "" {
			result.NextAction = "WebSearch: " + research.SuggestedQuery
			result.Context = map[string]string{
				"suggested_query": research.SuggestedQuery,
			}
		}
	}

	r.state.AddResult(result)
}

// finalize saves state and returns the final chain state.
func (r *Runner) finalize() *ChainState {
	r.saveState()
	return r.state
}

// saveState persists the chain state for debugging/audit.
func (r *Runner) saveState() {
	if r.cacheDir == "" {
		return
	}

	// Ensure directory exists
	os.MkdirAll(r.cacheDir, 0755)

	// Save state as JSON
	filename := fmt.Sprintf("chain_%s_%d.json", r.state.SessionID, time.Now().Unix())
	filepath := filepath.Join(r.cacheDir, filename)

	data, err := json.MarshalIndent(r.state, "", "  ")
	if err != nil {
		return
	}

	os.WriteFile(filepath, data, 0644)
}

// debug logs debug messages if debug mode is enabled.
func (r *Runner) debug(format string, args ...interface{}) {
	if r.debugMode {
		fmt.Fprintf(os.Stderr, "[CHAIN] "+format+"\n", args...)
	}
}

// GetState returns the current chain state.
func (r *Runner) GetState() *ChainState {
	return r.state
}

// ToTOON converts the chain state to TOON format for context injection.
func (r *Runner) ToTOON() string {
	toon := "[VERIFICATION_CHAIN]\n"
	toon += fmt.Sprintf("session: %s\n", r.state.SessionID)
	toon += fmt.Sprintf("status: %s\n", r.state.FinalStatus)
	toon += fmt.Sprintf("timestamp: %s\n", time.Now().Format(time.RFC3339))
	toon += "\n"

	for _, result := range r.state.Results {
		toon += fmt.Sprintf("[%s]\n", result.Gate)
		toon += fmt.Sprintf("status: %s\n", result.Status)
		toon += fmt.Sprintf("reason: %s\n", result.Reason)
		if result.NextAction != "" {
			toon += fmt.Sprintf("next_action: %s\n", result.NextAction)
		}
		toon += "\n"
	}

	return toon
}

// ToJSON converts the chain state to JSON.
func (r *Runner) ToJSON() string {
	data, _ := json.MarshalIndent(r.state, "", "  ")
	return string(data)
}
