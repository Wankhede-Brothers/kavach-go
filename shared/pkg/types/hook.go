// Package types provides consolidated type definitions for the umbrella CLI.
// These types eliminate duplication across 10+ servers.
package types

// HookInput represents JSON input passed to any hook.
// Supports both PreToolUse (tool_name/tool_input) and UserPromptSubmit (prompt).
type HookInput struct {
	ToolName  string                 `json:"tool_name,omitempty"`
	ToolInput map[string]interface{} `json:"tool_input,omitempty"`
	Prompt    string                 `json:"prompt,omitempty"` // UserPromptSubmit
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
// P2 FIX: Added ToolInput for PreToolUse input modification (Claude Code v2.0.10+).
type HookResponse struct {
	Decision          string                 `json:"decision"`
	Reason            string                 `json:"reason,omitempty"`
	AdditionalContext string                 `json:"additionalContext,omitempty"`
	ToolInput         map[string]interface{} `json:"tool_input,omitempty"` // P2: Modified input
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
