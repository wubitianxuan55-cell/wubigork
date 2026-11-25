package workcost

// export.go — 把解析出的项目工作簿重新导出成六表测算工作簿。版式只有一处，在
// render.go（与空白测算模版共用同一个渲染器）；本文件只负责把 ProjectBundle 装进
// 渲染输入。
//
// 清单项的定额块：bundle 带三级行组（QuotaBlocks）按块走；旧 1:1 形态（没有定额
// 块的 bundle）合成单块——码 = 清单码，与导入链的定额锚点一致。

import (
	"fmt"
	"strings"
)

// ExportOptions 导出可选附加内容。
type ExportOptions struct {
	// Project 项目名（写入封面标题；空则用 bundle 里的）。
	Project string
	// Location/Duration 封面基本信息覆盖。
	Location string
	Duration string
	// Pricing 组价口径自述（空则用封面解析到的，再空用缺省一句）。
	Pricing string
}

// ExportProjectWorkbook 把项目 bundle 渲染成六表工作簿字节。填好的簿可经
// import_project 原样回流（三级行组逐块还原成库的 1:N 挂接）。
func ExportProjectWorkbook(b *ProjectBundle, opt ExportOptions) ([]byte, error) {
	if b == nil {
		return nil, fmt.Errorf("项目数据为空")
	}
	if len(b.Resources) == 0 && len(b.Lines) == 0 {
		return nil, fmt.Errorf("项目没有资源与消耗量，无法导出（先解析项目工作簿或录入工料机）")
	}
	m := workbookModel{
		Title:        firstNonEmpty(opt.Project, b.Project, "项目"),
		Indicators:   true,
		Fee:          RateSet{ManagementRate: b.Fee.ManagementRate, RegulatoryRate: b.Fee.RegulatoryRate, ProfitRate: b.Fee.ProfitRate, TaxRate: b.Fee.TaxRate},
		ControlPrice: b.Fee.ControlPrice,
		Quantities:   b.Quantities,
	}
	if loc := firstNonEmpty(opt.Location, b.Location); loc != "" {
		m.CoverLines = append(m.CoverLines, "地点："+loc)
	}
	if dur := firstNonEmpty(opt.Duration, b.Duration); dur != "" {
		m.CoverLines = append(m.CoverLines, "工期："+dur)
	}
	m.CoverLines = append(m.CoverLines, "组价口径："+firstNonEmpty(opt.Pricing, b.Pricing,
		"综合单价=工序消耗量×基础资源价（只含人材机）；管理费、利润、税金在费用汇总单列。"))

	for _, r := range b.Resources {
		// 规格列在实测工作簿里承载「计算式」（如「折旧900+柴油×90L+司机」）：台班公式
		// 原文优先，便于追溯口径。
		m.Resources = append(m.Resources, resourceRow{
			Code: r.Code, GCode: r.GCode, Kind: r.Kind, Title: r.Title,
			Spec: firstNonEmpty(r.Spec, r.PriceFormula), Unit: r.Unit, Price: r.Price,
			Source: r.Source, Note: r.Note,
		})
	}
	for _, it := range b.Items {
		wi := workbookItem{Item: it}
		for _, blk := range blocksOfBundle(b, it.Code) {
			lines := linesOfBlock(b, it.Code, blk.Code)
			wi.Blocks = append(wi.Blocks, workbookBlock{Quota: blk, Lines: lines, DetailCount: len(lines)})
		}
		m.Items = append(m.Items, wi)
	}
	return renderWorkbook(m)
}

// blocksOfBundle 清单项的定额块：bundle 带三级行组按块走；旧 1:1 形态合成单块。
func blocksOfBundle(b *ProjectBundle, code string) []ProjectQuotaBlock {
	var out []ProjectQuotaBlock
	for _, blk := range b.QuotaBlocks {
		if blk.ItemCode == code && strings.TrimSpace(blk.Code) != "" {
			out = append(out, blk)
		}
	}
	if len(out) == 0 {
		return []ProjectQuotaBlock{{ItemCode: code, Code: code}}
	}
	return out
}

// linesOfBlock 该清单项该定额块的明细行。旧 1:1 形态的行不带定额码，归合成块
// （码 = 清单码）。
func linesOfBlock(b *ProjectBundle, itemCode, quotaCode string) []ProjectConsumption {
	var out []ProjectConsumption
	for _, l := range b.Lines {
		if l.ItemCode != itemCode {
			continue
		}
		if l.QuotaCode == quotaCode || (l.QuotaCode == "" && quotaCode == itemCode) {
			out = append(out, l)
		}
	}
	return out
}
