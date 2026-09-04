package contextview

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// TestNodeDetailForToolResult：完整输出 + 向最近一条同 id dispatch 回读参数；
// 配对失败（id 不同）参数诚实留空。
func TestNodeDetailForToolResult(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "tool_dispatch", map[string]any{"id": "t1", "name": "read_file", "args": `{"path":"a.go"}`, "partial": false}),
		entry(2, "tool_dispatch", map[string]any{"id": "t2", "name": "ls", "args": `{}`, "partial": false}),
		entry(3, "tool_result", map[string]any{"id": "t1", "name": "read_file", "output": "line1\nline2\nline3"}),
		entry(4, "tool_result", map[string]any{"id": "tx", "name": "grep", "output": "no pair"}),
	}
	d, ok := NodeDetailFor(entries, 3)
	if !ok {
		t.Fatal("tool_result 节点应可展开")
	}
	if d.Kind != "tool_result" || d.Tool != "read_file" {
		t.Fatalf("kind/tool = %s/%s", d.Kind, d.Tool)
	}
	if d.Args != `{"path":"a.go"}` {
		t.Fatalf("args 回读错误: %q", d.Args)
	}
	if d.Output != "line1\nline2\nline3" || d.Lines != 3 {
		t.Fatalf("output/lines = %q/%d", d.Output, d.Lines)
	}
	// 无同 id dispatch：参数留空（诚实缺失）
	d2, ok := NodeDetailFor(entries, 4)
	if !ok || d2.Args != "" {
		t.Fatalf("无配对 dispatch 应留空: %+v", d2)
	}
}

// TestNodeDetailForErrorAndMessages：错误结果 Err 语义位保留、正文可展示；
// user/assistant 正文可展开；seq 不存在/不可展开 kind 返回 false。
func TestNodeDetailForErrorAndMessages(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "request_header", headerPayload("sys")),
		entry(2, "tool_dispatch", map[string]any{"id": "t1", "name": "bash", "args": `{}`, "partial": false}),
		entry(3, "tool_result", map[string]any{"id": "t1", "name": "bash", "err": "denied by policy"}),
		entry(4, "user_message", map[string]any{"content": "请帮我看看"}),
		entry(5, "assistant_message", map[string]any{"text": "回答正文"}),
	}
	d, ok := NodeDetailFor(entries, 3)
	if !ok || d.Err == "" || d.Output != "denied by policy" {
		t.Fatalf("错误结果详情错误: %+v", d)
	}
	u, ok := NodeDetailFor(entries, 4)
	if !ok || u.Kind != "user_message" || u.Text != "请帮我看看" {
		t.Fatalf("user 详情错误: %+v", u)
	}
	a, ok := NodeDetailFor(entries, 5)
	if !ok || a.Kind != "assistant_message" || a.Lines != 1 {
		t.Fatalf("assistant 详情错误: %+v", a)
	}
	// request_header 聚合节点不可展开；未知 seq 同样 false
	if _, ok := NodeDetailFor(entries, 1); ok {
		t.Fatal("header 节点不应可展开")
	}
	if _, ok := NodeDetailFor(entries, 99); ok {
		t.Fatal("未知 seq 不应可展开")
	}
}

// TestNodeDetailClamp：超上限正文被截断并标 Clamped（UTF-8 rune 边界安全）。
func TestNodeDetailClamp(t *testing.T) {
	big := strings.Repeat("汉", MaxDetailBytes/3+100) // 3 字节/rune，必超 1MB
	entries := []session.LogEntry{
		entry(1, "user_message", map[string]any{"content": big}),
	}
	d, ok := NodeDetailFor(entries, 1)
	if !ok {
		t.Fatal("大正文节点应可展开")
	}
	if !d.Clamped || len(d.Text) > MaxDetailBytes {
		t.Fatalf("clamp 错误: clamped=%v len=%d", d.Clamped, len(d.Text))
	}
	// 截断点必须是合法 UTF-8（反序列化不炸）
	if !utf8.ValidString(d.Text) {
		t.Fatal("截断点破坏了 UTF-8 rune 边界")
	}
}

// TestNodeDetailForImageRefs（2.5b 后半）：工具参数 JSON 的 image_path 与
// 输出文本里的图片路径被确定性提取；user 正文 @附件引用同验。
func TestNodeDetailForImageRefs(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "tool_dispatch", map[string]any{"id": "t1", "name": "vision", "args": `{"image_path":"C:\\imgs\\wx.png","prompt":"描述"}`, "partial": false}),
		entry(2, "tool_result", map[string]any{"id": "t1", "name": "vision", "output": "图中是一张报表截图"}),
		entry(3, "tool_result", map[string]any{"id": "t2", "name": "chart_gen", "output": "已生成 ![图表](out/plot.webp)"}),
		entry(4, "user_message", map[string]any{"content": "看这张 @C:/pics/a.png"}),
	}
	d, ok := NodeDetailFor(entries, 2)
	if !ok {
		t.Fatal("节点应可展开")
	}
	if len(d.ImageRefs) != 1 || d.ImageRefs[0] != `C:\imgs\wx.png` {
		t.Fatalf("参数图片引用提取错误: %v", d.ImageRefs)
	}
	d2, _ := NodeDetailFor(entries, 3)
	if len(d2.ImageRefs) != 1 || d2.ImageRefs[0] != "out/plot.webp" {
		t.Fatalf("输出图片引用提取错误: %v", d2.ImageRefs)
	}
	u, _ := NodeDetailFor(entries, 4)
	if len(u.ImageRefs) != 1 || u.ImageRefs[0] != `C:/pics/a.png` {
		t.Fatalf("user 图片引用提取错误: %v", u.ImageRefs)
	}
}
