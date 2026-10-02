package app

// 审计 AP5-06 对照钉子：GaeaMemoryDuplicates（模糊 textsim 人审面板）与
// memory.DistillMergeCandidates（确定性蒸馏，宁漏勿误）对同一批办公记忆的
// 判定差异钉成机器可见事实。行为冻结口径（gaea_memory_meta.go 函数头注记）：
// 两套口径差异属预期，收敛与否在拍板池——本测试红 = 有人单方面改了其中
// 一套的判定口径或保留方向。

import (
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/memory"
)

func TestDuplicatesVsDistillDivergence(t *testing.T) {
	store := newOfficeMemoryTestEnv(t)
	a := &App{}

	// 组 1（模糊面板独有）：描述相似非逐字、名称无归一关系——蒸馏按
	// 「宁漏勿误」不报，面板按 textsim ≥ 阈值报。
	if _, err := store.Save(memory.Memory{Name: "fuzzy-a", Title: "桩基施工要点", Description: "振动锤选型", Body: "要点A"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Save(memory.Memory{Name: "fuzzy-b", Title: "桩基施工 要点", Description: "振动锤选型需匹配地质", Body: "要点B"}); err != nil {
		t.Fatal(err)
	}
	// 组 2（两套都报，保留方向相反）：归一化同名异写 a-b/ab；ab 严格较新
	// （Save 落秒精度 updated_at，隔 >1s 落第二条保证 UpdatedAt 可分先后）。
	if _, err := store.Save(memory.Memory{Name: "a-b", Title: "时区设定", Description: "用户时区为 UTC+8", Body: "body"}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond)
	if _, err := store.Save(memory.Memory{Name: "ab", Title: "时区设定", Description: "用户时区为 UTC+8", Body: "body"}); err != nil {
		t.Fatal(err)
	}
	facts := store.List()

	// 蒸馏（确定性）：只报组 2，保留 UpdatedAt 较新的 ab；组 1 不报。
	cands := memory.DistillMergeCandidates(facts)
	if len(cands) != 1 {
		t.Fatalf("蒸馏候选数 = %d (%+v)，want 1（仅同名异写对；模糊近似文必须不报）", len(cands), cands)
	}
	if cands[0].Keep != "ab" || cands[0].Archive != "a-b" {
		t.Errorf("蒸馏保留方向 = %s←%s，want ab←a-b（保 UpdatedAt 较新）", cands[0].Keep, cands[0].Archive)
	}

	// 模糊面板：组 1、组 2 都报；保留方向按名称靠前（a-b 在 ab 之前），
	// 与蒸馏的保较新相反——这就是两套口径的机器可见分叉。
	dups := a.GaeaMemoryDuplicates(0.5)
	if len(dups) != 2 {
		t.Fatalf("面板重复对数 = %d (%+v)，want 2（模糊近似文对 + 同名异写对）", len(dups), dups)
	}
	if dups[0].Score != 1.0 || dups[0].Keep != "a-b" || dups[0].Dup != "ab" {
		t.Errorf("面板同名异写对 = %+v，want a-b→ab（保名称靠前，与蒸馏保较新相反）", dups[0])
	}
	if dups[1].Keep != "fuzzy-a" || dups[1].Dup != "fuzzy-b" {
		t.Errorf("面板模糊近似文对 = %+v，want fuzzy-a→fuzzy-b（蒸馏按宁漏勿误不报）", dups[1])
	}
}
