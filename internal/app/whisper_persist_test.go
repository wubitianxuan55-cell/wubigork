package app

import (
	"testing"

	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db"
	"github.com/gaea/gaea/internal/whisper/db/repos"
)

// TestWhisperMemoryPersistRoundTrip 角色记忆隔离端到端：写入 → 持久化 → 同会话恢复
// 验证事实/情节/图谱按会话隔离：自己的记忆重启不丢，其他角色的记忆不串不覆盖
func TestWhisperMemoryPersistRoundTrip(t *testing.T) {
	dataRoot := t.TempDir()
	if _, err := db.GetDatabase(dataRoot); err != nil { // 初始化 hermes.db schema
		t.Fatalf("GetDatabase: %v", err)
	}
	defer db.CloseDatabase(dataRoot)

	// 会话 1：写入事实 + 情节 + 退役事实
	orch1 := whisper.NewOrchestrator("sess-1", whisper.PersonalityPresets[0])
	orch1.DataRoot = dataRoot
	f1 := orch1.FactStore.Add(whisper.MemoryFact{
		Domain: "preference", Subcategory: "FOOD", Subject: "用户",
		Summary: "喜欢吃辣", Weight: 2, Confidence: 0.9, SourceSessionID: "sess-1",
	})
	retired := orch1.FactStore.Add(whisper.MemoryFact{
		Domain: "user_state", Subcategory: "MOOD", Subject: "用户",
		Summary: "最近睡不好", Weight: 1, SourceSessionID: "sess-1",
	})
	orch1.FactStore.RetireFact(retired.ID)
	orch1.EpisodicStore.Add(whisper.Episode{
		Summary: "用户分享了吃辣爱好", DominantEmotion: "开心",
		EmotionalIntensity: 0.8, Keywords: []string{"辣", "美食"}, SourceSessionID: "sess-1",
	})
	orch1.KG.Add("用户", "喜欢", "辣", 1, []string{f1.ID})
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
	if got := orch2.KG.Size(); got != 1 {
		t.Fatalf("恢复后知识图谱三元组 = %d, want 1", got)
	}
	if got := len(orch2.KG.Query("辣", 5)); got != 1 {
		t.Fatalf("恢复后 KG 检索(辣) = %d, want 1", got)
	}

	// 多会话隔离：会话 3 新增自己的事实；写回后会话 1 的事实仍在 DB（合并而非全量覆盖）
	orch3 := whisper.NewOrchestrator("sess-2", whisper.PersonalityPresets[0])
	orch3.DataRoot = dataRoot
	if err := restoreWhisperState(orch3); err != nil {
		t.Fatalf("restoreWhisperState(sess-2): %v", err)
	}
	// 隔离：sess-2 不应看到 sess-1 的任何记忆
	if got := orch3.FactStore.Count(); got != 0 {
		t.Fatalf("sess-2 不应看到 sess-1 的事实, got %d", got)
	}
	if got := orch3.KG.Size(); got != 0 {
		t.Fatalf("sess-2 不应看到 sess-1 的三元组, got %d", got)
	}
	orch3.FactStore.Add(whisper.MemoryFact{
		Domain: "preference", Subcategory: "DRINK", Subject: "用户",
		Summary: "喜欢喝美式咖啡", Weight: 1, SourceSessionID: "sess-2",
	})
	persistWhisperState(orch3)

	// 会话 1 再次恢复：只看到自己的事实（sess-1 两条，含退役态）；sess-2 的事实隔离在外
	orch4 := whisper.NewOrchestrator("sess-1", whisper.PersonalityPresets[0])
	orch4.DataRoot = dataRoot
	if err := restoreWhisperState(orch4); err != nil {
		t.Fatalf("restoreWhisperState(sess-1 again): %v", err)
	}
	if got := len(orch4.FactStore.ListAll()); got != 2 {
		t.Fatalf("sess-1 恢复后事实总数 = %d, want 2（不含 sess-2 的事实）", got)
	}
	// DB 层合并校验：sess-1 两条 + sess-2 一条，全部保留
	if got := len(repos.LoadFactsFromDB(dataRoot)); got != 3 {
		t.Fatalf("DB 事实总数 = %d, want 3（两会话合并，互不覆盖）", got)
	}
}
