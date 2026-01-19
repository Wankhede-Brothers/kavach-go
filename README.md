# Kavach

**Universal enforcement layer for AI coding assistants**

Part of the **Brahmastra Stack**: Kavach CLI + Sutra Protocol (SP/1.0) + TOON Format + DACE

---

## What is Kavach?

Kavach (Sanskrit: कवच, "armor/shield") is a Go binary that enforces best practices for AI coding assistants through hook-based gates.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              KAVACH STACK                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│  DACE          Dynamic Agentic Context Engineering                          │
│  SP/1.0        Sutra Protocol - 75-80% token reduction                     │
│  TOON          Text-Only Object Notation - 40% savings vs JSON             │
│  Memory Bank   Persistent context across sessions                          │
│  Gates         Hook-based enforcement (TABULA_RASA, sanitization)          │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Prerequisites

Kavach enforces modern Rust CLI tools over legacy commands for better performance and UX.

### Required Tools

| Tool | Replaces | Purpose | Install |
|------|----------|---------|---------|
| **bat** | `cat`, `head`, `tail` | Syntax highlighting + paging | `cargo install bat` |
| **eza** | `ls` | Icons + git status + tree view | `cargo install eza` |
| **fd** | `find` | 10x faster file search | `cargo install fd-find` |
| **rg** | `grep` | Ripgrep (fastest grep) | `cargo install ripgrep` |

### Recommended Tools

| Tool | Replaces | Purpose | Install |
|------|----------|---------|---------|
| **sd** | `sed` | Simpler regex syntax | `cargo install sd` |
| **procs** | `ps` | Colorful process tree | `cargo install procs` |
| **dust** | `du` | Visual disk usage | `cargo install du-dust` |
| **btm** | `top` | Bottom (GPU + graphs) | `cargo install bottom` |
| **delta** | `diff` | Git-aware syntax diff | `cargo install git-delta` |

### Quick Install (All Tools)

**Linux/macOS:**
```bash
# Install Rust first (if not installed)
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# Install all required + recommended tools
cargo install bat eza fd-find ripgrep sd procs du-dust bottom git-delta
```

**macOS (Homebrew):**
```bash
brew install bat eza fd ripgrep sd procs dust bottom git-delta
```

**Windows (Scoop):**
```powershell
scoop install bat eza fd ripgrep sd procs dust bottom delta
```

**Windows (Chocolatey):**
```powershell
choco install bat eza fd ripgrep sd procs dust bottom delta
```

### Legacy Command Blocking

When Kavach detects legacy commands, it blocks them with suggestions:

```
LEGACY_BLOCKED:ls:USE:eza:icons + git status + tree
LEGACY_BLOCKED:find:USE:fd:10x faster + better syntax
LEGACY_BLOCKED:grep:USE:rg:ripgrep (fastest grep)
LEGACY_BLOCKED:cat:USE:bat:syntax highlighting + paging
```

### Allowed Legacy Commands

These commands are allowed (no modern replacement needed):
```
echo, printf, cd, pwd, mkdir, rm, cp, mv, chmod, chown, touch, which, env, export, source
```

---

## Quick Install

**Linux/macOS:**
```bash
curl -fsSL https://raw.githubusercontent.com/Wankhede-Brothers/kavach-go/main/install/install.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/Wankhede-Brothers/kavach-go/main/install/install.ps1 | iex
```

**With options:**
```bash
# Install with Rust CLI tools (bat, eza, fd, rg, etc.)
curl -fsSL .../install.sh | bash -s -- --rust-cli

# Install for OpenCode
curl -fsSL .../install.sh | bash -s -- --cli opencode
```

**Build from source:**
```bash
git clone https://github.com/Wankhede-Brothers/kavach-go.git
cd kavach-go
go build -o kavach ./cmd/kavach
cp kavach ~/.local/bin/
```

---

## Setup Guide

After installation, follow these steps to configure Kavach with your AI coding assistant.

### Step 1: Verify Binary Location

The installer places the binary at:

