package app

// gaea_cost_search.go — 造价库混合检索的应用侧装配。
//
// 模型依赖固定读 [retrieval] 配置（boot 注入 builtin 的检索运行时；零值 = localhost:8080 +
// bge-m3 / bge-reranker-v2-m3），与方案库同一套客户端。不走 resolveHerdsmanSearchModel
// 那条「引擎模型表自动升级」的路——旧的 semantic_vectors 里已经混着两种维度，新索引
// 不再掺和。
//
// 两个入口的口径：
//   - 模型工具 cost_project：带 keyword 一律走混合检索（编码精确 → 关键词 → 向量 →
//     融合 → 候选多时精排），向量路不可用时自然退化成关键词；
//   - 界面列表绑定：只是浏览（没给关键词）或类目索引还没建时，保持原列表口径（全量
//     返回，既有行为不变）；索引就绪且带关键词时才走混合检索。

import (
	"context"
	"strings"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/retrieval"
	"github.com/gaea/gaea/internal/gaea/tool/builtin"
	"github.com/gaea/gaea/internal/gaea/vecindex"
	"github.com/gaea/gaea/internal/gaea/workcost"
)

// costSearchListLimit 界面列表带关键词时的返回上限（原口径是不截断的 LIKE 全量；
// 混合检索是有序的相关度列表，给前 200 条足够）。
const costSearchListLimit = 200

// 测试注入缝（避免触碰真实库 / 引擎）：非 nil 时优先于真实取值。
var (
	costIndexStoreOverride    *vecindex.Store
	costIndexEmbedderOverride func() workcost.SearchEmbedder
	costIndexRerankerOverride func() workcost.SearchReranker
)

// SetCostIndexForTest 注入隔离的向量索引与模型依赖；返回还原函数。
func SetCostIndexForTest(vs *vecindex.Store, embed func() workcost.SearchEmbedder, rerank func() workcost.SearchReranker) func() {
	costIndexStoreOverride, costIndexEmbedderOverride, costIndexRerankerOverride = vs, embed, rerank
	return func() {
		costIndexStoreOverride, costIndexEmbedderOverride, costIndexRerankerOverride = nil, nil, nil
	}
}

// costIndexStore 造价库向量索引（与应用其它面共用同一个实例与内存缓存）；库没就绪时为 nil。
func (a *App) costIndexStore() *vecindex.Store {
	if costIndexStoreOverride != nil {
		return costIndexStoreOverride
	}
	if a == nil {
		return nil
	}
	return vecindex.Open(db.GetDatabase(config.MemoryUserDir()))
}

// costIndexReady 类目索引是否已建（里面有向量）。
func (a *App) costIndexReady(kind string) bool {
	vs := a.costIndexStore()
	if vs == nil || !vs.Available() {
		return false
	}
	m, err := vs.Meta(kind)
	return err == nil && m.Count > 0
}

// costIndexEmbedder 向量化能力：与方案库同一套客户端；检索后端 kind 未知时返回 nil
// （只走关键词）。注意别把 nil 的 *retrieval.Embedder 装进接口——那是「非 nil 的接口包着
// nil 指针」，调用方的 nil 判断会失效、一调就崩。
func (a *App) costIndexEmbedder() workcost.SearchEmbedder {
	if costIndexEmbedderOverride != nil {
		return costIndexEmbedderOverride()
	}
	e := builtin.LocalEmbedder()
	if e == nil {
		return nil
	}
	return e
}

// costIndexReranker 本地精排；模型当前不可用时 Rerank 返回 nil（检索器按融合分返回）。
func (a *App) costIndexReranker() workcost.SearchReranker {
	if costIndexRerankerOverride != nil {
		return costIndexRerankerOverride()
	}
	r := builtin.LocalReranker()
	if r == nil {
		return nil
	}
	return costRerankAdapter{r: r}
}

// costRerankAdapter 把 retrieval.Reranker 适配成检索器要的形状（只要下标序）。
type costRerankAdapter struct{ r *retrieval.Reranker }

func (c costRerankAdapter) Rerank(ctx context.Context, query string, docs []string, topN int) ([]int, error) {
	if !c.r.Available(ctx) {
		return nil, nil
	}
	scored, err := c.r.Rerank(ctx, query, docs, topN)
	if err != nil {
		return nil, err
	}
	out := make([]int, 0, len(scored))
	for _, s := range scored {
		out = append(out, s.Index)
	}
	return out, nil
}

// costSearcher 造价库混合检索器。依赖都是轻量取值（模型客户端由 builtin 侧缓存）。
func (a *App) costSearcher() *workcost.Searcher {
	if a == nil {
		return nil
	}
	return workcost.NewSearcher(a.hubWorkcostStore(), a.costIndexStore(), a.costIndexEmbedder, a.costIndexReranker)
}

// costSearchEnabled 界面检索是否走混合路：给了关键词，且该类目的向量索引已建。
func (a *App) costSearchEnabled(kind, keyword string) bool {
	return strings.TrimSpace(keyword) != "" && a.costIndexReady(kind)
}
