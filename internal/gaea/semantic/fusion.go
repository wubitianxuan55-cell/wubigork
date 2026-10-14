package semantic

// ── 三路融合排序（阶段九 9.2「三路融合权重可解释」）──────────
//
// 融合公式（权重常量+来源注释，可解释性的承诺：任何一条命中为什么排这里，
// 都能用三个分量分与权重复算出来）：
//
//	final = W_VEC*vec + W_FT*fulltext + W_NAME*name
//
//	向量路  W_VEC=0.55  cosine ∈ [0,1]，语义变体召回主力
//	全文路  W_FT =0.30  归一化 bm25/命中词数，精确词（型号/编号/专名）主力
//	名称路  W_NAME=0.15 名称含查询词（最强意图信号，命中即拉满 1.0）
//
// 权重和=1，final ∈ [0,1]。时序 tie-break：同分按 updated_at 新者优先（同一
// 事实的新版本比旧版本更有用），再新者相同按 kind+id 字典序（确定性）。

import (
	"sort"
	"strings"
)

// 融合权重（可解释性的单一声明点；调整须同步本注释与测试基线）。
const (
	FuseWeightVec  = 0.55
	FuseWeightFT   = 0.30
	FuseWeightName = 0.15
)

// FuseInput 单条命中的三路分量（各路负责归一到 [0,1]）。
type FuseInput struct {
	ID       string
	Kind     string
	VecScore float64 // cosine [0,1]；无向量路=0
	FTScore  float64 // 全文路归一分 [0,1]；无全文命中=0
	NameHit  bool    // 名称含查询词
	// UpdatedAtRFC3339 时序 tie-break（新者优先；空=最旧）。
	UpdatedAtRFC3339 string
}

// FusedHit 融合后的排序命中（含分量与加权分——可解释性的 wire 面）。
type FusedHit struct {
	ID       string
	Kind     string
	VecScore float64
	FTScore  float64
	NameHit  bool
	Final    float64
	// UpdatedAtRFC3339 时序 tie-break 的透出（新者优先；空=最旧）。
	UpdatedAtRFC3339 string
}

// FuseOrdered 三路融合+时序 tie-break。纯函数（输入即全部事实，零 IO）。
func FuseOrdered(in []FuseInput) []FusedHit {
	out := make([]FusedHit, 0, len(in))
	for _, it := range in {
		name := 0.0
		if it.NameHit {
			name = 1.0
		}
		out = append(out, FusedHit{
			ID:               it.ID,
			Kind:             it.Kind,
			VecScore:         it.VecScore,
			FTScore:          it.FTScore,
			NameHit:          it.NameHit,
			Final:            FuseWeightVec*it.VecScore + FuseWeightFT*it.FTScore + FuseWeightName*name,
			UpdatedAtRFC3339: it.UpdatedAtRFC3339,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Final != b.Final {
			return a.Final > b.Final
		}
		// 时序 tie-break：updated_at 新者优先（RFC3339 同构格式下字典序=时间序；
		// 空串=最旧）。再同则 kind+id 字典序（确定性）。
		if a.UpdatedAtRFC3339 != b.UpdatedAtRFC3339 {
			return a.UpdatedAtRFC3339 > b.UpdatedAtRFC3339
		}
		return a.Kind+a.ID < b.Kind+b.ID
	})
	return out
}

// FuseSemantic 三路融合入口（app 接线用）：向量命中 + 全文命中 → 名称路判定 →
// FuseOrdered。updatedAt 为 (kind+DELIM+id) → RFC3339 的时序表（Store.
// Timestamps 批量取），FTS 路自带者优先。
func FuseSemantic(query string, vecHits []Hit, ftHits []FTSHit, updatedAt map[string]string, topN int) []FusedHit {
	terms := SplitQueryTerms(query)
	maxFT := 0.0
	for _, h := range ftHits {
		if h.Score > maxFT {
			maxFT = h.Score
		}
	}
	inputs := map[string]*FuseInput{}
	var order []*FuseInput
	add := func(kind, id string) *FuseInput {
		key := kind + "\x00" + id
		if _, ok := inputs[key]; !ok {
			ts := updatedAt[key]
			in := &FuseInput{ID: id, Kind: kind, UpdatedAtRFC3339: ts}
			inputs[key] = in
			order = append(order, in)
		}
		return inputs[key]
	}
	for _, h := range vecHits {
		in := add(h.Kind, h.ID)
		v := h.Score
		if v < 0 {
			v = 0
		}
		if v > 1 {
			v = 1
		}
		in.VecScore = v
	}
	for _, h := range ftHits {
		in := add(h.Kind, h.ID)
		if maxFT > 0 {
			in.FTScore = h.Score / maxFT
		}
		if h.UpdatedAt != "" {
			in.UpdatedAtRFC3339 = h.UpdatedAt
		}
	}
	// 名称路：条目名（id）含任一查询词=最强意图信号
	for _, in := range order {
		name := strings.ToLower(in.ID)
		for _, t := range terms {
			if strings.Contains(name, strings.ToLower(t)) {
				in.NameHit = true
				break
			}
		}
	}
	ins := make([]FuseInput, len(order))
	for i, o := range order {
		ins[i] = *o
	}
	out := FuseOrdered(ins)
	if topN > 0 && len(out) > topN {
		out = out[:topN]
	}
	return out
}
