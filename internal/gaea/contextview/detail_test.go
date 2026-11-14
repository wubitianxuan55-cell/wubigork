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
	// request_header 聚合节点可展开 system 全文；未知 seq 返回 false
	h, ok := NodeDetailFor(entries, 1)
	if !ok || h.Kind != "header" || h.Text != "sys" {
		t.Fatalf("header 详情错误: %+v", h)
	}
	if _, ok := NodeDetailFor(entries, 99); ok {
		t.Fatal("未知 seq 不应可展开")
	}
}

// TestNodeDetailForHeaderTools：system 与每个工具原始 schema 同次返回；
// token 是固定密度展示估算，缺失 schema 仍保留工具名。
func TestNodeDetailForHeaderTools(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "request_header", map[string]any{
			"system": "系统提示词全文",
			"tools": []map[string]any{
				{"name": "read_file", "schema": `{"type":"object","properties":{"path":{"type":"string"}}}`, "source": "builtin-workspace"},
				{"name": "bash"},
			},
		}),
	}
	d, ok := NodeDetailFor(entries, 1)
	if !ok || d.Kind != "header" || d.Text != "系统提示词全文" {
		t.Fatalf("header 详情错误: %+v", d)
	}
	if len(d.Tools) != 2 || d.Tools[0].Name != "read_file" || d.Tools[0].Schema == "" || d.Tools[0].Tokens <= 0 || d.Tools[0].Source != "builtin-workspace" {
		t.Fatalf("工具 schema 详情错误: %+v", d.Tools)
	}
	if d.Tools[1].Name != "bash" || d.Tools[1].Schema != "" {
		t.Fatalf("缺 schema 工具应诚实保留名称: %+v", d.Tools[1])
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

func TestExtractStoredFileRefsOnlyPersistedNonImages(t *testing.T) {
	text := `查看 @.gaea/uploads/report-a.xlsx 和
<file path=".gaea/uploads/note.pdf">正文</file>
图片 @.gaea/attachments/pic.png 继续走图片卡；
普通工作区 docs/readme.md 不是附件。`
	refs := ExtractStoredFileRefs(text)
	want := []string{".gaea/uploads/report-a.xlsx", ".gaea/uploads/note.pdf"}
	if len(refs) != len(want) || refs[0] != want[0] || refs[1] != want[1] {
		t.Fatalf("stored file refs = %v, want %v", refs, want)
	}
}