| Platform | Binary Path | Memory Bank Path |
|----------|-------------|------------------|
| **Linux** | `~/.local/bin/kavach` | `~/.local/share/shared-ai/memory/` |
| **macOS** | `~/.local/bin/kavach` | `~/Library/Application Support/shared-ai/memory/` |
| **Windows** | `%USERPROFILE%\.local\bin\kavach.exe` | `%APPDATA%\shared-ai\memory\` |

Verify installation:
```bash
kavach status
```

### Step 2: Configure Hooks (Claude Code)

Create or update `~/.claude/settings.json`:

```json
{
  "hooks": {
    "SessionStart": [
      {"hooks": [{"type": "command", "command": "kavach session init"}]}
    ],
    "UserPromptSubmit": [
      {"hooks": [{"type": "command", "command": "kavach session init"}]}
    ],
    "PreToolUse": [
      {"matcher": "Task", "hooks": [{"type": "command", "command": "kavach gates ceo --hook"}]},
      {"matcher": "Bash", "hooks": [{"type": "command", "command": "kavach gates bash --hook"}]},
      {"matcher": "Read", "hooks": [{"type": "command", "command": "kavach gates read --hook"}]},
      {"matcher": "Write", "hooks": [{"type": "command", "command": "kavach gates enforcer --hook"}]},
      {"matcher": "Edit", "hooks": [{"type": "command", "command": "kavach gates enforcer --hook"}]}
    ],
    "Stop": [
      {"hooks": [{"type": "command", "command": "kavach session end"}]}
    ]
  }
}
```

**Minimal configuration** (TABULA_RASA enforcement only):
```json
{
  "hooks": {
    "SessionStart": [
      {"hooks": [{"type": "command", "command": "kavach session init"}]}
    ],
    "PreToolUse": [
      {"matcher": "Write", "hooks": [{"type": "command", "command": "kavach gates enforcer --hook"}]},
      {"matcher": "Edit", "hooks": [{"type": "command", "command": "kavach gates enforcer --hook"}]}
    ]
  }
}
```

### Step 3: Add CLAUDE.md (Optional)

Copy the example CLAUDE.md to your project or global config:

```bash
# Global (all projects)
cp configs/linux/CLAUDE.md ~/.claude/CLAUDE.md

# Project-specific
cp configs/linux/CLAUDE.md /path/to/your/project/CLAUDE.md
```

The CLAUDE.md file contains enforcement rules that Claude reads at session start:
- TABULA_RASA: Forces WebSearch before code
- Memory Bank integration
- Agent hierarchy rules

### Step 4: Configure OpenCode (Alternative)

If using OpenCode instead of Claude Code:

```bash
# Linux
cp configs/linux/AGENTS.md ~/.config/opencode/AGENTS.md

# macOS
cp configs/darwin/AGENTS.md ~/Library/Application\ Support/opencode/AGENTS.md

# Windows
Copy-Item configs\windows\AGENTS.md "$env:APPDATA\opencode\AGENTS.md"
```

### Step 5: Verify Setup

```bash
# Check system health
kavach status

# Test session initialization
kavach session init

# View Memory Bank
kavach memory bank

# List available agents
kavach agents

# List available skills
kavach skills
```

Expected `kavach status` output:
```
[STATUS]
binary: kavach v1.x.x
date: 2026-01-19
project: your-project
memory_bank: OK
hooks: configured
```

---

## Features

| Problem | Kavach Solution |
|---------|-----------------|
| LLMs use stale patterns (cutoff: 2025-01) | **TABULA_RASA**: WebSearch BEFORE code |
| No memory between sessions | **Memory Bank**: Project-isolated persistence |
| Dangerous command execution | **Bash Gate**: Command sanitization |
| Context pollution across projects | **Project Isolation**: Scope by active directory |
| Concurrent write conflicts | **File Locking**: Prevents race conditions |
| No telemetry/observability | **Event Bus**: Session, write, agent events |

---

## Commands

```bash
# Status & Health
kavach status                    # System health check
kavach memory bank               # Memory bank summary (project-scoped)
kavach memory bank --all         # All projects
kavach memory kanban             # Kanban board

# Agents & Skills
kavach agents                    # List all agents
kavach agents --get ceo          # Get specific agent
kavach agents --get ceo --inject # With Memory Bank context
kavach skills                    # List all skills
kavach skills --get rust         # Get specific skill

# Session
kavach session init              # Initialize session (date injection)
kavach session validate          # Validate session state
kavach session end               # Save session state

