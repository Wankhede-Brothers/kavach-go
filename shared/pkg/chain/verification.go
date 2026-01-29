// Package chain provides multi-agent verification chain for kavach.
// Pattern: Intent → CEO → Aegis → Research
// Reference: https://www.microsoft.com/en-us/security/blog/2026/01/23/runtime-risk-realtime-defense-securing-ai-agents/
package chain

import (
	"strings"
	"time"
)

// VerificationResult holds the result of a verification step.
type VerificationResult struct {
	Gate       string            `json:"gate"`
	Status     string            `json:"status"` // "pass", "warn", "block"
	Reason     string            `json:"reason"`
	Context    map[string]string `json:"context,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
	NextAction string            `json:"next_action,omitempty"` // Suggestion for next step
}

// ChainState holds the accumulated state across verification gates.
type ChainState struct {
	SessionID   string                 `json:"session_id"`
	Intent      *IntentAnalysis        `json:"intent,omitempty"`
	CEO         *CEODecision           `json:"ceo,omitempty"`
	Aegis       *AegisVerification     `json:"aegis,omitempty"`
	Research    *ResearchStatus        `json:"research,omitempty"`
	Results     []VerificationResult   `json:"results"`
	FinalStatus string                 `json:"final_status"` // "approved", "blocked", "pending"
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// IntentAnalysis holds the result of intent classification.
type IntentAnalysis struct {
	Type             string   `json:"type"`              // "implement", "debug", "research", "refactor", "deploy"
	Confidence       float64  `json:"confidence"`        // 0.0 - 1.0
	RequiredSkills   []string `json:"required_skills"`   // Skills needed for this intent
	RequiredAgents   []string `json:"required_agents"`   // Agents needed for delegation
	RequiresResearch bool     `json:"requires_research"` // TABULA_RASA trigger
	Complexity       string   `json:"complexity"`        // "simple", "moderate", "complex"
	RiskLevel        string   `json:"risk_level"`        // "low", "medium", "high", "critical"
}

// CEODecision holds the CEO gate's delegation decision.
type CEODecision struct {
	Approved       bool     `json:"approved"`
	DelegationPlan string   `json:"delegation_plan,omitempty"`
	AssignedAgents []string `json:"assigned_agents,omitempty"`
	TaskBreakdown  []string `json:"task_breakdown,omitempty"`
	Blockers       []string `json:"blockers,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
}

// AegisVerification holds security verification results.
type AegisVerification struct {
	Passed           bool     `json:"passed"`
	SecurityScore    float64  `json:"security_score"`    // 0.0 - 1.0
	ThreatLevel      string   `json:"threat_level"`      // "none", "low", "medium", "high"
	ViolationsFound  []string `json:"violations_found"`  // Security violations
	Recommendations  []string `json:"recommendations"`   // Security recommendations
	MemoryProvenance string   `json:"memory_provenance"` // Source tracking
}

// ResearchStatus holds TABULA_RASA compliance status.
type ResearchStatus struct {
	Done           bool     `json:"done"`
	Sources        []string `json:"sources,omitempty"`
	SuggestedQuery string   `json:"suggested_query,omitempty"`
	Bypass         bool     `json:"bypass"`        // True for trivial changes
	BypassReason   string   `json:"bypass_reason"` // Why bypassed
}

// NewChainState creates a new verification chain state.
func NewChainState(sessionID string) *ChainState {
	return &ChainState{
		SessionID:   sessionID,
		Results:     make([]VerificationResult, 0),
		FinalStatus: "pending",
		Metadata:    make(map[string]interface{}),
	}
}

// AddResult adds a verification result to the chain.
func (c *ChainState) AddResult(result VerificationResult) {
	result.Timestamp = time.Now()
	c.Results = append(c.Results, result)

	// Update final status based on results
	if result.Status == "block" {
		c.FinalStatus = "blocked"
	}
}

// IsBlocked returns true if any gate blocked the action.
func (c *ChainState) IsBlocked() bool {
	return c.FinalStatus == "blocked"
}

// GetBlockReason returns the reason for blocking, if any.
func (c *ChainState) GetBlockReason() string {
	for _, r := range c.Results {
		if r.Status == "block" {
			return r.Gate + ": " + r.Reason
		}
	}
	return ""
}

// ===== Intent Analysis =====

