package app

import (
	"strings"
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db/repos"
)

// 众神殿刀4（进度计划/gaea-chat-persona-optimization-20261105.md §1.1④）：重启
// 预热+发布序。① PromptHistory 重启回灌只取无动态包裹标记的干净行（脏行跳过，
// 不模糊清洗）；② getOrCreateOrch 返回的实例必然已完成状态恢复与回灌
//（先恢复后发布，并发首轮拿不到半成品）。

func seedChatHistory(t *testing.T, root, sessionID string, rows []map[string]interface{}) {
	t.Helper()
	if err := repos.SaveChatHistoryToDB(root, sessionID, rows); err != nil {
		t.Fatalf("SaveChatHistoryToDB: %v", err)
	}
}

func TestWarmPromptHistory_SkipsDirtyRows(t *testing.T) {
	a := newChatServiceTestApp(t)
	root := a.whisperState.whisperDataRoot
	sessionID := "whisper_warm_dirty"
	cleanupWhisperSession(t, "warm_dirty")
	// 首轮 userText 带增广标记（模拟旧库搜索包裹历史），其余 4 轮干净
	seedChatHistory(t, root, sessionID, []map[string]interface{}{
		{"turnIndex": 1, "userText": "帮我查一下北京天气\n\n[以下是关于此问题的实时搜索结果]\n天气数据", "assistantText": "好的。"},
		{"turnIndex": 2, "userText": "今天心情不错", "assistantText": "那很好呀。"},
		{"turnIndex": 3, "userText": "你喜欢什么音乐", "assistantText": "雨声。"},
		{"turnIndex": 4, "userText": "讲个故事", "assistantText": "从前。"},
		{"turnIndex": 5, "userText": "讲短一点", "assistantText": "从前有座山。"},
	})

	orch := whisper.NewOrchestrator(sessionID, *whisper.GetPreset("gaea"))
	orch.DataRoot = root
	if err := restoreWhisperState(orch); err != nil {
		t.Fatalf("restoreWhisperState: %v", err)
	}
	if len(orch.WM.GetRecent(sessionID)) != 5 {
		t.Fatalf("WM 应保留全部 5 轮（完整记录），got %d", len(orch.WM.GetRecent(sessionID)))
	}

	a.whisperState.warmPromptHistory(orch)
	if len(orch.PromptHistory) != 4 {
		t.Fatalf("回灌应取 4 轮干净历史，got %d", len(orch.PromptHistory))
	}
	for _, ex := range orch.PromptHistory {
		if containsTransientMarker(ex.UserText) || containsTransientMarker(ex.AssistantText) {
			t.Fatalf("回灌历史不得含动态包裹标记: %+v", ex)
		}
	}
	if got := orch.PromptHistory[len(orch.PromptHistory)-1].UserText; got != "讲短一点" {
		t.Fatalf("回灌末轮应为最近干净行，got %q", got)
	}
}

// TestGetOrCreateOrch_RestoredBeforePublish 并发首轮：所有调用（无论胜出还是
// 让位）拿到的都是已完成恢复+回灌的实例，且全局缓存里只有同一胜者。
func TestGetOrCreateOrch_RestoredBeforePublish(t *testing.T) {
	a := newChatServiceTestApp(t)
	root := a.whisperState.whisperDataRoot
	personalityID := "warm_publish_x"
	sessionID := "whisper_" + personalityID
	cleanupWhisperSession(t, personalityID)
	seedChatHistory(t, root, sessionID, []map[string]interface{}{
		{"turnIndex": 1, "userText": "第一轮", "assistantText": "回应一"},
		{"turnIndex": 2, "userText": "第二轮", "assistantText": "回应二"},
		{"turnIndex": 3, "userText": "第三轮", "assistantText": "回应三"},
	})

	const n = 8
	results := make([]*whisper.Orchestrator, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = a.getOrCreateOrch(personalityID)
		}(i)
	}
	wg.Wait()

	first := results[0]
	if first == nil {
		t.Fatal("getOrCreateOrch 返回 nil")
	}
	for i, o := range results {
		if o != first {
			t.Fatalf("并发调用应全部返回同一胜者实例，results[%d] 不同", i)
		}
		// 胜者发布前必然已完成恢复与回灌（让位者返回的就是胜者）
		if len(o.WM.GetRecent(sessionID)) != 3 {
			t.Fatalf("发布实例必须已完成状态恢复（WM=%d）", len(o.WM.GetRecent(sessionID)))
		}
	}
	if len(first.PromptHistory) != 3 {
		t.Fatalf("发布实例应已回灌 3 轮历史，got %d", len(first.PromptHistory))
	}
	if got := first.PromptHistory[0].UserText; strings.Contains(got, "【本轮上下文】") {
		t.Fatalf("回灌历史应为原文: %q", got)
	}
}
