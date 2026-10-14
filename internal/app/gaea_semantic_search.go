package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/knowledge"
	"github.com/gaea/gaea/internal/gaea/retrieval"
	"github.com/gaea/gaea/internal/gaea/semantic"
)

// SemanticHitView 是跨库统一语义检索的命中视图。
type SemanticHitView struct {
	Kind  string  `json:"kind"` // cost / knowledge / office / file
	Name  string  `json:"name"`
	Score float64 `json:"score"`
	Text  string  `json:"text"`
}

// SemanticIndexStatus 各库向量索引状态（D3-1：持久化向量索引可见性）。
type SemanticIndexStatus struct {
	Available bool              `json:"available"`
	Counts    map[string]int    `json:"counts"` // kind → 向量条数
	Detail    map[string]string `json:"detail,omitempty"`
	Error     string            `json:"error,omitempty"`
	// v4.196 成本类覆盖细化：total=正文非空的成本条目数；indexed=「向量在且
	// 正文快照一致」的数量（与 Ensure 缺失判定同口径，正文变了算未覆盖）。
	CostTotal   int `json:"costTotal,omitempty"`
	CostIndexed int `json:"costIndexed,omitempty"`
}

// GaeaSemanticIndexStatus 返回向量索引（semantic_vectors）各 kind 的条数，
// 供前端展示「跨库语义检索」索引健康度。
func (a *App) GaeaSemanticIndexStatus() SemanticIndexStatus {
	e := a.localSearchEmbedder()
	if e == nil {
		return SemanticIndexStatus{Available: false, Error: "本地 embedding 未配置（Herdsman bge-m3）"}
	}
	st := a.hubSemanticStore()
	if st == nil || !st.Available() {
		return SemanticIndexStatus{Available: false, Error: "向量索引存储不可用"}
	}
	counts, err := st.Counts()
	if err != nil {
		return SemanticIndexStatus{Available: false, Error: err.Error()}
	}
	view := SemanticIndexStatus{Available: true, Counts: counts}
	if cov, err := a.costIndexCoverage(); err == nil {
		view.CostTotal, view.CostIndexed = cov.Total, cov.Indexed
	}
	return view
}

// GaeaSemanticSearch 跨库统一语义检索（成本/知识/办公记忆/工作区资料），本地
// bge-m3。向量索引持久化在 Hephaestus.db semantic_vectors；查询路径按需/缓存：
// 索引就绪（向量条数与源文档数一致）时直接 SearchReady（只嵌 query），不再每
// 查询先 Ensure 全量扫描比对；未就绪（首次/数据增删）才按需增量 Ensure。
// 资料（file）依赖文件索引定时维护（每 10 分钟增量重建），检索时不再扫描。
// embedding 不可用/未配置时返回空（不阻断关键词检索）。
func (a *App) GaeaSemanticSearch(query string) ([]SemanticHitView, error) {
	// 绑定面缺省 topN=20（与既有硬截上限一致，wire 契约零变更）。
	return a.semanticSearchHitsOnDemand(query, 20)
}

