// Package enforce provides enforcement of core principles.
// DACE: Dynamic Agentic Context Engineering
// SP/1.0: Sutra Protocol 1.0 (TOON format)
// Tabula Rasa: No trust in training weights
package enforce

import (
	"time"

	"github.com/claude/shared/pkg/toon"
	"github.com/claude/shared/pkg/util"
)

// Principles defines the core enforcement rules.
type Principles struct {
	// TabulaRasa: Training weights are stale, must verify
	TabulaRasa bool
	// DateInjection: Always inject current date
	DateInjection bool
	// NoAmnesia: Must query memory bank
	NoAmnesia bool
	// NoAssumption: Verify before acting
	NoAssumption bool
	// DACE: Lazy load, skill-first
	DACE bool
}

// DefaultPrinciples returns all principles enabled.
func DefaultPrinciples() *Principles {
	return &Principles{
		TabulaRasa:    true,
		DateInjection: true,
		NoAmnesia:     true,
		NoAssumption:  true,
		DACE:          true,
	}
}

// Context provides enforcement context for hooks.
type Context struct {
	Today          string
	SessionID      string
	Project        string
	WorkDir        string
	MemoryBank     *toon.MemoryBank
	Governance     *toon.Document
	Index          *toon.Document
	TrainingCutoff string
}

// NewContext creates enforcement context with date injection.
func NewContext() *Context {
	ctx := &Context{
		Today:          time.Now().Format("2006-01-02"),
		WorkDir:        util.WorkingDir(),
		TrainingCutoff: "2025-01",
	}

	// Load memory bank (NO AMNESIA)
	ctx.MemoryBank = toon.NewMemoryBank()

	// Load governance rules (log but don't fail - may not exist yet)
	var err error
	ctx.Governance, err = ctx.MemoryBank.LoadGovernance()
	if err != nil {
		// Not fatal - governance may not exist in new projects
	}

	// Load index (log but don't fail - may not exist yet)
	ctx.Index, err = ctx.MemoryBank.LoadIndex()
	if err != nil {
		// Not fatal - index may not exist in new projects
	}

	return ctx
}

// DateBlock returns TOON block for date injection.
func (c *Context) DateBlock() string {
	return "[DATE]\n" +
		"today: " + c.Today + "\n" +
		"training_cutoff: " + c.TrainingCutoff + "\n" +
		"status: WEIGHTS_STALE\n"
}

// TabulaRasaBlock returns enforcement block.
func (c *Context) TabulaRasaBlock() string {
	return "[TABULA_RASA]\n" +
		"severity: CRITICAL\n" +
		"rule: NO_TRUST_TRAINING_WEIGHTS\n" +
		"today: " + c.Today + "\n" +
		"cutoff: " + c.TrainingCutoff + "\n" +
		"action: WebSearch BEFORE code\n"
}

// NoAmnesiaBlock returns memory enforcement block.
func (c *Context) NoAmnesiaBlock() string {
	return "[NO_AMNESIA]\n" +
		"severity: CRITICAL\n" +
		"memory_path: ~/.local/shared/shared-ai/memory/\n" +
		"rule: QUERY_MEMORY_BANK\n" +
		"forbidden: \"I have no memory\"\n"
}

// DACEBlock returns DACE enforcement block.
func (c *Context) DACEBlock() string {
	return "[DACE]\n" +
		"mode: LAZY_LOAD\n" +
		"routing: SKILL_FIRST\n" +
		"research: ON_DEMAND\n" +
		"anti_pattern: EAGER_LOAD_ALL\n"
}

// FullEnforcementContext returns complete context injection.
func (c *Context) FullEnforcementContext() string {
	return c.DateBlock() + "\n" +
		c.TabulaRasaBlock() + "\n" +
		c.NoAmnesiaBlock() + "\n" +
		c.DACEBlock()
}

// Dead code audit: ValidateResearchDone(map[string]bool) and BlockedPhrases()
// were never called. Research validation is done by agentic.ResearchGate and
// chain.ResearchCheck. Blocked phrases are enforced by intent_output.go directives.