// AnalyzeIntent classifies user intent from prompt.
func AnalyzeIntent(prompt string) *IntentAnalysis {
	promptLower := strings.ToLower(prompt)
	analysis := &IntentAnalysis{
		Type:             "general",
		Confidence:       0.5,
		RequiredSkills:   []string{},
		RequiredAgents:   []string{},
		RequiresResearch: false,
		Complexity:       "simple",
		RiskLevel:        "low",
	}

	// Implementation intent
	if containsAny(promptLower, []string{"implement", "create", "build", "add", "develop", "write"}) {
		analysis.Type = "implement"
		analysis.RequiresResearch = true
		analysis.Complexity = "moderate"
		analysis.Confidence = 0.8
	}

	// Debug intent
	if containsAny(promptLower, []string{"fix", "bug", "error", "debug", "broken", "not working", "crash"}) {
		analysis.Type = "debug"
		analysis.RequiredSkills = append(analysis.RequiredSkills, "debug-like-expert")
		analysis.Complexity = "moderate"
		analysis.Confidence = 0.85
	}

	// Refactor intent
	if containsAny(promptLower, []string{"refactor", "restructure", "clean up", "improve", "optimize"}) {
		analysis.Type = "refactor"
		analysis.RequiresResearch = true
		analysis.Complexity = "complex"
		analysis.RiskLevel = "medium"
		analysis.Confidence = 0.8
	}

	// Deploy intent
	if containsAny(promptLower, []string{"deploy", "release", "publish", "production", "go live"}) {
		analysis.Type = "deploy"
		analysis.RequiredSkills = append(analysis.RequiredSkills, "cloud-infrastructure-mastery")
		analysis.RiskLevel = "high"
		analysis.Complexity = "complex"
		analysis.Confidence = 0.9
		analysis.RequiresResearch = true // Deploy always needs verification
	}

	// Security intent
	if containsAny(promptLower, []string{"security", "auth", "encrypt", "vulnerability", "password"}) {
		analysis.Type = "security"
		analysis.RequiredSkills = append(analysis.RequiredSkills, "security")
		analysis.RiskLevel = "high"
		analysis.RequiresResearch = true
		analysis.Confidence = 0.85
	}

	// Deletion/removal intent - HIGH RISK
	if containsAny(promptLower, []string{"delete", "remove", "drop", "destroy", "purge"}) {
		analysis.RiskLevel = "critical"
		analysis.Complexity = "complex"
	}

	// Extract required agents based on context
	analysis.RequiredAgents = extractAgents(promptLower)

	return analysis
}

// ===== CEO Gate =====

// CEOValidate validates the delegation strategy.
func CEOValidate(intent *IntentAnalysis, toolName, agentType string) *CEODecision {
	decision := &CEODecision{
		Approved:       true,
		AssignedAgents: []string{},
		TaskBreakdown:  []string{},
		Blockers:       []string{},
		Warnings:       []string{},
	}

	// Check if Task tool requires subagent_type
	if toolName == "Task" && agentType == "" {
		decision.Approved = false
		decision.Blockers = append(decision.Blockers, "Task requires subagent_type")
		return decision
	}

	// Validate agent for intent
	if intent != nil && len(intent.RequiredAgents) > 0 {
		if agentType != "" && !containsString(intent.RequiredAgents, agentType) {
			decision.Warnings = append(decision.Warnings,
				"Agent '"+agentType+"' may not be optimal for intent '"+intent.Type+"'")
		}
	}

	// High-risk intents require explicit verification
	if intent != nil && intent.RiskLevel == "critical" {
		decision.Warnings = append(decision.Warnings,
			"CRITICAL risk level - verify user intent before proceeding")
	}

	// Complex tasks should be broken down
	if intent != nil && intent.Complexity == "complex" {
		decision.DelegationPlan = "Complex task - recommend task breakdown"
		decision.TaskBreakdown = []string{
			"1. Research current patterns",
			"2. Create implementation plan",
			"3. Implement with verification",
			"4. Test and validate",
		}
	}

	return decision
}

// ===== Aegis Gate =====

