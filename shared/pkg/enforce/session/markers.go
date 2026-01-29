// Package session provides session state management.
// markers.go: State marking functions.
// DACE: Single responsibility - marker functions only.
package session

import "time"

// MarkResearchDone marks that WebSearch was performed.
func (s *SessionState) MarkResearchDone() {
	s.ResearchDone = true
	s.Save()
}

// MarkMemoryQueried marks that memory bank was queried.
func (s *SessionState) MarkMemoryQueried() {
	s.MemoryQueried = true
	s.Save()
}

// MarkCEOInvoked marks that CEO was invoked.
func (s *SessionState) MarkCEOInvoked() {
	s.CEOInvoked = true
	s.Save()
}

// Dead code audit: MarkNLUParsed and MarkAegisVerified were never called.
// NLUParsed and AegisVerified fields are still persisted for future use.

// MarkPostCompact marks that context was compacted.
func (s *SessionState) MarkPostCompact() {
	s.PostCompact = true
	s.CompactedAt = time.Now().Format("2006-01-02T15:04:05")
	s.CompactCount++
	s.Save()
}

// ClearPostCompact clears post-compact mode after resume.
func (s *SessionState) ClearPostCompact() {
	s.PostCompact = false
	s.Save()
}

// IsPostCompact returns true if in post-compact mode.
func (s *SessionState) IsPostCompact() bool {
	return s.PostCompact
}

// IncrementTurn increments the conversation turn counter.
// Called on each UserPromptSubmit.
func (s *SessionState) IncrementTurn() {
	s.TurnCount++
	s.Save()
}

// NeedsReinforcement returns true if context reinforcement is needed.
// This mitigates "Lost in the Middle" attention decay in long conversations.
// Default: reinforce every 15 turns (~50k tokens estimated).
func (s *SessionState) NeedsReinforcement() bool {
	threshold := s.ReinforceEveryN
	if threshold == 0 {
		threshold = 15 // Default: every 15 turns
	}
	return s.TurnCount-s.LastReinforceTurn >= threshold
}

// MarkReinforcementDone marks that context was reinforced.
func (s *SessionState) MarkReinforcementDone() {
	s.LastReinforceTurn = s.TurnCount
	s.Save()
}

// Dead code audit: SetCurrentTask and ResetTaskResearch were never called.
// Task state is managed by task.go gate via TaskCreate/TaskUpdate hooks.

// StoreIntent persists intent classification for the CEO gate to read.
func (s *SessionState) StoreIntent(intentType, domain string, subAgents, skills []string) {
	s.IntentType = intentType
	s.IntentDomain = domain
	s.IntentSubAgents = subAgents
	s.IntentSkills = skills
	s.Save()
}
