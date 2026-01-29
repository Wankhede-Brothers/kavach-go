// Package config provides dynamic configuration loading.
// gates.go: Load security gates config from ~/.claude/gates/config.json
// DACE: JSON config for LLM-modifiable security rules.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// GatesConfig holds all gate configurations from config.json
type GatesConfig struct {
	Schema      string         `json:"$schema"`
	Description string         `json:"description"`
	Updated     string         `json:"updated"`
	Read        ReadConfig     `json:"read"`
	Bash        BashConfig     `json:"bash"`
	Write       WriteConfig    `json:"write"`
	Enforcer    EnforcerConfig `json:"enforcer"`
	Intent      IntentConfig   `json:"intent"`
	Research    ResearchConfig `json:"research"`
	Context     ContextConfig  `json:"context"`
	Quality     QualityConfig  `json:"quality"`
}

// ReadConfig defines file read gate rules
type ReadConfig struct {
	Enabled           bool     `json:"enabled"`
	BlockedPaths      []string `json:"blocked_paths"`
	BlockedExtensions []string `json:"blocked_extensions"`
	WarnExtensions    []string `json:"warn_extensions"`
	WarnPatterns      []string `json:"warn_patterns"`
}

// BashConfig defines bash command gate rules
type BashConfig struct {
	Enabled         bool     `json:"enabled"`
	BlockedCommands []string `json:"blocked_commands"`
	BlockedPatterns []string `json:"blocked_patterns"`
	WarnCommands    []string `json:"warn_commands"`
}

// WriteConfig defines file write gate rules
type WriteConfig struct {
	Enabled        bool     `json:"enabled"`
	BlockedPaths   []string `json:"blocked_paths"`
	ProtectedFiles []string `json:"protected_files"`
	SecretPatterns []string `json:"secret_patterns"`
}

// EnforcerConfig defines enforcer gate chain
type EnforcerConfig struct {
	Enabled  bool     `json:"enabled"`
	Chain    []string `json:"chain"`
	FailFast bool     `json:"fail_fast"`
}

// IntentConfig defines intent classification rules
type IntentConfig struct {
	Enabled          bool                `json:"enabled"`
	SkillTriggers    map[string][]string `json:"skill_triggers"`
	ResearchTriggers []string            `json:"research_triggers"`
}

// ResearchConfig defines research enforcement rules
type ResearchConfig struct {
	Enabled           bool     `json:"enabled"`
	RequireBeforeCode bool     `json:"require_before_code"`
	CodeTools         []string `json:"code_tools"`
	ResearchTools     []string `json:"research_tools"`
	BypassPatterns    []string `json:"bypass_patterns"`
}

// ContextConfig defines context tracking rules
type ContextConfig struct {
	Enabled       bool `json:"enabled"`
	TrackHotPaths bool `json:"track_hot_paths"`
	MaxHotFiles   int  `json:"max_hot_files"`
	PersistToSTM  bool `json:"persist_to_stm"`
}

// QualityConfig defines quality gate rules
type QualityConfig struct {
	Enabled       bool   `json:"enabled"`
	Comment       string `json:"comment,omitempty"`
	CheckSyntax   bool   `json:"check_syntax"`
	CheckImports  bool   `json:"check_imports"`
	MaxFileSizeKB int    `json:"max_file_size_kb"`
}

var (
	gatesConfig     *GatesConfig
	gatesConfigOnce sync.Once
	gatesConfigMu   sync.RWMutex
	gatesConfigTime time.Time
)

// GatesConfigPath returns the path to gates config.json
func GatesConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "gates", "config.json")
}

// LoadGatesConfig loads gates configuration from ~/.claude/gates/config.json
// Uses sync.Once for first load, then TTL-based cache invalidation.
func LoadGatesConfig() *GatesConfig {
	gatesConfigMu.RLock()
	if gatesConfig != nil && time.Since(gatesConfigTime) < CacheTTL {
		gatesConfigMu.RUnlock()
		return gatesConfig
	}
	gatesConfigMu.RUnlock()

	gatesConfigMu.Lock()
	defer gatesConfigMu.Unlock()

	// Double-check after acquiring write lock
	if gatesConfig != nil && time.Since(gatesConfigTime) < CacheTTL {
		return gatesConfig
	}

	cfg := loadGatesConfigFromFile()
	gatesConfig = cfg
	gatesConfigTime = time.Now()
	return cfg
}

func loadGatesConfigFromFile() *GatesConfig {
	cfg := &GatesConfig{}

	data, err := os.ReadFile(GatesConfigPath())
	if err != nil {
		// Return defaults if file not found
		return getDefaultGatesConfig()
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		// Return defaults if parse error
		return getDefaultGatesConfig()
	}

	// Merge with defaults for any missing fields
	mergeGatesDefaults(cfg)
	return cfg
}