# Gates (Hook Mode)
kavach gates enforcer --hook     # Full pipeline (TABULA_RASA + research gate)
kavach gates ceo --hook          # Task orchestration validation
kavach gates bash --hook         # Command sanitization
kavach gates read --hook         # Sensitive file blocking
```

---

## Hook Configuration

**Claude Code** (`~/.claude/settings.json`):
```json
{
  "hooks": {
    "SessionStart": [
      {"hooks": [{"type": "command", "command": "kavach session init"}]}
    ],
    "UserPromptSubmit": [
      {"hooks": [{"type": "command", "command": "kavach session init"}]}
    ],
    "PreToolUse": [
      {"matcher": "Task", "hooks": [{"type": "command", "command": "kavach gates ceo --hook"}]},
      {"matcher": "Bash", "hooks": [{"type": "command", "command": "kavach gates bash --hook"}]},
      {"matcher": "Read", "hooks": [{"type": "command", "command": "kavach gates read --hook"}]},
      {"matcher": "Write", "hooks": [{"type": "command", "command": "kavach gates enforcer --hook"}]},
      {"matcher": "Edit", "hooks": [{"type": "command", "command": "kavach gates enforcer --hook"}]}
    ],
    "Stop": [
      {"hooks": [{"type": "command", "command": "kavach session end"}]}
    ]
  }
}
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           KAVACH ARCHITECTURE                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐                     │
│  │ Claude Code │    │  OpenCode   │    │  Other CLI  │                     │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘                     │
│         │                  │                  │                             │
│         └──────────────────┼──────────────────┘                             │
│                            ▼                                                │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        KAVACH BINARY                                 │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │  gates/          memory/         session/        agents/   skills/  │   │
│  │  ├─ enforcer     ├─ bank         ├─ init         ├─ list   ├─ list │   │
│  │  ├─ ceo          ├─ write        ├─ validate     ├─ get    ├─ get  │   │
│  │  ├─ bash         ├─ kanban       └─ end          └─ inject └─ inject│   │
│  │  └─ read         └─ stm                                              │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                            │                                                │
│         ┌──────────────────┼──────────────────┐                            │
│         ▼                  ▼                  ▼                            │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐                    │
│  │ Memory Bank │    │  Event Bus  │    │ File Lock   │                    │
│  │ (TOON files)│    │ (telemetry) │    │ (concurrency)│                   │
│  └─────────────┘    └─────────────┘    └─────────────┘                    │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Project Structure

```
kavach/
├── cmd/kavach/                 # Main binary
│   └── internal/commands/
│       ├── gates/              # Enforcement gates
│       │   ├── enforcer.go     # Full pipeline (TABULA_RASA)
│       │   ├── ceo.go          # Task orchestration
│       │   ├── bash.go         # Command sanitization
│       │   └── read.go         # File access control
│       ├── memory/             # Memory operations
│       │   ├── bank.go         # Project-scoped queries
│       │   ├── write.go        # With file locking + events
│       │   └── kanban.go       # 5-stage pipeline
│       ├── session/            # Session management
│       │   └── init.go         # With event publishing
│       ├── agents/             # Agent management (micro-modular)
│       │   ├── types.go        # Agent struct
│       │   ├── builtin.go      # 10 built-in agents
│       │   ├── discover.go     # Find agents
│       │   ├── inject.go       # Memory Bank injection
│       │   └── output.go       # TOON/Sutra output
│       └── skills/             # Skill management (micro-modular)
│           └── (same structure as agents)
├── shared/                     # Shared packages
│   ├── pkg/
│   │   ├── hook/               # Hook I/O
│   │   ├── toon/               # TOON parser
│   │   ├── patterns/           # Dynamic patterns
│   │   ├── enforce/            # Enforcement logic
│   │   ├── agentic/            # Research gate (TABULA_RASA)
│   │   ├── dsa/                # LRU cache, data structures
│   │   └── util/               # Path utilities, project detection
│   ├── events/                 # Event bus (telemetry)
│   └── lock/                   # File locking (concurrency)
├── examples/                   # Reference examples
│   ├── agents/                 # 6 agents
│   └── skills/                 # 15 skills with DACE
└── configs/                    # Platform configs
    ├── linux/
    ├── darwin/
    └── windows/
```

---

## Core Concepts

### TABULA_RASA
LLM training data is stale (cutoff: 2025-01). Enforces WebSearch BEFORE code.

```
TABULA_RASA:ENFORCEMENT
  cutoff: 2025-01
  today: $(kavach status | grep today)
  rule: WebSearch BEFORE code
  forbidden: assumptions without verification
```

### Project Isolation
Memory Bank queries scoped to active project, preventing context pollution.

```
PROJECT:DETECTION (Priority Order)
  1. KAVACH_PROJECT env var
  2. .git root directory name
  3. .claude/project.json marker
  4. Memory Bank project match
  5. "global" fallback
```

