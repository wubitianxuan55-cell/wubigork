package app

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// badSchemaTool 模拟 MCP 服务器返回非法 inputSchema 的工具。
type badSchemaTool struct{}

func (*badSchemaTool) Name() string { return "mcp__mock__bad" }
func (*badSchemaTool) Description() string {
	return "模拟 MCP 非法 schema 工具"
}
func (*badSchemaTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object"}{"bad":1}`) // 双顶层值 → 非法 JSON
}
func (*badSchemaTool) ReadOnly() bool { return true }
func (*badSchemaTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	return "ok", nil
}

// TestGaeaChatRequestMarshalWithBadToolSchema 复现 MCP 非法 schema 场景：
// 非法 inputSchema 进入 registry 后，发送消息的 ChatRequest marshal 必须成功
// （修复前报 "json: error calling MarshalJSON for type json.RawMessage:
// invalid character '}' after top-level value"）。
func TestGaeaChatRequestMarshalWithBadToolSchema(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(&badSchemaTool{})

	schs := reg.Schemas()
	if len(schs) == 0 {
		t.Fatal("registry schemas 为空")
	}
	for _, s := range schs {
		if !json.Valid(s.Parameters) {
			t.Fatalf("工具 %s 的 schema 非法: %s", s.Name, string(s.Parameters))
		}
	}

	// 模拟 bridge.toChatTools + ChatStream 的 marshal 路径
	tools := make([]ai.ChatToolSchema, 0, len(schs))
	for _, s := range schs {
		tools = append(tools, ai.ChatToolSchema{Name: s.Name, Description: s.Description, Parameters: s.Parameters})
	}
	req := ai.ChatRequest{
		Model:    "gaea",
		Messages: []ai.ChatMessage{{Role: "user", Content: "hi"}},
		Tools:    tools,
	}
	if _, err := json.Marshal(req); err != nil {
		t.Fatalf("ChatRequest marshal 失败（非法 schema 未被兜底）: %v", err)
	}
}
