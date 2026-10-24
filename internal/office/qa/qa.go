// Package qa — docx 结构机检（Q1 结构不变量 + K3 WPS/Word 兼容黑名单）。
//
// 移植自 zai-org/ZCode documents-plugin postcheck.py 与 references/common-rules.md
// （Apache-2.0）：判定重写为 Go 直读 OOXML（zip→document.xml 正则断言），
// 不起 Python。规则取舍按蒸馏对话档第 4/5 轮收口——只抄「事后可从 XML
// 客观断言的结构不变量」：表格边距/cantSplit/CJK 缩进是生成时参数（归技能
// 自检域，机检是自我重复），清洁度/内容质量是语义判断（归模型/视觉平面），
// 均不进本包。
package qa

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gaea/gaea/internal/office/ooxml"
	"github.com/gaea/gaea/internal/office/standard"
)

const (
	specStructural = "docx 结构不变量"
	specCompat     = "WPS/Word 兼容"
)

// DocxStructureChecker 以 standard.Checker 形态挂载机检：只处理 .docx 路径，
// 读失败/非 docx 不产出（诚实边界：报不出而不是误报）。
type DocxStructureChecker struct{}

func (DocxStructureChecker) Name() string { return "docx 结构机检" }

func (DocxStructureChecker) Check(path, head, body string) []standard.Issue {
	if strings.ToLower(filepath.Ext(path)) != ".docx" {
		return nil
	}
	files, _, err := ooxml.ReadZip(path, "qa")
	if err != nil {
		return nil
	}
	doc := string(files["word/document.xml"])
	if !strings.Contains(doc, "<w:body") && !strings.Contains(doc, "<w:document") {
		return nil
	}
	var issues []standard.Issue
	for _, r := range structuralRules {
		issues = append(issues, runRule(r, doc))
	}
	for _, r := range compatRules {
		issues = append(issues, runRule(r, doc))
	}
	return issues
}

// docxRule 一条机检规则：check 返回 (是否通过, 未通过时的修复说明)。
type docxRule struct {
	id    string
	spec  string
	check func(doc string) (ok bool, note string)
}

func runRule(r docxRule, doc string) standard.Issue {
	ok, note := r.check(doc)
	return standard.Issue{Element: r.id, Found: ok, Note: note, Spec: r.spec}
}

// ── 共用抽取 ──

var (
	rePara        = regexp.MustCompile(`(?s)<w:p\b.*?</w:p>|<w:p\b[^>]*/>`)
	reText        = regexp.MustCompile(`(?s)<w:t[^>]*>(.*?)</w:t>`)
	reDrawing     = regexp.MustCompile(`(?s)<w:drawing>.*?</w:drawing>`)
	reExtent      = regexp.MustCompile(`<wp:extent[^>]*cx="(\d+)"[^>]*cy="(\d+)"`)
	reAExt        = regexp.MustCompile(`<a:ext[^>]*cx="(\d+)"[^>]*cy="(\d+)"`)
	reRow         = regexp.MustCompile(`(?s)<w:tr\b.*?</w:tr>`)
	reTable       = regexp.MustCompile(`(?s)<w:tbl>.*?</w:tbl>`)
	reHeading     = regexp.MustCompile(`w:val="(?:Heading|标题)\s?([1-9])"`)
	reLine        = regexp.MustCompile(`w:line="(\d+)"`)
	reParaStyle   = regexp.MustCompile(`<w:pStyle w:val="([^"]+)"`)
	reShdSolid    = regexp.MustCompile(`<w:shd\b[^>]*w:val="solid"`)
	reTcWDxa      = regexp.MustCompile(`<w:tcW\b[^>]*w:type="dxa"`)
	reInstr       = regexp.MustCompile(`(?s)<w:instrText[^>]*>(.*?)</w:instrText>`)
	reRun         = regexp.MustCompile(`(?s)<w:r\b[^>]*>.*?</w:r>|<w:r\b[^>]*/>`)
	reCJK         = regexp.MustCompile(`\p{Han}`)
	decoLineChars = "─━═—–―"
)

