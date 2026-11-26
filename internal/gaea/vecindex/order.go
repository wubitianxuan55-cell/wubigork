package vecindex

// order.go — 混合检索的融合排序：关键词命中 + 向量命中 → 一条可解释的顺序。

import "sort"

// Cand 混合检索里的一条候选，带各路的命中情况。
type Cand struct {
	ID string
	// Exact 编码 / 名称与查询词完全相同：定了就是它，恒排最前。
	Exact bool
	// Title 标题含查询词（最强的意图信号）。
	Title bool
	// Field 标题之外的字段含查询词（规格 / 章节 / 分类 / 特征…）。
	Field bool
	// Vec 向量余弦（向量路没命中或没开 = 0）。
	Vec float64
}

// 融合权重（与 semantic.FuseOrdered 同一套数，三路分量各归一到 [0,1]）：
//
//	final = 0.55·向量 + 0.30·关键词 + 0.15·标题 + 编码精确加 1.0
//
// 标题命中 = 关键词 1.0 + 标题 1.0 = 0.45，与余弦 0.82 的纯语义命中相当；两路都中
// 直接到 0.9 以上。只命中其他字段的 0.18——给个露脸的机会，排不到前面。
const (
	WVec   = 0.55
	WFT    = 0.30
	WName  = 0.15
	WExact = 1.0
)

// Score 算一条候选的融合分。
func Score(c Cand) float64 {
	vec := c.Vec
	if vec < 0 {
		vec = 0
	} else if vec > 1 {
		vec = 1
	}
	ft := 0.0
	if c.Title {
		ft = 1
	} else if c.Field {
		ft = 0.6
	}
	name := 0.0
	if c.Title {
		name = 1
	}
	s := WVec*vec + WFT*ft + WName*name
	if c.Exact {
		s += WExact
	}
	return s
}

// Keyword 这条候选是不是关键词路命中（编码精确 / 标题 / 其他字段）。
func (c Cand) Keyword() bool { return c.Exact || c.Title || c.Field }

// Order 按融合分降序排（改的是入参切片）。同分时关键词命中排在纯语义命中前面
// （说得出理由的优先），再按 ID 字典序，结果确定。
func Order(in []Cand) []Cand {
	sort.SliceStable(in, func(i, j int) bool {
		a, b := in[i], in[j]
		sa, sb := Score(a), Score(b)
		if sa != sb {
			return sa > sb
		}
		if a.Keyword() != b.Keyword() {
			return a.Keyword()
		}
		return a.ID < b.ID
	})
	return in
}
