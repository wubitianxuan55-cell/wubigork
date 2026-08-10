package retrieval

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/gaea/gaea/internal/gaea/cost"
)

// SemanticRecall 语义召回：关键词召回不足时，把候选库向量化后按与查询的
// 余弦相似度补召回 topN 条「已有结果之外」的条目（避免重复）。调用方负责
// 在 Embedder 不可用/失败时回退原结果。纯本地推理，不消耗云端 token。
func SemanticRecall(ctx context.Context, e *Embedder, query string, full, have []cost.Summary, topN int) []cost.Summary {
	if e == nil || strings.TrimSpace(query) == "" || len(full) == 0 || topN <= 0 {
		return nil
	}
	if !e.Available(ctx) {
		return nil
	}
	docs := make([]string, len(full))
	for i, s := range full {
		docs[i] = DocText(s)
	}
	vecs, err := e.Embed(ctx, docs)
	if err != nil || len(vecs) != len(full) {
		return nil
	}
	qvec, err := e.Embed(ctx, []string{query})
	if err != nil || len(qvec) == 0 {
		return nil
	}

	haveNames := make(map[string]bool, len(have))
	for _, s := range have {
		haveNames[s.Name] = true
	}
	type scored struct {
		idx   int
		score float64
	}
	var cand []scored
	for i, v := range vecs {
		if len(v) == 0 || haveNames[full[i].Name] {
			continue
		}
		if s := Cosine(qvec[0], v); s > 0.1 {
			cand = append(cand, scored{idx: i, score: s})
		}
	}
	sort.Slice(cand, func(i, j int) bool { return cand[i].score > cand[j].score })
	if len(cand) > topN {
		cand = cand[:topN]
	}
	out := append([]cost.Summary{}, have...)
	for _, c := range cand {
		out = append(out, full[c.idx])
	}
	return out
}

// DocText 把成本条目摘要拼成向量化/精排文档串。
func DocText(e cost.Summary) string {
	var b strings.Builder
	b.WriteString(e.Title)
	if e.Spec != "" {
		b.WriteString("（" + e.Spec + "）")
	}
	if e.Unit != "" {
		b.WriteString(" 单位" + e.Unit)
	}
	b.WriteString(" 单价" + formatPrice(e.Price) + "元")
	if e.Category != "" {
		b.WriteString(" 分类" + e.Category)
	}
	if e.Source != "" {
		b.WriteString(" 来源" + e.Source)
	}
	if len(e.Tags) > 0 {
		b.WriteString(" 标签" + strings.Join(e.Tags, ","))
	}
	return b.String()
}

func formatPrice(v float64) string {
	return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(v, 'f', 2, 64), "0"), ".")
}
