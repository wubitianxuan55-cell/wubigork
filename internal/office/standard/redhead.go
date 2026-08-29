// Package standard 提供中文公文规范校验（v4.1c 第一刀：GB/T 9704 红头要素 lint）。
// 纯函数：输入标题/正文文本，输出缺失要素清单与修复建议；不依赖文件类型
// （md/txt/docx 经调用方先提取文本）。
package standard

import (
	"regexp"
	"strings"
)

// Issue 是体检报告中的一条不合格项。
type Issue struct {
	Element string `json:"element"` // 要素名（发文机关标志 / 发文字号 / …）
	Found   bool   `json:"found"`
	Note    string `json:"note"` // 修复建议（缺失时给出）
}

// LintReport 是一次红头要素体检的结果。
type LintReport struct {
	Path    string  `json:"path"`
	Issues  []Issue `json:"issues"`
	Passed  bool    `json:"passed"`
	Summary string  `json:"summary"`
}

var (
	reDocNumber = regexp.MustCompile(`〔?[0-9]{4}〕?\s*[0-9]+号`)
	reDate      = regexp.MustCompile(`[0-9]{4}\s*年\s*[0-9]{1,2}\s*月\s*[0-9]{1,2}\s*日`)
)

// LintText 对标题+正文做红头要素体检（GB/T 9704 关键要素子集）。
// head 通常为前若干行（含发文机关标志与标题），body 为其余正文。
func LintText(path, head, body string) LintReport {
	all := head + "\n" + body
	issues := []Issue{
		{Element: "发文机关标志（红头）", Found: firstLine(head) != ""},
		{Element: "发文字号（如：×发〔2026〕3号）", Found: reDocNumber.MatchString(all)},
		{Element: "公文标题（含「关于…的通知/报告/请示」）", Found: strings.Contains(head, "关于") || strings.Contains(head, "报告")},
		{Element: "主送机关（正文开头的称谓行）", Found: hasSalutation(body)},
		{Element: "成文日期（落款）", Found: reDate.MatchString(all)},
		{Element: "印章/盖章位（落款处）", Found: strings.Contains(all, "印") || strings.Contains(all, "盖章")},
		{Element: "版记（抄送/印发机关/印发日期）", Found: strings.Contains(all, "抄送") || strings.Contains(all, "印发")},
	}
	// 缺失项补修复建议
	for i := range issues {
		if issues[i].Found {
			issues[i].Note = "符合"
			continue
		}
		issues[i].Note = fixHint(issues[i].Element)
	}
	missing := 0
	for _, it := range issues {
		if !it.Found {
			missing++
		}
	}
	return LintReport{
		Path:    path,
		Issues:  issues,
		Passed:  missing == 0,
		Summary: redHeadSummary(missing, len(issues)),
	}
}

func firstLine(head string) string {
	for _, l := range strings.Split(head, "\n") {
		if strings.TrimSpace(l) != "" {
			return strings.TrimSpace(l)
		}
	}
	return ""
}

func hasSalutation(body string) bool {
	for _, l := range strings.Split(body, "\n") {
		t := strings.TrimSpace(l)
		if t != "" && (strings.HasPrefix(t, "各") || strings.HasPrefix(t, "本") || strings.HasSuffix(t, "：")) {
			return true
		}
	}
	return false
}

func fixHint(element string) string {
	switch element {
	case "发文机关标志（红头）":
		return "首行应为本机关全称或规范化简称（红头字）"
	case "发文字号（如：×发〔2026〕3号）":
		return "标题下方应标注发文字号：机关代字〔年份〕序号号"
	case "公文标题（含「关于…的通知/报告/请示」）":
		return "标题应完整包含发文机关、事由与文种（如：关于××的通知）"
	case "主送机关（正文开头的称谓行）":
		return "正文开头应顶格写主送机关，如「各市、州人民政府：」"
	case "成文日期（落款）":
		return "落款应有成文日期（格式：2026年8月29日）"
	case "印章/盖章位（落款处）":
		return "成文日期上应加盖发文机关印章（或标注盖章位）"
	case "版记（抄送/印发机关/印发日期）":
		return "文末应加版记：抄送、印发机关与印发日期"
	}
	return "补齐该要素"
}

func redHeadSummary(missing, total int) string {
	if missing == 0 {
		return "全部要素齐备，符合 GB/T 9704 红头基本要求"
	}
	return "检出 " + itoa(missing) + " 项缺失（共 " + itoa(total) + " 项要素）"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
