// env_file.go: CLAUDE_ENV_FILE support for SessionStart.
// Persists session vars as environment variables for all subsequent Bash commands.
package session

import (
	"os"
	"path/filepath"

	"github.com/claude/shared/pkg/enforce"
)

// writeClaudeEnvFile writes session vars to CLAUDE_ENV_FILE if set.
// These become available as env vars in all subsequent Bash tool calls.
func writeClaudeEnvFile(session *enforce.SessionState) {
	envFile := os.Getenv("CLAUDE_ENV_FILE")
	if envFile == "" {
		return
	}

	homeDir, _ := os.UserHomeDir()
	memoryPath := filepath.Join(homeDir, ".local", "shared", "shared-ai", "memory")

	content := "KAVACH_SESSION_ID=" + session.ID + "\n"
	content += "KAVACH_PROJECT=" + session.Project + "\n"
	content += "KAVACH_MEMORY_BANK=" + memoryPath + "\n"
	content += "KAVACH_TODAY=" + session.Today + "\n"
	content += "KAVACH_RESEARCH_DONE=" + boolStr(session.ResearchDone) + "\n"

	// Append to env file (other hooks may also write to it)
	f, err := os.OpenFile(envFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(content)
}
