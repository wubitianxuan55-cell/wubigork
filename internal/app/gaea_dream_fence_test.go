package app

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/genui"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// 审计 #1（docs/gaea-genui-memoryfence-audit-2026-09.md）：自动做梦输入剥离
// genui/dsh-ui 围栏——含围栏回复 → 围栏 JSON 不进提炼输入且正文保留；无围栏
// 回复 → 输入逐字节不变（回归锚）。

// 回复以 >1500 字节的大围栏开头、真结论在围栏后：旧行为下 1500 字符窗口被
// JSON 占满，结论进不了输入；剥离后 JSON 不出现、正文与占位都在。
func TestDreamInputStripsGenuiFences(t *testing.T) {
	bigJSON := `{"title":"成本看板","pad":"` + strings.Repeat("围栏JSON噪声", 300) + `"}`
	fence := "```genui\n" + bigJSON + "\n```"
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: "帮我出个看板"},
		{Role: provider.RoleAssistant, Content: fence + "\n看板结论：营收 128 万，环比 +5%。"},
	}
	in := dreamInput(msgs)
	if strings.Contains(in, "围栏JSON噪声") || strings.Contains(in, `"title"`) {
		t.Fatal("围栏 JSON 不应进入提炼输入")
	}
	if strings.Contains(in, "```") {
		t.Fatal("围栏标记不应残留在提炼输入")
	}
	if !strings.Contains(in, "看板结论：营收 128 万，环比 +5%。") {
		t.Fatal("围栏外正文应保留并进入输入")
	}
	if !strings.Contains(in, genui.UIFencePlaceholder) {
		t.Fatal("围栏位置应有占位行")
	}
}

// user 消息同样过剥离（粘贴围栏的 user 文本也不入提炼）。
func TestDreamInputStripsFencesInUserMessage(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: "参考这个：\n```dsh-ui\n{\"items\":[{\"type\":\"stat\"}]}\n```\n照它的样式做"},
		{Role: provider.RoleAssistant, Content: strings.Repeat("好的，按样式完成。", 30)},
	}
	in := dreamInput(msgs)
	if strings.Contains(in, `"items"`) || strings.Contains(in, "```") {
		t.Fatalf("user 消息里的围栏不应进入提炼输入: %q", in)
	}
	if !strings.Contains(in, "照它的样式做") {
		t.Fatal("user 围栏外正文应保留")
	}
}

// 无围栏回复 → 输入与改造前逐字节相同（回归锚：剥离不得影响普通文本）。
func TestDreamInputWithoutFencesByteExact(t *testing.T) {
	in := dreamInput([]provider.Message{
		{Role: provider.RoleUser, Content: "帮我整理这份成本测算表"},
		{Role: provider.RoleAssistant, Content: "已完成，总额 120 万。"},
	})
	want := "以下是刚结束的一轮对话，请提炼值得长期记住的信息。\n\n" +
		"【user】\n帮我整理这份成本测算表\n\n" +
		"【assistant】\n已完成，总额 120 万。\n\n"
	if in != want {
		t.Fatalf("无围栏输入应逐字节不变:\n got %q\nwant %q", in, want)
	}
}
