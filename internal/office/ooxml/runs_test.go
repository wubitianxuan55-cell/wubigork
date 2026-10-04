package ooxml

import "testing"

// TestGroupRunsAndAffectedSpans 钉共享核数学面：分组保序、区间求交半开
// [s,e)、全不命中=ok=false。docx/pptx 两消费方的端到端用例覆盖发射端，
// 这里钉计算面本身。
func TestGroupRunsAndAffectedSpans(t *testing.T) {
	groups := GroupRuns([]TextSegCore{
		{Text: "ab", RunTagStart: 100, RunTagEnd: 105, RunEnd: 120, RPrStart: -1, RPrEnd: 104, TAttrs: ` xml:space="preserve"`},
		{Text: "cd", RunTagStart: 100, RunTagEnd: 105, RunEnd: 120, RPrStart: -1, RPrEnd: 104},
		{Text: "ef", RunTagStart: 200, RunTagEnd: 205, RunEnd: 220, RPrStart: -1, RPrEnd: 204},
	})
	if len(groups) != 2 {
		t.Fatalf("应按 runTag 分成 2 组，实际 %d", len(groups))
	}
	if len(groups[0].Segs) != 2 || groups[0].Segs[1].Text != "cd" {
		t.Fatalf("组内段应保序聚合: %+v", groups[0].Segs)
	}
	if groups[0].End != 120 || groups[0].TAttrs != ` xml:space="preserve"` {
		t.Fatalf("组应取首段的 end/attrs: %+v", groups[0])
	}

	// 段落文本 "abcdef"，选区 [2,5) = "cde"：跨第 1、2 组
	affected, ok := AffectedSpans(groups, 2, 5)
	if !ok || len(affected) != 2 {
		t.Fatalf("选区 [2,5) 应命中 2 组: ok=%v n=%d", ok, len(affected))
	}
	if affected[0].Start != 0 || affected[0].End != 4 || affected[0].DelFrom != 2 || affected[0].DelTo != 4 {
		t.Fatalf("首组（文本 abcd）应覆盖 [2,4)=cd: %+v", affected[0])
	}
	if affected[1].Start != 4 || affected[1].DelFrom != 0 || affected[1].DelTo != 1 {
		t.Fatalf("次组应覆盖 'e': %+v", affected[1])
	}

	// 全不命中
	if _, ok := AffectedSpans(groups, 10, 20); ok {
		t.Fatal("越界选区应 ok=false")
	}
}