// semanticSearchHitsOnDemand 跨库语义检索的按需/缓存实现（T7-3「名实相符」）：
// 避免每查询先 Ensure 全量扫描（vectorDocs 全表比对 + 潜在重嵌）。判断依据为
// kind 的向量条数与源文档数是否一致：
//   - 一致 → 索引已就绪（缓存命中），直接 SearchReady（每 kind 只嵌 query）；
//   - 不一致 → 数据有增删（或首次访问），按需增量 Ensure + Stale 清理后查询。
//
// 内容快照比对（Ensure 的正文变化检测）仍保留——在数据增删时自动重嵌；查询
// 热路径不再承担该扫描成本。file 向量由后台定时任务/文件监听维护，直接查库。
func (a *App) semanticSearchHitsOnDemand(query string, topN int) ([]SemanticHitView, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	st := a.hubSemanticStore()
	if st == nil || !st.Available() {
		return nil, nil
	}

	// ── 全文路（阶段九 9.2 混合召回）：零模型依赖，embedding 不可用时独立工作
	// （降级红利：检索不再被模型可用性绑架）。惰性 FTS 维护失败只 warn 不阻断。
	if _, err := st.EnsureFTS(); err != nil {
		slog.Warn("语义 FTS 索引维护失败（全文路降级跳过）", "error", err)
	}
	ftHits, ferr := st.FTSSearch(query, topN)
	if ferr != nil {
		slog.Warn("语义全文检索失败（全文路降级跳过）", "error", ferr)
		ftHits = nil
	}

	e := a.localSearchEmbedder()
	if e == nil {
		// 无 embedding：全文+名称两路独立工作（混合召回的降级语义）。
		ts, _ := st.Timestamps()
		fused := semantic.FuseSemantic(query, nil, ftHits, ts, topN)
		return fusedToViews(fused, st), nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// 一次 COUNT 查询拿到各 kind 现有向量条数（廉价信号，替代每查询全量扫描）。
	counts, err := st.Counts()
	if err != nil {
		return nil, err
	}

	// 成本库
	costList, cerr := a.hubCostStore().List()
	if cerr != nil {
		// 审计 GA6-09：读失败如实上抛——按空库跳过会把索引面静默清空。
		return nil, fmt.Errorf("成本库暂不可读（读取失败），可重试: %w", cerr)
	}
	if len(costList) > 0 {
		docs := make([]semantic.Doc, 0, len(costList))
		keep := make(map[string]bool, len(costList))
		for _, s := range costList {
			docs = append(docs, semantic.Doc{ID: s.Name, Text: retrieval.DocText(s)})
			keep[s.Name] = true
		}
		if counts["cost"] != len(docs) {
			if _, err := st.Ensure(ctx, e, "cost", docs); err != nil {
				return nil, err
			}
			_, _ = st.Stale("cost", keep)
		}
	}

	// 工程知识库
	if ks, err := knowledge.Global().Store(); err == nil {
		entries, rerr := ks.ReadAll()
		if rerr != nil {
			// 审计 P0#6 GA3-01：读失败如实上抛，不得当「知识库为空」跳过索引。
			return nil, fmt.Errorf("知识库暂不可读（读取失败），可重试: %w", rerr)
		}
		if len(entries) > 0 {
			docs := make([]semantic.Doc, 0, len(entries))
			keep := make(map[string]bool, len(entries))
			for _, e2 := range entries {
				docs = append(docs, semantic.Doc{
					ID:   e2.Name,
					Text: e2.Title + " " + e2.Category + " " + e2.Body,
				})
				keep[e2.Name] = true
			}
			if counts["knowledge"] != len(docs) {
				if _, err := st.Ensure(ctx, e, "knowledge", docs); err != nil {
					return nil, err
				}
				_, _ = st.Stale("knowledge", keep)
			}
		}
	}

	// 办公记忆 facts
	facts := a.hubOfficeStore().List()
	if len(facts) > 0 {
		docs := make([]semantic.Doc, 0, len(facts))
		keep := make(map[string]bool, len(facts))
		for _, m := range facts {
			docs = append(docs, semantic.Doc{
				ID:   m.Name,
				Text: m.Title + " " + m.Description + " " + m.Body,
			})
			keep[m.Name] = true
		}
		if counts["office"] != len(docs) {
			if _, err := st.Ensure(ctx, e, "office", docs); err != nil {
				return nil, err
			}
			_, _ = st.Stale("office", keep)
		}
	}

	// 工作区资料：文件索引由定时任务维护，docs=nil → 直接 SearchReady 查库。
	// 审计 P1 AP5-05：perKind 与总截由调用方 topN 推导（此前固定 6/20，
	// 调用方声明的召回口径形同虚设——评测 Recall@10 传 10 即得 10 条口径）。
	if topN <= 0 {
		topN = 20
	}
	kinds := []string{"cost", "knowledge", "office", "file"}
	perKind := topN / len(kinds)
	if perKind < 1 {
		perKind = 1
	}
	var vecHits []semantic.Hit
	for _, kind := range kinds {
		hits, err := st.SearchReady(ctx, e, kind, query, perKind)
		if err != nil {
			// 向量路单 kind 失败：降级继续（全文路已在手），不中断检索
			slog.Warn("语义向量检索单库失败（该库降级跳过）", "kind", kind, "error", err)
			continue
		}
		vecHits = append(vecHits, hits...)
	}

	// 三路融合（阶段九 9.2）：向量 cosine + 全文 bm25/LIKE + 名称关键词，
	// 权重可解释（semantic.FuseWeight*），时序 tie-break。
	ts, _ := st.Timestamps()
	fused := semantic.FuseSemantic(query, vecHits, ftHits, ts, topN)
	return fusedToViews(fused, st), nil
}

// fusedToViews 融合结果 → wire 视图（Text 从 FTS/向量全文取：FTS 命中走
// semantic_vectors.doc 单查缓存——条目级 lazy，命中集 ≤topN 小）。
func fusedToViews(fused []semantic.FusedHit, st *semantic.Store) []SemanticHitView {
	out := make([]SemanticHitView, 0, len(fused))
	for _, f := range fused {
		text := ""
		if st != nil {
			text, _ = st.DocOf(f.Kind, f.ID)
		}
		out = append(out, SemanticHitView{Kind: f.Kind, Name: f.ID, Score: f.Final, Text: text})
	}
	return out
}
