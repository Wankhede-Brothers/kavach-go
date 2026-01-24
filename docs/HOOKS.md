# Hooks Reference

Complete reference for Claude Code hook integration with kavach.

## Hook Events

| Event | Trigger | Kavach Command |
|-------|---------|----------------|
| `SessionStart` | Session begins | `kavach session init` |
| `UserPromptSubmit` | User sends prompt | `kavach session init` |
| `PreToolUse` | Before tool execution | `kavach gates <gate> --hook` |
| `PostToolUse` | After tool execution | Memory write (with events) |
| `Stop` | Session ends | `kavach session end` |
| `PreCompact` | Before context compaction | `kavach session compact` |

---

## Gate Pipeline

```
TOOL REQUEST (stdin JSON)
      │
      ▼
┌─────────────────────────────────────────────────────────────────┐
│                       ENFORCER GATE                              │
│                                                                  │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐    │
│  │  RESEARCH GATE │  │  PATTERN CHECK │  │ FORBIDDEN CHECK│    │
│  │  (pkg/agentic) │  │  (pkg/patterns)│  │ (phrases)      │    │
│  │                │  │                │  │                │    │
│  │ • Framework    │  │ • IsCodeFile() │  │ • No guessing  │    │
│  │   detection    │  │ • IsBlocked()  │  │ • No assuming  │    │
│  │ • Research     │  │ • IsSensitive()│  │ • Verify first │    │
│  │   required?    │  │ • ValidAgent() │  │                │    │
│  └────────────────┘  └────────────────┘  └────────────────┘    │
│                                                                  │
│  Decision: APPROVE (silent) | BLOCK (with reason)               │
└─────────────────────────────────────────────────────────────────┘
      │
      ├── Task Tool ──► CEO Gate (validate agent hierarchy)
      │
      ├── TaskCreate/Update/Get/List/Output ──► Task Gate (2.1.19+)
      │
      ├── Bash Tool ──► Bash Gate (command sanitization)
      │
      ├── Read Tool ──► Read Gate (sensitive file blocking)
      │
      └── Write/Edit ──► Enforcer Gate (TABULA_RASA check)
```

---

## Output Schemas

### PreToolUse Response

**Approve (silent):**
```json
{
  "decision": "approve",
  "reason": "Gate passed"
}
```

**Block (with reason):**
```json
{
  "decision": "block",
  "reason": "TABULA_RASA: WebSearch required before code"
}
```

**Modify (inject context):**
```json
{
  "decision": "modify",
  "reason": "Context injected",
  "additionalContext": "[CONTEXT]\ndate: 2026-01-16\nresearch: done\n"
}
```

### SessionStart/UserPromptSubmit Response

```json
{
  "hookEventName": "SessionStart",
  "additionalContext": "[SESSION]\nid: sess_abc123\ndate: 2026-01-16\nproject: my-project\n"
}
```

---

## Hook Commands

### SessionStart
```bash
kavach session init
```

**Output (TOON format):**
```toon
[SESSION]
id: sess_abc123
date: 2026-01-16
project: my-project
cutoff: 2025-01

[ENFORCE]
TABULA_RASA: WebSearch BEFORE code
NO_AMNESIA: kavach memory bank
DATE_INJECTION: 2026-01-16
```

### PreToolUse:Task
```bash
echo '{"tool_name":"Task","tool_input":{"subagent_type":"backend-engineer"}}' | kavach gates ceo --hook
```

### PreToolUse:TaskCreate (2.1.19+)
```bash
echo '{"tool_name":"TaskCreate","tool_input":{"subject":"Build API","description":"Implement REST endpoints"}}' | kavach gates task --hook
```

### PreToolUse:TaskUpdate (2.1.19+)
```bash
echo '{"tool_name":"TaskUpdate","tool_input":{"taskId":"abc123","status":"completed"}}' | kavach gates task --hook
```

### PreToolUse:Bash
```bash
echo '{"tool_name":"Bash","tool_input":{"command":"ls -la"}}' | kavach gates bash --hook
```

### PreToolUse:Read
```bash
echo '{"tool_name":"Read","tool_input":{"file_path":"/etc/passwd"}}' | kavach gates read --hook
```

### PreToolUse:Write/Edit
```bash
echo '{"tool_name":"Write","tool_input":{"file_path":"main.go"}}' | kavach gates enforcer --hook
```

### Stop
```bash
kavach session end
```

### PreCompact
```bash
kavach session compact
```

---

## settings.json Configuration

### Full Configuration (Claude Code 2.1.19+)

Location: `~/.claude/settings.json`

```json
{
  "env": {
    "CLAUDE_CODE_ENABLE_TASKS": "1",
    "CLAUDE_CODE_TASK_LIST_ID": "your-project"
  },
  "hooks": {
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [{"type": "command", "command": "kavach session init"}]
      }
    ],
    "UserPromptSubmit": [
      {
        "matcher": "",
        "hooks": [{"type": "command", "command": "kavach gates intent --hook"}]
      }
    ],
    "PreToolUse": [
      {"matcher": "Task", "hooks": [{"type": "command", "command": "kavach gates ceo --hook"}]},
      {"matcher": "TaskCreate", "hooks": [{"type": "command", "command": "kavach gates task --hook"}]},
      {"matcher": "TaskUpdate", "hooks": [{"type": "command", "command": "kavach gates task --hook"}]},
      {"matcher": "TaskGet", "hooks": [{"type": "command", "command": "kavach gates task --hook"}]},
      {"matcher": "TaskList", "hooks": [{"type": "command", "command": "kavach gates task --hook"}]},
      {"matcher": "TaskOutput", "hooks": [{"type": "command", "command": "kavach gates task --hook"}]},
      {"matcher": "Bash", "hooks": [{"type": "command", "command": "kavach gates bash --hook"}]},
      {"matcher": "Read", "hooks": [{"type": "command", "command": "kavach gates read --hook"}]},
      {"matcher": "Write", "hooks": [{"type": "command", "command": "kavach gates enforcer --hook"}]},
      {"matcher": "Edit", "hooks": [{"type": "command", "command": "kavach gates enforcer --hook"}]}
    ],
    "PostToolUse": [
      {"matcher": "TaskCreate", "hooks": [{"type": "command", "command": "kavach memory sync --hook"}]},
      {"matcher": "TaskUpdate", "hooks": [{"type": "command", "command": "kavach memory sync --hook"}]},
      {"matcher": "TaskOutput", "hooks": [{"type": "command", "command": "kavach gates task --hook"}]}
    ],
    "Stop": [
      {"matcher": "", "hooks": [{"type": "command", "command": "kavach session end"}]}
    ],
    "PreCompact": [
      {"matcher": "", "hooks": [{"type": "command", "command": "kavach session compact"}]}
    ]
  }
}
```

