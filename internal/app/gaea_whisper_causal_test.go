package app

import (
	"fmt"
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
	// 进程级 whisperSessions 缓存用完即删：固定 ID 在 -count 多次运行下会命中
	// 上次运行的 orch（跨 app 实例串扰）。
	cleanupWhisperSession(t, personalityID)
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
	cleanupWhisperSession(t, "pidEmpty")

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

func TestBuildCausalEvidence_TwoHopChain(t *testing.T) {
	// seedCausalApp 已种「加班 → 导致 → 睡不好」，此处只补更早一环。
	_, orch := seedCausalApp(t, "pidTwoHop")
	orch.KG.AddTriple(whisper.Triple{Subject: "甲方拖延", Predicate: "导致", Object: "加班", Confidence: 0.9})

	ev := buildCausalEvidence("睡不好", orch.KG, orch.AssocIndex, orch.FactStore)
	if !strings.Contains(ev, "记忆链：甲方拖延 → 导致 → 加班 → 导致 → 睡不好") {
		t.Errorf("应收集两跳因果链:\n%s", ev)
	}
	if strings.Contains(ev, "记忆：加班 → 导致 → 睡不好") {
		t.Errorf("被链覆盖的单跳不应重复出现:\n%s", ev)
	}
}

func TestBuildCausalChains_CycleNoRunaway(t *testing.T) {
	_, orch := seedCausalApp(t, "pidCycle")
	orch.KG.AddTriple(whisper.Triple{Subject: "甲乙相因", Predicate: "导致", Object: "乙果反复", Confidence: 0.9})
	orch.KG.AddTriple(whisper.Triple{Subject: "乙果反复", Predicate: "导致", Object: "甲乙相因", Confidence: 0.9})

	ev := buildCausalEvidence("甲乙相因", orch.KG, orch.AssocIndex, orch.FactStore)
	if strings.Contains(ev, "记忆链：") {
		t.Errorf("互为因果的两条单跳边不成链，不应出现记忆链:\n%s", ev)
	}
	if !strings.Contains(ev, "记忆：甲乙相因 → 导致 → 乙果反复") ||
		!strings.Contains(ev, "记忆：乙果反复 → 导致 → 甲乙相因") {
		t.Errorf("两条单跳边应原样保留:\n%s", ev)
	}
}

func TestBuildCausalEvidence_DepthCapThreeHops(t *testing.T) {
	_, orch := seedCausalApp(t, "pidDepth")
	orch.KG.AddTriple(whisper.Triple{Subject: "远因三层", Predicate: "导致", Object: "中因两层", Confidence: 0.9})
	orch.KG.AddTriple(whisper.Triple{Subject: "中因两层", Predicate: "导致", Object: "近因一层", Confidence: 0.9})
	orch.KG.AddTriple(whisper.Triple{Subject: "近因一层", Predicate: "导致", Object: "目标结果", Confidence: 0.9})

	ev := buildCausalEvidence("目标结果", orch.KG, orch.AssocIndex, orch.FactStore)
	if !strings.Contains(ev, "记忆链：中因两层 → 导致 → 近因一层 → 导致 → 目标结果") {
		t.Errorf("应收集两跳链:\n%s", ev)
	}
	if strings.Contains(ev, "远因三层") {
		t.Errorf("超出跳数上限的第三层不应进入证据:\n%s", ev)
	}
}

func TestBuildCausalChains_CapFour(t *testing.T) {
	_, orch := seedCausalApp(t, "pidCap")
	for i := 0; i < 6; i++ {
		orch.KG.AddTriple(whisper.Triple{Subject: fmt.Sprintf("事因%d", i), Predicate: "导致", Object: fmt.Sprintf("中间%d", i), Confidence: 0.9})
		orch.KG.AddTriple(whisper.Triple{Subject: fmt.Sprintf("中间%d", i), Predicate: "导致", Object: "目标结果", Confidence: 0.9})
	}

	ev := buildCausalEvidence("目标结果", orch.KG, orch.AssocIndex, orch.FactStore)
	if got := strings.Count(ev, "记忆链："); got != causalMaxChains {
		t.Errorf("记忆链应封顶 %d 条, got %d:\n%s", causalMaxChains, got, ev)
	}
}
