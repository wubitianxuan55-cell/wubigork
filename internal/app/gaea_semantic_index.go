package app

// 语义索引主动维护（刀路池第 5 项：检索索引被动维护——成本类向量此前只在
// 搜索时按需增量构建，首次检索大库会在超时内静默降级为仅关键词）。
// 显形 + 显式补齐：不做后台守护协程，覆盖数字与补齐动作都由用户看着发生。
// 判定内核 semantic.Coverage 与 Ensure 同口径（正文快照变了算未覆盖）。

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/retrieval"
	"github.com/gaea/gaea/internal/gaea/semantic"
)

// costIndexDocs 成本条目 → 向量文档（与 semanticCostRecall 的构建完全一致）。
func (a *App) costIndexDocs() []semantic.Doc {
	full := a.hubCostStore().List()
	docs := make([]semantic.Doc, len(full))
	for i, s := range full {
		docs[i] = semantic.Doc{ID: s.Name, Text: retrieval.DocText(s)}
	}
	return docs
}

// CostIndexView 成本条目向量索引覆盖视图（GaeaSemanticIndexStatus 的扩展字段）。
type CostIndexView struct {
	Total   int `json:"total"`   // 成本条目总数（正文非空）
	Indexed int `json:"indexed"` // 已覆盖（向量在且正文快照一致，与 Ensure 同口径）
}

// costIndexCoverage 成本类索引覆盖（供 GaeaSemanticIndexStatus 扩展字段；
// 非导出=不成绑定，仅状态汇报内部消费）。
func (a *App) costIndexCoverage() (CostIndexView, error) {
	docs := a.costIndexDocs()
	view := CostIndexView{}
	for _, d := range docs {
		if strings.TrimSpace(d.Text) != "" {
			view.Total++
		}
	}
	st := a.hubSemanticStore()
	if st != nil && st.Available() {
		var err error
		view.Indexed, err = st.Coverage("cost", docs)
		if err != nil {
			return view, err
		}
	}
	return view, nil
}

// GaeaSemanticIndexBackfill 显式补齐成本类向量索引：对全部成本条目增量
// 向量化（缺失或正文变化才重嵌，与搜索路径共用 Ensure，幂等）。返回
// {total, updated, indexed, missing}；模型不可用返回 error 说人话，
// 绝不静默假装补齐。
func (a *App) GaeaSemanticIndexBackfill() (map[string]interface{}, error) {
	e := a.localSearchEmbedder()
	if e == nil {
		return nil, fmt.Errorf("本地语义模型未配置（需 Herdsman bge-m3 或 qwen3-embedding），补齐不可用")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	st := a.hubSemanticStore()
	if st == nil || !st.Available() {
		return nil, fmt.Errorf("向量索引存储不可用")
	}
	docs := a.costIndexDocs()
	total := len(docs)
	updated, err := st.Ensure(ctx, e, "cost", docs)
	if err != nil {
		return nil, fmt.Errorf("向量化失败: %w", err)
	}
	indexed, err := st.Coverage("cost", docs)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"total":   total,
		"updated": updated,
		"indexed": indexed,
		"missing": total - indexed,
	}, nil
}
