package app

// 画室消耗（阶段一刀 E 轻量收尾，规格 进度计划/gaea-studio-usage-20260917.md）：
// 台账已逐条记 Cost（目录单价口径）+CreatedAt，本文件只做**只读月度聚合**——
// 登记侧零改动。口径：本地时区当月（CreatedAt 前缀 YYYY-MM 匹配）；按
// model+backend 分组计数；单价以**台账记录为准**（记录 Cost 与目录表不一致时
// 记录是事实源）；可解析「X CNY/张」才估算（张数×X），"未定价"/空计入
// unpriced 诚实可见不猜。文案「画室消耗/创作记录」，不用积分话术（规格原文）。

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ImageHubUsageModelRow 按模型聚合行。
type ImageHubUsageModelRow struct {
	Model    string `json:"model"`
	Backend  string `json:"backend,omitempty"`
	Count    int    `json:"count"`
	UnitCost string `json:"unitCost,omitempty"` // 台账口径（"0"/"0.18 CNY/张"/"未定价"/""）
	EstCost  string `json:"estCost,omitempty"`  // 可算时「X.XX CNY」；免费「0 CNY」；不可算留空
}

// ImageHubMonthlyUsageReport 月度消耗报告（camelCase typed）。
type ImageHubMonthlyUsageReport struct {
	Month     string                  `json:"month"` // "2026-09"（本地时区当月）
	Space     string                  `json:"space"` // 归一化空间
	Total     int                     `json:"total"` // 当月登记张数（图+视频）
	FreeCount int                     `json:"freeCount"`
	Unpriced  int                     `json:"unpriced"`  // 未定价张数（诚实可见）
	Estimated string                  `json:"estimated"` // 合计估算（可算部分）
	ByModel   []ImageHubUsageModelRow `json:"byModel"`
}

// unitCostRe 可解析单价「X CNY/张」（整数或小数）。
var unitCostRe = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)?)\s*CNY/张$`)

// parseUnitCostCNY 单价串 → 元/张；裸 "0" 与 "0 CNY/张" 均回 (0, true)=免费；
// 空/"未定价"/其它形态回 (-1, false)。
func parseUnitCostCNY(cost string) (float64, bool) {
	c := strings.TrimSpace(cost)
	if c == "0" {
		return 0, true
	}
	if c == "" || c == "未定价" {
		return -1, false
	}
	if m := unitCostRe.FindStringSubmatch(c); m != nil {
		v, err := strconv.ParseFloat(m[1], 64)
		if err != nil || v < 0 {
			return -1, false
		}
		return v, true
	}
	return -1, false
}

// ImageHubMonthlyUsage 当月消耗聚合（只读，绑定入口——cwd 走 gaeaCwd 与
// ImageHubAssets 同构）。空台账=零报告不算错（正常态）。
func (a *mediaState) ImageHubMonthlyUsage(space string) (ImageHubMonthlyUsageReport, error) {
	return imageHubMonthlyUsageAt(gaeaCwd(), space)
}

// imageHubMonthlyUsageAt 聚合实现（cwd 参数化供测试直调）。
func imageHubMonthlyUsageAt(cwd, space string) (ImageHubMonthlyUsageReport, error) {
	sp := normalizeImageHubSpace(space)
	led := newImageHubLedger(cwd)
	month := time.Now().Format("2006-01")

	report := ImageHubMonthlyUsageReport{
		Month:   month,
		Space:   sp,
		ByModel: []ImageHubUsageModelRow{},
	}
	type key struct{ model, backend string }
	rows := map[key]*ImageHubUsageModelRow{}
	var estimated float64
	anyPriced := false

	for _, rec := range led.list(sp, 0) {
		// 当月过滤：CreatedAt 是 RFC3339（带时区），前缀 YYYY-MM 即本地当月口径
		if !strings.HasPrefix(rec.Meta.CreatedAt, month) {
			continue
		}
		k := key{model: strings.TrimSpace(rec.Meta.Model), backend: strings.TrimSpace(rec.Meta.Backend)}
		row, ok := rows[k]
		if !ok {
			// 单价以台账记录为准；记录缺价时回落目录表（登记早期可能未记）
			unit := strings.TrimSpace(rec.Meta.Cost)
			if unit == "" {
				if meta, ok2 := imageModelCatalogMetaFor(rec.Meta.Model); ok2 {
					unit = meta.UnitCost
				}
			}
			row = &ImageHubUsageModelRow{Model: k.model, Backend: k.backend, UnitCost: unit}
			rows[k] = row
		}
		row.Count++
		report.Total++

		if unitCNY, ok := parseUnitCostCNY(row.UnitCost); !ok {
			report.Unpriced++
		} else if unitCNY == 0 {
			report.FreeCount++
			row.EstCost = "0 CNY"
			anyPriced = true
		} else {
			row.EstCost = fmt.Sprintf("%.2f CNY", float64(row.Count)*unitCNY)
			anyPriced = true
		}
	}

	for _, row := range rows {
		report.ByModel = append(report.ByModel, *row)
	}
	sort.Slice(report.ByModel, func(i, j int) bool {
		if report.ByModel[i].Count != report.ByModel[j].Count {
			return report.ByModel[i].Count > report.ByModel[j].Count
		}
		return report.ByModel[i].Model < report.ByModel[j].Model
	})

	// 合计：重算（行内 EstCost 已随 Count 更新），避免浮点累计口径不一致
	estimated = 0
	for _, row := range report.ByModel {
		if v, ok := parseUnitCostCNY(row.UnitCost); ok && v > 0 {
			estimated += float64(row.Count) * v
		}
	}
	switch {
	case report.Total == 0:
		report.Estimated = "本月还没有创作记录"
	case !anyPriced && report.Unpriced > 0:
		report.Estimated = "全部未定价"
	case estimated == 0:
		report.Estimated = "0 CNY（本地免费）"
	default:
		report.Estimated = fmt.Sprintf("%.2f CNY", estimated)
		if report.Unpriced > 0 {
			report.Estimated += fmt.Sprintf("（另有 %d 张未定价）", report.Unpriced)
		}
	}
	return report, nil
}
