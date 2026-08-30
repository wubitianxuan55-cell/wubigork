// v4.8 成本知识图谱：把成本条目/测算项目与明细/询价/复盘笔记/分类树组装成
// 关联图（组图器纯函数见 costref.BuildGraph，零 IO）。绑定门面 CostB
// （gen_bindings 按 GaeaCost* 前缀自动归入）。
package app

import (
	"encoding/json"

	"github.com/gaea/gaea/internal/gaea/costproject"
	"github.com/gaea/gaea/internal/gaea/costref"
)

// GaeaCostGraph 成本知识图谱视图（返回 JSON 串，前端 JSON.parse 后渲染）。
//
//   - scope="tree"（默认）：分类树聚合总览（每分类一个节点，Val=子树金额合计）
//     + 项目节点，不展开明细；
//   - scope="entry"：条目展开，focus=分类路径（如 综合单价/土建）或项目 ID，
//     展开该子树条目/该项目明细 + 匹配到的询价/指标/复盘笔记。
//
// limit 为节点上限；<=0 或 >600 归一为 600（组图器内部再按 300 默认/600 硬上限
// 截断并置 Truncated）。取数全部复用 hub*Store（成本库/测算项目/询价/复盘笔记）。
func (a *App) GaeaCostGraph(scope, focus string, limit int) (string, error) {
	if limit <= 0 || limit > costref.MaxGraphNodes {
		limit = costref.MaxGraphNodes
	}
	costStore := a.hubCostStore()
	projStore := a.hubCostProjectStore()
	projects := projStore.ListProjects()
	itemsByProject := make(map[string][]costproject.Item, len(projects))
	for _, p := range projects {
		itemsByProject[p.ID] = projStore.ListItems(p.ID)
	}
	view := costref.BuildGraph(
		projects,
		itemsByProject,
		costStore.List(),
		costStore.Categories(),
		a.hubCostInquiryStore().List("", 1000),
		a.hubCostRefStore().List("", ""),
		scope, focus, limit,
	)
	b, err := json.MarshalIndent(view, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
