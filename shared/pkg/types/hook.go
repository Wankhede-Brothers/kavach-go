// Package types provides consolidated type definitions for the umbrella CLI.
// These types eliminate duplication across 10+ servers.
package types

// HookInput represents JSON input passed to any hook.
// Reference: https://code.claude.com/docs/en/hooks
type HookInput struct {
	// Common fields - ALL hooks receive these
	SessionID      string `json:"session_id,omitempty"`
	TranscriptPath string `json:"transcript_path,omitempty"`
	Cwd            string `json:"cwd,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"` // "default","plan","acceptEdits","dontAsk","bypassPermissions"
	HookEventName  string `json:"hook_event_name,omitempty"`

	// PreToolUse / PostToolUse / PermissionRequest
	ToolName  string                 `json:"tool_name,omitempty"`
	ToolInput map[string]interface{} `json:"tool_input,omitempty"`
	ToolUseID string                 `json:"tool_use_id,omitempty"`

	// PostToolUse
	ToolResponse map[string]interface{} `json:"tool_response,omitempty"`

	// UserPromptSubmit
	Prompt string `json:"prompt,omitempty"`

	// Stop / SubagentStop
	StopHookActive bool `json:"stop_hook_active,omitempty"`

	// SubagentStart / SubagentStop
	AgentID             string `json:"agent_id,omitempty"`
	AgentType           string `json:"agent_type,omitempty"`
	AgentTranscriptPath string `json:"agent_transcript_path,omitempty"`

	// SessionStart
	Source string `json:"source,omitempty"` // "startup","resume","clear","compact"
	Model  string `json:"model,omitempty"`

	// SessionEnd
	Reason string `json:"reason,omitempty"` // "clear","logout","prompt_input_exit","exit","other"

	// PreCompact
	Trigger            string `json:"trigger,omitempty"` // "manual","auto"
	CustomInstructions string `json:"custom_instructions,omitempty"`

	// Notification
	Message          string `json:"message,omitempty"`
	NotificationType string `json:"notification_type,omitempty"` // "permission_prompt","idle_prompt","auth_success","elicitation_dialog"
}

// GetToolName returns the tool name.
func (h *HookInput) GetToolName() string {
	return h.ToolName
}

// GetToolInput returns the tool input map.
func (h *HookInput) GetToolInput() map[string]interface{} {
	return h.ToolInput
}

// GetPrompt returns the prompt (for UserPromptSubmit hooks).
func (h *HookInput) GetPrompt() string {
	return h.Prompt
}

