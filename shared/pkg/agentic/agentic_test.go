// Package agentic provides tests for Dynamic Agentic Context Engineering.
package agentic

import (
	"strings"
	"testing"
	"time"
)

// =============================================================================
// ResearchGate Tests
// =============================================================================

func TestNewResearchGate(t *testing.T) {
	gate := NewResearchGate()

	// Date should be today
	today := time.Now().Format("2006-01-02")
	if gate.Today() != today {
		t.Errorf("Today() = %v, want %v", gate.Today(), today)
	}

	// Year should be current year
	year := time.Now().Format("2006")
	if gate.Year() != year {
		t.Errorf("Year() = %v, want %v", gate.Year(), year)
	}
}

func TestRequireResearch_Frameworks(t *testing.T) {
	gate := NewResearchGate()
	year := gate.Year()

	tests := []struct {
		task      string
		wantTopic string
		mandatory bool
	}{
		{"implement axum handler", "axum", true},
		{"create react component", "react", true},
		{"setup kubernetes deployment", "kubernetes", true},
		{"write terraform config", "terraform", true},
		{"build tonic grpc service", "tonic", true},
		{"simple variable rename", "", false}, // No framework
	}

	for _, tt := range tests {
		req := gate.RequireResearch(tt.task)
		if tt.wantTopic == "" {
			if req != nil && req.Mandatory {
				t.Errorf("RequireResearch(%q) returned mandatory requirement, want nil", tt.task)
			}
			continue
		}

		if req == nil {
			t.Errorf("RequireResearch(%q) = nil, want requirement for %s", tt.task, tt.wantTopic)
			continue
		}

		if req.Topic != tt.wantTopic {
			t.Errorf("RequireResearch(%q).Topic = %v, want %v", tt.task, req.Topic, tt.wantTopic)
		}

		if req.Mandatory != tt.mandatory {
			t.Errorf("RequireResearch(%q).Mandatory = %v, want %v", tt.task, req.Mandatory, tt.mandatory)
		}

		// Keywords should include current year
		hasYear := false
		for _, kw := range req.Keywords {
			if strings.Contains(kw, year) {
				hasYear = true
				break
			}
		}
		if !hasYear {
			t.Errorf("RequireResearch(%q).Keywords missing year %s", tt.task, year)
		}
	}
}

func TestBuildSearchQuery(t *testing.T) {
	gate := NewResearchGate()
	year := gate.Year()

	query := gate.BuildSearchQuery("axum routing")

	if !strings.Contains(query, "axum routing") {
		t.Errorf("BuildSearchQuery missing topic, got %v", query)
	}

	if !strings.Contains(query, year) {
		t.Errorf("BuildSearchQuery missing year %s, got %v", year, query)
	}

	if !strings.Contains(query, "documentation") {
		t.Errorf("BuildSearchQuery missing 'documentation', got %v", query)
	}
}

func TestValidateResearchDone(t *testing.T) {
	gate := NewResearchGate()

	tests := []struct {
		context string
		want    bool
	}{
		{"I used WebSearch to find patterns", true},
		{"According to the documentation shows...", true},
		{"The latest docs say...", true},
		{"I think this is the pattern", false},
		{"Based on my experience", false},
	}

	for _, tt := range tests {
		got := gate.ValidateResearchDone(tt.context)
		if got != tt.want {
			t.Errorf("ValidateResearchDone(%q) = %v, want %v", tt.context, got, tt.want)
		}
	}
}

func TestCheckForForbiddenPhrases(t *testing.T) {
	gate := NewResearchGate()

	tests := []struct {
		text       string
		violations int
	}{
		{"Based on my knowledge, this is correct", 1},
		{"I think the pattern is X", 1},
		{"According to docs, use Y", 0},
		{"I believe the API is Z, but I recall it differently", 2},
	}

	for _, tt := range tests {
		violations := gate.CheckForForbiddenPhrases(tt.text)
		if len(violations) != tt.violations {
			t.Errorf("CheckForForbiddenPhrases(%q) = %v violations, want %v",
				tt.text, len(violations), tt.violations)
		}
	}
}

