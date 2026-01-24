// Package sync provides git-backed synchronization for Memory Bank.
// git_sync.go: Beads-inspired git sync with JSONL export/import.
//
// Reference: https://github.com/steveyegge/beads
// Pattern: Export to JSONL -> git commit -> git push
package sync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SyncState represents the current sync status
type SyncState struct {
	LastSync     time.Time `json:"last_sync"`
	LastCommit   string    `json:"last_commit"`
	IsDirty      bool      `json:"is_dirty"`
	Ahead        int       `json:"ahead"`
	Behind       int       `json:"behind"`
	Branch       string    `json:"branch"`
	Remote       string    `json:"remote"`
	AutoCommit   bool      `json:"auto_commit"`
	AutoPush     bool      `json:"auto_push"`
	SyncInterval int       `json:"sync_interval_seconds"`
}

// GitSync handles git-backed synchronization
type GitSync struct {
	workDir    string
	beadsDir   string // .kavach directory
	state      *SyncState
	debounceMs int
	lastExport time.Time
}

// NewGitSync creates a new git sync manager
func NewGitSync(workDir string) *GitSync {
	beadsDir := filepath.Join(workDir, ".kavach")
	return &GitSync{
		workDir:    workDir,
		beadsDir:   beadsDir,
		debounceMs: 30000, // 30 second debounce like Beads
		state: &SyncState{
			SyncInterval: 60,
		},
	}
}

// Init initializes the .kavach directory
func (g *GitSync) Init(stealth bool) error {
	// Create .kavach directory
	if err := os.MkdirAll(g.beadsDir, 0755); err != nil {
		return fmt.Errorf("failed to create .kavach: %w", err)
	}

	// Create subdirectories
	dirs := []string{"tasks", "memory", "cache"}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(g.beadsDir, dir), 0755); err != nil {
			return err
		}
	}

	// If stealth mode, add to .gitignore
	if stealth {
		return g.addToGitignore()
	}

	return nil
}

// addToGitignore adds .kavach to .gitignore for stealth mode
func (g *GitSync) addToGitignore() error {
	gitignorePath := filepath.Join(g.workDir, ".gitignore")

	// Read existing
	content, _ := os.ReadFile(gitignorePath)

	// Check if already present
	if strings.Contains(string(content), ".kavach") {
		return nil
	}

	// Append
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString("\n# Kavach (stealth mode)\n.kavach/\n")
	return err
}

// Export exports tasks to JSONL format
func (g *GitSync) Export(data interface{}, filename string) error {
	// Debounce check
	if time.Since(g.lastExport) < time.Duration(g.debounceMs)*time.Millisecond {
		return nil // Skip, too soon
	}

	path := filepath.Join(g.beadsDir, filename)

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}

	// Write atomically
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename: %w", err)
	}

	g.lastExport = time.Now()
	return nil
}

// Import imports tasks from JSONL format
func (g *GitSync) Import(filename string, target interface{}) error {
	path := filepath.Join(g.beadsDir, filename)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No file to import
		}
		return err
	}

	return json.Unmarshal(data, target)
}

// Status returns current git status
func (g *GitSync) Status() (*SyncState, error) {
	// Get current branch
	branch, err := g.runGit("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, err
	}
	g.state.Branch = strings.TrimSpace(branch)

	// Get remote
	remote, _ := g.runGit("remote")
	g.state.Remote = strings.TrimSpace(strings.Split(remote, "\n")[0])

	// Check if dirty
	status, _ := g.runGit("status", "--porcelain", g.beadsDir)
	g.state.IsDirty = len(strings.TrimSpace(status)) > 0

	// Get ahead/behind
	if g.state.Remote != "" {
		revList, _ := g.runGit("rev-list", "--left-right", "--count",
			fmt.Sprintf("%s/%s...HEAD", g.state.Remote, g.state.Branch))
		parts := strings.Fields(revList)
		if len(parts) == 2 {
			fmt.Sscanf(parts[0], "%d", &g.state.Behind)
			fmt.Sscanf(parts[1], "%d", &g.state.Ahead)
		}
	}

	// Get last commit
	lastCommit, _ := g.runGit("log", "-1", "--format=%H", "--", g.beadsDir)
	g.state.LastCommit = strings.TrimSpace(lastCommit)

	g.state.LastSync = time.Now()
	return g.state, nil
}

// Sync performs full sync: export -> commit -> push
func (g *GitSync) Sync(message string) error {
	// Stage .kavach changes
	if _, err := g.runGit("add", g.beadsDir); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	// Check if anything to commit
	status, _ := g.runGit("diff", "--cached", "--name-only")
	if strings.TrimSpace(status) == "" {
		return nil // Nothing to commit
	}

	// Commit
	if message == "" {
		message = fmt.Sprintf("kavach: sync at %s", time.Now().Format("2006-01-02 15:04:05"))
	}
	if _, err := g.runGit("commit", "-m", message); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}

	// Push if auto-push enabled
	if g.state.AutoPush && g.state.Remote != "" {
		if _, err := g.runGit("push"); err != nil {
			return fmt.Errorf("git push failed: %w", err)
		}
	}

	return nil
}

// Pull pulls latest changes and imports
func (g *GitSync) Pull() error {
	if g.state.Remote == "" {
		return nil
	}

	// Pull with rebase
	if _, err := g.runGit("pull", "--rebase"); err != nil {
		return fmt.Errorf("git pull failed: %w", err)
	}

	return nil
}

// runGit runs a git command and returns output
func (g *GitSync) runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = g.workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// IsGitRepo checks if workDir is a git repository
func (g *GitSync) IsGitRepo() bool {
	_, err := g.runGit("rev-parse", "--git-dir")
	return err == nil
}

// GetBeadsDir returns the .kavach directory path
func (g *GitSync) GetBeadsDir() string {
	return g.beadsDir
}
