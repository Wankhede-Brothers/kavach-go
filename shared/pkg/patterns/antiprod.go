// Package patterns provides anti-production pattern detection.
// antiprod.go: Hierarchical P0-P3 detection for production code quality.
package patterns

import (
	"regexp"
	"strings"
)

// AntiProdLevel represents priority levels for anti-production patterns.
type AntiProdLevel int

const (
	P0MockData   AntiProdLevel = iota // Fake data, placeholders
	P1ProdLeak                        // console.log, TODO, localhost, env
	P2ErrorBlind                      // Empty catch, non-null !, unwrap
	P3TypeLoose                       // as any, ts-ignore, eslint-disable
)

// AntiProdResult holds the detection result.
type AntiProdResult struct {
	Level   AntiProdLevel
	Code    string
	Match   string
	Message string
}

// P1: Production leak patterns
var (
	reConsoleLog   = regexp.MustCompile(`\bconsole\.(log|debug|info|warn|error)\s*\(`)
	reTodoComment  = regexp.MustCompile(`(?i)\b(TODO|FIXME|HACK|XXX)\b`)
	reLocalhost    = regexp.MustCompile(`https?://localhost\b`)
	reProcessEnv   = regexp.MustCompile(`process\.env\.\w+`)
	reEnvNoDefault = regexp.MustCompile(`std::env::var\(\s*"[^"]+"\s*\)\.unwrap\(\)`)
)

// P2: Error blindness patterns
var (
	reEmptyCatch    = regexp.MustCompile(`\.catch\s*\(\s*(?:\(\s*\)|_)\s*=>\s*\{\s*\}\s*\)`)
	reNonNullAssert = regexp.MustCompile(`[a-zA-Z_]\w*!\.[a-zA-Z_]`)
	reRustUnwrap    = regexp.MustCompile(`\.unwrap\(\)`)
	reFetchNoError  = regexp.MustCompile(`fetch\s*\([^)\n]+\)\s*(?:\.then|;)`)
)

// P3: Type looseness patterns
var (
	reAsAny     = regexp.MustCompile(`\bas\s+any\b`)
	reTsIgnore  = regexp.MustCompile(`@ts-ignore|@ts-expect-error`)
	reEslintDis = regexp.MustCompile(`eslint-disable(?:-next-line)?`)
	reTsNoCheck = regexp.MustCompile(`@ts-nocheck`)
)

