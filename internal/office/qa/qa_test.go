package qa

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/office/standard"
)

// writeDocx 把 document.xml 写成最小 docx（zip 单条目），返回临时路径。
func writeDocx(t *testing.T, documentXML string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "t.docx")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w := zip.NewWriter(f)
	entry, err := w.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte(documentXML)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return p
}

func wrap(body string) string {
	return `<?xml version="1.0"?><w:document xmlns:w="u"><w:body>` + body + `</w:body></w:document>`
}

func para(text, spacing, style string) string {
	p := "<w:p><w:pPr>"
	if style != "" {
		p += `<w:pStyle w:val="` + style + `"/>`
	}
	if spacing != "" {
		p += `<w:spacing w:line="` + spacing + `"/>`
	}
	p += "</w:pPr>"
	if text != "" {
		p += `<w:r><w:rPr><w:rFonts w:eastAsia="SimSun"/></w:rPr><w:t>` + text + `</w:t></w:r>`
	}
	return p + "</w:p>"
}

// findIssue 按规则 id 取 Issue（不存在=该规则未产出）。
func findIssue(issues []standard.Issue, id string) *standard.Issue {
	for i := range issues {
		if issues[i].Element == id {
			return &issues[i]
		}
	}
	return nil
}

func runDoc(t *testing.T, body string) []standard.Issue {
	return DocxStructureChecker{}.Check(writeDocx(t, wrap(body)), "", "")
}

// TestStructuralRules Q1 六条结构不变量：正例（违例 XML）与反例（干净文档）。
func TestStructuralRules(t *testing.T) {
	cases := []struct {
		name  string
		body  string
		rule  string
		found bool // true=检出门道（Issue.Found=false）
	}{
		{"空白页：三连空段", strings.Repeat("<w:p/>", 3), "空白页（连续空段）", true},
		{"空白页：两个空段不报", strings.Repeat("<w:p/>", 2), "空白页（连续空段）", false},
		{"标题跳级 H1→H3", para("a", "", "Heading1") + para("b", "", "Heading3"), "标题跳级", true},
		{"标题递进不报", para("a", "", "Heading1") + para("b", "", "Heading2"), "标题跳级", false},
		{"行距两种且样本足", para("a", "312", "") + para("b", "312", "") + para("c", "360", "") + para("d", "360", "") + para("e", "360", ""), "行距一致性", true},
		{"行距样本不足不报", para("a", "312", "") + para("b", "360", ""), "行距一致性", false},
		{"eastAsia 全缺", strings.Repeat(`<w:r><w:t>汉字</w:t></w:r>`, 5), "CJK 字体 eastAsia", true},
		{"eastAsia 齐备不报", para("汉字内容一", "", "") + para("汉字内容二", "", ""), "CJK 字体 eastAsia", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			it := findIssue(runDoc(t, tc.body), tc.rule)
			if it == nil {
				t.Fatal("规则未产出 Issue")
			}
			if it.Found == tc.found {
				t.Fatalf("期望检出=%v，实得 Found=%v（note=%q）", tc.found, it.Found, it.Note)
			}
		})
	}
}