func TestExtractFrameworkFromTask(t *testing.T) {
	tests := []struct {
		task string
		want []string
	}{
		{"implement axum handler", []string{"axum"}},
		{"build react frontend with vue fallback", []string{"react", "vue"}},
		{"setup kubernetes with terraform", []string{"kubernetes", "terraform"}},
		{"simple refactor", nil},
		{"AXUM and axum", []string{"axum"}}, // Deduplication
	}

	for _, tt := range tests {
		got := ExtractFrameworkFromTask(tt.task)
		if len(got) != len(tt.want) {
			t.Errorf("ExtractFrameworkFromTask(%q) = %v, want %v", tt.task, got, tt.want)
			continue
		}
		for i, fw := range tt.want {
			if got[i] != fw {
				t.Errorf("ExtractFrameworkFromTask(%q)[%d] = %v, want %v", tt.task, i, got[i], fw)
			}
		}
	}
}

// =============================================================================
// SkillFirstRouter Tests
// =============================================================================

func TestNewSkillFirstRouter(t *testing.T) {
	router := NewSkillFirstRouter(nil)
	if router == nil {
		t.Fatal("NewSkillFirstRouter returned nil")
	}
}

func TestRegisterSkillTrigger(t *testing.T) {
	router := NewSkillFirstRouter(nil)
	router.RegisterSkillTrigger("rust", "backend")
	router.RegisterSkillTrigger("REACT", "frontend") // Test case-insensitivity

	decision := router.Route("", []string{"rust"})
	if !decision.UseSkill {
		t.Error("Route with 'rust' trigger should use skill")
	}
	if decision.SkillName != "backend" {
		t.Errorf("SkillName = %v, want 'backend'", decision.SkillName)
	}

	decision = router.Route("", []string{"react"}) // lowercase
	if !decision.UseSkill {
		t.Error("Route with 'react' trigger should use skill")
	}
	if decision.SkillName != "frontend" {
		t.Errorf("SkillName = %v, want 'frontend'", decision.SkillName)
	}
}

// TestRoute_IntentMapping and TestRoute_ComplexTask removed: permanently skipped stubs (dead code audit 2026-01-29)

func TestRoute_DefaultAgent(t *testing.T) {
	router := NewSkillFirstRouter(nil)

	// Task that doesn't match any skill or complexity indicator
	decision := router.Route("review code style", nil)

	if decision.UseSkill {
		t.Error("Unmatched task should not use skill")
	}
	if decision.AgentName != "backend-engineer" {
		t.Errorf("Default agent should be 'backend-engineer', got %v", decision.AgentName)
	}
}

// TestGetSkillForAgent and TestShouldPreferSkill removed: permanently skipped stubs (dead code audit 2026-01-29)

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkRequireResearch(b *testing.B) {
	gate := NewResearchGate()
	task := "implement axum handler with tonic grpc"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gate.RequireResearch(task)
	}
}

func BenchmarkRoute(b *testing.B) {
	router := NewSkillFirstRouter(nil)
	router.RegisterSkillTrigger("rust", "backend")
	router.RegisterSkillTrigger("react", "frontend")

	keywords := []string{"rust", "api", "backend"}
	intent := "implement backend api"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.Route(intent, keywords)
	}
}

func BenchmarkExtractFramework(b *testing.B) {
	task := "build react frontend with kubernetes deployment and terraform infrastructure"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractFrameworkFromTask(task)
	}
}

func BenchmarkCheckForbiddenPhrases(b *testing.B) {
	gate := NewResearchGate()
	text := "I think based on my knowledge this is the best approach but I recall differently"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gate.CheckForForbiddenPhrases(text)
	}
}
