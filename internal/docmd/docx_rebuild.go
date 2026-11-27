package docmd

// docx_rebuild.go — docx→Markdown 表格重建器（沿江包1 复盘 T1 的根修）。
//
// 背景：由表格软件从 PDF 转出的 Word（WPS pdf2word 同病）有三种结构性缺陷：
// 单元格内每个视觉折行都是独立 w:p、垂直居中产生大量空 w:p、目录域/目录残表
// 与正文无别。通用转换器（markitdown）与旧内置提取都会把这类文档转散架
// （沿江包1 实测列数矛盾 42/49 块、合并信息全丢、目录混正文）。
//
// 算法照搬沿江包1 项目目录里验证过的重建器（方案库录入/tools/docx2md_v2.py）：
//  1. 按 tblGrid 逻辑列展开 gridSpan/vMerge，每行列数与网格严格一致；
//  2. 单元格内折行做语义重拼（中文直接续接、列表项/句末断行用 <br> 分行）；
//  3. 目录域(sdt) 与目录残表识别剔除；图形文本框误包装成的表格（列数≤2 且
//     非空单元格全是图）转图片引用而非表格；
//  4. 图片按 rels 解析为 data URI 内嵌——与 markitdown keep-images 同契约，
//     ExternalizeDataImages 可照常落盘。
//
// 文字本身不在此修（错字/上下标以 PDF 原件对账为准）——本器只保证结构正确。

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

// ── 轻量 XML 树 ─────────────────────────────────────────────────────────────

// xmlNode 只留局部名与属性值：OOXML 的命名空间 URL 恒定，按局部名匹配足够。
type xmlNode struct {
	name     string
	attrs    map[string]string
	children []*xmlNode
	text     strings.Builder
}

func (n *xmlNode) attr(local string) string { return n.attrs[local] }

// find 返回直接子节点里第一个名为 local 的（表格/单元格的语义按直接子节点算）。
func (n *xmlNode) find(local string) *xmlNode {
	for _, c := range n.children {
		if c.name == local {
			return c
		}
	}
	return nil
}

func (n *xmlNode) findAll(local string) []*xmlNode {
	var out []*xmlNode
	for _, c := range n.children {
		if c.name == local {
			out = append(out, c)
		}
	}
	return out
}

// walk 深度优先遍历全部子孙。
func (n *xmlNode) walk(fn func(*xmlNode)) {
	for _, c := range n.children {
		fn(c)
		c.walk(fn)
	}
}

// parseXMLTree 把 XML 解成轻量树（encoding/xml 分词，命名空间只留局部名）。
func parseXMLTree(data []byte) (*xmlNode, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false // WPS 产物常见实体瑕疵，别为声明问题整份拒
	var stack []*xmlNode
	var root *xmlNode
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			node := &xmlNode{name: t.Name.Local, attrs: map[string]string{}}
			for _, a := range t.Attr {
				if a.Name.Local == "xmlns" {
					continue
				}
				node.attrs[a.Name.Local] = a.Value
			}
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.children = append(parent.children, node)
			} else if root == nil {
				root = node
			}
			stack = append(stack, node)
		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].text.Write(t)
			}
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	if root == nil {
		return nil, fmt.Errorf("document.xml 没解析出元素")
	}
	return root, nil
}

// ── 重建报告（回执的结构完整性指标，复盘 P0-2）─────────────────────────────