// TestCompatRules K3 六条兼容黑名单。
func TestCompatRules(t *testing.T) {
	exactRow := `<w:tbl><w:tblPr><w:tblBorders><w:top w:val="none"/></w:tblBorders></w:tblPr>` +
		`<w:tr><w:trPr><w:trHeight w:val="16838" w:hRule="exact"/></w:trPr><w:tc><w:p/></w:tc></w:tr></w:tbl>`
	cases := []struct {
		name  string
		body  string
		rule  string
		found bool
	}{
		{"SOLID 底纹", `<w:tbl><w:tc><w:tcPr><w:shd w:val="solid"/></w:tcPr><w:p/></w:tc></w:tbl>`, "ShadingType.SOLID", true},
		{"clear 底纹不报", `<w:tbl><w:tc><w:tcPr><w:shd w:val="clear" w:fill="EEEEEE"/></w:tcPr><w:p/></w:tc></w:tbl>`, "ShadingType.SOLID", false},
		{"exact+center", `<w:tbl><w:tr><w:trPr><w:trHeight w:val="1000" w:hRule="exact"/></w:trPr><w:tc><w:tcPr><w:vAlign w:val="center"/></w:tcPr><w:p/></w:tc></w:tr></w:tbl>`, "exact 行竖排对齐", true},
		{"exact+top 不报", `<w:tbl><w:tr><w:trPr><w:trHeight w:val="1000" w:hRule="exact"/></w:trPr><w:tc><w:tcPr><w:vAlign w:val="top"/></w:tcPr><w:p/></w:tc></w:tr></w:tbl>`, "exact 行竖排对齐", false},
		{"DXA 列宽", `<w:tbl><w:tr><w:tc><w:tcPr><w:tcW w:w="3000" w:type="dxa"/></w:tcPr><w:p/></w:tc></w:tr></w:tbl>`, "DXA 列宽", true},
		{"PCT 列宽不报", `<w:tbl><w:tr><w:tc><w:tcPr><w:tcW w:w="30" w:type="pct"/></w:tcPr><w:p/></w:tc></w:tr></w:tbl>`, "DXA 列宽", false},
		{"TOC decimal", `<w:p><w:r><w:instrText> TOC \o "1-3" \* decimal </w:instrText></w:r></w:p>`, "TOC 页码 decimal", true},
		{"TOC arabic 不报", `<w:p><w:r><w:instrText> TOC \o "1-3" \* arabic </w:instrText></w:r></w:p>`, "TOC 页码 decimal", false},
		{"封面表 single 边框", `<w:tbl><w:tblPr><w:tblBorders><w:top w:val="single" w:sz="4"/></w:tblBorders></w:tblPr>` +
			`<w:tr><w:trPr><w:trHeight w:val="16838" w:hRule="exact"/></w:trPr><w:tc><w:p/></w:tc></w:tr></w:tbl>`, "封面表默认边框", true},
		{"封面表全无边框不报", exactRow, "封面表默认边框", false},
		{"字符画线", para("───────", "", ""), "字符画装饰线", true},
		{"普通正文不报", para("正常段落", "", ""), "字符画装饰线", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			it := findIssue(runDoc(t, tc.body), tc.rule)
			if it == nil {
				t.Fatal("规则未产出 Issue")
			}
			if it.Found == tc.found {
				t.Fatalf("期望检出=%v，实得 Found=%v（note=%q）", tc.found, it.Found, it.Note)
			}
		})
	}
}

// TestCleanDocumentPass 干净长文：全部规则 Found（含短文免检面不误报）。
func TestCleanDocumentPass(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 6; i++ {
		b.WriteString(para("这是第"+string(rune('A'+i))+"段正文内容，行距一致字体齐备。", "312", ""))
	}
	issues := runDoc(t, b.String())
	for _, it := range issues {
		if !it.Found {
			t.Fatalf("干净文档被误报：%s note=%q", it.Element, it.Note)
		}
	}
}

// TestNonDocxSkipped 非 docx/坏 zip 不产出（诚实边界）。
func TestNonDocxSkipped(t *testing.T) {
	if got := (DocxStructureChecker{}).Check("x.txt", "", ""); got != nil {
		t.Fatalf("非 docx 应跳过，实得 %d 项", len(got))
	}
	if got := (DocxStructureChecker{}).Check(filepath.Join(t.TempDir(), "bad.docx"), "", ""); got != nil {
		t.Fatalf("坏 zip 应跳过，实得 %d 项", len(got))
	}
}

// TestRegisteredIntegration 注册面集成：Register 后 standard.LintDocument
// 对 docx 路径产出租机检 Issue（顺带钉死「调用方须传绝对路径」的接线前提）。
func TestRegisteredIntegration(t *testing.T) {
	Register()
	p := writeDocx(t, wrap(strings.Repeat("<w:p/>", 3)))
	rep := standard.LintDocument(p, "", "")
	var hit bool
	for _, it := range rep.Issues {
		if it.Spec == specStructural && !it.Found {
			hit = true
		}
	}
	if !hit {
		t.Fatal("注册后 LintDocument 未产出机检问题项")
	}
}
