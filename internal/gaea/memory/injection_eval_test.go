package memory

// 注入体检纯核测试（市场调研候选2；2026-11-25 体检对象收成索引块）：四条不变量
// 逐形态覆盖 + 固化覆盖信息面 + 真实渲染的全合规通路。索引输入一律来自真实
// 渲染器（renderIndex / IndexForInjection），除故意造违例的用例。

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

func evalFixture() (work []Memory, other map[string]bool, archived []string) {
	work = []Memory{
		{Name: "general-fact", Description: "通用条目", Body: "b", Type: TypeProject},
		{Name: "pinned-rule", Description: "固化规范", Body: "b", Type: TypeProject, Pinned: true},
	}
	return work, map[string]bool{"play-note": true}, []string{"old-one"}
}

func TestEvalIndexInjectionAllClean(t *testing.T) {
	work, other, archived := evalFixture()
	block := renderIndex(work)
	rep := EvalIndexInjection(InjectionEvalInput{Work: work, OtherNames: other, ArchivedNames: archived, IndexBlock: block})
	if !rep.Passed || len(rep.Violations) != 0 {
		t.Fatalf("全合规索引应 Passed，violations=%v（block=%q）", rep.Violations, block)
	}
	if !rep.IndexPresent || rep.EntryCount != 2 || rep.Truncated {
		t.Errorf("索引形态不对：%+v", rep)
	}
	if rep.PinnedTotal != 1 || rep.PinnedInIndex != 1 || len(rep.MissingPinned) != 0 {
		t.Errorf("固化覆盖 = (%d,%d,%v), want (1,1,[])", rep.PinnedTotal, rep.PinnedInIndex, rep.MissingPinned)
	}
	if rep.IndexBudget != EvalIndexBudget {
		t.Errorf("预算 = %d, want %d", rep.IndexBudget, EvalIndexBudget)
	}
}

func TestEvalIndexInjectionViolations(t *testing.T) {
	work, other, archived := evalFixture()
	line := func(m Memory) string {
		return strings.TrimSpace(strings.TrimPrefix(renderIndex([]Memory{m}), "# Memory"))
	}
	cases := []struct {
		name  string
		extra Memory
		want  string
	}{
		{"他空间泄漏", Memory{Name: "play-note", Description: "p"}, "属他空间"},
		{"归档泄漏", Memory{Name: "old-one", Description: "o"}, "是归档记忆"},
		{"悬空条目", Memory{Name: "ghost", Description: "g"}, "不在 work 视图"},
	}
	for _, c := range cases {
		block := renderIndex(append(append([]Memory(nil), work...), c.extra))
		rep := EvalIndexInjection(InjectionEvalInput{Work: work, OtherNames: other, ArchivedNames: archived, IndexBlock: block})
		if rep.Passed || !strings.Contains(strings.Join(rep.Violations, "\n"), c.want) {
			t.Errorf("%s：应判违并点名「%s」，got passed=%v violations=%v", c.name, c.want, rep.Passed, rep.Violations)
		}
	}

	// 重复出现：同一条目被列两遍（回归钉——晨报/事实单停注就是为了不再重复）。
	block := renderIndex(work)
	block = strings.TrimSpace(block + "\n" + line(work[0]))
	rep := EvalIndexInjection(InjectionEvalInput{Work: work, IndexBlock: block})
	if rep.Passed || !strings.Contains(strings.Join(rep.Violations, "\n"), "重复出现") {
		t.Errorf("重复条目应判违：passed=%v violations=%v", rep.Passed, rep.Violations)
	}

	// 超预算（无截断提示）判违；带截断提示时后缀计入容差且 Truncated=true。
	base := renderIndex(work)
	over := base + strings.Repeat("x", EvalIndexBudget)
	rep = EvalIndexInjection(InjectionEvalInput{Work: work, IndexBlock: over})
	if rep.Passed || !strings.Contains(strings.Join(rep.Violations, "\n"), "超预算") {
		t.Errorf("超预算应判违：passed=%v violations=%v", rep.Passed, rep.Violations)
	}
	capped := strings.Repeat("a", EvalIndexBudget) + memoryIndexHint
	rep = EvalIndexInjection(InjectionEvalInput{Work: work, IndexBlock: capped})
	if !rep.Passed || !rep.Truncated {
		t.Errorf("截断后缀应计入容差且 Truncated=true：passed=%v truncated=%v violations=%v", rep.Passed, rep.Truncated, rep.Violations)
	}
}

func TestEvalIndexInjectionEmpty(t *testing.T) {
	rep := EvalIndexInjection(InjectionEvalInput{Work: nil, IndexBlock: ""})
	if !rep.Passed || rep.IndexPresent || rep.EntryCount != 0 {
		t.Errorf("空索引=未注入，应只统计不判违：%+v", rep)
	}
}

// TestIndexForInjectionTruncates 真实库 + 真实渲染：条目多到超预算时截断、不切半行。
func TestIndexForInjectionTruncates(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	t.Cleanup(func() { _ = db.CloseDatabase(dir) })
	store := SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	for i := 0; i < 160; i++ {
		m := Memory{Name: "fact-" + string(rune('a'+i%26)) + string(rune('a'+(i/26)%26)) + string(rune('a'+(i/676)%26)) + "-" + strings.Repeat("z", 1+i%3),
			Description: strings.Repeat("描述", 30), Body: "b"}
		if _, err := store.Save(m); err != nil {
			t.Fatal(err)
		}
	}
	block := IndexForInjection(store)
	if len(block) > EvalIndexBudget+len(memoryIndexHint) {
		t.Fatalf("注入索引 %d 字节超预算 %d", len(block), EvalIndexBudget)
	}
	rep := EvalIndexInjection(InjectionEvalInput{Work: store.List(), IndexBlock: block})
	if !rep.Passed || !rep.Truncated || rep.EntryCount == 0 || rep.EntryCount >= len(store.List()) {
		t.Fatalf("应截断且部分进索引：%+v / 库内 %d", rep, len(store.List()))
	}
}