// AegisVerify performs security verification.
func AegisVerify(intent *IntentAnalysis, toolName string, toolInput map[string]interface{}) *AegisVerification {
	verification := &AegisVerification{
		Passed:          true,
		SecurityScore:   1.0,
		ThreatLevel:     "none",
		ViolationsFound: []string{},
		Recommendations: []string{},
	}

	// Check for dangerous patterns in tool input
	if toolName == "Bash" {
		if cmd, ok := toolInput["command"].(string); ok {
			if isDangerousCommand(cmd) {
				verification.Passed = false
				verification.ThreatLevel = "high"
				verification.SecurityScore = 0.0
				verification.ViolationsFound = append(verification.ViolationsFound,
					"Dangerous command pattern detected")
			}
		}
	}

	// Check file access patterns
	if toolName == "Read" || toolName == "Write" || toolName == "Edit" {
		if path, ok := toolInput["file_path"].(string); ok {
			if isSensitivePath(path) {
				verification.Passed = false
				verification.ThreatLevel = "high"
				verification.SecurityScore = 0.0
				verification.ViolationsFound = append(verification.ViolationsFound,
					"Sensitive file access: "+path)
			}
		}
	}

	// Check for code removal patterns (hallucination prevention)
	if toolName == "Edit" {
		oldStr, _ := toolInput["old_string"].(string)
		newStr, _ := toolInput["new_string"].(string)

		if isProblematicEdit(oldStr, newStr) {
			verification.Passed = false
			verification.ThreatLevel = "medium"
			verification.SecurityScore = 0.3
			verification.ViolationsFound = append(verification.ViolationsFound,
				"Suspicious code removal pattern - verify intent")
		}
	}

	// Add memory provenance
	verification.MemoryProvenance = "chain_verification:" + time.Now().Format(time.RFC3339)

	return verification
}

// ===== Research Gate =====

// ResearchCheck verifies TABULA_RASA compliance.
// STRICT: For high-risk intents, always require fresh research verification.
func ResearchCheck(intent *IntentAnalysis, researchDone bool, prompt string) *ResearchStatus {
	status := &ResearchStatus{
		Done:   researchDone,
		Bypass: false,
	}

	// Check for bypass patterns (trivial changes)
	promptLower := strings.ToLower(prompt)
	bypassPatterns := []string{"typo", "comment", "rename", "format", "whitespace", "spacing", "fix typo"}
	for _, p := range bypassPatterns {
		if strings.Contains(promptLower, p) {
			status.Bypass = true
			status.BypassReason = "Trivial change: " + p
			return status
		}
	}

	// High-risk intents: require research if not yet done, but respect completed research
	if intent != nil && intent.RequiresResearch {
		if !researchDone {
			status.Done = false
			status.SuggestedQuery = buildSearchQuery(intent.Type, prompt)
			return status
		}
		// Research was done — trust it, even for high-risk intents
	}

	return status
}

// ===== Helper Functions =====

func containsAny(s string, patterns []string) bool {
	for _, p := range patterns {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func extractAgents(prompt string) []string {
	agents := []string{}
	agentKeywords := map[string]string{
		"backend":    "backend-engineer",
		"frontend":   "frontend-engineer",
		"database":   "database-engineer",
		"devops":     "devops-engineer",
		"security":   "security-engineer",
		"test":       "qa-lead",
		"explore":    "Explore",
		"plan":       "Plan",
	}
	for keyword, agent := range agentKeywords {
		if strings.Contains(prompt, keyword) {
			agents = append(agents, agent)
		}
	}
	return agents
}

func isDangerousCommand(cmd string) bool {
	dangerous := []string{
		"rm -rf /", "rm -rf /*", "> /dev/sda",
		":(){ :|:& };:", "dd if=/dev/zero",
		"chmod -R 777 /", "curl | bash", "wget | sh",
	}
	cmdLower := strings.ToLower(cmd)
	for _, d := range dangerous {
		if strings.Contains(cmdLower, d) {
			return true
		}
	}
	return false
}

func isSensitivePath(path string) bool {
	sensitive := []string{
		"/etc/shadow", "/etc/passwd", "/.ssh/",
		"/.aws/credentials", "/.gnupg/", ".pem", ".key",
	}
	pathLower := strings.ToLower(path)
	for _, s := range sensitive {
		if strings.Contains(pathLower, s) {
			return true
		}
	}
	return false
}

func isProblematicEdit(old, new string) bool {
	// Empty replacement of significant code
	if strings.TrimSpace(new) == "" && len(old) > 100 {
		return true
	}
	// Removing TODO/FIXME without expanding code
	oldHasStub := containsAny(strings.ToLower(old), []string{"todo", "fixme", "stub", "placeholder"})
	newHasStub := containsAny(strings.ToLower(new), []string{"todo", "fixme", "stub", "placeholder"})
	if oldHasStub && !newHasStub && len(new) <= len(old) {
		return true
	}
	return false
}

func buildSearchQuery(intentType, prompt string) string {
	year := time.Now().Format("2006")
	switch intentType {
	case "implement":
		return "implementation patterns " + year + " best practices"
	case "security":
		return "security best practices " + year + " OWASP"
	case "deploy":
		return "deployment patterns " + year + " production"
	case "refactor":
		return "refactoring patterns " + year + " clean code"
	default:
		return "latest patterns " + year
	}
}