// DocxRebuildReport docx 结构体检与重建计数。列数一致率 = GridConformant/GridRows
// （源表口径：行展开后列数与 tblGrid 一致的行占比——脏文档的量化证据）；重建路线
// 的输出恒列数一致（展开+补齐是构造性的）。
type DocxRebuildReport struct {
	Rebuilt           bool `json:"rebuilt"`           // 产物是否出自重建路线（false=markitdown，报告只当体检）
	Paragraphs        int  `json:"paragraphs"`        // 保留的正文段落
	EmptyParasDropped int  `json:"emptyParasDropped"` // 剔除的空段落（垂直居中噪声）
	Headings          int  `json:"headings"`
	Tables            int  `json:"tables"`          // 保留为 md 表格的张数
	CatalogSkipped    int  `json:"catalogSkipped"`  // 目录残表剔除（编号+短标题形态）
	PseudoTables      int  `json:"pseudoTables"`    // 图形文本框误包装成的表格 → 转图片引用
	TocSkipped        int  `json:"tocSkipped"`      // 目录域(sdt)整块剔除
	GridRows          int  `json:"gridRows"`        // 全部源表的行数（展开前）
	GridConformant    int  `json:"gridConformant"`  // 其中展开后列数=网格列数的行
	MergedCells       int  `json:"mergedCells"`     // 识别并按网格展开的 gridSpan/vMerge 单元格
	FragmentedCells   int  `json:"fragmentedCells"` // 单元格内 ≥2 个非空段落的（折行碎片）
	Images            int  `json:"images"`          // 解析到的内嵌图（重建路线=已内嵌 data URI）
}

// dirtyDocx 报告该 docx 是否带「表格软件从 PDF 转出」的结构缺陷签名，值得走重建
// 路线而不是 markitdown：有合并网格（不展开列数必散）、有折行碎片单元格、有目录
// 残表、有行列数与网格不一致的行（缺格/溢出——markitdown 原样吐，必散架）。目录
// 域(sdt) 不算数——干净的现代 Word 也带 TOC sdt，单凭它换路线会白白丢掉
// markitdown 的保真。
func (r *DocxRebuildReport) dirtyDocx() bool {
	return r.MergedCells > 0 || r.FragmentedCells > 0 || r.CatalogSkipped > 0 || r.GridConformant < r.GridRows
}

// GridConformanceRate 列数一致率（0~1；无表格行时为 1）。
func (r *DocxRebuildReport) GridConformanceRate() float64 {
	if r.GridRows == 0 {
		return 1
	}
	return float64(r.GridConformant) / float64(r.GridRows)
}

// ── 文本规整（docx2md_v2.py 逐条移植）───────────────────────────────────────

const brMark = "\u2028" // w:br/w:cr 的换行标记（非打印位，重拼后必被消费掉）

const imgPh = "@@IMG@"

var (
	reSpaces = regexp.MustCompile(`[ \t\x{00a0}]+`)
	reCJKGap = regexp.MustCompile("([\\x{4e00}-\\x{9fff}\\x{3000}-\\x{303f}\\x{ff00}-\\x{ffef}])[ \\t\\x{00a0}]+([\\x{4e00}-\\x{9fff}\\x{3000}-\\x{303f}\\x{ff00}-\\x{ffef}])")
	reBRPad  = regexp.MustCompile(`\s*(<br>)\s*`)
	// 列表项头（RE2 不支持 (?!...) 负向断言，数字分支的「排除小数」在 isListHead 补判）。
	reListHeadNum   = regexp.MustCompile(`^[0-9]{1,2}\s*[.．、)）]`)
	reListHeadOther = regexp.MustCompile(`^\s*(?:[（(]\s*[0-9一二三四五六七八九十]{1,3}\s*[）)]|[\x{2460}-\x{2473}]|[一二三四五六七八九十]{1,3}\s*[、．.]|[·•▪●○◆※]|[A-Za-z]\s*[)）.．])`)
	reEndPunct      = regexp.MustCompile("[。；！？：;”\"」』）)】》]\\s*$")
	reTabCap        = regexp.MustCompile(`^\s*表\s*\d+`)
	reFigCap        = regexp.MustCompile(`^\s*图\s*\d+`)
	reHeading       = regexp.MustCompile(`^\s*(\d{1,2}(?:\.\d{1,3}){0,3})\s+(\S.*)$`)
	// 「（1）xxx」被转成残形「1 ）xxx」——括号残留，是列举项不是标题。
	reBrokenParen = regexp.MustCompile(`^\s*[）)、]`)
	reTOCLine     = regexp.MustCompile(`^\s*\d+(\.\d{1,3}){0,3}\s+\S`)
	reTOCDots     = regexp.MustCompile(`\.{4,}\s*[0-9]+$`)
	reTOCCell     = regexp.MustCompile(`^\s*(\d{1,2}(?:\.\d{1,3}){0,3})\.?\s*(\S.*)$`)
	rePureNum     = regexp.MustCompile(`^[0-9]{1,4}$`)
)