// GetString extracts a string value from tool input by key.
// Also checks Prompt field for UserPromptSubmit hooks.
func (h *HookInput) GetString(key string) string {
	if key == "prompt" && h.Prompt != "" {
		return h.Prompt
	}
	if h.ToolInput == nil {
		return ""
	}
	if val, ok := h.ToolInput[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// IsEvent checks if this hook input is for a specific event.
func (h *HookInput) IsEvent(event string) bool {
	return h.HookEventName == event
}

// IsSubagentEvent returns true for SubagentStart/SubagentStop events.
func (h *HookInput) IsSubagentEvent() bool {
	return h.HookEventName == "SubagentStart" || h.HookEventName == "SubagentStop"
}

// GetBool extracts a boolean value from tool input by key.
func (h *HookInput) GetBool(key string) bool {
	if h.ToolInput == nil {
		return false
	}
	if val, ok := h.ToolInput[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// HookResponse represents the hook's decision output.
// Updated for Claude Code 2026 hook format with hookSpecificOutput.
type HookResponse struct {
	// Legacy fields (still supported)
	Decision          string                 `json:"decision,omitempty"`
	Reason            string                 `json:"reason,omitempty"`
	AdditionalContext string                 `json:"additionalContext,omitempty"`
	ToolInput         map[string]interface{} `json:"tool_input,omitempty"`

	// New Claude Code 2026 format
	HookSpecificOutput *HookSpecificOutput `json:"hookSpecificOutput,omitempty"`

	// Control fields
	Continue       *bool  `json:"continue,omitempty"`
	StopReason     string `json:"stopReason,omitempty"`
	SuppressOutput bool   `json:"suppressOutput,omitempty"`
	SystemMessage  string `json:"systemMessage,omitempty"`
}

// HookSpecificOutput provides structured output per hook event type.
// Reference: https://code.claude.com/docs/en/hooks
type HookSpecificOutput struct {
	HookEventName            string                 `json:"hookEventName"`
	PermissionDecision       string                 `json:"permissionDecision,omitempty"`       // "allow", "deny", "ask"
	PermissionDecisionReason string                 `json:"permissionDecisionReason,omitempty"` // Shown to Claude on deny
	UpdatedInput             map[string]interface{} `json:"updatedInput,omitempty"`             // Modify tool input
	AdditionalContext        string                 `json:"additionalContext,omitempty"`        // Context for Claude
}

// NewApprove creates an approve response.
func NewApprove(reason string) *HookResponse {
	return &HookResponse{Decision: "approve", Reason: reason}
}

// NewBlock creates a block response.
func NewBlock(reason string) *HookResponse {
	return &HookResponse{Decision: "block", Reason: reason}
}

// NewModify creates an approve response with additional context injection.
func NewModify(reason, context string) *HookResponse {
	return &HookResponse{
		Decision:          "approve",
		Reason:            reason,
		AdditionalContext: context,
	}
}

// NewModifyInput creates a response that modifies tool input (PreToolUse only).
// P2 FIX: Supports Claude Code v2.0.10+ input modification.
func NewModifyInput(reason string, modifiedInput map[string]interface{}) *HookResponse {
	return &HookResponse{
		Decision:  "approve",
		Reason:    reason,
		ToolInput: modifiedInput,
	}
}

// === Claude Code 2026 Format Helpers ===

// NewPreToolUseAllow creates an allow decision for PreToolUse hooks.
func NewPreToolUseAllow(reason string) *HookResponse {
	return &HookResponse{
		HookSpecificOutput: &HookSpecificOutput{
			HookEventName:            "PreToolUse",
			PermissionDecision:       "allow",
			PermissionDecisionReason: reason,
		},
	}
}

// NewPreToolUseDeny creates a deny decision for PreToolUse hooks.
func NewPreToolUseDeny(reason string) *HookResponse {
	return &HookResponse{
		HookSpecificOutput: &HookSpecificOutput{
			HookEventName:            "PreToolUse",
			PermissionDecision:       "deny",
			PermissionDecisionReason: reason,
		},
	}
}

// NewPreToolUseAsk creates an ask decision for PreToolUse hooks.
func NewPreToolUseAsk(reason string) *HookResponse {
	return &HookResponse{
		HookSpecificOutput: &HookSpecificOutput{
			HookEventName:            "PreToolUse",
			PermissionDecision:       "ask",
			PermissionDecisionReason: reason,
		},
	}
}

// NewPreToolUseWithContext creates an allow with additional context.
func NewPreToolUseWithContext(reason, context string) *HookResponse {
	return &HookResponse{
		HookSpecificOutput: &HookSpecificOutput{
			HookEventName:            "PreToolUse",
			PermissionDecision:       "allow",
			PermissionDecisionReason: reason,
			AdditionalContext:        context,
		},
	}
}

// NewPreToolUseModifyInput creates an allow with modified input.
func NewPreToolUseModifyInput(reason string, updatedInput map[string]interface{}) *HookResponse {
	return &HookResponse{
		HookSpecificOutput: &HookSpecificOutput{
			HookEventName:            "PreToolUse",
			PermissionDecision:       "allow",
			PermissionDecisionReason: reason,
			UpdatedInput:             updatedInput,
		},
	}
}

// NewPostToolUseBlock creates a block for PostToolUse (tool already ran).
func NewPostToolUseBlock(reason, context string) *HookResponse {
	return &HookResponse{
		Decision: "block",
		Reason:   reason,
		HookSpecificOutput: &HookSpecificOutput{
			HookEventName:     "PostToolUse",
			AdditionalContext: context,
		},
	}
}

// NewUserPromptSubmitContext creates a UserPromptSubmit response with context.
func NewUserPromptSubmitContext(context string) *HookResponse {
	return &HookResponse{
		HookSpecificOutput: &HookSpecificOutput{
			HookEventName:     "UserPromptSubmit",
			AdditionalContext: context,
		},
	}
}

// NewUserPromptSubmitBlock blocks a user prompt.
func NewUserPromptSubmitBlock(reason string) *HookResponse {
	return &HookResponse{
		Decision: "block",
		Reason:   reason,
		HookSpecificOutput: &HookSpecificOutput{
			HookEventName: "UserPromptSubmit",
		},
	}
}

// NewStopBlock prevents Claude from stopping.
func NewStopBlock(reason string) *HookResponse {
	return &HookResponse{
		Decision: "block",
		Reason:   reason,
	}
}

// === PermissionRequest Helpers ===

// NewPermissionAllow auto-approves a permission request.
func NewPermissionAllow(reason string) *HookResponse {
	return &HookResponse{
		HookSpecificOutput: &HookSpecificOutput{
			HookEventName:            "PermissionRequest",
			PermissionDecision:       "allow",
			PermissionDecisionReason: reason,
		},
	}
}

// NewPermissionDeny auto-denies a permission request.
func NewPermissionDeny(reason string, interrupt bool) *HookResponse {
	return &HookResponse{
		HookSpecificOutput: &HookSpecificOutput{
			HookEventName:            "PermissionRequest",
			PermissionDecision:       "deny",
			PermissionDecisionReason: reason,
		},
	}
}

// NewPermissionAllowWithInput auto-approves with modified input.
func NewPermissionAllowWithInput(reason string, updatedInput map[string]interface{}) *HookResponse {
	return &HookResponse{
		HookSpecificOutput: &HookSpecificOutput{
			HookEventName:            "PermissionRequest",
			PermissionDecision:       "allow",
			PermissionDecisionReason: reason,
			UpdatedInput:             updatedInput,
		},
	}
}

// === SessionEnd / SubagentStop Helpers ===

// NewSessionEndContext creates a SessionEnd response with cleanup context.
func NewSessionEndContext(context string) *HookResponse {
	return &HookResponse{
		HookSpecificOutput: &HookSpecificOutput{
			HookEventName:    "SessionEnd",
			AdditionalContext: context,
		},
	}
}

// NewSubagentStartContext creates a SubagentStart response with context.
func NewSubagentStartContext(context string) *HookResponse {
	return &HookResponse{
		HookSpecificOutput: &HookSpecificOutput{
			HookEventName:    "SubagentStart",
			AdditionalContext: context,
		},
	}
}

// NewSubagentStopContext creates a SubagentStop response with context.
func NewSubagentStopContext(context string) *HookResponse {
	return &HookResponse{
		HookSpecificOutput: &HookSpecificOutput{
			HookEventName:    "SubagentStop",
			AdditionalContext: context,
		},
	}
}

// === Setup Hook Helper ===

// NewSetupContext creates a Setup response with additional context.
func NewSetupContext(context string) *HookResponse {
	return &HookResponse{
		HookSpecificOutput: &HookSpecificOutput{
			HookEventName:    "Setup",
			AdditionalContext: context,
		},
	}
}