// DetectAntiProd runs hierarchical P0→P3 checks.
// Returns on first detection at each level (P0 first, then P1, etc.).
// All levels are checked — caller decides whether to stop at first hit.
func DetectAntiProd(filePath, content string) []AntiProdResult {
	if content == "" || isAllowlisted(filePath) {
		return nil
	}

	var results []AntiProdResult

	// P0: Mock data (delegate to existing DetectMockData)
	if detected, reason := DetectMockData(filePath, content); detected {
		results = append(results, AntiProdResult{
			Level:   P0MockData,
			Code:    "MOCK_DATA",
			Match:   reason,
			Message: "Replace hardcoded data with real API fetch. Use: fetch('/api/...') or sqlx::query_as.",
		})
	}

	// P1: Production leaks (frontend + backend)
	if isFrontendFile(filePath) {
		if m := reConsoleLog.FindString(content); m != "" {
			results = append(results, AntiProdResult{
				Level:   P1ProdLeak,
				Code:    "PROD_LEAK",
				Match:   "console.log",
				Message: "Remove debug output or use structured logger.",
			})
		}
	}
	if m := reTodoComment.FindString(content); m != "" {
		results = append(results, AntiProdResult{
			Level:   P1ProdLeak,
			Code:    "PROD_LEAK",
			Match:   "TODO/FIXME",
			Message: "Implement or create ticket. Do not ship TODO comments.",
		})
	}
	if isNonConfigFile(filePath) && reLocalhost.MatchString(content) {
		results = append(results, AntiProdResult{
			Level:   P1ProdLeak,
			Code:    "PROD_LEAK",
			Match:   "http://localhost",
			Message: "Use config/environment variable for URLs.",
		})
	}
	if isFrontendFile(filePath) {
		for _, line := range strings.Split(content, "\n") {
			if reProcessEnv.MatchString(line) && !hasEnvFallback(line) {
				results = append(results, AntiProdResult{
					Level:   P1ProdLeak,
					Code:    "PROD_LEAK",
					Match:   "process.env without fallback",
					Message: "Add fallback: process.env.X ?? 'default' or process.env.X || 'default'.",
				})
				break
			}
		}
	}
	if isBackendFile(filePath) && reEnvNoDefault.MatchString(content) {
		results = append(results, AntiProdResult{
			Level:   P1ProdLeak,
			Code:    "PROD_LEAK",
			Match:   "env::var().unwrap()",
			Message: "Use env::var().unwrap_or_else() or env::var().ok().",
		})
	}

	// P2: Error blindness
	if isFrontendFile(filePath) {
		if reEmptyCatch.MatchString(content) {
			results = append(results, AntiProdResult{
				Level:   P2ErrorBlind,
				Code:    "ERROR_BLIND",
				Match:   ".catch(() => {})",
				Message: "Handle errors: log, show user feedback, or re-throw.",
			})
		}
		if reNonNullAssert.MatchString(content) {
			results = append(results, AntiProdResult{
				Level:   P2ErrorBlind,
				Code:    "ERROR_BLIND",
				Match:   "non-null assertion (!.)",
				Message: "Use optional chaining (?.) instead of non-null assertion.",
			})
		}
		if reFetchNoError.MatchString(content) && !strings.Contains(content, ".catch") && !strings.Contains(content, "try") {
			results = append(results, AntiProdResult{
				Level:   P2ErrorBlind,
				Code:    "ERROR_BLIND",
				Match:   "fetch() without error handling",
				Message: "Wrap fetch in try/catch or add .catch() handler.",
			})
		}
	}
	if isBackendFile(filePath) && strings.HasSuffix(strings.ToLower(filePath), ".rs") {
		// Only flag .unwrap() in handler files, not tests
		if reRustUnwrap.MatchString(content) && isHandlerFile(filePath) {
			results = append(results, AntiProdResult{
				Level:   P2ErrorBlind,
				Code:    "ERROR_BLIND",
				Match:   ".unwrap() in handler",
				Message: "Use ? operator instead of .unwrap() in request handlers.",
			})
		}
	}

	// P3: Type looseness (frontend only)
	if isFrontendFile(filePath) {
		if reAsAny.MatchString(content) {
			results = append(results, AntiProdResult{
				Level:   P3TypeLoose,
				Code:    "TYPE_LOOSE",
				Match:   "as any",
				Message: "Use proper type narrowing instead of 'as any'.",
			})
		}
		if reTsIgnore.MatchString(content) {
			results = append(results, AntiProdResult{
				Level:   P3TypeLoose,
				Code:    "TYPE_LOOSE",
				Match:   "@ts-ignore",
				Message: "Fix the type error instead of suppressing it.",
			})
		}
		if reEslintDis.MatchString(content) {
			results = append(results, AntiProdResult{
				Level:   P3TypeLoose,
				Code:    "TYPE_LOOSE",
				Match:   "eslint-disable",
				Message: "Fix the lint error instead of disabling the rule.",
			})
		}
		if reTsNoCheck.MatchString(content) {
			results = append(results, AntiProdResult{
				Level:   P3TypeLoose,
				Code:    "TYPE_LOOSE",
				Match:   "@ts-nocheck",
				Message: "Remove @ts-nocheck and fix type errors.",
			})
		}
	}

	return results
}

// isNonConfigFile returns true for files that should not contain localhost.
func isNonConfigFile(path string) bool {
	p := strings.ToLower(path)
	configPatterns := []string{
		"config", ".env", "astro.config", "vite.config", "next.config",
		"wrangler.toml", "docker-compose", "Caddyfile", ".toml",
		"dev_ports", "constants",
	}
	for _, pat := range configPatterns {
		if strings.Contains(p, pat) {
			return false
		}
	}
	return true
}

// isHandlerFile returns true for Rust handler/route files.
func isHandlerFile(path string) bool {
	p := strings.ToLower(path)
	return strings.Contains(p, "handler") || strings.Contains(p, "routes") ||
		strings.Contains(p, "lib.rs") || strings.Contains(p, "main.rs")
}

// hasEnvFallback checks if a line with process.env already has a fallback.
func hasEnvFallback(line string) bool {
	return strings.Contains(line, "??") || strings.Contains(line, "||") ||
		strings.Contains(line, "import.meta.env")
}