// cleanSpaces 规整一段文本：吞换行标记、折叠空白、吃掉汉字间残留空格。
func cleanSpaces(s string) string {
	s = strings.ReplaceAll(s, brMark, "")
	s = reSpaces.ReplaceAllString(s, " ")
	s = reCJKGap.ReplaceAllString(s, "$1$2")
	s = reBRPad.ReplaceAllString(s, "$1")
	return strings.TrimSpace(s)
}

// headingText 标题文本规整：只折叠空白，不吃汉字间空格——标题是身份文本，
// 方案库录入按标题对行（与正文断词重拼的口径分开）。
func headingText(s string) string {
	s = strings.ReplaceAll(s, brMark, "")
	s = reSpaces.ReplaceAllString(s, " ")
	s = reBRPad.ReplaceAllString(s, "$1")
	return strings.TrimSpace(s)
}

// smartJoin 单元格内折行语义重拼：同一单元格里的视觉折行按语义接回去。中文接
// 中文直接相连；边界任一侧是 ASCII 字母数字时补一个空格；新片段是列表项、或上一
// 片段以句末标点收尾时用 <br> 分行。这一步是断词问题（「按照相关技术导」+
// 「则及生态环境行政主管部门要」）的解药。
func smartJoin(lines []string) string {
	var frags []string
	for _, l := range lines {
		for _, x := range strings.Split(l, brMark) {
			if x = strings.TrimSpace(x); x != "" {
				frags = append(frags, x)
			}
		}
	}
	if len(frags) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(frags[0])
	for _, f := range frags[1:] {
		prev := b.String()
		if isListHead(f) || reEndPunct.MatchString(prev) {
			b.WriteString("<br>")
			b.WriteString(f)
			continue
		}
		if isASCIILetterDigit(lastRune(prev)) || isASCIILetterDigit(firstRune(f)) {
			b.WriteString(" ")
		}
		b.WriteString(f)
	}
	return cleanSpaces(b.String())
}

func firstRune(s string) string {
	for _, r := range s {
		return string(r)
	}
	return ""
}

func lastRune(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return ""
	}
	return string(r[len(r)-1])
}

// isListHead 列表项头：数字分支排除小数（「1.5 倍」不是列表项，「1、场地平整」是）。
func isListHead(f string) bool {
	t := strings.TrimLeft(f, " \t")
	if m := reListHeadNum.FindString(t); m != "" {
		rest := t[len(m):]
		return len(rest) == 0 || rest[0] < '0' || rest[0] > '9'
	}
	return reListHeadOther.MatchString(t)
}

func isASCIILetterDigit(s string) bool {
	if s == "" {
		return false
	}
	c := s[0]
	return c < 0x80 && (('0' <= c && c <= '9') || ('A' <= c && c <= 'Z') || ('a' <= c && c <= 'z'))
}

// paraText 提取一个 w:p 的文本：w:t 文字、w:tab→空格、w:br/w:cr→换行标记。
func paraText(p *xmlNode) string {
	var b strings.Builder
	var walk func(*xmlNode)
	walk = func(n *xmlNode) {
		switch n.name {
		case "t":
			b.WriteString(n.text.String())
		case "tab":
			b.WriteString(" ")
		case "br", "cr":
			b.WriteString(brMark)
		}
		for _, c := range n.children {
			walk(c)
		}
	}
	walk(p)
	return b.String()
}

// paraImages 提取节点里全部图片 rId（a:blip 的 r:embed）。
func paraImages(n *xmlNode) []string {
	var ids []string
	var walk func(*xmlNode)
	walk = func(x *xmlNode) {
		if x.name == "blip" {
			if rid := x.attr("embed"); rid != "" {
				ids = append(ids, rid)
			}
		}
		for _, c := range x.children {
			walk(c)
		}
	}
	walk(n)
	return ids
}

