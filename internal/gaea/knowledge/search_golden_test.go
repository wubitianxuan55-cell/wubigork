package knowledge

import (
	"testing"
)

// goldenCorpus 是 golden 榜单的固定语料。条目形状是为合分口径敏感性设计的：
//
//	f/g-twin      同文异名孪生：kw=10 且 vec 逐位相等 → 总分精确并列，
//	              验证并列分按 Name 升序的稳定序（sort.Slice 的次级键）。
//	c-mixed       标题命中(+10)+正文命中(+1)+向量分 → 混合口径榜首梯队。
//	a-kw-title    仅标题命中(+10)、向量分低 → 纯关键词口径。
//	v-body-strongvec 正文命中(+1)但向量分极高(≈0.594) → 靠 vec*vecScale
//	              压过 t-tag-only(kw=5,vec=0)。vecScale 8→4 会让两者换位，
//	              golden 榜单因此对 vecScale 敏感（变异即红）。
//	t-tag-only    仅标签命中(+5)；标签不入索引 → vec=0，纯关键词对照面。
//	b-vec-only    关键词零命中但 vec≈0.137 ∈ [0.1, 0.3) → 恰好不被淘汰；
//	              淘汰线 0.1→0.3 会把它踢出榜单（变异即红）。
//	e-near-boundary 关键词零命中且 vec≈0.080 < 0.1 → 被淘汰线剔除；
//	              单独查询「氧化铁皮」时标题命中 kw=11 → 又是纯关键词面。
//	d-unrelated   与查询零词元重叠，vec=0 → 淘汰线剔除。
func goldenCorpus() []Entry {
	return []Entry{
		{Name: "a-kw-title", Title: "化学氧化技术导则", Category: CatStandard, Body: "本导则规定药剂投加与反应时长的一般要求。"},
		{Name: "b-vec-only", Title: "场地修复药剂选型手册", Category: CatCase, Body: "药剂选型需结合氧化还原电位与化学相容性试验，避免二次污染。"},
		{Name: "c-mixed", Title: "化学氧化修复工程案例", Category: CatCase, Body: "某化工地块采用化学氧化工艺，药剂投加前先做氧化还原电位测定。"},
		{Name: "d-unrelated", Title: "语音陪伴机器人", Category: CatOther, Body: "语音识别与合成实现对话交互，支持离线唤醒。"},
		{Name: "e-near-boundary", Title: "氧化铁皮形成机理", Category: CatMaterial, Body: "热轧表面氧化铁皮由三层铁氧化物构成。"},
		{Name: "f-twin-1", Title: "化学氧化", Category: CatDesign, Body: "工艺要点汇编。"},
		{Name: "g-twin-2", Title: "化学氧化", Category: CatDesign, Body: "工艺要点汇编。"},
		{Name: "t-tag-only", Title: "药剂管理制度", Category: CatStandard, Body: "入库登记与领用台账要求。", Tags: []string{"化学氧化"}},
		{Name: "v-body-strongvec", Title: "工艺选型笔记", Category: CatOther, Body: "化学氧化。化学。氧化。"},
	}
}

func newGoldenStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range goldenCorpus() {
		if err := s.Save(e); err != nil {
			t.Fatalf("Save(%s): %v", e.Name, err)
		}
	}
	return s
}

