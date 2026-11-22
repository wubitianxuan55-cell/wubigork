package memory

import (
	"testing"
)

func vec3(a, b, c float32) []float32 { return []float32{a, b, c} }

// VectorSimilarPairs：阈值、确定性排序、零向量跳过、perNameCap 与 maxPairs 护栏。
func TestVectorSimilarPairs(t *testing.T) {
	vecs := map[string][]float32{
		"a": vec3(1, 0, 0),
		"b": vec3(1, 0, 0),     // 与 a 同向 → cos=1
		"c": vec3(0.9, 0.1, 0), // 与 a 高相似
		"d": vec3(0, 0, 1),     // 正交
		"e": vec3(0, 0, 0),     // 零向量跳过
	}
	pairs := VectorSimilarPairs([]string{"a", "b", "c", "d", "e"}, vecs, 0.85, 20000, 8)
	// 期望：a-b(1.0)、a-c/b-c 高分；a-d、b-d 正交不报；e 无向量不报。
	if len(pairs) != 3 {
		t.Fatalf("len=%d want 3: %+v", len(pairs), pairs)
	}
	if pairs[0].A != "a" || pairs[0].B != "b" || pairs[0].Score < 0.9999 {
		t.Fatalf("top=%+v want a-b 1.0", pairs[0])
	}
	for _, p := range pairs {
		if p.Score < 0.85 {
			t.Fatalf("低于阈值混入：%+v", p)
		}
	}
	// 确定性：再跑一遍逐项相等。
	again := VectorSimilarPairs([]string{"a", "b", "c", "d", "e"}, vecs, 0.85, 20000, 8)
	for i := range pairs {
		if pairs[i] != again[i] {
			t.Fatalf("非确定性：%+v vs %+v", pairs[i], again[i])
		}
	}
}

// perNameCap：巨簇收敛——3 条全同向只产 2 对（每名 ≤2）。
func TestVectorSimilarPairsPerNameCap(t *testing.T) {
	vecs := map[string][]float32{
		"x1": vec3(1, 0, 0),
		"x2": vec3(1, 0, 0),
		"x3": vec3(1, 0, 0),
	}
	pairs := VectorSimilarPairs([]string{"x1", "x2", "x3"}, vecs, 0.85, 20000, 2)
	// 契约=每名参与 ≤2 对（3 条全同向恰 3 对、每名恰 2）——巨簇按名收敛而非
	// 全局截断；全局截断走 maxPairs。
	if len(pairs) != 3 {
		t.Fatalf("perNameCap 语义漂移：%+v", pairs)
	}
	counts := map[string]int{}
	for _, p := range pairs {
		counts[p.A]++
		counts[p.B]++
	}
	for name, n := range counts {
		if n > 2 {
			t.Fatalf("%s 参与 %d 对超 cap", name, n)
		}
	}
	// maxPairs 早停：阈值 0 → 全对候选，上限 1。
	all := VectorSimilarPairs([]string{"x1", "x2", "x3"}, vecs, 0.01, 1, 8)
	if len(all) != 1 {
		t.Fatalf("maxPairs 早停失效：%+v", all)
	}
}