### Minimal Configuration

```json
{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [{"type": "command", "command": "kavach session init"}]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "Write",
        "hooks": [{"type": "command", "command": "kavach gates enforcer --hook"}]
      },
      {
        "matcher": "Edit",
        "hooks": [{"type": "command", "command": "kavach gates enforcer --hook"}]
      }
    ]
  }
}
```

---

## Gate Reference

| Gate | Matcher | Purpose | Package |
|------|---------|---------|---------|
| `enforcer` | Write, Edit | Full pipeline (TABULA_RASA + research) | `pkg/agentic` |
| `ceo` | Task | Validate agent hierarchy | `pkg/patterns` |
| `task` | TaskCreate, TaskUpdate, TaskGet, TaskList, TaskOutput | Task management (2.1.19+) | `gates/task` |
| `bash` | Bash | Command sanitization | `pkg/enforce` |
| `read` | Read | Sensitive file blocking | `pkg/patterns` |

### Enforcer Gate Checks

1. **TABULA_RASA**: Requires WebSearch before code generation
2. **Pattern Check**: Validates file patterns and blocked paths
3. **Forbidden Phrases**: Blocks assumptions without verification
4. **Agent Validation**: Ensures valid agent hierarchy

### CEO Gate Checks

1. **Agent Hierarchy**: L-1 → L0 → L1 → L2 flow
2. **Valid Agent Types**: Only registered agents allowed
3. **Model Assignment**: Opus/Sonnet/Haiku per agent level

### Bash Gate Checks

1. **Command Sanitization**: Blocks dangerous commands
2. **Path Validation**: Prevents unauthorized access
3. **Environment Safety**: Protects sensitive variables

### Read Gate Checks

1. **Sensitive Files**: Blocks `/etc/passwd`, credentials, keys
2. **Pattern Blocking**: Configurable blocked patterns
3. **Access Control**: Project-scoped file access

### Task Gate Checks (Claude Code 2.1.19+)

1. **Parameter Validation**: Required fields (subject, taskId)
2. **Status Validation**: Valid status transitions
3. **Health Tracking**: Records task creation/completion
4. **Zombie Detection**: Monitors stale in_progress tasks
5. **Multi-Session**: Coordinates via `CLAUDE_CODE_TASK_LIST_ID`

**Detected Issues:**

| Issue Type | GitHub | Detection |
|------------|--------|-----------|
| `STALE_TASK_COUNT` | #19894 | UI count vs actual mismatch |
| `ZOMBIE_TASK` | #17542 | in_progress >30min without update |
| `HEADLESS_MODE_TASK_TOOLS` | #20463 | Task tools unavailable in pipe mode |
| `SILENT_COMPLETION` | #20525 | Background task completed without notification |

**Health Check:**
```bash
kavach orch task-health
```

---

## Event Bus Integration

Hooks publish events for telemetry:

```go
// Events published by hooks
EventSessionStart  // SessionStart hook
EventMemoryWrite   // PostToolUse memory write
EventAgentInvoke   // Task tool (CEO gate)
EventSkillInvoke   // Skill invocation
```

---

## Error Handling

### Hook Errors

If a hook fails, Claude Code shows:
```
PreToolUse hook error: <message>
```

### Common Issues

| Error | Cause | Fix |
|-------|-------|-----|
| `unknown command` | Binary not found | Check `which kavach` |
| `JSON validation failed` | Wrong output schema | Verify JSON format |
| `decision: block` | Gate rejected request | Check block reason |

### Debugging

```bash
# Test hook manually
echo '{"tool_name":"Write","tool_input":{"file_path":"test.go"}}' | kavach gates enforcer --hook

# Check binary
which kavach
kavach --version

# Verify settings.json
cat ~/.claude/settings.json | jq '.hooks'

# Check Memory Bank
kavach memory bank
kavach status
```

---

## TABULA_RASA Enforcement

```toon
TABULA_RASA:ENFORCEMENT
  cutoff: 2025-01
  today: $(kavach status | grep today)
  rule: WebSearch BEFORE code
  forbidden: Assumptions without verification
```

### How It Works

1. **Write/Edit hook** triggers enforcer gate
2. **Research gate** (pkg/agentic) checks if research was done
3. **Block** if no WebSearch for new frameworks/patterns
4. **Approve** if research verified or known pattern

---

## File Locking

Memory writes use file locking to prevent concurrent access:

```go
lockMgr := lock.GetLockManager()
err := lockMgr.AcquireWithTimeout(path, 5*time.Second)
defer lockMgr.Release(path)
```

This ensures consistent Memory Bank state across parallel sessions.