// paragraphHeading 判标题级别：先看段落样式（干净文档的主信号），无样式再看编号
// 形态（WPS 从 PDF 转出的文档段落全是 Normal，编号是唯一信号）。表题/图题永远
// 不是标题。
func paragraphHeading(p *xmlNode, text string) int {
	if reTabCap.MatchString(text) || reFigCap.MatchString(text) {
		return 0
	}
	if pPr := p.find("pPr"); pPr != nil {
		if st := pPr.find("pStyle"); st != nil {
			switch v := st.attr("val"); v {
			case "Title", "title", "Heading1", "heading1", "1":
				return 1
			case "Heading2", "heading2", "2":
				return 2
			case "Heading3", "heading3", "3":
				return 3
			default:
				if strings.HasPrefix(v, "Heading") || strings.HasPrefix(v, "heading") {
					if lvl := headingLevelFromStyle(v); lvl > 0 {
						return lvl
					}
				}
			}
		}
	}
	t := []rune(strings.TrimSpace(text))
	if len(t) > 45 || strings.ContainsAny(string(t), "。；，！？") {
		return 0
	}
	m := reHeading.FindStringSubmatch(string(t))
	if m == nil {
		return 0
	}
	rest := m[2]
	if reBrokenParen.MatchString(rest) || strings.HasPrefix(rest, "章节") || strings.HasPrefix(rest, "条") || strings.HasPrefix(rest, "款") {
		return 0
	}
	depth := strings.Count(m[1], ".") + 1
	if depth > 4 {
		return 0
	}
	return depth
}

func headingLevelFromStyle(v string) int {
	digits := strings.TrimLeft(strings.TrimPrefix(strings.TrimPrefix(v, "Heading"), "heading"), " ")
	if len(digits) == 1 && digits[0] >= '1' && digits[0] <= '9' {
		lvl := int(digits[0] - '0')
		if lvl > 6 {
			lvl = 6
		}
		return lvl
	}
	return 0
}

// ── docx 打开与素材 ─────────────────────────────────────────────────────────

type docxDoc struct {
	root *xmlNode          // document.xml 树
	zr   *zip.Reader       // 媒体懒加载用
	rels map[string]string // rId → media 目标（如 media/image1.png）
}

func openDocxDoc(path string) (*docxDoc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("不是有效的 docx 文件: %w", err)
	}
	var docXML, relsXML []byte
	for _, f := range zr.File {
		switch f.Name {
		case "word/document.xml":
			if docXML, err = readZipEntry(f); err != nil {
				return nil, err
			}
		case "word/_rels/document.xml.rels":
			if relsXML, err = readZipEntry(f); err != nil {
				return nil, err
			}
		}
	}
	if docXML == nil {
		return nil, fmt.Errorf("未找到 word/document.xml")
	}
	root, err := parseXMLTree(docXML)
	if err != nil {
		return nil, fmt.Errorf("解析 document.xml 失败: %w", err)
	}
	d := &docxDoc{root: root, zr: zr, rels: map[string]string{}}
	if relNode, err2 := parseXMLTree(relsXML); err2 == nil {
		for _, rel := range relNode.findAll("Relationship") {
			if target := rel.attr("Target"); strings.HasPrefix(target, "media/") {
				d.rels[rel.attr("Id")] = target
			}
		}
	}
	return d, nil
}

func readZipEntry(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// mediaBytes 懒加载某个 rId 对应的媒体字节；缺失返回 nil。
func (d *docxDoc) mediaBytes(rid string) []byte {
	target, ok := d.rels[rid]
	if !ok {
		return nil
	}
	for _, f := range d.zr.File {
		if f.Name == "word/"+target {
			b, err := readZipEntry(f)
			if err != nil {
				return nil
			}
			return b
		}
	}
	return nil
}

// imageDataURI 把一个 rId 转成 data URI；媒体缺失返回 ""。
func (d *docxDoc) imageDataURI(rid string) string {
	b := d.mediaBytes(rid)
	if len(b) == 0 {
		return ""
	}
	mime := imageMIME(strings.ToLower(path.Ext(d.rels[rid])))
	if mime == "" {
		mime = "application/octet-stream"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b)
}

func imageMIME(ext string) string {
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	case ".webp":
		return "image/webp"
	case ".tif", ".tiff":
		return "image/tiff"
	case ".emf":
		return "image/x-emf"
	case ".wmf":
		return "image/x-wmf"
	default:
		return ""
	}
}

