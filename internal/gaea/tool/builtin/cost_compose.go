package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() {
	tool.RegisterBuiltin(costCompose{})
}

// costCompose AI 组价（长期规划阶段三「造价开口」）：清单描述 → 相似条目检索
// （关键词+语义补召回+本地精排，与 cost_search 同款组合）→ 价格带推荐
// （P25/中位/P75+样本数+置信度+离群标注）+ 证据链（来源/地区/期数/口径溯源
// 五元组）。只读不落库：人材机拆解由会话模型自行完成（管家本身就是 LLM，
// 不再嵌套一次 LLM 调用）；采用哪个价由模型给结论后经 cost_save 确认沉淀
// （cost_save 落盘门=确认进库，不新造盖章）。与 UI 组价全链（GaeaCostCompose
// 含嵌套 LLM 拆解+V16 留痕快照）互为两路：会话开口走本工具，造价页走完整链。
type costCompose struct{}

func (costCompose) Name() string { return "cost_compose" }
func (costCompose) Description() string {
	return "AI 组价：按清单描述检索相似成本条目，计算价格带（P25/中位/P75+样本数+置信度+离群数）并给出推荐价与证据链（来源/地区/期数/口径）。只读不落库：人材机拆解与定价结论由你给出，采用后用 cost_save 确认沉淀。"
}
func (costCompose) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "description":{"type":"string","description":"清单描述，如「HP300 高频液压振动锤 300kW 履带式」"},
  "unit":{"type":"string","description":"单位（可选）：台班/吨/m³/工日等；提供时只统计同单位样本"},
  "limit":{"type":"integer","description":"证据链条数上限（默认 8，最大 20）"},
   "mode":{"type":"string","description":"推荐档：median（默认）/p25/p75/mean/conservative；未知按中位数"}
},
"required":["description"]
}`)
}
func (costCompose) ReadOnly() bool                 { return true }
func (costCompose) CompactDescription() string     { return compactDesc["cost_compose"] }
func (costCompose) CompactSchema() json.RawMessage { return compactSchema["cost_compose"] }

func (costCompose) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Description string `json:"description"`
		Unit        string `json:"unit,omitempty"`
		Limit       int    `json:"limit,omitempty"`
		Mode        string `json:"mode,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	desc := strings.TrimSpace(p.Description)
	if desc == "" {
		return "", fmt.Errorf("description 为必填项")
	}
	store, err := openCostStore()
	if err != nil {
		return "", err
	}
	if !store.Available() {
		return "", fmt.Errorf("成本库不可用（用户目录为空或数据库打开失败）")
	}

	// 相似检索：与 cost_search 同款组合（SQL → 语义补召回 → 本地精排）。
	similar := store.Search(desc, "", "现行")
	if len(similar) < 3 {
		if sem := semanticCostRecall(ctx, desc, similar, store, 10); len(sem) > 0 {
			similar = sem
		}
	}
	if len(similar) == 0 {
		return fmt.Sprintf("成本库中没有「%s」的相似条目，无法组价。可先按合理估价测算，完成后用 cost_save 沉淀，下次即可组价引用。", desc), nil
	}
	if reranked := rerankCostResults(ctx, desc, similar, 12); len(reranked) > 0 {
		similar = reranked
	}

	band := cost.ComputePriceBand(similar, p.Unit)
	if band == nil {
		// 单位过滤把样本全排除时 band 为 nil：提示去掉 unit 重试。
		return fmt.Sprintf("「%s」的相似条目在单位 %q 下无样本。去掉 unit 参数重试可跨单位参考。", desc, p.Unit), nil
	}
	rec, reason := cost.RecommendPrice(band, strings.ToLower(strings.TrimSpace(p.Mode)))

	limit := p.Limit
	if limit <= 0 {
		limit = 8
	}
	if limit > 20 {
		limit = 20
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## AI 组价：%s\n\n", desc)
	if u := strings.TrimSpace(p.Unit); u != "" {
		fmt.Fprintf(&b, "（单位过滤：%s）\n\n", u)
	}
	b.WriteString("**价格带**\n\n")
	b.WriteString("| P25 | 中位数 | P75 | 均值 | 最低 | 最高 | 样本 | 离散度 | 离群 | 置信度 |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|---|\n")
	fmt.Fprintf(&b, "| %.2f | %.2f | %.2f | %.2f | %.2f | %.2f | %d | %.1f%% | %d | %s |\n\n",
		band.P25, band.Median, band.P75, band.Mean, band.Min, band.Max,
		band.Samples, band.SpreadPct, band.Outliers, band.Confidence)
	fmt.Fprintf(&b, "**三档对照**：P25 %.2f · 中位 %.2f · P75 %.2f\n\n", band.P25, band.Median, band.P75)
	fmt.Fprintf(&b, "**推荐价**: %.2f 元（%s）\n\n", rec, reason)
	b.WriteString("**证据链**（溯源五元组，含离群样本——定价时注意甄别）\n\n")
	b.WriteString("| 标题 | 分类 | 单价(元) | 单位 | 来源 | 地区 | 期数 | 口径 |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|\n")
	shown := 0
	for _, s := range band.Sources {
		if shown >= limit {
			break
		}
		fmt.Fprintf(&b, "| %s | %s | %.2f | %s | %s | %s | %s | %s |\n",
			cell(s.Title), cell(s.Category), s.Price, cell(s.Unit), cell(s.Source),
			cell(s.Region), cell(s.PriceDate), cell(s.PriceType))
		shown++
	}
	fmt.Fprintf(&b, "\n共 %d 条样本，上表列 %d 条；置信度 %s。", band.Samples, shown, band.Confidence)
	if band.Outliers > 0 {
		fmt.Fprintf(&b, "其中 %d 条离群（P25-1.5IQR/P75+1.5IQR 之外）。", band.Outliers)
	}
	b.WriteString("\n\n人材机拆解由你基于以上证据给出；**采用某个价后用 cost_save 沉淀**")
	b.WriteString("（title=清单描述、source 建议标「AI组价」+本次上下文；保存会走确认门），下次 cost_search/组价即可直接引用。")
	return tool.WrapText(b.String()), nil
}
