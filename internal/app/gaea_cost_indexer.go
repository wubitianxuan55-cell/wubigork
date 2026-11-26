package app

// gaea_cost_indexer.go — 造价库向量索引的后台索引器（随应用启动）与状态 / 补齐绑定。
//
// 索引器（internal/gaea/vecindex）：启动先对一轮；之后库的写入经 workcost 的句柄打脏
// 标记唤醒（合并窗口 3 秒），另每 10 分钟兜底对账；模型不可用时待办保留、1 分钟后重试。
// 搜索路径从不建索引，所以索引落后只会少几条语义命中，不会拖慢检索。
//
// 模型固定读 [retrieval] 配置（缺省 bge-m3 / bge-reranker-v2-m3），与方案库同一套客户端。

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/tool/builtin"
	"github.com/gaea/gaea/internal/gaea/vecindex"
	"github.com/gaea/gaea/internal/gaea/workcost"
)

// costIndexKindLabels 类目的人话名。
var costIndexKindLabels = map[string]string{
	workcost.IndexKindQuota:    "定额",
	workcost.IndexKindResource: "工料机",
	workcost.IndexKindBill:     "清单",
}

// CostIndexKindView 一个类目（定额 / 工料机 / 清单）的索引状态。
type CostIndexKindView struct {
	Kind  string `json:"kind"`
	Label string `json:"label"`
	// Total 库里条目数（索引器最近一轮对账看到的）；没对账过时为 0。
	Total int `json:"total"`
	// Count 已建向量的条数。
	Count int `json:"count"`
	// UpdatedAt 最近一次写入索引的时间（空 = 还没建过）。
	UpdatedAt string `json:"updatedAt"`
	// LastError 最近一轮对账的错误（空 = 正常）。
	LastError string `json:"lastError,omitempty"`
}

// CostIndexStatusView 造价库语义索引的状态。
type CostIndexStatusView struct {
	Kinds []CostIndexKindView `json:"kinds"`
	// Model 索引使用的向量模型名。
	Model string `json:"model"`
	// Running 后台索引器在跑。
	Running bool `json:"running"`
}

// startCostIndexer 启动后台索引器（幂等）。在任务调度器与工料法存储都就绪之后调用。
func (a *App) startCostIndexer() {
	if a == nil || a.officeState == nil {
		return
	}
	a.officeState.costIndexOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		a.officeState.costIndexStop = cancel
		ix := a.newCostIndexer()
		a.officeState.costIndexer = ix
		// 库的写入点经包级默认索引器打脏标记。
		vecindex.SetDefault(ix)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("cost index: 后台索引器异常退出", "panic", r)
				}
			}()
			ix.Run(ctx)
		}()
	})
}

// newCostIndexer 装配索引器：向量索引与造价库各自取应用侧的同一份实例。
func (a *App) newCostIndexer() *vecindex.Indexer {
	return vecindex.NewIndexer(a.costIndexStore(), builtin.LocalEmbedModel(), func() vecindex.Embedder {
		if e := a.costIndexEmbedder(); e != nil {
			return e
		}
		return nil
	}, a.hubWorkcostStore().IndexSources(), workcost.IndexKinds...)
}

// costIndexer 取后台索引器；没启动时为 nil。
func (a *App) costIndexer() *vecindex.Indexer {
	if a == nil || a.officeState == nil {
		return nil
	}
	return a.officeState.costIndexer
}

// GaeaCostIndexStatus 造价库语义索引的状态（只读）。直接读索引元信息与索引器最近一轮的
// 结果，不探测模型——状态面要的是「此刻记在案的情况」，不是一次可能卡住的网络往返。
func (a *App) GaeaCostIndexStatus() CostIndexStatusView {
	v := CostIndexStatusView{Kinds: []CostIndexKindView{}, Model: builtin.LocalEmbedModel()}
	ix := a.costIndexer()
	if ix != nil {
		v.Running = true
		v.Model = ix.Model()
	}
	vs := a.costIndexStore()
	for _, kind := range workcost.IndexKinds {
		kv := CostIndexKindView{Kind: kind, Label: costIndexKindLabels[kind]}
		if vs != nil {
			if m, err := vs.Meta(kind); err == nil {
				kv.Count, kv.UpdatedAt = m.Count, m.UpdatedAt
			}
		}
		if ix != nil {
			if st, ok := ix.Last(kind); ok {
				kv.Total = st.Total
			}
			if err, ok := ix.LastError(kind); ok && err != nil {
				kv.LastError = err.Error()
			}
		}
		v.Kinds = append(v.Kinds, kv)
	}
	return v
}

// GaeaCostIndexBackfill 显式补齐造价库语义索引：立刻对账（缺的补、正文变了的重嵌、换了
// 模型整类目重嵌），返回补齐后的状态。kind 空 = 全部三类。模型不可用时如实报错，
// 不静默假装补齐。
func (a *App) GaeaCostIndexBackfill(kind string) (CostIndexStatusView, error) {
	kinds := workcost.IndexKinds
	if k := strings.TrimSpace(kind); k != "" {
		if !slices.Contains(workcost.IndexKinds, k) {
			return CostIndexStatusView{}, fmt.Errorf("不支持的类目 %q（%s）", k, strings.Join(workcost.IndexKinds, " / "))
		}
		kinds = []string{k}
	}
	ix := a.costIndexer()
	if ix == nil {
		return CostIndexStatusView{}, fmt.Errorf("造价库索引器没有启动")
	}
	// 首次建大索引要十几分钟（本机实测约 100 条/秒），上限放到一小时。
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	for _, k := range kinds {
		st, err := ix.ApplyNow(ctx, k)
		if err != nil {
			return a.GaeaCostIndexStatus(), fmt.Errorf("%s索引补齐失败: %w", costIndexKindLabels[k], err)
		}
		slog.Info("cost index: 显式补齐完成", "kind", k, "total", st.Total, "embedded", st.Embedded,
			"removed", st.Removed, "count", st.Count, "dur", st.Dur.Round(time.Millisecond))
	}
	return a.GaeaCostIndexStatus(), nil
}

// GaeaWorkcostBillSearch 清单行检索（跨项目）：带关键词走混合检索（相关度序，含同义
// 说法召回）；不带关键词按列表顺序返回前 200 条。project_id 非 0 时只看该项目。
func (a *App) GaeaWorkcostBillSearch(keyword, specialty string, projectID int64) []workcost.BillItem {
	hits, err := a.costSearcher().BillItems(context.Background(), workcost.SearchQuery{
		Text: keyword, Specialty: specialty, ProjectID: projectID, Limit: costSearchListLimit,
	})
	if err != nil {
		return []workcost.BillItem{}
	}
	return nonNilSlice(hitsItems(hits))
}

func hitsItems(hits []workcost.BillItemHit) []workcost.BillItem {
	out := make([]workcost.BillItem, len(hits))
	for i, h := range hits {
		out[i] = h.Item
	}
	return out
}
