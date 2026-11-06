package app

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db/repos"
)

// 众神殿来源字段契约（进度计划/gaea-chat-persona-optimization-20261105.md
// §三.3）：新写入 chat_history 行带 source="raw"；遗留旧行（无 source 键）
// 经 restore→重落库全程零改写、零标记（可审计、可逆）；未知来源形态不回灌。

func TestSourceContract_NewRowsMarkedRaw(t *testing.T) {
	a := newChatServiceTestApp(t)
	root := a.whisperState.whisperDataRoot
	sessionID := "whisper_source_new"
	cleanupWhisperSession(t, "source_new")

	orch := whisper.NewOrchestrator(sessionID, *whisper.GetPreset("gaea"))
	orch.DataRoot = root
	orch.WM.Push(sessionID, whisper.Exchange{
		TurnIndex: 1, UserText: "帮我查一下成都天气", AssistantText: "成都今天晴。",
		Source: whisper.ExchangeSourceRaw,
	})
	if err := persistWhisperState(orch); err != nil {
		t.Fatalf("persistWhisperState: %v", err)
	}
	rows, err := repos.LoadChatHistoryFromDB(root, sessionID)
	if err != nil || len(rows) != 1 {
		t.Fatalf("load: %v rows=%d", err, len(rows))
	}
	if got, _ := rows[0]["source"].(string); got != whisper.ExchangeSourceRaw {
		t.Fatalf("新写入行 source = %q, want %q", got, whisper.ExchangeSourceRaw)
	}
	if got, _ := rows[0]["userText"].(string); got != "帮我查一下成都天气" {
		t.Fatalf("userText 应为原文: %q", got)
	}
}

func TestSourceContract_LegacyRowsUntouched(t *testing.T) {
	a := newChatServiceTestApp(t)
	root := a.whisperState.whisperDataRoot
	sessionID := "whisper_source_legacy"
	cleanupWhisperSession(t, "source_legacy")

	// 种一行「遗留形态」：无 source 键（v4.532 旧数据口径）
	if err := repos.SaveChatHistoryToDB(root, sessionID, []map[string]interface{}{
		{"turnIndex": 1, "userText": "你好", "assistantText": "嗨呀。"},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// restore 载入（Source 空）→ 原样重落库 → 仍无 source 键
	orch := whisper.NewOrchestrator(sessionID, *whisper.GetPreset("gaea"))
	orch.DataRoot = root
	if err := restoreWhisperState(orch); err != nil {
		t.Fatalf("restoreWhisperState: %v", err)
	}
	recent := orch.WM.GetRecent(sessionID)
	if len(recent) != 1 || recent[0].Source != "" {
		t.Fatalf("遗留行 restore 后 Source 应为空: %+v", recent)
	}
	if err := persistWhisperState(orch); err != nil {
		t.Fatalf("re-persist: %v", err)
	}
	rows, err := repos.LoadChatHistoryFromDB(root, sessionID)
	if err != nil || len(rows) != 1 {
		t.Fatalf("reload: %v rows=%d", err, len(rows))
	}
	if _, has := rows[0]["source"]; has {
		t.Fatalf("遗留行重落库不得被盖上来源标记: %+v", rows[0])
	}
	if got, _ := rows[0]["userText"].(string); got != "你好" {
		t.Fatalf("遗留正文零改写: %q", got)
	}
}

func TestSourceContract_UnknownSourceNotWarmed(t *testing.T) {
	a := newChatServiceTestApp(t)
	root := a.whisperState.whisperDataRoot
	sessionID := "whisper_source_warm"
	cleanupWhisperSession(t, "source_warm")

	seedChatHistory(t, root, sessionID, []map[string]interface{}{
		{"turnIndex": 1, "userText": "未知来源轮", "assistantText": "回应", "source": "future-form"},
		{"turnIndex": 2, "userText": "干净原文轮", "assistantText": "回应二", "source": whisper.ExchangeSourceRaw},
		{"turnIndex": 3, "userText": "无标记旧轮", "assistantText": "回应三"},
	})
	orch := whisper.NewOrchestrator(sessionID, *whisper.GetPreset("gaea"))
	orch.DataRoot = root
	if err := restoreWhisperState(orch); err != nil {
		t.Fatalf("restoreWhisperState: %v", err)
	}
	a.whisperState.warmPromptHistory(orch)
	// 未知形态跳过；raw 与无标记旧行都回灌
	if len(orch.PromptHistory) != 2 {
		t.Fatalf("回灌应取 2 轮（raw+无标记旧行），got %d", len(orch.PromptHistory))
	}
	for _, ex := range orch.PromptHistory {
		if strings.Contains(ex.UserText, "未知来源") {
			t.Fatalf("未知来源形态不得回灌: %+v", ex)
		}
	}
}