### DACE (Dynamic Agentic Context Engineering)
```
DACE:PRINCIPLES
  lazy_load:    Load context on-demand
  skill_first:  Use kavach binary before spawning agents
  on_demand:    Inject research only when needed
  no_hardcode:  WebSearch for current patterns
```

### Sutra Protocol (SP/1.0)
Agent communication with 75-80% token reduction:
```
[AGENTS:HIERARCHY]
L-1: nlu-intent-analyzer (haiku)  - Fast intent parsing
L0:  ceo, research-director (opus) - Decision makers
L1:  backend, frontend (sonnet)    - Engineers
L2:  aegis-guardian (opus)         - Final verification
```

### Memory Bank
Persistent storage at `~/.local/shared/shared-ai/memory/`:
```
memory/
├── decisions/    # Architecture decisions
├── research/     # Research findings (TABULA_RASA)
├── patterns/     # Code patterns
├── kanban/       # Task boards (5-stage pipeline)
├── proposals/    # Feature proposals
├── roadmaps/     # Project roadmaps
├── graph/        # Knowledge graph
├── STM/          # Short-term memory (scratchpad)
├── GOVERNANCE.toon
├── index.toon
└── volatile.toon
```

### Kanban Pipeline
5-stage production pipeline with Aegis-Guard verification:
```
Backlog → In-Progress → Testing → Verified → Done
                           ↓          ↓
                      lint/bugs    algorithm
                      warnings     dead code
                      unit tests   suppressed
```

---
## Memory Bank

Persistent memory across Claude Code sessions using TOON format with project isolation.

### Storage Location

| Platform | Path |
|----------|------|
| **Linux** | `~/.local/shared/shared-ai/memory/` |
| **macOS** | `~/Library/Application Support/shared-ai/memory/` |
| **Windows** | `%%APPDATA%%\shared-ai\memory\` |

### Directory Structure

```
~/.local/shared/shared-ai/memory/
├── GOVERNANCE.toon     # Rules and enforcement policies
├── index.toon          # Project index with metadata
├── volatile.toon       # Session state (cleared on restart)
│
├── decisions/          # Architecture decisions (ADRs)
│   ├── global/         # Shared across all projects
│   └── {project}/      # Project-specific decisions
│
├── patterns/           # Code patterns with TTL
│   └── {project}/
│
├── research/           # Research cache (TABULA_RASA)
│   └── {project}/      # Framework docs, API patterns
│
├── kanban/             # 5-stage task pipeline
│   └── {project}/      # Backlog → InProgress → Testing → Verified → Done
│       └── kanban.toon
│
├── proposals/          # Feature proposals
├── roadmaps/           # Project roadmaps
├── graph/              # Knowledge graphs
└── STM/                # Short-term memory
    └── scratchpad.toon # Session scratchpad
```

### Project Isolation

Memory Bank queries are scoped to the active project:

```
PROJECT:DETECTION (Priority Order)
1. KAVACH_PROJECT env var      # Explicit override
2. .git root directory name    # Git repository name
3. .claude/project.json        # Project marker file
4. Memory Bank project match   # Match against known projects
5. "global" fallback           # Shared context
```

### Commands

```bash
# Query Memory Bank
kavach memory bank               # Project-scoped summary
kavach memory bank --all         # All projects
kavach memory bank --category X  # Specific category

# Kanban Operations
kavach memory kanban             # View task board
kavach memory kanban --status    # Pipeline status

# Short-term Memory
kavach memory stm                # Session scratchpad
```

### TOON Format Example

```toon
# Research Cache - kavach

FACT:cloudflare-bun-version
  category: config
  verified: 2026-01-18
  ttl: 5d
  expires: 2026-01-23
  src: https://developers.cloudflare.com/pages/
  val: BUN_VERSION=1.1.33 (V2 build system)

FACT:axum-0.8-router
  category: syntax
  verified: 2026-01-18
  ttl: 7d
  src: https://docs.rs/axum/0.8/
  val: Router::new().route("/", get(handler))
