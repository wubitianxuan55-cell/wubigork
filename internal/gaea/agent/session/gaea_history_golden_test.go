package session

// 3.0 Step 1 兼容红线：GaeaHistory 输出必须逐字节不变。
// internal/app 当前被阶段 7 在途改动阻断编译（internal/office/docmd/pdf.go
// 语法错误，非本任务引入），故本包以「逐字节同构副本」复算 GaeaHistory 输出
// 并钉死 golden 字节；internal/app/gaea_history_golden_test.go 使用同一
// fixture 断言真实实现，二者产出必须一致。
// 额外断言事件日志往返：fixture 消息 → ToLogEntries → ProjectMessages →
// 同款 GaeaHistory 输出逐字节一致（事件日志投影不得改变前端所见）。

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// goldenFixtureMessages 与 internal/app/gaea_history_golden_test.go 的
// goldenFixtureSession 完全一致（同 fixture、同顺序），保证 golden 字节可互通。
func goldenFixtureMessages() []provider.Message {
	return []provider.Message{
		{Role: provider.RoleSystem, Content: "golden system prompt"},
		{Role: provider.RoleUser, Content: "请帮我调试 auth 模块"},
		{Role: provider.RoleAssistant, Content: "好的，我先读取相关文件。", ToolCalls: []provider.ToolCall{
			{ID: "call-1", Name: "read_file", Arguments: `{"path":"internal/auth.go"}`},
		}},
		{Role: provider.RoleTool, Content: "package auth\n\nfunc Login() {}", ToolCallID: "call-1", Name: "read_file"},
		{Role: provider.RoleAssistant, Content: "找到问题了：Login 缺少参数校验。"},
	}
}

// gaeaHistoryMessage 是 internal/app.HistoryMessage 的逐字节同构副本
// （字段名、json tag、声明顺序完全一致）。
type gaeaHistoryMessage struct {
	Role       string `json:"role"`
	Content    string `json:"content"`
	ToolName   string `json:"toolName,omitempty"`
	ToolArgs   string `json:"toolArgs,omitempty"`
	ToolID     string `json:"toolId,omitempty"`
	ToolOutput string `json:"toolOutput,omitempty"`
}

// toGaeaHistory 复刻 internal/app.GaeaHistory 的消息转换循环
// （gaea_ui.go:99-124）：tool 角色 → tool_result 条目；assistant 消息的
// ToolCalls 逐个展开为 tool 条目；其余按 role 原样。
func toGaeaHistory(msgs []provider.Message) []gaeaHistoryMessage {
	out := []gaeaHistoryMessage{}
	for _, m := range msgs {
		switch m.Role {
		case provider.RoleTool:
			out = append(out, gaeaHistoryMessage{
				Role:       "tool_result",
				Content:    m.Content,
				ToolName:   m.Name,
				ToolID:     m.ToolCallID,
				ToolOutput: m.Content,
			})
		default:
			out = append(out, gaeaHistoryMessage{Role: string(m.Role), Content: m.Content})
			if m.Role == provider.RoleAssistant {
				for _, tc := range m.ToolCalls {
					out = append(out, gaeaHistoryMessage{
						Role:     "tool",
						ToolName: tc.Name,
						ToolArgs: tc.Arguments,
						ToolID:   tc.ID,
					})
				}
			}
		}
	}
	return out
}

func goldenBytes(t *testing.T) []byte {
	t.Helper()
	b, err := json.Marshal(toGaeaHistory(goldenFixtureMessages()))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

// TestGaeaHistoryGoldenReplica 钉死 GaeaHistory 输出字节（golden 文件缺失时
// 自动生成基线；之后任何运行都必须逐字节一致）。
func TestGaeaHistoryGoldenReplica(t *testing.T) {
	got := goldenBytes(t)
	goldenPath := filepath.Join("testdata", "gaea_history.golden.json")
	if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("golden file generated: %s (%d bytes)", goldenPath, len(got))
		return
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("GaeaHistory replica output drifted from golden:\n got: %s\nwant: %s", got, want)
	}
}

// TestGaeaHistoryGoldenEventLogRoundTrip 验证事件日志投影与 legacy 消息流
// 产出逐字节一致的 GaeaHistory 输出（投影层不改变前端所见）。
func TestGaeaHistoryGoldenEventLogRoundTrip(t *testing.T) {
	fixture := goldenFixtureMessages()
	entries := ToLogEntries(fixture)
	projected := ProjectMessages(entries)
	if !reflect.DeepEqual(projected, fixture) {
		t.Fatalf("projection round-trip mismatch:\n got: %+v\nwant: %+v", projected, fixture)
	}
	got, err := json.Marshal(toGaeaHistory(projected))
	if err != nil {
		t.Fatal(err)
	}
	want, err := json.Marshal(toGaeaHistory(fixture))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("event-log projected GaeaHistory output drifted:\n got: %s\nwant: %s", got, want)
	}
}
