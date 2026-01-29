package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/claude/shared/pkg/types"
)

// Output writes a hook response as JSON to stdout.
func Output(resp *types.HookResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		OutputError("failed to marshal response: " + err.Error())
		return
	}
	fmt.Println(string(data))
}

// Approve outputs an approve decision with reason.
func Approve(reason string) {
	Output(types.NewApprove(reason))
}

// Block outputs a block decision with reason.
func Block(reason string) {
	Output(types.NewBlock(reason))
}

// Modify outputs a modify decision with context injection.
func Modify(reason, context string) {
	Output(types.NewModify(reason, context))
}

// OutputError outputs a block decision for errors.
func OutputError(message string) {
	Output(&types.HookResponse{
		Decision: "block",
		Reason:   "error: " + message,
	})
}

// OutputJSON writes any value as JSON to stdout.
func OutputJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// ExitApprove outputs approve and exits with code 0.
func ExitApprove(reason string) {
	Approve(reason)
	os.Exit(0)
}

// ExitBlock outputs block and exits with code 0.
func ExitBlock(reason string) {
	Block(reason)
	os.Exit(0)
}

// ExitModify outputs modify and exits with code 0.
func ExitModify(reason, context string) {
	Modify(reason, context)
	os.Exit(0)
}

// TOON-aware functions for SP/1.0 compliance

// Today returns current date for injection.
func Today() string {
	return time.Now().Format("2006-01-02")
}

// TOONBlock creates a TOON block string.
func TOONBlock(name string, kvs map[string]string) string {
	result := "[" + name + "]\n"
	for k, v := range kvs {
		result += k + ": " + v + "\n"
	}
	return result
}

// ExitApproveTOON outputs approve with TOON context.
func ExitApproveTOON(gate string) {
	ctx := TOONBlock("GATE", map[string]string{
		"name":   gate,
		"status": "approve",
		"date":   Today(),
	})
	Modify(gate+" passed", ctx)
	os.Exit(0)
}

// ExitBlockTOON outputs block with TOON context.
// Uses hookSpecificOutput format (Claude Code 2026).
func ExitBlockTOON(gate, reason string) {
	ctx := TOONBlock("BLOCK", map[string]string{
		"gate":   gate,
		"reason": reason,
		"date":   Today(),
	})
	Output(&types.HookResponse{
		HookSpecificOutput: &types.HookSpecificOutput{
			HookEventName:            "PreToolUse",
			PermissionDecision:       "deny",
			PermissionDecisionReason: reason,
			AdditionalContext:        ctx,
		},
	})
	os.Exit(0)
}

// ExitModifyTOON outputs modify with TOON context.
func ExitModifyTOON(gate string, kvs map[string]string) {
	kvs["date"] = Today()
	ctx := TOONBlock(gate, kvs)
	Modify(gate, ctx)
	os.Exit(0)
}

// UserPromptSubmit output format for Claude Code hooks
type UserPromptSubmitResponse struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

// ExitUserPromptSubmit outputs the correct format for UserPromptSubmit hooks.
func ExitUserPromptSubmit(context string) {
	resp := &UserPromptSubmitResponse{
		HookEventName:     "UserPromptSubmit",
		AdditionalContext: context,
	}
	data, _ := json.Marshal(resp)
	fmt.Println(string(data))
	os.Exit(0)
}

// ExitUserPromptSubmitTOON outputs UserPromptSubmit with TOON context.
func ExitUserPromptSubmitTOON(gate string, kvs map[string]string) {
	kvs["date"] = Today()
	ctx := TOONBlock(gate, kvs)
	ExitUserPromptSubmit(ctx)
}

// ExitModifyTOONWithModule outputs modify with TOON context and lazy-loaded module.
// DACE: Module is only loaded when relevant tool is used.
func ExitModifyTOONWithModule(gate string, kvs map[string]string, moduleContent string) {
	kvs["date"] = Today()
	ctx := TOONBlock(gate, kvs)
	if moduleContent != "" {
		ctx += "\n[MODULE:LAZY_LOADED]\n" + moduleContent
	}
	Modify(gate, ctx)
	os.Exit(0)
}

// === SessionEnd / SubagentStart / SubagentStop output helpers ===

// ExitSessionEnd outputs SessionEnd context and exits.
func ExitSessionEnd(context string) {
	Output(types.NewSessionEndContext(context))
	os.Exit(0)
}

// ExitSessionEndTOON outputs SessionEnd with TOON context.
func ExitSessionEndTOON(kvs map[string]string) {
	kvs["date"] = Today()
	ctx := TOONBlock("SESSION_END", kvs)
	ExitSessionEnd(ctx)
}

// ExitSubagentStart outputs SubagentStart context and exits.
func ExitSubagentStart(context string) {
	Output(types.NewSubagentStartContext(context))
	os.Exit(0)
}

// ExitSubagentStop outputs SubagentStop context and exits.
func ExitSubagentStop(context string) {
	Output(types.NewSubagentStopContext(context))
	os.Exit(0)
}

// ExitPermissionAllow auto-approves a permission request.
func ExitPermissionAllow(reason string) {
	Output(types.NewPermissionAllow(reason))
	os.Exit(0)
}

// ExitPermissionDeny auto-denies a permission request.
func ExitPermissionDeny(reason string) {
	Output(types.NewPermissionDeny(reason, false))
	os.Exit(0)
}

// DACE: Zero-context functions for silent passes

// ExitSilent exits with approve and NO context injection.
// Use this when hook should pass without adding to context.
func ExitSilent() {
	Approve("ok")
	os.Exit(0)
}

// ExitUserPromptSubmitSilent outputs minimal UserPromptSubmit.
// DACE: ~5 tokens instead of ~50.
func ExitUserPromptSubmitSilent() {
	resp := &UserPromptSubmitResponse{
		HookEventName:     "UserPromptSubmit",
		AdditionalContext: "",
	}
	data, _ := json.Marshal(resp)
	fmt.Println(string(data))
	os.Exit(0)
}

// ExitUserPromptSubmitWithContext outputs UserPromptSubmit with context string.
// DACE: For directive injection (e.g., BINARY_FIRST for status queries).
func ExitUserPromptSubmitWithContext(context string) {
	resp := &UserPromptSubmitResponse{
		HookEventName:     "UserPromptSubmit",
		AdditionalContext: context,
	}
	data, _ := json.Marshal(resp)
	fmt.Println(string(data))
	os.Exit(0)
}
