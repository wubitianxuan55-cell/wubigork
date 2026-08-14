package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/whisper"
)

// ── V11 FTS 全量同步失败 → slog 日志（T6-5.5）─────────────────────

// TestPersistFactsFTS_FailureLogs FTS 事实索引全量重建失败必须记录 slog
// （不再 `_ = repos.RebuildFactsFTS(...)` 静默丢弃）。
func TestPersistFactsFTS_FailureLogs(t *testing.T) {
	// dataRoot 被普通文件占位 → GetDatabase 失败 → RebuildFactsFTS 返回错误
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("写占位文件: %v", err)
	}

	orch := whisper.NewOrchestrator("fts-sess", whisper.PersonalityPresets[0])
	orch.DataRoot = blocker
	orch.FactStore.Add(whisper.MemoryFact{
		Domain: "preference", Subcategory: "FOOD", Subject: "用户",
		Summary: "喜欢吃辣", Weight: 2, Confidence: 0.9,
	})

	records := captureLogs(t, func() {
		persistFactsToDB(orch)
	})
	if !logContainsMsg(records, "FTS 事实索引重建失败") {
		t.Fatalf("未记录 FTS 事实索引重建失败日志, got %v", logMsgs(records))
	}
}

// TestPersistEpisodesFTS_FailureLogs FTS 情节索引全量重建失败必须记录 slog。
func TestPersistEpisodesFTS_FailureLogs(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("写占位文件: %v", err)
	}

	orch := whisper.NewOrchestrator("fts-ep-sess", whisper.PersonalityPresets[0])
	orch.DataRoot = blocker
	orch.EpisodicStore.Add(whisper.Episode{
		Summary: "一起吃辣", DominantEmotion: "开心",
		EmotionalIntensity: 0.8, Keywords: []string{"辣"},
	})

	records := captureLogs(t, func() {
		persistEpisodesToDB(orch)
	})
	if !logContainsMsg(records, "FTS 情节索引重建失败") {
		t.Fatalf("未记录 FTS 情节索引重建失败日志, got %v", logMsgs(records))
	}
}
