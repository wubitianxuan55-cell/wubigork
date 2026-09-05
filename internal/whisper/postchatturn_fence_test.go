// Package whisper — postchatturn_fence_test.go
// 审计 #3（docs/gaea-genui-memoryfence-audit-2026-09.md）：companion_reply
// 摘要剥离 genui/dsh-ui 围栏——围栏 JSON 不进 FactStore，正文保留；无围栏
// 回复逐字节不变（回归锚）。

package whisper

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/genui"
)

func newReplyLogOrch() *Orchestrator {
	return &Orchestrator{FactStore: NewFactStore()}
}

// 围栏开头 + 正文在后：摘要应保留正文与占位，不含围栏 JSON/标记。
func TestCompanionReplyLogStripsGenuiFence(t *testing.T) {
	orch := newReplyLogOrch()
	fence := "```genui\n{\"title\":\"心情\",\"items\":[{\"type\":\"stat\",\"label\":\"今日\",\"value\":\"好\"}]}\n```"
	ids := writeCompanionReplyLog(orch, PostTurnContext{SessionID: "s1", TurnIndex: 1, AssistantText: fence + "\n今天聊得开心，明天想一起听歌。"})
	if len(ids) != 1 {
		t.Fatalf("应写入一条事实, got %d", len(ids))
	}
	sum := orch.FactStore.ListActive()[0].Summary
	if !strings.HasPrefix(sum, "gaea回复：") {
		t.Fatalf("摘要前缀应不变: %q", sum)
	}
	if strings.Contains(sum, `"items"`) || strings.Contains(sum, "```") {
		t.Fatalf("围栏 JSON 不应进入摘要: %q", sum)
	}
	if !strings.Contains(sum, "今天聊得开心") {
		t.Fatalf("围栏外正文应保留: %q", sum)
	}
	if !strings.Contains(sum, genui.UIFencePlaceholder) {
		t.Fatalf("围栏位置应有占位行: %q", sum)
	}
}

// 回复仅由围栏构成 → 剥离结果即占位行（非空），摘要为占位而非空体。
func TestCompanionReplyLogFenceOnlyYieldsPlaceholder(t *testing.T) {
	orch := newReplyLogOrch()
	writeCompanionReplyLog(orch, PostTurnContext{SessionID: "s2", TurnIndex: 2, AssistantText: "```dsh-ui\n{\"items\":[]}\n```"})
	sum := orch.FactStore.ListActive()[0].Summary
	if !strings.Contains(sum, genui.UIFencePlaceholder) {
		t.Fatalf("纯围栏回复摘要应保留占位: %q", sum)
	}
	if strings.Contains(sum, `"items"`) || strings.Contains(sum, "```") {
		t.Fatalf("围栏 JSON 不应进入摘要: %q", sum)
	}
}

// 剥离后为空白（回复只有空白）→ 保留占位，不产生空摘要体。
func TestCompanionReplyLogBlankReplyUsesPlaceholder(t *testing.T) {
	orch := newReplyLogOrch()
	writeCompanionReplyLog(orch, PostTurnContext{SessionID: "s3", TurnIndex: 3, AssistantText: "   \n\t "})
	sum := orch.FactStore.ListActive()[0].Summary
	if sum != "gaea回复："+genui.UIFencePlaceholder {
		t.Fatalf("空白回复摘要应为占位: %q", sum)
	}
}

// 无围栏回复 → 摘要与改造前逐字节相同（回归锚）。
func TestCompanionReplyLogNoFenceRegression(t *testing.T) {
	orch := newReplyLogOrch()
	text := "今晚聊了喜欢的电影，还约定下周一起爬山。"
	writeCompanionReplyLog(orch, PostTurnContext{SessionID: "s4", TurnIndex: 4, AssistantText: text})
	sum := orch.FactStore.ListActive()[0].Summary
	if sum != "gaea回复："+text {
		t.Fatalf("无围栏摘要应逐字节不变:\n got %q\nwant %q", sum, "gaea回复："+text)
	}
}
