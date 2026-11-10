// Package tool — V8.9: ToolEnvelope provides a unified JSON result format
// for all tool calls. Every tool result is wrapped as {"ok":bool, "code":..., ...}
// so the model can programmatically distinguish success/failure/timeout/denied
// instead of parsing free-form error strings.
//
// Cache safety: MarshalToolJSON uses SetEscapeHTML(false) and deterministic
// map ordering to ensure identical input always produces identical output bytes.
package tool

import (
	"bytes"
	"encoding/json"
	"strings"
)

// ToolEnvelope is the JSON envelope every tool result is wrapped in.
// The model sees this as the tool_result message Content.
//
// B2 失败即值（dsh-ops 蒸馏 2026-11-10）：失败信封追加 Retryable/
// EffectState/NextAction 三字段（omitempty，纯加法——旧消费方零感知），
// 由 tool.Failure 结构化错误经 WrapFailure 渲染。
type ToolEnvelope struct {
	OK      bool   `json:"ok"`
	Success bool   `json:"success"`
	Code    string `json:"code,omitempty"`    // machine-readable: "ok", "timeout", "denied", "not_found", "exec_error", "validation_error"
	Subcode string `json:"subcode,omitempty"` // 细分码（D3）：同 code 下的具体阶段/成因（如 SANDBOX_UNAVAILABLE/START_FAILED/SOURCE_REMOVE_FAILED），直接回答「重试有没有用」
	Error   string `json:"error,omitempty"`   // human-readable short error (for the model)
	// 失败即值三字段（仅失败信封携带；刻意排在 Data 前——大 detail 触发
	// backstop 截断时头部优先保住结构字段，D1 2026-11-10）：
	Retryable   bool   `json:"retryable,omitempty"`    // 原样重试是否可能成功
	EffectState string `json:"effect_state,omitempty"` // none / unknown（unknown=先核实再重试）
	NextAction  string `json:"next_action,omitempty"`  // 下一步短句（一句以内，token 纪律）
	Message     string `json:"message,omitempty"`      // friendly summary (for the model)
	Data        any    `json:"data,omitempty"`         // structured payload when applicable
}

// Pre-defined outcome codes.
const (
	CodeOK              = "ok"
	CodeTimeout         = "timeout"
	CodeDenied          = "denied"
	CodeNotFound        = "not_found"
	CodeExecError       = "exec_error"
	CodeValidationError = "validation_error"
	CodeUnknownTool     = "unknown_tool"
	CodeBlocked         = "blocked"
	// CodeConflict revision CAS 冲突（B2：原为 REVISION_CONFLICT 文本前缀，
	// 结构化后进信封码；前缀文本保留为兼容层）。
	CodeConflict = "conflict"
)

// WrapResult envelopes a successful tool result.
func WrapResult(code string, data any) string {
	if code == "" {
		code = CodeOK
	}
	return mustMarshal(ToolEnvelope{
		OK:      true,
		Success: true,
		Code:    code,
		Data:    data,
	})
}

// WrapError envelopes a tool error with a machine-readable code.
func WrapError(code, errMsg string, data any) string {
	if code == "" {
		code = CodeExecError
	}
	return mustMarshal(ToolEnvelope{
		OK:      false,
		Success: false,
		Code:    code,
		Error:   strings.TrimSpace(errMsg),
		Data:    data,
	})
}

// WrapText wraps free-form text (legacy tool output) in a success envelope.
// Used when a tool's output is informational rather than structured.
func WrapText(text string) string {
	return mustMarshal(ToolEnvelope{
		OK:      true,
		Success: true,
		Code:    CodeOK,
		Message: strings.TrimSpace(text),
	})
}

// ParseEnvelope attempts to parse a tool result as a ToolEnvelope.
// Returns (envelope, true) on success, (zero, false) on any failure.
func ParseEnvelope(raw string) (ToolEnvelope, bool) {
	var env ToolEnvelope
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &env); err != nil {
		return ToolEnvelope{}, false
	}
	return env, true
}

// MarshalEnvelope 序列化信封（agent 收口点复用 mustMarshal 的确定性口径：
// SetEscapeHTML(false) + TrimSpace）。
func MarshalEnvelope(env ToolEnvelope) string {
	return mustMarshal(env)
}

// mustMarshal serialises without HTML escaping (SetEscapeHTML(false)) so that
// payload fragments containing & < > survive byte-for-byte. Panics on marshal
// failure — which never happens for ToolEnvelope (all fields are basic types).
func mustMarshal(env ToolEnvelope) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(env); err != nil {
		panic("tool: marshal ToolEnvelope: " + err.Error())
	}
	return strings.TrimSpace(buf.String())
}