// ── 重建主流程 ─────────────────────────────────────────────────────────────

// RebuildDocx 重建路线的 docx→Markdown（含 data URI 图片内嵌）。
func RebuildDocx(path string) (string, DocxRebuildReport, error) {
	d, err := openDocxDoc(path)
	if err != nil {
		return "", DocxRebuildReport{}, err
	}
	md, rep := d.render(true)
	return md, rep, nil
}

// AnalyzeDocx 只做结构体检不产产物（图片不内嵌，毫秒级）：路由决策与回执指标用。
func AnalyzeDocx(path string) (DocxRebuildReport, error) {
	d, err := openDocxDoc(path)
	if err != nil {
		return DocxRebuildReport{}, err
	}
	_, rep := d.render(false)
	return rep, nil
}

func (d *docxDoc) render(withImages bool) (string, DocxRebuildReport) {
	var rep DocxRebuildReport
	body := d.root.find("body")
	if body == nil {
		return "", rep
	}
	rep.Rebuilt = true
	// 预扫：收集正文编号标题清单（目录残表交叉比对用——复盘终版口径：形态占比
	// 超四成且单元格对得上正文标题才判目录，防误吃「编号+短标题」的真内容表）。
	hs := d.collectHeadings(body)
	var parts []string
	d.emitBlocks(body, withImages, hs, &parts, &rep)
	return strings.Join(parts, "\n\n") + "\n", rep
}

// headingIndex 正文编号标题清单（目录残表交叉比对的对照面）。
type headingIndex struct {
	full   map[string]bool // 归一化全文（去空格/点/、）
	nums   map[string]bool // 编号（如 "2"、"2.1"）
	titles map[string]bool // 编号后的标题（归一化）
}

func (h *headingIndex) add(text string) {
	m := reHeading.FindStringSubmatch(strings.TrimSpace(text))
	if m == nil {
		return
	}
	h.nums[m[1]] = true
	title := normHeadingKey(m[2])
	if title != "" {
		h.titles[title] = true
	}
	h.full[normHeadingKey(m[1]+m[2])] = true
}

// tocLikeCell 单元格是否是「对得上正文标题的目录行」。编号命中要求多段编号
// （如 2.1）——单段章号（"1"）谁都可能是，纯数字序号列不数。
func (h *headingIndex) tocLikeCell(c string) bool {
	t := strings.TrimSpace(c)
	if t == "" {
		return false
	}
	if rePureNum.MatchString(t) {
		return strings.Contains(t, ".") && h.nums[t]
	}
	if m := reTOCCell.FindStringSubmatch(t); m != nil {
		title := m[2]
		if utf8RuneCount(title) <= 40 && !strings.ContainsAny(title, "。，；") {
			if (strings.Contains(m[1], ".") && h.nums[m[1]]) || h.titles[normHeadingKey(title)] {
				return true
			}
		}
	}
	return h.full[normHeadingKey(t)]
}

// collectHeadings 收集正文层编号标题（表格/内容控件里的段落不算——目录残表
// 自己的行不能给判定当对照面）。
func (d *docxDoc) collectHeadings(body *xmlNode) *headingIndex {
	hs := &headingIndex{full: map[string]bool{}, nums: map[string]bool{}, titles: map[string]bool{}}
	var walk func(n *xmlNode)
	walk = func(n *xmlNode) {
		switch n.name {
		case "tbl", "sdt":
			return
		case "p":
			if text := cleanSpaces(paraText(n)); text != "" {
				hs.add(text)
			}
			return
		}
		for _, c := range n.children {
			walk(c)
		}
	}
	walk(body)
	return hs
}

func normHeadingKey(s string) string {
	repl := strings.NewReplacer(" ", "", "\t", "", ".", "", "．", "", "、", "", "　", "")
	return repl.Replace(strings.TrimSpace(s))
}