// paragraphs 切出段落块（含自闭合）。
func paragraphs(doc string) []string { return rePara.FindAllString(doc, -1) }

// visibleText 取段落可见文本（w:t 拼接，不解实体——断言用字符面足够）。
func visibleText(para string) string {
	var sb strings.Builder
	for _, m := range reText.FindAllStringSubmatch(para, -1) {
		sb.WriteString(m[1])
	}
	return sb.String()
}

// isHeadingPara 段落是否带 Heading 样式。
func isHeadingPara(para string) bool {
	st := reParaStyle.FindStringSubmatch(para)
	return st != nil && (strings.HasPrefix(st[1], "Heading") || strings.HasPrefix(st[1], "标题"))
}

// ── Q1：结构不变量（6 条）──

var structuralRules = []docxRule{
	{
		id:   "空白页（连续空段）",
		spec: specStructural,
		check: func(doc string) (bool, string) {
			empty := 0
			for _, p := range paragraphs(doc) {
				if strings.TrimSpace(visibleText(p)) == "" {
					empty++
					if empty >= 3 {
						return false, "连续 ≥3 个空段落——多余分页/空白页的常见根因，删除空段或分页符"
					}
				} else {
					empty = 0
				}
			}
			return true, ""
		},
	},
	{
		id:   "长文目录域",
		spec: specStructural,
		check: func(doc string) (bool, string) {
			n := len(paragraphs(doc))
			if n < 40 {
				return true, "" // 短文档免检（Found=true=通过）
			}
			for _, m := range reInstr.FindAllStringSubmatch(doc, -1) {
				if strings.Contains(m[1], "TOC") {
					return true, ""
				}
			}
			return false, "正文 ≥40 段但无 TOC 域——插入目录（域指令 TOC \\* arabic）"
		},
	},
	{
		id:   "标题跳级",
		spec: specStructural,
		check: func(doc string) (bool, string) {
			prev := 0
			for _, p := range paragraphs(doc) {
				m := reHeading.FindStringSubmatch(p)
				if m == nil {
					continue
				}
				lv := int(m[1][0] - '0')
				if prev > 0 && lv > prev+1 {
					return false, "标题层级跳跃（如 H1 直接到 H3）——补齐中间层级"
				}
				prev = lv
			}
			return true, ""
		},
	},
	{
		id:   "行距一致性",
		spec: specStructural,
		check: func(doc string) (bool, string) {
			seen := map[string]int{}
			for _, p := range paragraphs(doc) {
				if isHeadingPara(p) || strings.TrimSpace(visibleText(p)) == "" {
					continue
				}
				if m := reLine.FindStringSubmatch(p); m != nil {
					seen[m[1]]++
				}
			}
			if len(seen) < 2 {
				return true, ""
			}
			total := 0
			for _, c := range seen {
				total += c
			}
			if total < 5 {
				return true, "" // 样本太少不判（避免特例段拉响）
			}
			return false, "正文段落出现 ≥2 种行距值（" + keysJoined(seen) + "）——统一正文行距"
		},
	},
	{
		id:   "图片纵横比",
		spec: specStructural,
		check: func(doc string) (bool, string) {
			for _, d := range reDrawing.FindAllString(doc, -1) {
				e := reExtent.FindStringSubmatch(d)
				a := reAExt.FindStringSubmatch(d)
				if e == nil || a == nil || e[1] == "0" || a[2] == "0" {
					continue
				}
				rExt := parseFloat(e[1]) / parseFloat(e[2])
				rA := parseFloat(a[1]) / parseFloat(a[2])
				if rA == 0 {
					continue
				}
				dev := rExt/rA - 1
				if dev < 0 {
					dev = -dev
				}
				if dev > 0.10 {
					return false, "图片显示尺寸与原生尺寸纵横比偏差 >10%——拉伸变形，按原比例缩放"
				}
			}
			return true, ""
		},
	},
	{
		id:   "CJK 字体 eastAsia",
		spec: specStructural,
		check: func(doc string) (bool, string) {
			cjk, missing := 0, 0
			for _, r := range reRun.FindAllString(doc, -1) {
				if !reCJK.MatchString(visibleText(r)) {
					continue
				}
				cjk++
				if !strings.Contains(r, "w:eastAsia") {
					missing++
				}
			}
			if cjk < 4 || missing*2 < cjk { // 过半缺失才报：rPr 可从样式继承，避免误报
				return true, ""
			}
			return false, "过半 CJK run 缺 w:eastAsia 字体声明——目标机器可能回退错字体，逐 run 显式声明"
		},
	},
}