// getDefaultGatesConfig returns built-in security defaults
func getDefaultGatesConfig() *GatesConfig {
	return &GatesConfig{
		Schema:      "kavach-gates/1.0",
		Description: "Default kavach gates config",
		Read: ReadConfig{
			Enabled: true,
			BlockedPaths: []string{
				"/etc/shadow", "/etc/passwd", "/.ssh/id_rsa",
				"/.ssh/id_ed25519", "/.aws/credentials",
				"/.gnupg/", "/.bitcoin/wallet.dat",
			},
			BlockedExtensions: []string{".pem", ".key", ".p12", ".pfx"},
			WarnExtensions:    []string{".env", ".secret"},
			WarnPatterns:      []string{"credentials", "password", "token"},
		},
		Bash: BashConfig{
			Enabled: true,
			BlockedCommands: []string{
				"rm -rf /", "rm -rf /*", "> /dev/sda",
				":(){ :|:& };:", "curl | bash", "wget | sh",
			},
			WarnCommands: []string{"sudo", "rm -rf", "chmod 777"},
		},
		Write: WriteConfig{
			Enabled: true,
			BlockedPaths: []string{
				"/etc/", "/usr/", "/bin/", "/.ssh/", "/.aws/",
			},
			ProtectedFiles: []string{".gitignore", ".env", "Cargo.lock"},
		},
		Enforcer: EnforcerConfig{
			Enabled:  true,
			Chain:    []string{"read", "bash", "write"},
			FailFast: true,
		},
		Intent: IntentConfig{
			Enabled: true,
			SkillTriggers: map[string][]string{
				"implement": {"rust", "backend"},
				"debug":     {"debug-like-expert"},
				"security":  {"security"},
			},
		},
		Research: ResearchConfig{
			Enabled:           true,
			RequireBeforeCode: true,
			CodeTools:         []string{"Write", "Edit"},
			ResearchTools:     []string{"WebSearch", "WebFetch"},
		},
		Context: ContextConfig{
			Enabled:       true,
			TrackHotPaths: true,
			MaxHotFiles:   10,
		},
		Quality: QualityConfig{
			Enabled: false,
		},
	}
}

// mergeGatesDefaults fills in missing fields with defaults
func mergeGatesDefaults(cfg *GatesConfig) {
	defaults := getDefaultGatesConfig()

	if len(cfg.Read.BlockedPaths) == 0 {
		cfg.Read.BlockedPaths = defaults.Read.BlockedPaths
	}
	if len(cfg.Bash.BlockedCommands) == 0 {
		cfg.Bash.BlockedCommands = defaults.Bash.BlockedCommands
	}
	if len(cfg.Write.BlockedPaths) == 0 {
		cfg.Write.BlockedPaths = defaults.Write.BlockedPaths
	}
}

// ReloadGatesConfig forces reload of gates config
func ReloadGatesConfig() *GatesConfig {
	gatesConfigMu.Lock()
	gatesConfig = nil
	gatesConfigTime = time.Time{}
	gatesConfigMu.Unlock()
	return LoadGatesConfig()
}

// Helper functions for gate checks

// IsBlockedPath checks if path matches any blocked path pattern
func IsBlockedPath(path string) bool {
	cfg := LoadGatesConfig()
	if !cfg.Read.Enabled {
		return false
	}

	pathLower := strings.ToLower(path)
	for _, blocked := range cfg.Read.BlockedPaths {
		if strings.Contains(pathLower, strings.ToLower(blocked)) {
			return true
		}
	}
	return false
}

// IsBlockedExtension checks if path has a blocked extension
func IsBlockedExtension(path string) bool {
	cfg := LoadGatesConfig()
	if !cfg.Read.Enabled {
		return false
	}

	pathLower := strings.ToLower(path)
	for _, ext := range cfg.Read.BlockedExtensions {
		if strings.HasSuffix(pathLower, strings.ToLower(ext)) {
			return true
		}
	}
	return false
}

// IsWarnPath checks if path should trigger a warning
func IsWarnPath(path string) bool {
	cfg := LoadGatesConfig()
	pathLower := strings.ToLower(path)

	for _, ext := range cfg.Read.WarnExtensions {
		if strings.HasSuffix(pathLower, strings.ToLower(ext)) {
			return true
		}
	}

	for _, pattern := range cfg.Read.WarnPatterns {
		if strings.Contains(pathLower, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// IsBlockedCommand checks if command matches any blocked pattern
func IsBlockedCommand(cmd string) bool {
	cfg := LoadGatesConfig()
	if !cfg.Bash.Enabled {
		return false
	}

	cmdLower := strings.ToLower(cmd)
	for _, blocked := range cfg.Bash.BlockedCommands {
		if strings.Contains(cmdLower, strings.ToLower(blocked)) {
			return true
		}
	}
	return false
}

// IsBlockedWritePath checks if write path is blocked
func IsBlockedWritePath(path string) bool {
	cfg := LoadGatesConfig()
	if !cfg.Write.Enabled {
		return false
	}

	for _, blocked := range cfg.Write.BlockedPaths {
		if strings.HasPrefix(path, blocked) {
			return true
		}
	}
	return false
}

// GetSkillsForIntent returns skills matching an intent keyword
func GetSkillsForIntent(prompt string) []string {
	cfg := LoadGatesConfig()
	if !cfg.Intent.Enabled {
		return nil
	}

	promptLower := strings.ToLower(prompt)
	var skills []string

	for trigger, triggerSkills := range cfg.Intent.SkillTriggers {
		if strings.Contains(promptLower, trigger) {
			skills = append(skills, triggerSkills...)
		}
	}

	return skills
}

// RequiresResearch checks if prompt requires research before code
func RequiresResearch(prompt string) bool {
	cfg := LoadGatesConfig()
	if !cfg.Research.Enabled || !cfg.Research.RequireBeforeCode {
		return false
	}

	promptLower := strings.ToLower(prompt)

	// Check bypass patterns
	for _, bypass := range cfg.Research.BypassPatterns {
		if strings.Contains(promptLower, bypass) {
			return false
		}
	}

	// Check research triggers
	for _, trigger := range cfg.Intent.SkillTriggers {
		for _, skill := range trigger {
			if strings.Contains(promptLower, skill) {
				return true
			}
		}
	}

	return false
}
