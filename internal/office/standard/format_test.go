package standard

// GB/T 9704 排版细则测试：真实 document.xml 片段解析（边距/字体/字号/行距/
// 缩进/层级标题）+ 合规全过/违规带实测建议/数据不足不适用三态。

import (
	"strings"
	"testing"
)

func paraXML(text, font, size, line, rule, indChars, ind, jc string) string {
	var b strings.Builder
	b.WriteString("<w:p><w:pPr>")
	if line != "" || rule != "" {
		b.WriteString(`<w:spacing w:line="` + line + `" w:lineRule="` + rule + `"/>`)
	}
	if indChars != "" || ind != "" {
		b.WriteString(`<w:ind w:firstLineChars="` + indChars + `" w:firstLine="` + ind + `"/>`)
	}
	if jc != "" {
		b.WriteString(`<w:jc w:val="` + jc + `"/>`)
	}
	b.WriteString(`</w:pPr><w:r><w:rPr><w:rFonts w:eastAsia="` + font + `"/><w:sz w:val="` + size + `"/></w:rPr><w:t>` + text + `</w:t></w:r></w:p>`)
	return b.String()
}

const marginsOK = `<w:sectPr><w:pgMar w:top="2098" w:right="1474" w:bottom="1985" w:left="1588"/></w:sectPr>`

func compliantDoc() DocxLayout {
	xml := `<w:document><w:body>` +
		paraXML("××市人民政府文件", "小标宋", "44", "", "", "", "", "center") +
		paraXML("关于加快数字政府建设的通知", "小标宋", "44", "", "", "", "", "center") +
		paraXML("各区、县人民政府：", "仿宋_GB2312", "32", "", "", "", "", "") +
		paraXML("一、总体要求", "黑体", "32", "560", "exact", "", "", "") +
		paraXML("（一）坚持统筹推进", "楷体", "32", "560", "exact", "", "", "") +
		paraXML("坚持以人民为中心的发展思想，扎实推进各项任务落地见效。", "仿宋_GB2312", "32", "560", "exact", "200", "640", "") +
		paraXML("坚持以人民为中心的发展思想，扎实推进各项任务落地见效。", "仿宋_GB2312", "32", "560", "exact", "200", "640", "") +
		marginsOK +
		`</w:body></w:document>`
	l, err := ParseDocxLayout([]byte(xml))
	if err != nil {
		panic(err)
	}
	return l
}

func TestParseDocxLayoutFacts(t *testing.T) {
	l := compliantDoc()
	if !l.MarginsFound || l.MarginTopMM != 37 || l.MarginBottomMM != 35 || l.MarginLeftMM != 28 || l.MarginRightMM != 26 {
		t.Fatalf("边距提取错误: %+v", l)
	}
	if len(l.Paras) != 7 {
		t.Fatalf("段落数错误: %d", len(l.Paras))
	}
	body := l.Paras[5]
	if body.Font != "仿宋_GB2312" || body.SizeHalfPt != 32 || body.LineTwips != 560 ||
		body.LineRule != "exact" || body.FirstLineChars != 200 || body.FirstLineTwips != 640 {
		t.Fatalf("段落事实提取错误: %+v", body)
	}
	if body.Text != "坚持以人民为中心的发展思想，扎实推进各项任务落地见效。" {
		t.Fatalf("文本提取错误: %q", body.Text)
	}
}

func TestFormatCheckerCompliant(t *testing.T) {
	issues := FormatChecker{}.CheckLayout(compliantDoc())
	if len(issues) != 6 {
		t.Fatalf("检查项数量错误: %d", len(issues))
	}
	for _, it := range issues {
		if !it.Found {
			t.Fatalf("%s 应符合，实测: %s", it.Element, it.Note)
		}
		if it.Spec != "GB/T 9704 排版细则" {
			t.Fatalf("Spec 归属缺失: %+v", it)
		}
	}
}

func TestFormatCheckerViolations(t *testing.T) {
	xml := `<w:document><w:body>` +
		paraXML("关于加快数字政府建设的通知", "宋体", "36", "", "", "", "", "center") +
		paraXML("各区、县人民政府：", "宋体", "28", "360", "auto", "", "", "") +
		paraXML("一、总体要求", "宋体", "28", "360", "auto", "", "", "") +
		paraXML("扎实推进各项任务落地见效确保完成全年目标。", "宋体", "28", "360", "auto", "", "", "") +
		`<w:sectPr><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440"/></w:sectPr>` +
		`</w:body></w:document>`
	l, err := ParseDocxLayout([]byte(xml))
	if err != nil {
		t.Fatal(err)
	}
	issues := FormatChecker{}.CheckLayout(l)
	byElem := map[string]Issue{}
	for _, it := range issues {
		byElem[it.Element] = it
	}
	if it := byElem["页边距（上37 下35 左28 右26 mm）"]; it.Found || !strings.Contains(it.Note, "上边实测") {
		t.Fatalf("边距违规应带实测: %+v", it)
	}
	if it := byElem["公文标题字号（二号 22pt）"]; it.Found {
		t.Fatalf("标题 18pt 应违规: %+v", it)
	}
	if it := byElem["正文字体字号（三号仿宋）"]; it.Found || !strings.Contains(it.Note, "14 磅") {
		t.Fatalf("正文宋体14pt 应违规并注明实测字号: %+v", it)
	}
	if it := byElem["正文行距（28~30 磅）"]; it.Found {
		t.Fatalf("行距 18磅 应违规: %+v", it)
	}
	if it := byElem["正文首行缩进（2 字符）"]; it.Found {
		t.Fatalf("无缩进应违规: %+v", it)
	}
	if it := byElem["层级标题字体（一、黑体；（一）楷体）"]; it.Found || !strings.Contains(it.Note, "黑体") {
		t.Fatalf("层级标题宋体应违规: %+v", it)
	}
}

func TestFormatCheckerNotApplicable(t *testing.T) {
	// 无标题/无层级标题/正文不足：对应项不适用（不误报为缺失）。
	xml := `<w:document><w:body>` +
		paraXML("随便写点内容", "仿宋", "32", "560", "exact", "200", "640", "") +
		marginsOK +
		`</w:body></w:document>`
	l, err := ParseDocxLayout([]byte(xml))
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range (FormatChecker{}).CheckLayout(l) {
		if !it.Found {
			t.Fatalf("单项不足文档不应判缺失: %+v", it)
		}
	}
}

// 无边距定义（sectPr 缺失）：边距项缺失并给出规范值，其余照常。
func TestFormatCheckerNoMargins(t *testing.T) {
	xml := `<w:document><w:body>` +
		paraXML("坚持以人民为中心扎实推进各项任务落地见效。", "仿宋", "32", "560", "exact", "200", "640", "") +
		`</w:body></w:document>`
	l, err := ParseDocxLayout([]byte(xml))
	if err != nil {
		t.Fatal(err)
	}
	if l.MarginsFound {
		t.Fatal("不应误报边距在场")
	}
	var margin Issue
	foundMargin := false
	for _, it := range (FormatChecker{}).CheckLayout(l) {
		if strings.HasPrefix(it.Element, "页边距") {
			margin, foundMargin = it, true
		}
	}
	if !foundMargin || margin.Found {
		t.Fatalf("无边距应判缺失: %+v", margin)
	}
}
