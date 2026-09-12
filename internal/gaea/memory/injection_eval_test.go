package memory

// 注入体检纯核测试（市场调研候选2）：五条不变量逐形态覆盖 + 固化覆盖信息面 +
// 真实构建器组合的全合规通路。块输入一律来自真实构建器（除故意造违例）。

import (
	"strings"
	"testing"
	"time"
)

func TestEvalInjectionBlocksAllClear(t *testing.T) {
	now := time.Now()
	work := []Memory{
		{Name: "pricing-rule", Title: "组价口径", Description: "综合单价按 2026 定额", Type: TypeProject, Kind: KindSemantic},
		{Name: "fresh-fact", Title: "新鲜事实", Description: "昨天刚沉淀", Type: TypeUser, Kind: KindEpisodic, LastUsedAt: now},
	}
	work[0].Pinned = true
	preload := BuildMorningPreloadBlock(work, now, 0)
	brief := BuildProjectBrief(work, now, 0)
	if preload == "" || brief == "" {
		t.Fatalf("真实构建器产出空块 preload=%q brief=%q", preload, brief)
	}

	rep := EvalInjectionBlocks(InjectionEvalInput{
		Work:         work,
		PreloadBlock: preload,
		BriefBlock:   brief,
	})
	if !rep.Passed || len(rep.Violations) != 0 {
		t.Fatalf("全合规场景应 Passed，violations=%v", rep.Violations)
	}
	if !rep.PreloadPresent || !rep.BriefPresent {
		t.Fatal("两块都应标记 Present")
	}
	if rep.EntryCount != 2 {
		t.Errorf("EntryCount = %d, want 2", rep.EntryCount)
	}
	if rep.RefCount < 1 {
		t.Errorf("RefCount = %d, want ≥1（[MEM:pricing-rule] 应被引用）", rep.RefCount)
	}
	if rep.PinnedTotal != 1 || rep.PinnedInBrief != 1 || len(rep.MissingPinned) != 0 {
		t.Errorf("固化覆盖 = (%d,%d,%v), want (1,1,[])", rep.PinnedTotal, rep.PinnedInBrief, rep.MissingPinned)
	}
	if rep.PreloadRunes > EvalPreloadBudget || rep.BriefRunes > EvalBriefBudget {
		t.Errorf("预算口径错误 preload=%d/%d brief=%d/%d",
			rep.PreloadRunes, rep.PreloadBudget, rep.BriefRunes, rep.BriefBudget)
	}
}

func TestEvalInjectionBlocksViolations(t *testing.T) {
	work := []Memory{{Name: "ok-fact", Title: "T", Description: "D", Type: TypeUser, Kind: KindSemantic}}
	// 预算超限（伪造超长块）
	rep := EvalInjectionBlocks(InjectionEvalInput{
		Work:         work,
		PreloadBlock: strings.Repeat("超", EvalPreloadBudget+10),
	})
	if rep.Passed || !strings.Contains(strings.Join(rep.Violations, ";"), "超预算") {
		t.Errorf("预算超限应 violation，got %v", rep.Violations)
	}

	// 悬空引用（[MEM:ghost] 不在 work 视图）
	rep = EvalInjectionBlocks(InjectionEvalInput{
		Work:       work,
		BriefBlock: "- [MEM:ghost] 幽灵引用",
	})
	if rep.Passed || !strings.Contains(strings.Join(rep.Violations, ";"), "悬空注入") {
		t.Errorf("悬空引用应 violation，got %v", rep.Violations)
	}

	// 归档泄漏（精确路径）：归档名仍以预载条目形态出现（库状态不一致形态）
	// → 判「是归档记忆」。
	rep = EvalInjectionBlocks(InjectionEvalInput{
		Work:          []Memory{{Name: "ghost-arch", Title: "T", Description: "D", Pinned: true}},
		ArchivedNames: []string{"ghost-arch"},
		PreloadBlock:  "- ghost-arch：已归档内容",
	})
	if rep.Passed || !strings.Contains(strings.Join(rep.Violations, ";"), "是归档记忆") {
		t.Errorf("归档条目应 violation，got %v", rep.Violations)
	}

	// 跨空间泄漏（精确路径）：他空间名以预载条目形态出现 → 判「不在 work 视图」。
	rep = EvalInjectionBlocks(InjectionEvalInput{
		Work:         []Memory{{Name: "ok-fact", Title: "T", Description: "D"}},
		OtherNames:   map[string]bool{"play-secret": true},
		PreloadBlock: "- play-secret：闲庭记忆",
	})
	if rep.Passed || !strings.Contains(strings.Join(rep.Violations, ";"), "不在 work 视图") {
		t.Errorf("跨空间条目应 violation，got %v", rep.Violations)
	}
}

func TestEvalInjectionBlocksPinnedCoverageIsInformational(t *testing.T) {
	now := time.Now()
	pinned := Memory{Name: "pinned-big", Title: "大条", Description: strings.Repeat("长", 300), Type: TypeProject, Kind: KindSemantic}
	pinned.Pinned = true
	brief := BuildProjectBrief([]Memory{pinned}, now, 80) // 预算挤到装不下
	rep := EvalInjectionBlocks(InjectionEvalInput{
		Work:       []Memory{pinned},
		BriefBlock: brief,
	})
	// 预算挤兑=设计内行为：MissingPinned 透出但不判死（violations 只可能来自预算行）
	if len(rep.MissingPinned) != 1 || rep.MissingPinned[0] != "pinned-big" {
		t.Errorf("MissingPinned = %v, want [pinned-big]", rep.MissingPinned)
	}
	for _, v := range rep.Violations {
		if strings.Contains(v, "固化") {
			t.Errorf("固化未全收不应作为 violation：%q", v)
		}
	}
}

func TestEvalInjectionBlocksEmpty(t *testing.T) {
	rep := EvalInjectionBlocks(InjectionEvalInput{})
	if !rep.Passed {
		t.Errorf("空输入应 Passed，violations=%v", rep.Violations)
	}
	if rep.PreloadPresent || rep.BriefPresent || rep.EntryCount != 0 || rep.RefCount != 0 {
		t.Error("空输入各计数应为零/Present 为假")
	}
	if rep.PreloadBudget != EvalPreloadBudget || rep.BriefBudget != EvalBriefBudget {
		t.Error("预算缺省应取 Eval*Budget")
	}
}
