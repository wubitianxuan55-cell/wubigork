package standard

// GB/T 9704 排版细则检查器（v4.223 中文文书规范包第二刀，roadmap 办公#5）：
// 在红头要素（文本层）之上补**排版层**——页边距/标题与正文字体字号/行距/
// 首行缩进/层级标题字体。输入=docx 主文档 XML 提取的排版事实（ParseDocxLayout），
// 全部纯函数；检查口径按 GB/T 9704-2012，容差内视为符合，实测值写进修复建议。

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// twipsPerMM 换算常数（1mm ≈ 56.6929 twip）。
const twipsPerMM = 56.6929

// DocxLayout 是从 docx 主文档 XML 提取的排版事实（检查器的全部输入）。
type DocxLayout struct {
	MarginTopMM    float64
	MarginBottomMM float64
	MarginLeftMM   float64
	MarginRightMM  float64
	MarginsFound   bool
	Paras          []ParaFact
}

// ParaFact 单段排版事实：东亚字体/字号（半磅）/行距/首行缩进/对齐。
type ParaFact struct {
	Text           string
	Font           string // w:rFonts w:eastAsia（仿宋_GB2312 / 黑体 / 楷体…）
	SizeHalfPt     int    // w:sz（半磅：32=三号16pt，44=二号22pt）
	LineTwips      int    // w:spacing w:line
	LineRule       string // exact / atLeast / auto
	FirstLineChars int    // w:ind w:firstLineChars（百分之一字符：200=2字符）
	FirstLineTwips int    // w:ind w:firstLine
	Center         bool   // w:jc w:val="center"
}

// ParseDocxLayout 从 word/document.xml 字节提取排版事实。容错子集解析：
// 只认 pgMar/spacing/ind/jc/rFonts/sz/t，未知元素跳过；表格内段落一并计入
// （正文多数票口径下影响可忽略）。run 级 rPr 优先于段级 rPr（段落标记缺省）。
func ParseDocxLayout(data []byte) (DocxLayout, error) {
	var layout DocxLayout
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	var cur *ParaFact
	var inRun, inSectPr, inText bool
	var runFont string
	var runSize int

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return layout, fmt.Errorf("解析 document.xml: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p":
				cur = &ParaFact{}
				runFont, runSize = "", 0
			case "r":
				inRun = cur != nil
			case "sectPr":
				inSectPr = true
			case "pgMar":
				if inSectPr {
					layout.MarginsFound = true
					layout.MarginTopMM = attrMM(t, "top")
					layout.MarginBottomMM = attrMM(t, "bottom")
					layout.MarginLeftMM = attrMM(t, "left")
					layout.MarginRightMM = attrMM(t, "right")
				}
			case "spacing":
				if cur != nil {
					cur.LineTwips = attrInt(t, "line")
					cur.LineRule = attrStr(t, "lineRule")
				}
			case "ind":
				if cur != nil {
					cur.FirstLineChars = attrInt(t, "firstLineChars")
					cur.FirstLineTwips = attrInt(t, "firstLine")
				}
			case "jc":
				if cur != nil && attrStr(t, "val") == "center" {
					cur.Center = true
				}
			case "rFonts":
				f := attrStr(t, "eastAsia")
				if f == "" {
					f = attrStr(t, "ascii")
				}
				if f == "" || cur == nil {
					break
				}
				if inRun {
					if runFont == "" {
						runFont = f
					}
				} else if cur.Font == "" {
					cur.Font = f
				}
			case "sz":
				v := attrInt(t, "val")
				if v <= 0 || cur == nil {
					break
				}
				if inRun {
					if runSize == 0 {
						runSize = v
					}
				} else if cur.SizeHalfPt == 0 {
					cur.SizeHalfPt = v
				}
			case "t":
				inText = true
			}
		case xml.CharData:
			if inText && cur != nil {
				cur.Text += string(t)
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "t":
				inText = false
			case "r":
				inRun = false
			case "p":
				if cur != nil {
					if runFont != "" {
						cur.Font = runFont
					}
					if runSize != 0 {
						cur.SizeHalfPt = runSize
					}
					layout.Paras = append(layout.Paras, *cur)
					cur = nil
				}
			case "sectPr":
				inSectPr = false
			}
		}
	}
	return layout, nil
}