// ── K3：WPS/Word 兼容黑名单（XML 可断言子集，6 条）──

var compatRules = []docxRule{
	{
		id:   "ShadingType.SOLID",
		spec: specCompat,
		check: func(doc string) (bool, string) {
			if reShdSolid.MatchString(doc) {
				return false, "w:shd w:val=\"solid\" 在 WPS 渲染为纯黑——改用 clear+fill"
			}
			return true, ""
		},
	},
	{
		id:   "exact 行竖排对齐",
		spec: specCompat,
		check: func(doc string) (bool, string) {
			for _, tr := range reRow.FindAllString(doc, -1) {
				if !strings.Contains(tr, `w:hRule="exact"`) {
					continue
				}
				if strings.Contains(tr, `<w:vAlign w:val="center"`) || strings.Contains(tr, `<w:vAlign w:val="bottom"`) {
					return false, "exact 高度行内 verticalAlign center/bottom 在 WPS 被忽略——改 top+spacing.before"
				}
			}
			return true, ""
		},
	},
	{
		id:   "DXA 列宽",
		spec: specCompat,
		check: func(doc string) (bool, string) {
			if reTcWDxa.MatchString(doc) {
				return false, "单元格 DXA 列宽触发 WPS tblGrid 失配——统一 PERCENTAGE 列宽"
			}
			return true, ""
		},
	},
	{
		id:   "TOC 页码 decimal",
		spec: specCompat,
		check: func(doc string) (bool, string) {
			for _, m := range reInstr.FindAllStringSubmatch(doc, -1) {
				if strings.Contains(m[1], "* decimal") {
					return false, "TOC 域用 \\* decimal 页码格式跨端不稳——改 \\* arabic"
				}
			}
			return true, ""
		},
	},
	{
		id:   "封面表默认边框",
		spec: specCompat,
		check: func(doc string) (bool, string) {
			for _, tbl := range reTable.FindAllString(doc, -1) {
				if !strings.Contains(tbl, `w:hRule="exact"`) {
					continue
				}
				if strings.Contains(tbl, "<w:tblBorders") && strings.Contains(tbl, `w:val="single"`) {
					return false, "exact 高度外层表带 single 边框（allNoBorders 缺失）——MS Office 计入边框厚度可致封面溢出空白页"
				}
			}
			return true, ""
		},
	},
	{
		id:   "字符画装饰线",
		spec: specCompat,
		check: func(doc string) (bool, string) {
			for _, p := range paragraphs(doc) {
				t := strings.TrimSpace(visibleText(p))
				if len([]rune(t)) < 3 {
					continue
				}
				if strings.Trim(t, decoLineChars) == "" {
					return false, "用字符（───/━━/═══）画装饰线——两端渲染宽度不一，改段落边框"
				}
			}
			return true, ""
		},
	},
}

func keysJoined(m map[string]int) string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sortStrings(ks)
	return strings.Join(ks, "/")
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func parseFloat(s string) float64 {
	var v float64
	for _, c := range s {
		if c < '0' || c > '9' {
			continue
		}
		v = v*10 + float64(c-'0')
	}
	return v
}

// 编译期断言：满足 standard.Checker 注册面。
var _ standard.Checker = DocxStructureChecker{}

// Register 把机检挂进 standard 默认注册表（app 装配层调用一次；同名幂等）。
func Register() { standard.RegisterDefault(DocxStructureChecker{}) }