// emitBlocks 处理一组块级节点（body 或 sdtContent 的直接子节点）。
func (d *docxDoc) emitBlocks(parent *xmlNode, withImages bool, hs *headingIndex, parts *[]string, rep *DocxRebuildReport) {
	for _, ch := range parent.children {
		switch ch.name {
		case "p":
			d.emitParagraph(ch, withImages, parts, rep)
		case "tbl":
			d.emitTable(ch, withImages, hs, parts, rep)
		case "sdt":
			// 目录域：内容以「编号+短标题/点线+页码」形态为主时整块剔除；
			// 别的内容控件（模板里常见的正文包裹）降级进内容照常处理。
			content := ch.find("sdtContent")
			if content == nil {
				continue
			}
			if sdtLooksLikeTOC(content) {
				rep.TocSkipped++
				continue
			}
			d.emitBlocks(content, withImages, hs, parts, rep)
		}
	}
}

// sdtLooksLikeTOC 目录域签名：非空段落过半是「编号 标题」或「点线…页码」短行。
func sdtLooksLikeTOC(content *xmlNode) bool {
	total, toc := 0, 0
	content.walk(func(n *xmlNode) {
		if n.name != "p" {
			return
		}
		t := strings.TrimSpace(cleanSpaces(paraText(n)))
		if t == "" {
			return
		}
		total++
		if (reTOCLine.MatchString(t) && utf8RuneCount(t) <= 40 && !strings.ContainsAny(t, "。；，")) || reTOCDots.MatchString(t) {
			toc++
		}
	})
	return total > 0 && toc*2 > total
}

func (d *docxDoc) emitParagraph(p *xmlNode, withImages bool, parts *[]string, rep *DocxRebuildReport) {
	rids := paraImages(p)
	raw := paraText(p)
	text := cleanSpaces(raw)
	if text == "" && len(rids) == 0 {
		rep.EmptyParasDropped++
		return
	}
	if text != "" {
		if lvl := paragraphHeading(p, text); lvl > 0 {
			// 标题是身份文本：从原始文本规整（只折叠空白、不吃字间空格）——
			// 「第一章 工艺」不能被规整成「第一章工艺」，方案库 outline 按标题对行会断。
			*parts = append(*parts, strings.Repeat("#", lvl)+" "+headingText(raw))
			rep.Headings++
			text = ""
		}
	}
	if text != "" {
		*parts = append(*parts, text)
		rep.Paragraphs++
	}
	for _, rid := range rids {
		if uri := d.imageDataURI(rid); uri != "" {
			rep.Images++
			if withImages {
				*parts = append(*parts, "![]("+uri+")")
			}
		}
	}
}

// emitTable 按 tblGrid 展开一张表，走目录残表剔除与伪表格识别。
func (d *docxDoc) emitTable(tbl *xmlNode, withImages bool, hs *headingIndex, parts *[]string, rep *DocxRebuildReport) {
	ncols := 0
	if grid := tbl.find("tblGrid"); grid != nil {
		ncols = len(grid.findAll("gridCol"))
	}
	var rows [][]string
	for _, tr := range tbl.findAll("tr") {
		var expanded []string
		conformant := true
		for _, tc := range tr.findAll("tc") {
			gridSpan := 1
			if tcPr := tc.find("tcPr"); tcPr != nil {
				if g := tcPr.find("gridSpan"); g != nil {
					if v, err := strconv.Atoi(g.attr("val")); err == nil && v > 1 {
						gridSpan = v
						rep.MergedCells++
					}
				}
				if v := tcPr.find("vMerge"); v != nil {
					rep.MergedCells++ // restart 与 continue 都计入网格语义
				}
			}
			paras, images := cellParasAndImages(tc)
			rep.EmptyParasDropped += len(paras) - countNonEmpty(paras)
			if countNonEmpty(paras) >= 2 {
				rep.FragmentedCells++
			}
			txt := smartJoin(paras)
			for _, rid := range images {
				if uri := d.imageDataURI(rid); uri != "" {
					rep.Images++
					if withImages {
						if txt != "" {
							txt += " "
						}
						txt += imgPh + uri + imgPh
					}
				}
			}
			expanded = append(expanded, txt)
			for i := 1; i < gridSpan; i++ {
				expanded = append(expanded, "")
			}
		}
		rep.GridRows++
		if ncols > 0 && len(expanded) != ncols {
			conformant = false // 展开后列数与网格不一致（源表结构错误）
		}
		if conformant {
			rep.GridConformant++
		}
		rows = append(rows, expanded)
	}
	rows = normalizeRows(rows, ncols)
	if len(rows) == 0 {
		return
	}
	// 目录残表：形态占比 ≥75%（无对照面也成立的强形态）或 ≥40% 且单元格对得上
	// 正文标题清单（复盘终版口径——防误吃编号+短标题的真内容表）。
	if isCatalogTable(rows) || (len(hs.nums) > 0 && catalogByHeadings(rows, hs)) {
		rep.CatalogSkipped++
		return
	}
	// 伪表格：图形文本框被包装成单行少列表——列数≤2 且非空单元格全是图 → 图片引用。
	if ncols <= 2 && tableIsImageOnly(rows) {
		rep.PseudoTables++
		for _, r := range rows {
			for _, c := range r {
				if uri := extractImgURI(c); uri != "" {
					*parts = append(*parts, "![]("+uri+")")
				}
			}
		}
		return
	}
	// 单元格内图片占位换成真实图引。
	for _, r := range rows {
		for i := range r {
			if strings.Contains(r[i], imgPh) {
				r[i] = cleanSpaces(replaceImgPlaceholders(r[i]))
			}
		}
	}
	if md := renderTableMD(ncols, rows); md != "" {
		*parts = append(*parts, md)
		rep.Tables++
	}
}

