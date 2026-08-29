package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// CostComposeEvidence 证据链一条:相似清单条目快照。溯源字段(来源/地区/期数/
// 口径)天然是证据格式——AI 组价的核心护城河。
type CostComposeEvidence struct {
	Name      string  `json:"name"`
	Title     string  `json:"title"`
	Category  string  `json:"category"`
	Unit      string  `json:"unit"`
	Spec      string  `json:"spec"`
	Price     float64 `json:"price"`
	Source    string  `json:"source"`
	Region    string  `json:"region"`
	PriceDate string  `json:"priceDate"`
	PriceType string  `json:"priceType"`
}

// CostComposeView AI 组价建议视图(无确认不落库;确认走 GaeaCostComposeApply)。
// Band 为 nil 表示成本库无相似条目(建议不可用,前端提示先建库)。
type CostComposeView struct {
	Description      string                `json:"description"`
	Unit             string                `json:"unit"`
	Band             *cost.PriceBand       `json:"band"`
	RecommendedPrice float64               `json:"recommendedPrice"`
	Reason           string                `json:"reason"`
	Components       []CostComponentView   `json:"components,omitempty"`
	ComponentsNote   string                `json:"componentsNote,omitempty"`
	LLMUsed          bool                  `json:"llmUsed"`
	Evidence         []CostComposeEvidence `json:"evidence"`
}

// GaeaCostCompose AI 组价:清单描述 → 相似清单检索(关键词+语义+精排)→ 价格带
// 推荐(P25-P75+置信度+证据链)→ LLM 人材机拆解(可选,失败降级为顶部相似条目
// 组件参考)。描述/单位来自用户(如测算明细行),全程无确认不落库。
func (a *App) GaeaCostCompose(desc, unit string) (CostComposeView, error) {
	desc = strings.TrimSpace(desc)
	if desc == "" {
		return CostComposeView{}, fmt.Errorf("清单描述不能为空")
	}
	store := a.hubCostStore()
	if !store.Available() {
		return CostComposeView{}, fmt.Errorf("成本库不可用")
	}

	// 1. 相似清单检索:关键词 + 语义补召回 + 本地精排(与 GaeaCostSearch 同款组合)。
	similar := store.Search(desc, "", "现行")
	if len(similar) < 3 {
		if sem := a.semanticCostRecall(desc, similar, 10); len(sem) > 0 {
			similar = sem
		}
	}
	if reranked := a.rerankCostSearch(desc, similar, 12); len(reranked) > 0 {
		similar = reranked
	}
	if len(similar) == 0 {
		return CostComposeView{Description: desc, Unit: unit}, nil
	}

	// 2. 价格带推荐(按单位过滤;单位未知时不过滤)。
	band := cost.ComputePriceBand(similar, unit)
	if band == nil {
		return CostComposeView{Description: desc, Unit: unit}, nil
	}
	rec, reason := cost.RecommendPrice(band, "median")

	// 3. 证据链(全部相似条目快照,含离群——前端用 Outliers 标注)。
	evidence := make([]CostComposeEvidence, 0, len(band.Sources))
	for _, s := range band.Sources {
		evidence = append(evidence, CostComposeEvidence{
			Name: s.Name, Title: s.Title, Category: s.Category, Unit: s.Unit,
			Spec: s.Spec, Price: s.Price, Source: s.Source,
			Region: s.Region, PriceDate: s.PriceDate, PriceType: s.PriceType,
		})
	}

	// 4. LLM 人材机拆解(敏感域本地化路由,失败降级)。
	comps, note, llmUsed := a.composeLLMDecompose(desc, similar, band)

	return CostComposeView{
		Description: desc, Unit: unit, Band: band,
		RecommendedPrice: rec, Reason: reason,
		Components: toCostComponentViews(comps), ComponentsNote: note,
		LLMUsed: llmUsed, Evidence: evidence,
	}, nil
}

// GaeaCostComposeApply 确认组价建议并回写成本库(UPSERT):
// 标题=描述首行,单价=推荐价,人材机=预览组件(可编辑),来源标注 AI 组价。
// 返回条目 name(SlugName 稳定键)。
func (a *App) GaeaCostComposeApply(v CostComposeView) (string, error) {
	title := strings.TrimSpace(v.Description)
	if title == "" {
		return "", fmt.Errorf("清单描述不能为空")
	}
	// 多行描述取首行为标题(其余进备注)。
	body := ""
	if lines := strings.Split(title, "\n"); len(lines) > 1 {
		title = strings.TrimSpace(lines[0])
		body = strings.TrimSpace(strings.Join(lines[1:], "\n"))
	}
	price := v.RecommendedPrice
	if price <= 0 {
		return "", fmt.Errorf("推荐价无效,请先在成本库录入相似条目后再组价")
	}
	e := cost.Entry{
		Name:         cost.SlugName(title),
		Title:        title,
		Category:     leafCategoryOf(v.Unit, title),
		CategoryPath: "",
		Unit:         strings.TrimSpace(v.Unit),
		Price:        price,
		Components:   fromCostComponentViews(v.Components),
		Source:       "AI组价",
		Status:       "现行",
		Body:         body,
	}
	if err := a.hubCostStore().Save(e); err != nil {
		return "", fmt.Errorf("回写成本库失败: %w", err)
	}
	return e.Name, nil
}

