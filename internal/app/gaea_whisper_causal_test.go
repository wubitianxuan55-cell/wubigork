package app

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/whisper"
)

// seedCausalApp 构造测试 App 并给 orch 种入：一条「导致」三元组 +
// 一条 event_chain 关联。返回 App 与 orch 供绑定/证据测试使用。
func seedCausalApp(t *testing.T, personalityID string) (*App, *whisper.Orchestrator) {
	t.Helper()
	a := newChatServiceTestApp(t)
	orch := a.whisperState.getOrCreateOrch(personalityID)
	orch.KG.AddTriple(whisper.Triple{Subject: "加班", Predicate: "导致", Object: "睡不好", Confidence: 0.9})
	fA := orch.FactStore.Add(whisper.MemoryFact{ID: "fCausalA", Subject: "加班", Summary: "最近项目赶进度"})
	fB := orch.FactStore.Add(whisper.MemoryFact{ID: "fCausalB", Subject: "睡不好", Summary: "夜里总是醒"})
	orch.AssocIndex.Add(whisper.Association{
		FactIDA: fA.ID, FactIDB: fB.ID,
		AssociationType: "event_chain", Strength: 0.6,
	})
	return a, orch
}

func TestGaeaWhisperCausalExplain_WithEvidence(t *testing.T) {
	a, _ := seedCausalApp(t, "pidCausal")
	got, err := a.whisperState.GaeaWhisperCausalExplain("睡不好", "pidCausal")
	if err != nil {
		t.Fatalf("GaeaWhisperCausalExplain: %v", err)
	}
	if !strings.Contains(got, "你好呀") {
		t.Errorf("应返回模型回复, got %q", got)
	}
}

func TestGaeaWhisperCausalExplain_NoEvidenceHonestFallback(t *testing.T) {
	a := newChatServiceTestApp(t)
	a.whisperState.getOrCreateOrch("pidEmpty")

	got, err := a.whisperState.GaeaWhisperCausalExplain("完全陌生的话题", "pidEmpty")
	if err != nil {
		t.Fatalf("GaeaWhisperCausalExplain: %v", err)
	}
	if !strings.Contains(got, "还没有足够的记忆") {
		t.Errorf("无证据应返回诚实回退文案, got %q", got)
	}
	if strings.Contains(got, "你好呀") {
		t.Errorf("无证据不应调 LLM, got %q", got)
	}
}

func TestGaeaWhisperCausalExplain_EmptyEntity(t *testing.T) {
	a := newChatServiceTestApp(t)
	if _, err := a.whisperState.GaeaWhisperCausalExplain("  ", "pidCausal"); err == nil ||
		!strings.Contains(err.Error(), "entity 为空") {
		t.Fatalf("空 entity 应报错, got %v", err)
	}
}

func TestBuildCausalEvidence(t *testing.T) {
	_, orch := seedCausalApp(t, "pidEvidence")
	ev := buildCausalEvidence("睡不好", orch.KG, orch.AssocIndex, orch.FactStore)
	if !strings.Contains(ev, "加班 → 导致 → 睡不好") {
		t.Errorf("应含「导致」三元组证据:\n%s", ev)
	}
	if !strings.Contains(ev, "关联：加班") || !strings.Contains(ev, "睡不好") {
		t.Errorf("应含 event_chain 关联证据:\n%s", ev)
	}
	if strings.Contains(ev, "完全无关") {
		t.Errorf("不应出现无关内容:\n%s", ev)
	}
}

func TestBuildCausalEvidence_UnrelatedEntityEmpty(t *testing.T) {
	_, orch := seedCausalApp(t, "pidEvidence2")
	if got := buildCausalEvidence("完全陌生的话题", orch.KG, orch.AssocIndex, orch.FactStore); got != "" {
		t.Errorf("无关实体应无证据, got %q", got)
	}
}
