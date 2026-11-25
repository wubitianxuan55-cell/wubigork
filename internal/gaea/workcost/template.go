package workcost

// template.go — 空白成本测算模版（六表；2026-11-25 用户拍板「成本测算模版要依据
// 现在的造价数据库进行程序设计，使其对应」，行组为「清单 → 定额 → 工料机」三级）。
//
// 模版不是另一套版式：它就是 renderWorkbook 的一次渲染——没有清单、带若干空白
// 组、工料机价格表按库预填。所以填好的簿走 import_project 能原样回流。
//
// 怎么填（每列含义、系数怎么取、常见错法）是测算知识，在 cost-estimate 技能；
// 这里只有「空白组长什么样」。

import (
	"fmt"
	"strings"
)

// 空白模版的形态常量。
const (
	// TemplateQuotaBlocksPerGroup 综合单价每组预置的定额块数（不够在 Excel 里复制
	// 一组定额行+明细行插入，公式区间按行号引用，插入后自动扩）。
	TemplateQuotaBlocksPerGroup = 2
	// TemplateDetailRowsPerBlock 每个定额块预置的工料机明细行数。
	TemplateDetailRowsPerBlock = 4
	// TemplateBlankResourceRows 工料机价格表预填之后留的空行数（VLOOKUP 区间
	// 覆盖它，后来加的行也查得到价）。
	TemplateBlankResourceRows = 60
	// TemplateDefaultGroups 综合单价空白组数（Groups=0 时的缺省）。
	TemplateDefaultGroups = 30
)

// TemplateOptions 空白测算模版的口径：预填什么、封面写什么。
type TemplateOptions struct {
	// Project 封面标题（空=「成本测算模版」）。
	Project string
	// Specialty/Region/Period 模版口径三件（封面写明）。专业/地区是字典值；
	// 期数 YYYY-MM（空=不写）。解析器不替文件填这三个值——回导时的口径由读表的
	// 人或模型提给导入。
	Specialty string
	Region    string
	Period    string
	// Fee 取费预填（费率链数值；零值=默认口径 DefaultRateSet）。
	Fee RateSet
	// FeeSource 费率出处说明（封面写明，用模版的人知道预填的费率从哪来）。
	FeeSource string
	// Resources 预填的工料机（调用方按口径从库选好；带自有码则写进自有码列）。
	Resources []Resource
	// Groups 综合单价空白组数（0=TemplateDefaultGroups）。
	Groups int
}

// ExportProjectTemplate 渲染空白成本测算模版（六表）工作簿字节。
func ExportProjectTemplate(opt TemplateOptions) ([]byte, error) {
	groups := opt.Groups
	if groups <= 0 {
		groups = TemplateDefaultGroups
	}
	fee := opt.Fee
	if fee == (RateSet{}) {
		fee = DefaultRateSet()
	}
	m := workbookModel{
		Title:       firstNonEmpty(opt.Project, "成本测算模版"),
		Scope:       [3]string{opt.Specialty, opt.Region, opt.Period},
		PriceBlank:  TemplateBlankResourceRows,
		BlankGroups: groups,
		Fee:         fee,
		Blank:       true,
	}
	m.CoverLines = append(m.CoverLines, "组价口径：综合单价=工序消耗量×基础资源价（只含人材机）；管理费、利润、税金在费用汇总单列。")
	if v := strings.TrimSpace(opt.FeeSource); v != "" {
		m.CoverLines = append(m.CoverLines, "费率："+v)
	}
	if n := len(opt.Resources); n > 0 {
		m.CoverLines = append(m.CoverLines, fmt.Sprintf(
			"工料机：已从造价库预填 %d 条（带库内自有码与现行价）；综合单价明细行的资源编码填工料机价格表的「编码」列，库里没有的自己加行。", n))
	}
	m.CoverLines = append(m.CoverLines,
		"填法：「清单」表填清单行；综合单价每组 = 1 清单汇总行 + 定额块（定额行 + 明细行），一个清单套几条定额就建几个块；没填的组导回库时自动跳过。")
	for _, r := range opt.Resources {
		m.Resources = append(m.Resources, resourceRow{
			Code: r.Code, GCode: r.GCode, Kind: r.Kind, Title: r.Title, Spec: r.Spec, Unit: r.Unit,
			Price: r.EffectivePrice(), Source: r.Source, Note: r.Note,
		})
	}
	return renderWorkbook(m)
}
