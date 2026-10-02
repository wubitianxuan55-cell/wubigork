package app

// board_id_single_source_test.go — 审计 AP8-08：板块 ID 清单只能有一个来源
// （manifest）。收敛前三份手写：intent_llm.go 的 LLM 兜底提示词、wx_agent.go 的
// 工具 schema 参数说明、wx_agent.go 的「打开失败」回执——都含已删除的 code
// （v4.439 删编程板块）、都缺 schedule/sin/knowledge。
//
// 本用例把「提示词/schema/回执里出现的 ID 集合」与 manifest ID 集合做双向相等
// 断言：少一个（如漏 sin）红，多一个（如手写回 code）也红。

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/app/board"
)

// boardIDSet 取 manifest 的 id 集合（唯一期望值来源）。
func boardIDSet(a *App) map[string]bool {
	out := map[string]bool{}
	for _, m := range a.GetBoardManifests() {
		out[m.ID] = true
	}
	return out
}

// extractBoardIDs 从文案里按「前缀……后缀」截出空格分隔的 id 清单。
func extractBoardIDs(t *testing.T, text, prefix, suffix string) map[string]bool {
	t.Helper()
	i := strings.Index(text, prefix)
	if i < 0 {
		t.Fatalf("文案缺少板块清单起点 %q（清单可能被删或换写法）: %q", prefix, text)
	}
	rest := text[i+len(prefix):]
	j := strings.Index(rest, suffix)
	if j < 0 {
		t.Fatalf("文案缺少板块清单终点 %q: %q", suffix, rest)
	}
	out := map[string]bool{}
	for _, f := range strings.Fields(rest[:j]) {
		out[f] = true
	}
	if len(out) == 0 {
		t.Fatalf("板块清单为空（%q…%q 之间无 id）", prefix, suffix)
	}
	return out
}

// assertSameBoardIDs 双向断言：缺的（manifest 有而文案没有）与多的（文案有而
// manifest 没有）都要点名。
func assertSameBoardIDs(t *testing.T, what string, got, want map[string]bool) {
	t.Helper()
	for id := range want {
		if !got[id] {
			t.Errorf("%s 缺板块 %q（manifest 有而文案没有）", what, id)
		}
	}
	for id := range got {
		if !want[id] {
			t.Errorf("%s 多出板块 %q（文案有而 manifest 没有：清单漂移/残留已删板块）", what, id)
		}
	}
}

// TestIntentFallbackPromptBoardIDsMatchManifest LLM 兜底提示词的板块白名单 ==
// manifest 清单（不是第二份手写清单）。
func TestIntentFallbackPromptBoardIDsMatchManifest(t *testing.T) {
	a := &App{}
	prompt := intentFallbackSystemPrompt(a.boardIDList())
	got := extractBoardIDs(t, prompt, "只能是：", " 之一")
	assertSameBoardIDs(t, "LLM 兜底提示词", got, boardIDSet(a))

	// 已删板块不得回流（用户拍板 v4.439 删编程板块）
	if got["code"] {
		t.Error("提示词出现已删除的 code 板块（v4.439 删除编程板块）")
	}
	// 三类曾漏的板块必须在场（LLM 否则永不输出，用户「打开原罪」落回聊天管道）
	for _, id := range []string{"schedule", "sin", "knowledge"} {
		if !got[id] {
			t.Errorf("提示词缺板块 %q（曾漏：LLM 永不输出该类导航）", id)
		}
	}
}

// TestWxAgentToolSchemaBoardIDsMatchManifest 微信智能体的工具参数说明与失败回执
// 同源：两者列举的 id 集合 == manifest 清单，且说明里的中文名枚举同样来自 manifest。
func TestWxAgentToolSchemaBoardIDsMatchManifest(t *testing.T) {
	a := &App{}
	want := boardIDSet(a)

	var navDesc, navParams string
	for _, tl := range a.wxAgentToolSchemas() {
		if tl.Function.Name == "navigate_board" {
			navDesc, navParams = tl.Function.Description, string(tl.Function.Parameters)
		}
	}
	if navParams == "" {
		t.Fatal("工具 schema 缺 navigate_board")
	}
	assertSameBoardIDs(t, "微信工具 schema 参数说明",
		extractBoardIDs(t, navParams, "优先用英文 id：", "；也可以传中文板块名"), want)

	msg, _ := a.wxAgentExecTool("", ai.ChatToolCall{
		Function: ai.ChatToolFunction{Name: "navigate_board", Arguments: `{"board":"不存在的板块"}`},
	}, "随便")
	assertSameBoardIDs(t, "微信工具失败回执",
		extractBoardIDs(t, msg, "可用板块 id：", "。"), want)

	// 说明文案里的中文板块名同样来自 manifest（此前写死「编程」）
	if strings.Contains(navDesc, "编程") {
		t.Error("工具说明仍列举已删除的编程板块")
	}
	for _, m := range a.GetBoardManifests() {
		if m.Label != "" && !strings.Contains(navDesc, m.Label) {
			t.Errorf("工具说明缺板块展示名 %q", m.Label)
		}
	}
}

// TestBoardIDListDegradesWithoutManifest 清单拼装不出（nil/全空 id）时返回空串，
// 提示词退回「不打白名单约束」而不是 panic 或编造清单。
func TestBoardIDListDegradesWithoutManifest(t *testing.T) {
	if got := boardIDsFromManifests(nil); got != "" {
		t.Fatalf("nil 清单拼装 = %q, want 空串", got)
	}
	if got := boardIDsFromManifests([]board.Manifest{{ID: ""}, {ID: "   "}}); got != "" {
		t.Fatalf("全空 id 拼装 = %q, want 空串", got)
	}
	if got := boardIDsFromManifests([]board.Manifest{{ID: "chat"}, {ID: " sin "}}); got != "chat sin" {
		t.Fatalf("拼装 = %q, want %q（trim + 空格分隔）", got, "chat sin")
	}

	p := intentFallbackSystemPrompt("")
	if !strings.Contains(p, "navigate") || !strings.Contains(p, "confidence") {
		t.Fatalf("降级提示词必须保留动作语义: %q", p)
	}
	if strings.Contains(p, "只能是：") {
		t.Error("空清单不得写白名单约束（不编造清单）")
	}
	if p2 := intentFallbackSystemPrompt("chat sin"); !strings.Contains(p2, "只能是：chat sin 之一") {
		t.Errorf("非空清单应逐字进白名单行: %q", p2)
	}
}