// TestSearchGoldenRanking 冻结固定语料 + 固定查询集的期望榜单顺序（golden）。
// 期望顺序来源于当前合分口径（vecScale=8、淘汰线 0.1、索引 minScore 0.02、
// scoreEntry 权重 10/5/3/3/1）的实测分值；任何常数或合分口径改动都会使
// 榜单顺序变化、本测试变红 —— 这是 GA3-07「行为不变」的验收基线。
func TestSearchGoldenRanking(t *testing.T) {
	s := newGoldenStore(t)

	cases := []struct {
		name  string
		query string
		param Filter
		want  []string // 冻结的期望榜单（按 Name）
	}{
		{
			// 混合榜单：孪生并列居首（Name 稳定序）→ 混合 → 纯关键词 →
			// 向量增强的正文命中 → 纯标签 → 恰好过淘汰线的纯向量。
			// v-body-strongvec(1+0.594*8=5.75) 压过 t-tag-only(5+0*8=5.00)，
			// b-vec-only(kw=0,vec=0.137≥0.1) 恰好不被淘汰；
			// e-near-boundary(kw=0,vec=0.080<0.1) 与 d-unrelated(vec=0) 被剔除。
			name:  "混合口径榜单",
			query: "化学氧化",
			want:  []string{"f-twin-1", "g-twin-2", "c-mixed", "a-kw-title", "v-body-strongvec", "t-tag-only", "b-vec-only"},
		},
		{
			// 纯关键词面：仅 e-near-boundary 词面命中(kw=11)；其余候选
			// kw=0 且 vec<0.1 全部被淘汰线剔除。
			name:  "纯关键词命中其余淘汰",
			query: "氧化铁皮",
			want:  []string{"e-near-boundary"},
		},
		{
			// 空查询：全部候选保底分 kw=1、vec=0，总分精确并列 →
			// 整张榜单退化为 Name 升序的稳定序（不含淘汰）。
			name:  "空查询全量稳定序",
			query: "",
			want:  []string{"a-kw-title", "b-vec-only", "c-mixed", "d-unrelated", "e-near-boundary", "f-twin-1", "g-twin-2", "t-tag-only", "v-body-strongvec"},
		},
		{
			// 空查询 + 分类过滤：过滤后的候选同样按 Name 稳定序。
			name:  "空查询分类过滤稳定序",
			query: "",
			param: Filter{Category: CatDesign},
			want:  []string{"f-twin-1", "g-twin-2"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Search(s, tc.query, tc.param)
			if err != nil {
				t.Fatalf("Search(%q): %v", tc.query, err)
			}
			names := make([]string, len(got))
			for i, e := range got {
				names[i] = e.Name
			}
			if len(names) != len(tc.want) {
				t.Fatalf("golden 榜单长度 = %d (%v), want %d\n实际顺序: %v", len(names), names, len(tc.want), names)
			}
			for i := range tc.want {
				if names[i] != tc.want[i] {
					t.Fatalf("golden 榜单顺序变化：位置 %d = %q, want %q\n实际顺序: %v\n期望顺序: %v",
						i, names[i], tc.want[i], names, tc.want)
				}
			}
		})
	}
}

// TestCombineScorePins 合分口径的数值钉：vecScale=8 的精确含义。
// vecScale 改成任何其它值，这里的期望值即不再成立（红）。
func TestCombineScorePins(t *testing.T) {
	cases := []struct {
		kw   int
		vec  float64
		want float64
	}{
		{0, 0, 0},
		{1, 0, 1},                       // 纯正文命中无向量贡献
		{10, 0, 10},                     // 纯标题命中无向量贡献
		{0, 1, 8},                       // 满分向量 ≡ vecScale
		{10, 0.5, 14},                   // 10 + 0.5*8
		{5, 0.25, 7},                    // 5 + 0.25*8 = 7（恰好追平标题档的临界形状）
		{11, 0.230150, 11 + 8*0.230150}, // 实测 golden 语料 c-mixed 的量级
	}
	for _, tc := range cases {
		if got := combineScore(tc.kw, tc.vec); got != tc.want {
			t.Errorf("combineScore(%d, %v) = %v, want %v（合分口径或 vecScale 被改动）", tc.kw, tc.vec, got, tc.want)
		}
	}
}

// TestIsIrrelevantBoundary 淘汰线的边界口径：vec 恰等于 0.1 不淘汰（<），
// 略低于即淘汰；任一关键词命中即不淘汰。淘汰线改动（如 0.1→0.3）即红。
func TestIsIrrelevantBoundary(t *testing.T) {
	cases := []struct {
		kw   int
		vec  float64
		want bool
	}{
		{0, 0, true},         // 零命中零向量：无关
		{0, 0.0999, true},    // 略低于淘汰线：无关
		{0, 0.1, false},      // 恰在淘汰线上：保留（< 严格小于）
		{0, 0.137296, false}, // golden 语料 b-vec-only 的量级：保留
		{1, 0, false},        // 有关键词命中：保留
		{5, 0, false},        // 纯标签命中：保留
	}
	for _, tc := range cases {
		if got := isIrrelevant(tc.kw, tc.vec); got != tc.want {
			t.Errorf("isIrrelevant(%d, %v) = %v, want %v（淘汰线口径被改动）", tc.kw, tc.vec, got, tc.want)
		}
	}
}
