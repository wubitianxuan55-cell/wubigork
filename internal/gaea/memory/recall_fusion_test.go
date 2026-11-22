package memory

import (
	"math"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// FuseRecallArms：加权、臂归因、排序（Final 降序→pinned 优先→name 字典序）。
func TestFuseRecallArmsWeightsAndAttribution(t *testing.T) {
	in := []RecallFuseInput{
		{Name: "alpha", Lexical: 1.0, Semantic: 0.5},
		{Name: "beta", Semantic: 1.0},
		{Name: "gamma", Pinned: true},
		{Name: "delta", Graph: 1.0},
		{Name: "epsilon", Lexical: 0.25, Semantic: 0.25, Graph: 0.25, Pinned: true},
	}
	out := FuseRecallArms(in)
	if len(out) != len(in) {
		t.Fatalf("len=%d want %d", len(out), len(in))
	}
	byName := map[string]RecallFusedHit{}
	for _, h := range out {
		byName[h.Name] = h
	}
	// alpha：0.40*1 + 0.35*0.5 = 0.575，双臂归因。
	a := byName["alpha"]
	if !approx(a.Final, 0.575) {
		t.Fatalf("alpha Final=%v want 0.575", a.Final)
	}
	if len(a.Arms) != 2 || a.Arms[0] != RecallArmLexical || a.Arms[1] != RecallArmSemantic {
		t.Fatalf("alpha Arms=%v", a.Arms)
	}
	// beta：单语义臂 0.35。
	if b := byName["beta"]; !approx(b.Final, RecallWeightSemantic) {
		t.Fatalf("beta Final=%v want %v", b.Final, RecallWeightSemantic)
	}
	// gamma：纯置顶 0.10 + pinned 臂归因。
	g := byName["gamma"]
	if !approx(g.Final, RecallWeightPinned) || !g.Pinned {
		t.Fatalf("gamma Final=%v pinned=%v", g.Final, g.Pinned)
	}
	// epsilon：三臂+置顶 = (0.4+0.35+0.15)*0.25 + 0.10 = 0.325。
	e := byName["epsilon"]
	if !approx(e.Final, 0.325) {
		t.Fatalf("epsilon Final=%v want 0.325", e.Final)
	}
	// 排序：alpha(0.575) > epsilon(0.325) > beta(0.35)?——beta 0.35 > 0.325，
	// 实际序 alpha > beta > epsilon > delta(0.15) > gamma(0.10)。
	want := []string{"alpha", "beta", "epsilon", "delta", "gamma"}
	for i, name := range want {
		if out[i].Name != name {
			t.Fatalf("rank %d = %s want %s", i, out[i].Name, name)
		}
	}
}

// 同分 tie-break：pinned 优先，再 name 字典序。
func TestFuseRecallArmsTieBreak(t *testing.T) {
	in := []RecallFuseInput{
		{Name: "zeta", Semantic: 1.0},
		{Name: "alpha", Semantic: 1.0},
		{Name: "pinned-equal", Semantic: 1.0, Pinned: true},
	}
	out := FuseRecallArms(in)
	// pinned-equal 带置顶加成分更高（0.45），应排第一；alpha/zeta 同分按名字序。
	want := []string{"pinned-equal", "alpha", "zeta"}
	for i, name := range want {
		if out[i].Name != name {
			t.Fatalf("rank %d = %s want %s", i, out[i].Name, name)
		}
	}
	// 纯置顶同分对：pinned 优先于未置顶。
	tie := FuseRecallArms([]RecallFuseInput{
		{Name: "aaa", Pinned: true, Lexical: 0.25},
		{Name: "bbb", Lexical: 0.25, Semantic: 0.1},
	})
	if tie[0].Name != "aaa" || !tie[0].Pinned {
		t.Fatalf("tie0=%s pinned=%v", tie[0].Name, tie[0].Pinned)
	}
}

// 休眠惩罚：×0.55 且 pinned 豁免；臂归因不受影响。
func TestFuseRecallArmsDormantPenalty(t *testing.T) {
	out := FuseRecallArms([]RecallFuseInput{
		{Name: "sleepy", Lexical: 1.0, Dormant: true},
		{Name: "pinned-sleepy", Lexical: 1.0, Dormant: true, Pinned: true},
		{Name: "awake", Lexical: 1.0},
	})
	byName := map[string]RecallFusedHit{}
	for _, h := range out {
		byName[h.Name] = h
	}
	if !approx(byName["awake"].Final, RecallWeightLexical) {
		t.Fatalf("awake=%v", byName["awake"].Final)
	}
	if !approx(byName["sleepy"].Final, RecallWeightLexical*DormantPenalty) {
		t.Fatalf("sleepy=%v", byName["sleepy"].Final)
	}
	if !approx(byName["pinned-sleepy"].Final, (RecallWeightLexical+RecallWeightPinned)) {
		t.Fatalf("pinned-sleepy=%v（pinned 应豁免休眠惩罚）", byName["pinned-sleepy"].Final)
	}
	if byName["sleepy"].Arms[0] != RecallArmLexical {
		t.Fatalf("sleepy Arms=%v", byName["sleepy"].Arms)
	}
}

// 分量越界夹取：负分与 >1 分都不放大。
func TestFuseRecallArmsClamp(t *testing.T) {
	out := FuseRecallArms([]RecallFuseInput{{Name: "x", Lexical: -1, Semantic: 5}})
	h := out[0]
	if h.Lexical != 0 || h.Semantic != 1 {
		t.Fatalf("lex=%v sem=%v", h.Lexical, h.Semantic)
	}
	if !approx(h.Final, RecallWeightSemantic) {
		t.Fatalf("Final=%v", h.Final)
	}
}

// ExpandGraphSeeds：1 跳扩散、×0.5 衰减、同邻居取最高种子分、种子自身不产出。
func TestExpandGraphSeeds(t *testing.T) {
	adj := map[string][]string{
		"seed-a": {"nb-1", "nb-2"},
		"seed-b": {"nb-2", "seed-a"}, // 回边指向种子自身 → 不产出
	}
	seeds := []RecallArmHit{
		{Name: "seed-a", Score: 1.0},
		{Name: "seed-b", Score: 0.6},
	}
	out := ExpandGraphSeeds(seeds, adj)
	if len(out) != 2 {
		t.Fatalf("len=%d want 2（seed-a 自身与回边不产出）", len(out))
	}
	if out["nb-1"] != GraphArmDecay*1.0 {
		t.Fatalf("nb-1=%v", out["nb-1"])
	}
	if out["nb-2"] != GraphArmDecay*1.0 {
		t.Fatalf("nb-2 应取最高种子分（seed-a 1.0）=%v，got %v", GraphArmDecay, out["nb-2"])
	}
	if _, ok := out["seed-a"]; ok {
		t.Fatal("种子自身不应出现在邻居结果里")
	}
	// 无邻接表的记忆零产出。
	empty := ExpandGraphSeeds([]RecallArmHit{{Name: "lonely", Score: 1}}, adj)
	if len(empty) != 0 {
		t.Fatalf("lonely 产出 %v", empty)
	}
}
