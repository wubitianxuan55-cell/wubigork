package standard

// 造价工程表式规范包（v4.6.1 补课：审计 §C ③「造价工程表式」零实现）。
// 面向造价文档/测算表/报价书文书的「表式齐全性」体检：工程名称、编制依据、
// 单位造价口径、人材机组成、汇总、编制说明六要素。纯函数、字符串启发式，
// 与红头 lint 同档（检出缺失并给可执行建议；非 AI 判定）。

import (
	"regexp"
	"strings"
)

var (
	reProjectName = regexp.MustCompile(`(工程|项目)\s*(名称|概况|全称)`)
	reCompileInfo = regexp.MustCompile(`编制\s*(单位|人|依据|日期)`)
	reUnitPrice   = regexp.MustCompile(`单价|单位造价|元/(m2|m²|㎡|吨|t|个|套|m)`)
	reTotals      = regexp.MustCompile(`合计|总计|小计|汇总`)
)

// CostTableChecker 是造价工程表式规范包。
type CostTableChecker struct{}

// Name 规范包名。
func (CostTableChecker) Name() string { return "造价工程表式" }

// Check 对造价文书做表式齐全性体检。
func (c CostTableChecker) Check(path, head, body string) []Issue {
	all := head + "\n" + body
	issues := []Issue{
		{Element: "工程/项目名称（如：××工程预算书 / 项目概况）", Found: reProjectName.MatchString(all)},
		{Element: "编制单位/编制人/编制依据", Found: reCompileInfo.MatchString(all)},
		{Element: "单位造价口径（单价/单位造价，如 元/m²）", Found: reUnitPrice.MatchString(all)},
		{Element: "人材机组成（人工费/材料费/机械费）", Found: strings.Contains(all, "人工") || strings.Contains(all, "人材机") || strings.Contains(all, "材料费") || strings.Contains(all, "机械费")},
		{Element: "合计/总计汇总表", Found: reTotals.MatchString(all)},
		{Element: "编制说明/备注", Found: strings.Contains(all, "编制说明") || strings.Contains(all, "备注") || strings.Contains(all, "说明：")},
	}
	for i := range issues {
		issues[i].Spec = c.Name()
		if issues[i].Found {
			issues[i].Note = "符合"
			continue
		}
		issues[i].Note = costFixHint(issues[i].Element)
	}
	return issues
}

func costFixHint(element string) string {
	switch element {
	case "工程/项目名称（如：××工程预算书 / 项目概况）":
		return "文首应标注工程/项目全称（如「××小区 1#楼 预算书」）"
	case "编制单位/编制人/编制依据":
		return "应标注编制单位、编制人与编制依据（图纸/清单/定额号）"
	case "单位造价口径（单价/单位造价，如 元/m²）":
		return "表格应有单价列或单位造价口径（如 元/m²、元/t）"
	case "人材机组成（人工费/材料费/机械费）":
		return "综合单价应展开人材机：人工费/材料费/机械费（或注明人材机合计）"
	case "合计/总计汇总表":
		return "应有合计/总计汇总（分项小计或全表汇总行）"
	case "编制说明/备注":
		return "应有编制说明/备注（计价口径、未含项、调整说明）"
	}
	return "补齐该要素"
}
