package app

import (
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db/repos"
)

// seedAnchorScenario 写入一条生日锚点 + 关联事实 + 覆盖该轮次的情节 + 原始对话。
func seedAnchorScenario(t *testing.T, a *App) {
	t.Helper()
	now := time.Now()
	if err := repos.InsertFact(a.whisperDataRoot, whisper.MemoryFact{
		ID: "factBirthday", Domain: "user_profile", Subcategory: "BASIC_PROFILE",
		Subject: "生日", Summary: "我的生日是 5 月 20 日",
		Weight: 1, Confidence: 0.9, Status: "active",
		SourceSessionID: "whisper_anchorA", SourceTurnIndex: 3,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("InsertFact: %v", err)
	}
	if err := repos.InsertEpisode(a.whisperDataRoot, whisper.Episode{
		ID: "epAnchorA", Summary: "聊到生日那天", EmotionalIntensity: 0.6,
		DominantEmotion: "开心", Keywords: []string{"生日"},
		SourceSessionID: "whisper_anchorA", StartTurn: 2, EndTurn: 4, CreatedAt: now,
	}); err != nil {
		t.Fatalf("InsertEpisode: %v", err)
	}
	rows := []map[string]interface{}{
		{"turnIndex": float64(2), "userText": "第二轮：我下周过生日", "assistantText": "第二轮：提前祝你生日快乐"},
		{"turnIndex": float64(3), "userText": "第三轮：其实是 5 月 20 日", "assistantText": "第三轮：我记下了"},
		{"turnIndex": float64(4), "userText": "第四轮：到时候提醒我", "assistantText": "第四轮：好"},
	}
	if err := repos.SaveChatHistoryToDB(a.whisperDataRoot, "whisper_anchorA", rows); err != nil {
		t.Fatalf("SaveChatHistoryToDB: %v", err)
	}
	repo, err := repos.OpenTemporalAnchorsRepo(a.whisperDataRoot)
	if err != nil {
		t.Fatalf("OpenTemporalAnchorsRepo: %v", err)
	}
	if err := repo.SaveAll([]whisper.TemporalAnchor{{
		ID: "ancBirthday", AnchorDate: "2026-05-20", AnchorType: whisper.AnchorRecurring,
		LinkedFactIDs: []string{"factBirthday"}, EmotionalValence: 0.4,
		EmotionalIntensity: 0.6, Domain: "user_profile", Summary: "我的生日是 5 月 20 日",
	}}); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}
}

func TestGaeaWhisperAnchors_ListSortedByDateDesc(t *testing.T) {
	a := newChatServiceTestApp(t)
	repo, err := repos.OpenTemporalAnchorsRepo(a.whisperDataRoot)
	if err != nil {
		t.Fatalf("OpenTemporalAnchorsRepo: %v", err)
	}
	if err := repo.SaveAll([]whisper.TemporalAnchor{
		{ID: "ancOld", AnchorDate: "2025-06-01", AnchorType: whisper.AnchorMilestone, Summary: "旧锚点"},
		{ID: "ancNew", AnchorDate: "2026-05-20", AnchorType: whisper.AnchorRecurring, Summary: "新锚点"},
	}); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}

	list := a.GaeaWhisperAnchors()
	if len(list) != 2 {
		t.Fatalf("应返回 2 个锚点, got %d", len(list))
	}
	if list[0].ID != "ancNew" || list[1].ID != "ancOld" {
		t.Errorf("应按日期降序: got %s, %s", list[0].ID, list[1].ID)
	}
}

func TestGaeaWhisperAnchorReplay_ResolvesEpisodeAndReplay(t *testing.T) {
	a := newChatServiceTestApp(t)
	seedAnchorScenario(t, a)

	view, err := a.GaeaWhisperAnchorReplay("ancBirthday")
	if err != nil {
		t.Fatalf("GaeaWhisperAnchorReplay: %v", err)
	}
	if !view.Replayable {
		t.Fatalf("锚点应命中情节并可回放, got Replayable=false: %+v", view)
	}
	if view.EpisodeReplay == nil {
		t.Fatalf("应带 EpisodeReplay")
	}
	if view.EpisodeReplay.ID != "epAnchorA" {
		t.Errorf("命中的情节应为 epAnchorA, got %s", view.EpisodeReplay.ID)
	}
	if len(view.EpisodeReplay.Dialogue) != 6 {
		t.Fatalf("第 2-4 轮应重建 6 行对话, got %d: %+v", len(view.EpisodeReplay.Dialogue), view.EpisodeReplay.Dialogue)
	}
	if len(view.LinkedFactSummaries) != 1 || view.LinkedFactSummaries[0] != "我的生日是 5 月 20 日" {
		t.Errorf("LinkedFactSummaries 不符: %+v", view.LinkedFactSummaries)
	}
}

func TestGaeaWhisperAnchorReplay_NoEpisodeFallback(t *testing.T) {
	a := newChatServiceTestApp(t)
	now := time.Now()
	if err := repos.InsertFact(a.whisperDataRoot, whisper.MemoryFact{
		ID: "factOrphan", Domain: "user_profile", Subcategory: "BASIC_PROFILE",
		Subject: "纪念日", Summary: "我们第一次见面", Weight: 1, Confidence: 0.9,
		Status: "active", SourceSessionID: "whisper_anchorB", SourceTurnIndex: 9,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("InsertFact: %v", err)
	}
	if err := repos.InsertEpisode(a.whisperDataRoot, whisper.Episode{
		ID: "epAnchorB", Summary: "别的片段", SourceSessionID: "whisper_anchorB",
		StartTurn: 2, EndTurn: 4, CreatedAt: now,
	}); err != nil {
		t.Fatalf("InsertEpisode: %v", err)
	}
	repo, err := repos.OpenTemporalAnchorsRepo(a.whisperDataRoot)
	if err != nil {
		t.Fatalf("OpenTemporalAnchorsRepo: %v", err)
	}
	if err := repo.SaveAll([]whisper.TemporalAnchor{{
		ID: "ancOrphan", AnchorDate: "2026-01-01", AnchorType: whisper.AnchorMilestone,
		LinkedFactIDs: []string{"factOrphan"}, Summary: "我们第一次见面",
	}}); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}

	view, err := a.GaeaWhisperAnchorReplay("ancOrphan")
	if err != nil {
		t.Fatalf("GaeaWhisperAnchorReplay: %v", err)
	}
	if view.Replayable {
		t.Fatalf("无覆盖情节时应 Replayable=false")
	}
	if view.EpisodeReplay != nil {
		t.Fatalf("无覆盖情节时 EpisodeReplay 应为 nil")
	}
	if len(view.LinkedFactSummaries) != 1 {
		t.Fatalf("应回退展示关联事实摘要, got %+v", view.LinkedFactSummaries)
	}
}

func TestGaeaWhisperAnchorReplay_NotFoundAndEmptyID(t *testing.T) {
	a := newChatServiceTestApp(t)
	if _, err := a.GaeaWhisperAnchorReplay("nope"); err == nil || !strings.Contains(err.Error(), "时间锚点不存在") {
		t.Fatalf("未找到锚点应报错, got %v", err)
	}
	if _, err := a.GaeaWhisperAnchorReplay(""); err == nil || !strings.Contains(err.Error(), "anchorID 为空") {
		t.Fatalf("空 ID 应报错, got %v", err)
	}
}