```

### Fact TTL (Time-To-Live)

Research facts expire based on volatility:

| Category | TTL | Use Case |
|----------|-----|----------|
| `syntax` | 7d | Code patterns, API usage |
| `config` | 5d | Configuration, env vars |
| `behavior` | 3d | Runtime behavior |
| `migration` | 30d | Breaking changes |
| `security` | 1d | CVEs, advisories |

**Rule**: If `today > verified + ttl` → Fact is STALE → Re-research required

See [MEMORY_BANK.md](docs/MEMORY_BANK.md) for complete documentation.

---

## Examples

See [examples/](examples/) for:
- **6 Agents**: ceo, aegis-guardian, backend-engineer, frontend-engineer, nlu-intent-analyzer, research-director
- **15 Skills**: api-design, arch, cloud, debug, dsa, frontend, heal, data-processing, resume, rust, security, sql, sutra, testing, create-components

Each skill includes:
- `SKILL.md` with KAVACH:DYNAMIC block
- `references.toon` with dynamic WebSearch queries

---

## Documentation

**Website**: [https://wankhedebrothers.com/docs/kavach/](https://wankhedebrothers.com/docs/kavach/)

| Document | Description |
|----------|-------------|
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | System design and flow |
| [HOOKS.md](docs/HOOKS.md) | Hook configuration guide |
| [MEMORY_BANK.md](docs/MEMORY_BANK.md) | Memory Bank structure |
| [INSTALLATION.md](docs/INSTALLATION.md) | Installation guide |
| [API.md](docs/API.md) | Command reference |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Contribution guide |

---

## Contributing

We welcome contributions! The CI/CD pipeline handles validation autonomously.

### Quick Start

```bash
# 1. Fork and clone
git clone https://github.com/YOUR_USERNAME/kavach-go.git
cd kavach-go

# 2. Create feature branch
git checkout -b feature/my-feature

# 3. Make changes (follow DACE: max 100 lines per file)

# 4. Test locally
go build -o kavach ./cmd/kavach
go test ./...

# 5. Submit PR
git push origin feature/my-feature
```

### Autonomous CI/CD Pipeline

When you submit a PR, GitHub Actions runs automatically:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        PR SUBMITTED                                         │
│                            │                                                │
│                            ▼                                                │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │                      CI PIPELINE (ci.yml)                             │  │
│  ├──────────────────────────────────────────────────────────────────────┤  │
│  │  ✓ Build        │  go build ./cmd/kavach                             │  │
│  │  ✓ Test         │  go test ./... (cmd + shared)                      │  │
│  │  ✓ Lint         │  go vet + gofmt check                              │  │
│  │  ✓ Cross-compile│  linux/darwin/windows (amd64 + arm64)              │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│                            │                                                │
│                            ▼                                                │
│                    All checks pass? → Merge                                 │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Release Process (Maintainers)

Releases are fully automated:

```bash
# Tag a release
git tag v0.2.0
git push origin v0.2.0
```

This triggers the release pipeline:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      RELEASE PIPELINE (release.yml)                         │
├─────────────────────────────────────────────────────────────────────────────┤
│  1. Build binaries for all platforms:                                       │
│     • kavach-linux-amd64                                                    │
│     • kavach-linux-arm64                                                    │
│     • kavach-darwin-amd64 (Intel Mac)                                       │
│     • kavach-darwin-arm64 (Apple Silicon)                                   │
│     • kavach-windows-amd64.exe                                              │
│                                                                             │
│  2. Create archives (.tar.gz / .zip)                                        │
│  3. Generate SHA256SUMS.txt                                                 │
│  4. Publish GitHub Release with auto-generated notes                        │
│  5. Install scripts automatically fetch latest release                      │
└─────────────────────────────────────────────────────────────────────────────┘
```

### What to Contribute

| Type | Location | Guide |
|------|----------|-------|
| **New Gate** | `cmd/kavach/internal/commands/gates/` | [CONTRIBUTING.md](CONTRIBUTING.md#adding-a-new-gate) |
| **New Skill** | `examples/skills/` | [CONTRIBUTING.md](CONTRIBUTING.md#adding-a-new-skill) |
| **New Agent** | `examples/agents/` | Follow existing agent structure |
| **Bug Fix** | Relevant package | Include test case |
| **Documentation** | `docs/` | Keep concise |

### PR Checklist

- [ ] `go test ./...` passes
- [ ] `go fmt ./...` applied
- [ ] Max 100 lines per file (DACE)
- [ ] Tests added for new functionality
- [ ] README updated if adding new command

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

---

## License

MIT License - see [LICENSE](LICENSE)

---

**Author**: Gaurav Wankhede
**Stack**: Brahmastra
**Protocol**: SP/1.0
**Binary**: kavach
