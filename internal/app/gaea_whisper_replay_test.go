package app

import (
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db/repos"
)

// seedReplayEpisode 写入一条情节 + 对应的会话原始对话（轮次 1-4）。
func seedReplayEpisode(t *testing.T, root, sessionID, episodeID string) {
	t.Helper()
	now := time.Now()
	if err := repos.InsertEpisode(root, whisper.Episode{
		ID: episodeID, Summary: "深夜一起改 bug 到天亮",
		EmotionalIntensity: 0.85, DominantEmotion: "兴奋",
		Keywords: []string{"debug", "熬夜"},
		SourceSessionID: sessionID, StartTurn: 2, EndTurn: 3,
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("InsertEpisode: %v", err)
	}
	rows := []map[string]interface{}{
		{"turnIndex": float64(1), "userText": "第一轮：睡不着", "assistantText": "第一轮：我在"},
		{"turnIndex": float64(2), "userText": "第二轮：帮我看看这个 bug", "assistantText": "第二轮：好，贴出来"},
		{"turnIndex": float64(3), "userText": "第三轮：修好了，谢啦", "assistantText": "第三轮：一起熬的夜值得"},
		{"turnIndex": float64(4), "userText": "第四轮：晚安", "assistantText": "第四轮：晚安"},
	}
	if err := repos.SaveChatHistoryToDB(root, sessionID, rows); err != nil {
		t.Fatalf("SaveChatHistoryToDB: %v", err)
	}
}

func TestGaeaWhisperEpisodeReplay_ReconstructsDialogueInTurnRange(t *testing.T) {
	a := newChatServiceTestApp(t)
	seedReplayEpisode(t, a.whisperDataRoot, "whisper_replayA", "epReplayA")

	view, err := a.GaeaWhisperEpisodeReplay("epReplayA")
	if err != nil {
		t.Fatalf("GaeaWhisperEpisodeReplay: %v", err)
	}
	if !view.Replayable {
		t.Fatalf("应有可回放对话，got Replayable=false")
	}
	if len(view.Dialogue) != 4 {
		t.Fatalf("第 2-3 轮应重建 4 行对话（user+assistant×2），got %d: %+v", len(view.Dialogue), view.Dialogue)
	}
	want := []WhisperReplayLine{
		{TurnIndex: 2, Role: "user", Text: "第二轮：帮我看看这个 bug"},
		{TurnIndex: 2, Role: "assistant", Text: "第二轮：好，贴出来"},
		{TurnIndex: 3, Role: "user", Text: "第三轮：修好了，谢啦"},
		{TurnIndex: 3, Role: "assistant", Text: "第三轮：一起熬的夜值得"},
	}
	for i, w := range want {
		if view.Dialogue[i] != w {
			t.Errorf("dialogue[%d] = %+v, want %+v", i, view.Dialogue[i], w)
		}
	}
	if view.Summary != "深夜一起改 bug 到天亮" || view.StartTurn != 2 || view.EndTurn != 3 {
		t.Errorf("元数据不符: %+v", view)
	}
}

func TestGaeaWhisperEpisodeReplay_NoHistoryFallsBackToSummaryOnly(t *testing.T) {
	a := newChatServiceTestApp(t)
	now := time.Now()
	if err := repos.InsertEpisode(a.whisperDataRoot, whisper.Episode{
		ID: "epNoHistory", Summary: "爬山途中的闲聊",
		SourceSessionID: "whisper_replayB", StartTurn: 1, EndTurn: 2, CreatedAt: now,
	}); err != nil {
		t.Fatalf("InsertEpisode: %v", err)
	}

	view, err := a.GaeaWhisperEpisodeReplay("epNoHistory")
	if err != nil {
		t.Fatalf("GaeaWhisperEpisodeReplay: %v", err)
	}
	if view.Replayable {
		t.Fatalf("无原始对话时应 Replayable=false")
	}
	if len(view.Dialogue) != 0 {
		t.Fatalf("无原始对话时应空 Dialogue, got %+v", view.Dialogue)
	}
	if view.Summary != "爬山途中的闲聊" {
		t.Fatalf("摘要应保留, got %q", view.Summary)
	}
}

func TestGaeaWhisperEpisodeReplay_NotFoundAndEmptyID(t *testing.T) {
	a := newChatServiceTestApp(t)

	if _, err := a.GaeaWhisperEpisodeReplay("nope"); err == nil || !strings.Contains(err.Error(), "情节不存在") {
		t.Fatalf("未找到情节应报错, got %v", err)
	}
	if _, err := a.GaeaWhisperEpisodeReplay(""); err == nil || !strings.Contains(err.Error(), "episodeID 为空") {
		t.Fatalf("空 ID 应报错, got %v", err)
	}
}
