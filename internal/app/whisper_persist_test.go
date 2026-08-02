package app

import (
	"testing"

	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db"
)

// TestWhisperMemoryPersistRoundTrip 记忆贯通端到端：写入 → 持久化 → 新会话恢复
// 验证轻语事实库/情节库与 hermes.db 真正打通（此前只写内存，重启即丢、记忆中枢永远空）
func TestWhisperMemoryPersistRoundTrip(t *testing.T) {
	dataRoot := t.TempDir()
	db.GetDatabase(dataRoot) // 初始化 hermes.db schema
	defer db.CloseDatabase(dataRoot)

	// 会话 1：写入事实 + 情节 + 退役事实
	orch1 := whisper.NewOrchestrator("sess-1", whisper.PersonalityPresets[0])
	orch1.DataRoot = dataRoot
	orch1.FactStore.Add(whisper.MemoryFact{
		Domain: "preference", Subcategory: "FOOD", Subject: "用户",
		Summary: "喜欢吃辣", Weight: 2, Confidence: 0.9,
	})
	retired := orch1.FactStore.Add(whisper.MemoryFact{
		Domain: "user_state", Subcategory: "MOOD", Subject: "用户",
		Summary: "最近睡不好", Weight: 1,
	})
	orch1.FactStore.RetireFact(retired.ID)
	orch1.EpisodicStore.Add(whisper.Episode{
		Summary: "用户分享了吃辣爱好", DominantEmotion: "开心",
		EmotionalIntensity: 0.8, Keywords: []string{"辣", "美食"},
	})
	persistWhisperState(orch1)

	// 会话 2（同 session）：恢复后事实/情节/退役态都在
	orch2 := whisper.NewOrchestrator("sess-1", whisper.PersonalityPresets[0])
	orch2.DataRoot = dataRoot
	if err := restoreWhisperState(orch2); err != nil {
		t.Fatalf("restoreWhisperState: %v", err)
	}
	if got := orch2.FactStore.Count(); got != 1 {
		t.Fatalf("恢复后活跃事实 = %d, want 1（退役的不算）", got)
	}
	if got := len(orch2.FactStore.ListAll()); got != 2 {
		t.Fatalf("ListAll 应含退役事实 = 2, got %d", got)
	}
	if got := orch2.EpisodicStore.Count(); got != 1 {
		t.Fatalf("恢复后情节 = %d, want 1", got)
	}
	if ep := orch2.EpisodicStore.Latest(); ep == nil || ep.Summary != "用户分享了吃辣爱好" {
		t.Fatalf("情节内容未恢复: %+v", ep)
	}

	// 多会话不互踩：会话 3 新增事实，写回后会话 1 的事实仍在（合并而非全量覆盖）
	orch3 := whisper.NewOrchestrator("sess-2", whisper.PersonalityPresets[0])
	orch3.DataRoot = dataRoot
	if err := restoreWhisperState(orch3); err != nil {
		t.Fatalf("restoreWhisperState(sess-2): %v", err)
	}
	orch3.FactStore.Add(whisper.MemoryFact{
		Domain: "preference", Subcategory: "DRINK", Subject: "用户",
		Summary: "喜欢喝美式咖啡", Weight: 1,
	})
	persistWhisperState(orch3)

	// 会话 1 再次恢复：应同时看到自己的事实 + 会话 3 的新事实（全局用户档案）
	orch4 := whisper.NewOrchestrator("sess-1", whisper.PersonalityPresets[0])
	orch4.DataRoot = dataRoot
	if err := restoreWhisperState(orch4); err != nil {
		t.Fatalf("restoreWhisperState(sess-1 again): %v", err)
	}
	if got := len(orch4.FactStore.ListAll()); got != 3 {
		t.Fatalf("合并后事实总数 = %d, want 3（sess-1 两条 + sess-3 一条）", got)
	}
}