// normalizeRows 把每行列数规整到 ncols：短行补空，长行把溢出内容并进最后一列。
// 输出列数矛盾=0 是构造性保证。
func normalizeRows(rows [][]string, ncols int) [][]string {
	if ncols <= 0 {
		for _, r := range rows {
			if len(r) > ncols {
				ncols = len(r)
			}
		}
	}
	if ncols <= 0 {
		return rows
	}
	out := make([][]string, len(rows))
	for i, r := range rows {
		switch {
		case len(r) < ncols:
			row := append([]string{}, r...)
			for len(row) < ncols {
				row = append(row, "")
			}
			out[i] = row
		case len(r) > ncols:
			row := append([]string{}, r[:ncols-1]...)
			var extra []string
			for _, x := range r[ncols-1:] {
				if strings.TrimSpace(x) != "" {
					extra = append(extra, x)
				}
			}
			out[i] = append(row, strings.Join(extra, " "))
		default:
			out[i] = r
		}
	}
	return out
}

// cellParasAndImages 收集单元格的直接段落与图片 rId；嵌套小表的单元格文本拍平
// 成段落追加（信息不丢，形态退化为 <br> 分行文本——与验证过的重建器同口径）。
func cellParasAndImages(tc *xmlNode) (paras []string, images []string) {
	var walk func(n *xmlNode, inNested bool)
	walk = func(n *xmlNode, inNested bool) {
		switch n.name {
		case "p":
			paras = append(paras, paraText(n))
			images = append(images, paraImages(n)...)
			return
		case "tbl":
			if inNested {
				for _, tr := range n.findAll("tr") {
					for _, sub := range tr.findAll("tc") {
						sp, im := cellParasAndImages(sub)
						paras = append(paras, sp...)
						images = append(images, im...)
					}
				}
				return
			}
		case "tc":
			if inNested {
				// 嵌套表的单元格：其直接子节点由上面的 tr/tc 循环处理。
				for _, c := range n.children {
					walk(c, true)
				}
				return
			}
		}
		for _, c := range n.children {
			walk(c, inNested || n.name == "tbl")
		}
	}
	for _, c := range tc.children {
		walk(c, false)
	}
	return paras, images
}

// isCatalogTable 目录残表判定（验证过的重建器同阈值）：非空单元格里 ≥75% 是
// 「编号+短标题」或纯数字形态。空表（全空单元格）也剔除。
func isCatalogTable(rows [][]string) bool {
	var nonempty []string
	for _, r := range rows {
		for _, c := range r {
			if strings.TrimSpace(c) != "" {
				nonempty = append(nonempty, strings.TrimSpace(c))
			}
		}
	}
	if len(nonempty) == 0 {
		return true
	}
	shortLike := 0
	for _, c := range nonempty {
		if reTOCLine.MatchString(c) && utf8RuneCount(c) <= 40 && !strings.ContainsAny(c, "。，；：") {
			shortLike++
		} else if rePureNum.MatchString(c) {
			shortLike++
		}
	}
	return shortLike*4 >= len(nonempty)*3
}