func attrStr(t xml.StartElement, name string) string {
	for _, a := range t.Attr {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

func attrInt(t xml.StartElement, name string) int {
	v, _ := strconv.Atoi(attrStr(t, name))
	return v
}

func attrMM(t xml.StartElement, name string) float64 {
	tw := attrInt(t, name)
	if tw == 0 {
		return 0
	}
	return math.Round(float64(tw)/twipsPerMM*10) / 10
}

// reTitle 识别公文标题段落（机关+事由+文种）。
var reTitle = regexp.MustCompile(`(关于|印发|转发).*(通知|报告|请示|函|决定|意见|批复|纪要)|((通知|报告|请示|决定|批复|纪要)\s*$)`)

// FormatChecker 是 GB/T 9704 排版细则规范包（v4.223）。
type FormatChecker struct{}

// Name 规范包名。
func (FormatChecker) Name() string { return "GB/T 9704 排版细则" }

// CheckLayout 对排版事实做细则体检，每条 Issue 带 Spec 归属。
func (c FormatChecker) CheckLayout(l DocxLayout) []Issue {
	var issues []Issue
	issues = append(issues, checkMargins(l))
	issues = append(issues, checkTitle(l))
	issues = append(issues, checkBody(l))
	issues = append(issues, checkLineSpacing(l))
	issues = append(issues, checkIndent(l))
	issues = append(issues, checkHeadingFonts(l))
	for i := range issues {
		issues[i].Spec = c.Name()
	}
	return issues
}

// checkMargins 页边距：上37 下35 左28 右26 mm（±2mm 容差，四边聚一条）。
func checkMargins(l DocxLayout) Issue {
	const tol = 2.0
	it := Issue{Element: "页边距（上37 下35 左28 右26 mm）", Found: true}
	if !l.MarginsFound {
		it.Found = false
		it.Note = "未找到页面设置（sectPr/pgMar）；GB/T 9704 要求上37 下35 左28 右26 mm"
		return it
	}
	var bad []string
	for _, m := range []struct {
		name string
		got  float64
		want float64
	}{
		{"上", l.MarginTopMM, 37},
		{"下", l.MarginBottomMM, 35},
		{"左", l.MarginLeftMM, 28},
		{"右", l.MarginRightMM, 26},
	} {
		if m.got <= 0 || math.Abs(m.got-m.want) > tol {
			bad = append(bad, fmt.Sprintf("%s边实测 %.1fmm（应 %gmm）", m.name, m.got, m.want))
		}
	}
	if len(bad) > 0 {
		it.Found = false
		it.Note = "调整页边距：" + strings.Join(bad, "；")
	} else {
		it.Note = "符合"
	}
	return it
}

// checkTitle 公文标题字号：二号（44 半磅）。字体（小标宋）变体名多，只作建议
// 不参与判定——宁缺勿误。
func checkTitle(l DocxLayout) Issue {
	it := Issue{Element: "公文标题字号（二号 22pt）", Found: true}
	for _, p := range l.Paras {
		if reTitle.MatchString(p.Text) && len([]rune(p.Text)) <= 60 {
			if p.SizeHalfPt == 0 {
				it.Found = false
				it.Note = fmt.Sprintf("标题段落未检出字号定义，应设二号字（22pt）：%s", trunc(p.Text))
				return it
			}
			if p.SizeHalfPt != 44 {
				it.Found = false
				it.Note = fmt.Sprintf("标题实测 %s，应设二号字（22pt）", halfPtDesc(p.SizeHalfPt))
				return it
			}
			it.Note = "符合"
			if !strings.Contains(p.Font, "小标宋") {
				it.Note += "（标题字体建议用小标宋体）"
			}
			return it
		}
	}
	it.Note = "未识别到公文标题段落（关于…文种），本项不适用"
	return it
}

// checkBody 正文字体字号：三号（32 半磅）仿宋。主体=非标题非层级标题、长度
// ≥4 字的段落多数票。
func checkBody(l DocxLayout) Issue {
	it := Issue{Element: "正文字体字号（三号仿宋）", Found: true}
	vote := majorityFont(l)
	if vote == nil {
		it.Note = "正文段落不足，本项不适用"
		return it
	}
	if vote.size != 32 {
		it.Found = false
		it.Note = fmt.Sprintf("正文主体实测 %s，GB/T 9704 要求三号字（16pt）", halfPtDesc(vote.size))
		return it
	}
	if !isFangSong(vote.font) {
		it.Found = false
		it.Note = fmt.Sprintf("正文字体实测「%s」，应为仿宋（GB/T 9704）", vote.font)
		return it
	}
	it.Note = "符合"
	return it
}

// checkLineSpacing 行距：28~30 磅固定值（line=560~600 twip，lineRule=exact/atLeast）。
// 多数段落符合即视为符合（行距常在样式层统一）。
func checkLineSpacing(l DocxLayout) Issue {
	it := Issue{Element: "正文行距（28~30 磅）", Found: true}
	n, ok := 0, 0
	for _, p := range bodyParas(l) {
		n++
		if p.LineTwips >= 560 && p.LineTwips <= 600 && (p.LineRule == "exact" || p.LineRule == "atLeast") {
			ok++
		}
	}
	if n == 0 {
		it.Note = "正文段落不足，本项不适用"
		return it
	}
	if ok*2 >= n {
		it.Note = "符合"
		return it
	}
	it.Found = false
	it.Note = "正文行距多数不在 28~30 磅固定值（段落→行距→固定值 28 磅即可）"
	return it
}

// checkIndent 首行缩进：2 字符（firstLineChars=200，或 firstLine 560~720 twip）。
func checkIndent(l DocxLayout) Issue {
	it := Issue{Element: "正文首行缩进（2 字符）", Found: true}
	n, ok := 0, 0
	for _, p := range bodyParas(l) {
		n++
		if p.FirstLineChars == 200 || (p.FirstLineTwips >= 560 && p.FirstLineTwips <= 720) {
			ok++
		}
	}
	if n == 0 {
		it.Note = "正文段落不足，本项不适用"
		return it
	}
	if ok*2 >= n {
		it.Note = "符合"
		return it
	}
	it.Found = false
	it.Note = "正文多数段落未设置首行缩进 2 字符（段落→特殊格式→首行缩进 2 字符）"
	return it
}

// checkHeadingFonts 层级标题字体：一级「一、」黑体，二级「（一）」楷体
// （三级及以下变体多，不查；字体名缺失不判，宁缺勿误）。
func checkHeadingFonts(l DocxLayout) Issue {
	it := Issue{Element: "层级标题字体（一、黑体；（一）楷体）", Found: true}
	var bad []string
	seen := 0
	for _, p := range l.Paras {
		text := strings.TrimSpace(p.Text)
		switch {
		case strings.HasPrefix(text, "一、") || strings.HasPrefix(text, "二、") || strings.HasPrefix(text, "三、"):
			seen++
			if p.Font != "" && !strings.Contains(p.Font, "黑") {
				bad = append(bad, fmt.Sprintf("「%s…」实测「%s」应黑体", trunc(text), p.Font))
			}
		case strings.HasPrefix(text, "（一）") || strings.HasPrefix(text, "（二）"):
			seen++
			if p.Font != "" && !strings.Contains(p.Font, "楷") {
				bad = append(bad, fmt.Sprintf("「%s…」实测「%s」应楷体", trunc(text), p.Font))
			}
		}
	}
	if seen == 0 {
		it.Note = "未检出「一、/（一）」层级标题，本项不适用"
		return it
	}
	if len(bad) > 0 {
		it.Found = false
		it.Note = "调整层级标题字体：" + strings.Join(bad, "；")
	} else {
		it.Note = "符合"
	}
	return it
}

// bodyParas 参与正文投票的段落：非空、非标题、非层级标题、长度 ≥4 字。
func bodyParas(l DocxLayout) []ParaFact {
	var out []ParaFact
	for _, p := range l.Paras {
		text := strings.TrimSpace(p.Text)
		if len([]rune(text)) < 4 || reTitle.MatchString(text) {
			continue
		}
		if strings.HasPrefix(text, "一、") || strings.HasPrefix(text, "二、") ||
			strings.HasPrefix(text, "三、") || strings.HasPrefix(text, "（一）") ||
			strings.HasPrefix(text, "（二）") {
			continue
		}
		out = append(out, p)
	}
	return out
}

type fontVote struct {
	font  string
	size  int
	count int
}

// majorityFont 正文主体字体字号多数票。
func majorityFont(l DocxLayout) *fontVote {
	counts := map[string]*fontVote{}
	var order []string
	for _, p := range bodyParas(l) {
		key := p.Font + "|" + strconv.Itoa(p.SizeHalfPt)
		v, ok := counts[key]
		if !ok {
			v = &fontVote{font: p.Font, size: p.SizeHalfPt}
			counts[key] = v
			order = append(order, key)
		}
		v.count++
	}
	if len(order) == 0 {
		return nil
	}
	best := counts[order[0]]
	for _, k := range order[1:] {
		if counts[k].count > best.count {
			best = counts[k]
		}
	}
	return best
}

func isFangSong(font string) bool {
	return strings.Contains(font, "仿宋") || strings.Contains(strings.ToLower(font), "fangsong")
}

func halfPtDesc(half int) string {
	return fmt.Sprintf("%g 磅", float64(half)/2)
}

func trunc(s string) string {
	r := []rune(s)
	if len(r) > 12 {
		return string(r[:12])
	}
	return s
}
