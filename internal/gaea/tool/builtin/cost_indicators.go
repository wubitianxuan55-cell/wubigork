package builtin

// cost_indicators 造价参考指标工具（zaojia-database 蒸馏收口：数据库就是数据库，
// 测算/对标留给办公 agent）。对「已保存版本/已沉淀」测算项目的明细单价实时聚合，
// 按科目或一级分类返回样本数/均值/中位数/P25/P75/区间，供测算前对标与引用依据。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/costproject"
	"github.com/gaea/gaea/internal/gaea/costref"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() {
	tool.RegisterBuiltin(costIndicators{})
}

// costProjectStoreOverride 测试注入的隔离测算项目存储（避免触碰真实用户库）。
var costProjectStoreOverride *costproject.Store

// SetCostProjectStoreForTest 注入隔离的测算项目存储（测试用）。
func SetCostProjectStoreForTest(s *costproject.Store) { costProjectStoreOverride = s }

// openCostProjectStore 打开测算项目存储（可测试注入）。
func openCostProjectStore() (*costproject.Store, error) {
	if costProjectStoreOverride != nil {
		return costProjectStoreOverride, nil
	}
	return costproject.Open(db.GetDatabase(config.MemoryUserDir())), nil
}

// costIndicators 查询造价参考指标（只读）。
type costIndicators struct{}

func (costIndicators) Name() string { return "cost_indicators" }
func (costIndicators) Description() string {
	return "查询造价参考指标：对已保存版本/已沉淀测算项目的明细单价聚合（按科目 title 或一级分类 category），返回样本数/均值/中位数/P25/P75/最低~最高区间，供测算对标。样本数<3 标注「样本少」。"
}
func (costIndicators) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "group":{"type":"string","description":"分组方式：title（按科目标题，默认）或 category（按一级分类，如 材料/机械）"},
  "category":{"type":"string","description":"只统计该一级分类下的明细（可选，如 材料）"},
  "limit":{"type":"integer","description":"返回条数上限（默认20，最大50）"}
}
}`)
}
func (costIndicators) ReadOnly() bool                 { return true }
func (costIndicators) CompactDescription() string     { return "查询造价参考指标（案例单价分位数/均值，按科目或分类），供测算对标。" }
func (costIndicators) CompactSchema() json.RawMessage { return json.RawMessage(`{"type":"object","properties":{"group":{"type":"string"},"category":{"type":"string"}}}`) }

func (costIndicators) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Group    string `json:"group,omitempty"`
		Category string `json:"category,omitempty"`
		Limit    int    `json:"limit,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	projStore, err := openCostProjectStore()
	if err != nil {
		return "", err
	}
	if !projStore.Available() {
		return "", fmt.Errorf("测算项目存储不可用")
	}
	var items []costproject.Item
	for _, pr := range projStore.ListProjects() {
		if pr.VersionCount <= 0 {
			continue // 临时工作稿不参与对标
		}
		items = append(items, projStore.ListItems(pr.ID)...)
	}
	if cat := strings.TrimSpace(p.Category); cat != "" {
		filtered := items[:0]
		for _, i := range items {
			first := ""
			if parts := strings.Split(i.CategoryPath, "/"); len(parts) > 0 {
				first = strings.TrimSpace(parts[0])
			}
			if first == cat {
				filtered = append(filtered, i)
			}
		}
		items = filtered
	}
	group := p.Group
	if group != "category" {
		group = "title"
	}
	inds := costref.ComputeIndicators(items, group)
	if len(inds) == 0 {
		return "暂无造价参考指标。测算项目保存过版本或沉淀后，明细单价会自动聚合为对标数据。", nil
	}
	limit := p.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	if len(inds) > limit {
		inds = inds[:limit]
	}
	groupLabel := "科目标题"
	if group == "category" {
		groupLabel = "一级分类"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## 造价参考指标（按%s，%d 组）\n\n", groupLabel, len(inds))
	b.WriteString("| 科目/分类 | 单位 | 样本数 | 均值 | 中位数 | P25 | P75 | 最低~最高 | 质量 |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|\n")
	for _, ind := range inds {
		quality := "可用"
		if ind.Samples < 3 {
			quality = "样本少"
		}
		fmt.Fprintf(&b, "| %s | %s | %d | %.2f | %.2f | %.2f | %.2f | %.2f~%.2f | %s |\n",
			cell(ind.Key), cell(ind.Unit), ind.Samples, ind.Mean, ind.Median, ind.P25, ind.P75,
			ind.Min, ind.Max, quality)
	}
	b.WriteString("\n参考指标仅用于对标，正式定价仍以 cost_search 命中的条目与来源为准。")
	return tool.WrapText(b.String()), nil
}