// catalogByHeadings 目录残表的交叉比对判定：非空单元格里 ≥40% 是「编号+短标题」
// 且标题/编号对得上正文标题清单。
func catalogByHeadings(rows [][]string, hs *headingIndex) bool {
	var nonempty []string
	for _, r := range rows {
		for _, c := range r {
			if strings.TrimSpace(c) != "" {
				nonempty = append(nonempty, c)
			}
		}
	}
	if len(nonempty) == 0 {
		return false
	}
	hits := 0
	for _, c := range nonempty {
		if hs.tocLikeCell(c) {
			hits++
		}
	}
	return hits*5 >= len(nonempty)*2
}

// tableIsImageOnly 伪表格签名：有图，且每个非空单元格都是「纯图无文字」——
// 图形文本框被转表软件包成单行少列表的形态。有文字的单元格一律不算。
func tableIsImageOnly(rows [][]string) bool {
	anyImg := false
	for _, r := range rows {
		for _, c := range r {
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}
			if uri := extractImgURI(c); uri != "" {
				anyImg = true
				if strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(c, imgPh+uri+imgPh, ""), imgPh, "")) != "" {
					return false // 图外还有文字：真表格
				}
			} else {
				return false // 有不含图的文字单元格：真表格
			}
		}
	}
	return anyImg
}

func extractImgURI(s string) string {
	i := strings.Index(s, imgPh)
	if i < 0 {
		return ""
	}
	rest := s[i+len(imgPh):]
	j := strings.Index(rest, imgPh)
	if j < 0 {
		return ""
	}
	return rest[:j]
}

// replaceImgPlaceholders 把 @@IMG@uri@@ 占位换成 md 图引。
func replaceImgPlaceholders(s string) string {
	var b strings.Builder
	for {
		i := strings.Index(s, imgPh)
		if i < 0 {
			b.WriteString(s)
			return b.String()
		}
		b.WriteString(s[:i])
		rest := s[i+len(imgPh):]
		j := strings.Index(rest, imgPh)
		if j < 0 {
			b.WriteString(imgPh)
			b.WriteString(rest)
			return b.String()
		}
		if uri := rest[:j]; uri != "" {
			b.WriteString("![](")
			b.WriteString(uri)
			b.WriteString(")")
		}
		s = rest[j+len(imgPh):]
	}
}

// renderTableMD 渲染 md 管道表：竖线转义、全空数据行剔除、首行当表头。
func renderTableMD(ncols int, rows [][]string) string {
	if len(rows) == 0 || ncols <= 0 {
		return ""
	}
	escape := func(c string) string { return strings.ReplaceAll(c, "|", `\|`) }
	var b strings.Builder
	b.WriteString("| ")
	for i := 0; i < ncols; i++ {
		if i > 0 {
			b.WriteString(" | ")
		}
		b.WriteString(escape(cellAt(rows[0], i)))
	}
	b.WriteString(" |\n|")
	for i := 0; i < ncols; i++ {
		b.WriteString(" --- |")
	}
	b.WriteString("\n")
	for _, r := range rows[1:] {
		empty := true
		for _, c := range r {
			if strings.TrimSpace(c) != "" {
				empty = false
				break
			}
		}
		if empty {
			continue
		}
		b.WriteString("| ")
		for i := 0; i < ncols; i++ {
			if i > 0 {
				b.WriteString(" | ")
			}
			b.WriteString(escape(cellAt(r, i)))
		}
		b.WriteString(" |\n")
	}
	return b.String()
}

func cellAt(r []string, i int) string {
	if i < len(r) {
		return r[i]
	}
	return ""
}

func countNonEmpty(ss []string) int {
	n := 0
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			n++
		}
	}
	return n
}

func utf8RuneCount(s string) int { return len([]rune(s)) }