// leafCategoryOf 组价条目的默认分类:描述含「综合单价」关键词或带人材机组成时
// 归 综合单价(叶子取第一个斜杠段),否则归 其他。
func leafCategoryOf(unit, title string) string {
	if strings.Contains(title, "综合单价") {
		return "综合单价"
	}
	return "其他"
}

// composeLLMDecompose 用办公功能模型把清单描述拆解为人材机组成(综合单价子目)。
// 路由 routeSensitiveLocal(成本/报价属敏感域:开关开启时强制本地 Herdsman)。
// 任何一步不可用(开关关/本地引擎缺失/请求失败/输出无效)返回 (nil, 说明, false),
// 由调用方降级为「参考顶部相似条目组件」(在 GaeaCostCompose 调用侧处理)。
func (a *App) composeLLMDecompose(desc string, similar []cost.Summary, band *cost.PriceBand) ([]cost.Component, string, bool) {
	if a == nil || a.core == nil || a.cfg == nil || a.engineMgr == nil {
		return nil, "", false
	}
	featEng, featModel, _ := a.routeSensitiveLocal("office")
	if featEng == "" || featModel == "" {
		return nil, "", false
	}
	prov, err := provider.NewLLM("", provider.Config{Name: "cost-compose", Model: featModel, Engine: featEng})
	if err != nil {
		return nil, "", false
	}

	// 参考上下文:最多 3 条相似条目的标题/单价/人材机组成(带完整组件需逐条 Get)。
	store := a.hubCostStore()
	var ref strings.Builder
	for i, s := range similar {
		if i >= 3 {
			break
		}
		fmt.Fprintf(&ref, "- %s（%s）单价 %.2f 元", s.Title, s.Unit, s.Price)
		if e, err := store.Get(s.Name); err == nil && e != nil && len(e.Components) > 0 {
			var comps []string
			for _, c := range e.Components {
				if strings.TrimSpace(c.Title) == "" {
					continue
				}
				comps = append(comps, fmt.Sprintf("%s %s %.4f×%.2f", c.Kind, c.Title, c.Quantity, c.Price))
			}
			if len(comps) > 0 {
				ref.WriteString(" 组成:" + strings.Join(comps, "; "))
			}
		}
		ref.WriteString("\n")
	}

	const sysPrompt = "你是造价组价助手。根据清单描述与相似成本条目,拆解该清单项综合单价的人材机组成。\n" +
		"输出 JSON 数组,每项字段: kind(人工/材料/机械), title(资源名称), unit(单位,人工=工日/机械=台班), " +
		"quantity(含量,每单位工程量), price(资源单价,元), amount(金额=quantity*price)。\n" +
		"规则: 只输出 JSON 数组,不要代码块标记与解释;无法确定的资源用相似条目组成参考;宁少勿编造。"
	user := fmt.Sprintf("清单描述: %s\n相似条目参考(价格带中位数 %.2f 元):\n%s请拆解人材机组成:", desc, band.Median, ref.String())

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	ch, err := prov.Stream(ctx, provider.Request{
		Messages: []provider.Message{
			{Role: provider.RoleSystem, Content: sysPrompt},
			{Role: provider.RoleUser, Content: user},
		},
		Temperature: 0,
	})
	if err != nil {
		return nil, "", false
	}
	var out strings.Builder
	for {
		select {
		case <-ctx.Done():
			return nil, "", false
		case chunk, ok := <-ch:
			if !ok {
				comps, ok2 := parseComposeComponents(out.String())
				if !ok2 {
					return nil, "AI 拆解输出无效,已降级为规则参考", false
				}
				return comps, "AI 拆解完成,请核对含量与单价", true
			}
			switch chunk.Type {
			case provider.ChunkText:
				out.WriteString(chunk.Text)
			case provider.ChunkError:
				return nil, "", false
			}
		}
	}
}

// parseComposeComponents 解析 AI 拆解 JSON 数组为组件行;无效/空返回 false。
func parseComposeComponents(raw string) ([]cost.Component, bool) {
	start := strings.IndexByte(raw, '[')
	end := strings.LastIndexByte(raw, ']')
	if start < 0 || end <= start {
		return nil, false
	}
	var rows []struct {
		Kind     string  `json:"kind"`
		Title    string  `json:"title"`
		Unit     string  `json:"unit"`
		Quantity float64 `json:"quantity"`
		Price    float64 `json:"price"`
		Amount   float64 `json:"amount"`
	}
	if err := json.Unmarshal([]byte(raw[start:end+1]), &rows); err != nil || len(rows) == 0 {
		return nil, false
	}
	out := make([]cost.Component, 0, len(rows))
	seen := map[string]bool{}
	for i, r := range rows {
		title := strings.TrimSpace(r.Title)
		if title == "" {
			continue
		}
		kind := strings.TrimSpace(r.Kind)
		if kind != "人工" && kind != "材料" && kind != "机械" {
			kind = "材料"
		}
		key := kind + "\x00" + title
		if seen[key] {
			continue // 去重(同资源合并)
		}
		seen[key] = true
		amount := r.Amount
		if amount <= 0 {
			amount = r.Quantity * r.Price
		}
		out = append(out, cost.Component{
			Kind: kind, Title: title, Unit: strings.TrimSpace(r.Unit),
			Quantity: r.Quantity, Price: r.Price, Amount: amount,
			Note: "AI组价", Sort: i,
		})
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}
