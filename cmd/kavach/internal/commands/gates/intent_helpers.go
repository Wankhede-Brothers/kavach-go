// Package gates provides hook gates for Claude Code.
// intent_helpers.go: Helper functions for intent classification.
// DACE: Micro-modular split from intent.go
package gates

import "strings"

func isSimpleQuery(prompt string) bool {
	simple := []string{"hello", "hi", "hey", "thanks", "thank you", "bye", "yes", "no", "ok", "okay"}
	trimmed := strings.TrimSpace(prompt)
	for _, s := range simple {
		if trimmed == s {
			return true
		}
	}
	return false
}

func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

func isStatusQuery(prompt string) bool {
	triggers := []string{"status", "project status", "what is the status", "show status", "check status"}
	for _, t := range triggers {
		if strings.Contains(prompt, t) {
			return true
		}
	}
	return false
}

// Dead code removed: isImplementationIntent, containsTechnicalTerms
// Both were defined but never called from any gate or hook.
// Intent classification now handled entirely by classifyIntentFromConfig() in intent_nlu.go.
