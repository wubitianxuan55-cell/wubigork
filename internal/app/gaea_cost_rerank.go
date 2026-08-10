package app

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/retrieval"
	"github.com/gaea/gaea/internal/gaea/semantic"
)

// appEmbedderOverride / appSemanticStoreOverride 测试注入（避免触碰真实库/引擎）。
var (
	appEmbedderOverride      *retrieval.Embedder
	appSemanticStoreOverride *semantic.Store
)

// SetAppEmbedderForTest 注入隔离的 embedding 客户端（测试用）。
func SetAppEmbedderForTest(e *retrieval.Embedder) { appEmbedderOverride = e }

// SetAppSemanticStoreForTest 注入隔离的向量索引存储（测试用）。
func SetAppSemanticStoreForTest(s *semantic.Store) { appSemanticStoreOverride = s }

// localSearchReranker 构造本地语义精排客户端：优先用 Herdsman 引擎配置的
// BaseURL（与模型中心一致），模型名默认 bge-reranker-v2-m3，可用环境变量覆盖。
func (a *App) localSearchReranker() *retrieval.Reranker {
	if a.engineMgr != nil {
		if eng, ok := a.engineMgr.GetEngine("herdsman"); ok && eng.Enabled && eng.BaseURL != "" {
			model := strings.TrimSpace(os.Getenv("HERDSMAN_RERANK_MODEL"))
			if model == "" {
				model = "bge-reranker-v2-m3"
			}
			return retrieval.New(eng.BaseURL, model)
		}
	}
	return nil
}

// localSearchEmbedder 构造本地 embedding 客户端（bge-m3，引擎配置取 Herdsman）。
func (a *App) localSearchEmbedder() *retrieval.Embedder {
	if appEmbedderOverride != nil {
		return appEmbedderOverride
	}
	if a.engineMgr != nil {
		if eng, ok := a.engineMgr.GetEngine("herdsman"); ok && eng.Enabled && eng.BaseURL != "" {
			model := strings.TrimSpace(os.Getenv("HERDSMAN_EMBED_MODEL"))
			if model == "" {
				model = "bge-m3"
			}
			return retrieval.NewEmbedder(eng.BaseURL, model)
		}
	}
	return nil
}

// semanticCostRecall 语义召回：持久化向量索引（增量向量化 + 查询只嵌 query）。
func (a *App) semanticCostRecall(query string, have []cost.Summary, topN int) []cost.Summary {
	e := a.localSearchEmbedder()
	if e == nil {
		return nil
	}
	full := a.hubCostStore().List()
	if len(full) == 0 {
		return nil
	}
	st := a.hubSemanticStore()
	if st == nil || !st.Available() {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	docs := make([]semantic.Doc, len(full))
	keep := make(map[string]bool, len(full))
	for i, s := range full {
		docs[i] = semantic.Doc{ID: s.Name, Text: retrieval.DocText(s)}
		keep[s.Name] = true
	}
	_, _ = st.Stale("cost", keep)
	hits, err := st.Search(ctx, e, "cost", docs, query, topN)
	if err != nil || len(hits) == 0 {
		return nil
	}
	haveNames := make(map[string]bool, len(have))
	for _, s := range have {
		haveNames[s.Name] = true
	}
	byName := make(map[string]cost.Summary, len(full))
	for _, s := range full {
		byName[s.Name] = s
	}
	out := append([]cost.Summary{}, have...)
	for _, h := range hits {
		if s, ok := byName[h.ID]; ok && !haveNames[h.ID] {
			out = append(out, s)
		}
	}
	return out
}

// hubSemanticStore 构造共享向量索引存储（Hephaestus.db SchemaV5）。
func (a *App) hubSemanticStore() *semantic.Store {
	if appSemanticStoreOverride != nil {
		return appSemanticStoreOverride
	}
	return semantic.Open(db.GetDatabase(config.MemoryUserDir()))
}

// rerankCostSearch 对 SQL 粗召回结果做本地精排（topN）；模型不可用或失败时
// 返回 nil，调用方回退原顺序。候选少于 9 条或空查询不精排（开销不值得）。
func (a *App) rerankCostSearch(query string, list []cost.Summary, limit int) []cost.Summary {
	if len(list) <= 8 || strings.TrimSpace(query) == "" || limit <= 0 {
		return nil
	}
	r := a.localSearchReranker()
	if r == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if !r.Available(ctx) {
		return nil
	}
	docs := make([]string, len(list))
	for i, e := range list {
		docs[i] = costDocString(e)
	}
	scored, err := r.Rerank(ctx, query, docs, limit)
	if err != nil || len(scored) == 0 {
		return nil
	}
	out := make([]cost.Summary, 0, len(scored))
	for _, s := range scored {
		if s.Index >= 0 && s.Index < len(list) {
			out = append(out, list[s.Index])
		}
	}
	return out
}

func costDocString(e cost.Summary) string {
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
	return strconv.FormatFloat(v, 'f', 2, 64)
}
