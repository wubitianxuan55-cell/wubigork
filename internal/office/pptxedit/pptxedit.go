// Package pptxedit — pptx 文本框文本的 run 级直接替换（pptx 真编辑刀1 数据层，
// docs/gaea-pptx-edit-design-2026-09.md；与 internal/office/docxedit 同构）。
//
// 设计要点：
//   - 只动命中 slide 的 slide XML 内被选区覆盖的 a:t 文本；run 的 a:rPr 原字节
//     原样保留（新文本继承最后一个受影响 run 的 rPr）→ 天然保格式；其余 zip
//     条目字节不动（母版/主题/动画零扰动）。
//   - 无修订制（pptx 无 w:ins/w:del 等价物）：直接替换，回滚走应用层快照。
//   - 段落解析器 = docxedit WML 版的 DrawingML 移植：a:p/a:r/a:t/a:rPr，
//     blocker 换 a:br/a:fld/a:tab；a:fld 内文本可见但不可编辑（记 blocker）。
//   - 表格/组合形状内的 a:p 解析器自然覆盖（设计拍板项 3：包含）。
//   - 原子写回：临时文件 + fileutil.RenameWithRetry（带瞬时占用重试）；zip 条目顺序原样保留。
package pptxedit

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/gaea/gaea/internal/gaea/fileutil"
)

// SlideText 是单页幻灯片的可编辑文本定位结果（页码 1 起）。
type SlideText struct {
	Index      int      `json:"index"`
	Paragraphs []string `json:"paragraphs"`
}

// LocateText 按页码顺序返回全部 slide 的段落文本（供 agent 定位改写目标）。
func LocateText(path string) ([]SlideText, error) {
	doc, err := readPptx(path)
	if err != nil {
		return nil, err
	}
	parts, err := slideParts(doc)
	if err != nil {
		return nil, err
	}
	out := make([]SlideText, 0, len(parts))
	for i, part := range parts {
		paras, err := parseSlideParagraphs(doc.files[part])
		if err != nil {
			return nil, fmt.Errorf("解析 %s 失败: %w", part, err)
		}
		st := SlideText{Index: i + 1}
		for _, p := range paras {
			if strings.TrimSpace(p.text) != "" {
				st.Paragraphs = append(st.Paragraphs, p.text)
			}
		}
		out = append(out, st)
	}
	return out, nil
}

// ApplyTextReplace 在第 slideIdx 页（1 起）查找 target 并替换为 replacement
//（首个命中）。跨 run 文本可命中；新文本继承最后受影响 run 的 rPr 原字节。
// 返回人类可读摘要。
func ApplyTextReplace(path string, slideIdx int, target, replacement string) (string, error) {
	if slideIdx < 1 {
		return "", fmt.Errorf("页码从 1 起")
	}
	if strings.TrimSpace(target) == "" {
		return "", fmt.Errorf("替换目标为空")
	}
	doc, err := readPptx(path)
	if err != nil {
		return "", err
	}
	parts, err := slideParts(doc)
	if err != nil {
		return "", err
	}
	if slideIdx > len(parts) {
		return "", fmt.Errorf("第 %d 页不存在（共 %d 页）", slideIdx, len(parts))
	}
	part := parts[slideIdx-1]
	slideXML := doc.files[part]
	paras, err := parseSlideParagraphs(slideXML)
	if err != nil {
		return "", fmt.Errorf("解析 %s 失败: %w", part, err)
	}
	// 找首个能安全命中的段落
	for _, p := range paras {
		s, e, ok := locateSpan(p, target)
		if !ok {
			continue
		}
		for _, b := range p.blockers {
			if b.pos >= s && b.pos < e {
				return "", fmt.Errorf("选区包含特殊元素（换行/字段/制表符），暂不支持，请缩小目标后重试")
			}
		}
		newPara, err := rebuildParagraph(slideXML, p, s, e, replacement)
		if err != nil {
			return "", err
		}
		out := append([]byte{}, slideXML[:p.start]...)
		out = append(out, newPara...)
		out = append(out, slideXML[p.end:]...)
		doc.files[part] = out
		if err := writePptx(path, doc); err != nil {
			return "", err
		}
		return fmt.Sprintf("第 %d 页替换「%s」→「%s」", slideIdx, target, replacement), nil
	}
	return "", fmt.Errorf("第 %d 页未找到目标文本：%s", slideIdx, target)
}

// ── zip 读写（同 docxedit：条目序保留 + 临时文件原子替换） ──────────

type pptxFile struct {
	files map[string][]byte
	order []string
}

func readPptx(path string) (*pptxFile, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("打开 pptx 失败: %w", err)
	}
	defer r.Close()
	doc := &pptxFile{files: map[string][]byte{}}
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("读取 %s 失败: %w", f.Name, err)
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("读取 %s 失败: %w", f.Name, err)
		}
		doc.files[f.Name] = b
		doc.order = append(doc.order, f.Name)
	}
	if doc.files["ppt/presentation.xml"] == nil {
		return nil, fmt.Errorf("pptx 缺少 ppt/presentation.xml")
	}
	return doc, nil
}

func writePptx(path string, doc *pptxFile) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".gaea-pptxedit-*.pptx")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	zw := zip.NewWriter(tmp)
	for _, name := range doc.order {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate})
		if err != nil {
			zw.Close()
			tmp.Close()
			return fmt.Errorf("写回 %s 失败: %w", name, err)
		}
		if _, err := w.Write(doc.files[name]); err != nil {
			zw.Close()
			tmp.Close()
			return fmt.Errorf("写回 %s 失败: %w", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		tmp.Close()
		return fmt.Errorf("打包 pptx 失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("关闭临时文件失败: %w", err)
	}
	if err := fileutil.RenameWithRetry(tmpName, path); err != nil {
		return fmt.Errorf("替换原文件失败: %w", err)
	}
	return nil
}

// ── slide 序映射：presentation.xml sldIdLst + rels → 页码 → part 路径 ──

const relsNS = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"

// slideParts 返回按放映顺序的 slide part 路径（"ppt/slides/slide1.xml"）。
func slideParts(doc *pptxFile) ([]string, error) {
	presRaw := doc.files["ppt/presentation.xml"]
	relsRaw := doc.files["ppt/_rels/presentation.xml.rels"]
	if relsRaw == nil {
		return nil, fmt.Errorf("pptx 缺少 ppt/_rels/presentation.xml.rels（结构异常）")
	}
	// sldIdLst 顺序里的 r:id（Token 模式属性命名空间已解析为 URL，可靠）
	var rids []string
	dec := xml.NewDecoder(bytes.NewReader(presRaw))
	inSldIdLst := false
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("解析 presentation.xml 失败: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "sldIdLst" {
				inSldIdLst = true
				continue
			}
			if inSldIdLst && t.Name.Local == "sldId" {
				for _, a := range t.Attr {
					if a.Name.Local == "id" && a.Name.Space == relsNS {
						rids = append(rids, a.Value)
					}
				}
			}
		case xml.EndElement:
			if t.Name.Local == "sldIdLst" {
				inSldIdLst = false
			}
		}
	}
	if len(rids) == 0 {
		return nil, fmt.Errorf("presentation.xml 无幻灯片（sldIdLst 为空）")
	}
	// rels: Id → Target（属性无前缀，Space 为空）
	targets := map[string]string{}
	d2 := xml.NewDecoder(bytes.NewReader(relsRaw))
	for {
		tok, err := d2.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("解析 presentation.xml.rels 失败: %w", err)
		}
		if t, ok := tok.(xml.StartElement); ok && t.Name.Local == "Relationship" {
			var id, target string
			for _, a := range t.Attr {
				switch {
				case a.Name.Local == "Id" && a.Name.Space == "":
					id = a.Value
				case a.Name.Local == "Target" && a.Name.Space == "":
					target = a.Value
				}
			}
			if id != "" && target != "" {
				targets[id] = target
			}
		}
	}
	parts := make([]string, 0, len(rids))
	for i, rid := range rids {
		target, ok := targets[rid]
		if !ok {
			return nil, fmt.Errorf("presentation.xml.rels 缺少 %s 的映射", rid)
		}
		t := strings.ReplaceAll(target, "\\", "/")
		switch {
		case strings.HasPrefix(t, "/"):
			t = strings.TrimPrefix(t, "/")
		case strings.HasPrefix(t, "ppt/"):
		default:
			t = "ppt/" + t
		}
		if doc.files[t] == nil {
			return nil, fmt.Errorf("第 %d 页的 %s 不在包内", i+1, t)
		}
		parts = append(parts, t)
	}
	return parts, nil
}

// ── slide XML 字节级手术（DrawingML） ─────────────────────────────

const dmlNS = "http://schemas.openxmlformats.org/drawingml/2006/main"

func isDML(name xml.Name, local string) bool {
	return name.Local == local && strings.Contains(name.Space, "drawingml")
}

// textSeg 是段落中一个可编辑文本单元（a:t 的文本内容，隶属 a:r run）。
type textSeg struct {
	text              string
	runTagStart       int // a:r 起始标签起点
	runTagEnd         int // a:r 起始标签终点
	runEnd            int // </a:r> 终点
	rPrStart, rPrEnd  int // a:rPr 原始字节区间；无则 -1
	tAttrs            string
}

type blocker struct {
	pos int // 段落拼接文本中的 rune 位置
}

type paragraph struct {
	start, end int
	segs       []textSeg
	blockers   []blocker
	text       string
}

type xmlTokenRange struct {
	start, end int
}

// parseSlideParagraphs 解析 slide XML 的全部 a:p 段落（表格/组合形状内的自然覆盖；
// a:fld 内文本可见但记 blocker 不可编辑；备注页/图表文本是独立 part，天然不在范围）。
func parseSlideParagraphs(data []byte) ([]paragraph, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var paras []paragraph
	var cur *paragraph
	var runStack []xmlTokenRange
	var curRunPr xmlTokenRange
	var curRun xmlTokenRange
	var curT xmlTokenRange
	var curTText string
	var curTLocked bool // a:fld 内的 a:t：文本可见但不可编辑
	fldDepth := 0
	prevEnd := 0

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("解析 slide XML 失败: %w", err)
		}
		curEnd := int(dec.InputOffset())
		raw := xmlTokenRange{start: prevEnd, end: curEnd}
		prevEnd = curEnd

		switch t := tok.(type) {
		case xml.StartElement:
			if isDML(t.Name, "p") {
				paras = append(paras, paragraph{start: raw.start})
				cur = &paras[len(paras)-1]
				continue
			}
			if cur == nil {
				continue
			}
			switch {
			case isDML(t.Name, "r"):
				runStack = append(runStack, raw)
				curRun = raw
				curRunPr = xmlTokenRange{start: -1}
			case isDML(t.Name, "rPr") && len(runStack) > 0:
				curRunPr = raw
			case isDML(t.Name, "t") && (len(runStack) > 0 || fldDepth > 0):
				// a:fld 内的 a:t 没有 a:r 包装（ECMA-376 真实形状），也要采到
				curT = raw
				curTText = ""
				curTLocked = fldDepth > 0
			case isDML(t.Name, "fld"):
				fldDepth++
			case isDML(t.Name, "br"), isDML(t.Name, "tab"):
				cur.blockers = append(cur.blockers, blocker{pos: len([]rune(cur.text))})
			}
		case xml.EndElement:
			if cur == nil {
				continue
			}
			if isDML(t.Name, "t") {
				if curT.start >= 0 && !curTLocked {
					cur.segs = append(cur.segs, textSeg{
						text:        curTText,
						runTagStart: curRun.start, runTagEnd: curRun.end, runEnd: curRun.end,
						rPrStart: curRunPr.start, rPrEnd: curRunPr.end,
						tAttrs: extractAttrs(data[curT.start:curT.end]),
					})
				}
				if curT.start >= 0 {
					// fld 内文本：可见、进拼接文本、但不可编辑（记 blocker 覆盖其区间）
					start := len([]rune(cur.text))
					cur.text += curTText
					if curTLocked {
						cur.blockers = append(cur.blockers,
							blocker{pos: start}, blocker{pos: start + len([]rune(curTText))})
					}
				}
				curT = xmlTokenRange{start: -1}
				continue
			}
			if isDML(t.Name, "rPr") {
				curRunPr.end = curEnd
				continue
			}
			if isDML(t.Name, "r") {
				if len(runStack) > 0 {
					last := runStack[len(runStack)-1]
					for i := range cur.segs {
						seg := &cur.segs[i]
						if seg.runTagStart == last.start && seg.runEnd == last.end {
							seg.runEnd = curEnd
						}
					}
					runStack = runStack[:len(runStack)-1]
				}
				continue
			}
			if isDML(t.Name, "fld") && fldDepth > 0 {
				fldDepth--
				continue
			}
			if isDML(t.Name, "p") {
				cur.end = curEnd
				cur = nil
			}
		case xml.CharData:
			if cur != nil && curT.start >= 0 {
				curTText += string(t)
			}
		}
	}
	return paras, nil
}

// extractAttrs 从起始标签原始字节提取属性子串（不含标签名与 >）。
func extractAttrs(raw []byte) string {
	s := string(raw)
	idx := strings.IndexAny(s, " \t\n>")
	if idx < 0 {
		return ""
	}
	if s[idx] == '>' {
		return ""
	}
	rest := s[idx:]
	end := strings.Index(rest, ">")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(rest[:end])
}

// locateSpan 返回 target 在段落拼接文本中的 rune 区间 [s,e)；精确优先，失败折叠空白兜底。
func locateSpan(p paragraph, target string) (int, int, bool) {
	if idx := strings.Index(p.text, target); idx >= 0 {
		s := len([]rune(p.text[:idx]))
		return s, s + len([]rune(target)), true
	}
	runes := []rune(p.text)
	tRunes := []rune(target)
	norm := make([]int, 0, len(runes))
	for i, r := range runes {
		if unicode.IsSpace(r) {
			if i > 0 && unicode.IsSpace(runes[i-1]) {
				continue
			}
		}
		norm = append(norm, i)
	}
	normText := make([]rune, 0, len(norm))
	for _, i := range norm {
		normText = append(normText, runes[i])
	}
	var tNorm []rune
	prevSpace := false
	for _, r := range tRunes {
		if unicode.IsSpace(r) {
			if prevSpace {
				continue
			}
			prevSpace = true
			tNorm = append(tNorm, ' ')
		} else {
			prevSpace = false
			tNorm = append(tNorm, r)
		}
	}
	if len(tNorm) == 0 {
		return 0, 0, false
	}
	ni := indexRunes(normText, tNorm)
	if ni < 0 {
		return 0, 0, false
	}
	s := norm[ni]
	e := norm[ni+len(tNorm)-1] + 1
	for s < e && unicode.IsSpace(runes[s]) {
		s++
	}
	for e > s && unicode.IsSpace(runes[e-1]) {
		e--
	}
	if s >= e {
		return 0, 0, false
	}
	return s, e, true
}

func indexRunes(hay, needle []rune) int {
	if len(needle) == 0 {
		return 0
	}
	for i := 0; i+len(needle) <= len(hay); i++ {
		ok := true
		for j := range needle {
			if hay[i+j] != needle[j] {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}

// rebuildParagraph 直接替换（无修订制）：被选区覆盖的 run 重建为
// 「前缀 +（最后受影响 run 处）replacement」，新 a:t 继承最后受影响 run 的
// a:rPr 原字节；完全覆盖且非最后的 run 整体省略；其余字节原样。
func rebuildParagraph(data []byte, p paragraph, s, e int, replacement string) ([]byte, error) {
	type runGroup struct {
		tagStart, tagEnd, end int
		rPrStart, rPrEnd      int
		tAttrs                string
		segs                  []textSeg
	}
	var groups []*runGroup
	groupByRun := map[[2]int]*runGroup{}
	for _, seg := range p.segs {
		key := [2]int{seg.runTagStart, seg.runTagEnd}
		g := groupByRun[key]
		if g == nil {
			g = &runGroup{tagStart: seg.runTagStart, tagEnd: seg.runTagEnd, end: seg.runEnd,
				rPrStart: -1, rPrEnd: seg.rPrEnd, tAttrs: seg.tAttrs}
			groupByRun[key] = g
			groups = append(groups, g)
		}
		g.segs = append(g.segs, seg)
		if seg.runEnd > g.end {
			g.end = seg.runEnd
		}
		if seg.rPrStart >= 0 && (g.rPrStart < 0 || seg.rPrStart < g.rPrStart) {
			g.rPrStart = seg.rPrStart
		}
		if seg.rPrEnd > g.rPrEnd {
			g.rPrEnd = seg.rPrEnd
		}
	}

	type runSpan struct {
		g                    *runGroup
		start, end           int
		delFrom, delTo       int
	}
	var affected []*runSpan
	cursor := 0
	for _, g := range groups {
		runText := ""
		for _, seg := range g.segs {
			runText += seg.text
		}
		runLen := len([]rune(runText))
		rs := runSpan{g: g, start: cursor, end: cursor + runLen}
		df := maxInt(rs.start, s) - rs.start
		dt := minInt(rs.end, e) - rs.start
		if dt > df {
			rs.delFrom, rs.delTo = df, dt
			affected = append(affected, &rs)
		}
		cursor += runLen
	}
	if len(affected) == 0 {
		return nil, fmt.Errorf("选区未命中任何文本")
	}
	lastAffected := affected[len(affected)-1].g

	var out bytes.Buffer
	pos := p.start
	insInserted := false
	for _, g := range groups {
		runText := ""
		for _, seg := range g.segs {
			runText += seg.text
		}
		runRunes := []rune(runText)
		var delFrom, delTo int
		isAffected := false
		for _, rs := range affected {
			if rs.g == g {
				delFrom, delTo, isAffected = rs.delFrom, rs.delTo, true
				break
			}
		}
		out.Write(data[pos:g.tagStart])
		if !isAffected {
			out.Write(data[g.tagStart:g.end])
			pos = g.end
			continue
		}
		before := string(runRunes[:delFrom])
		after := string(runRunes[delTo:])
		rPrRaw := ""
		if g.rPrStart >= 0 && g.rPrEnd > g.rPrStart {
			rPrRaw = string(data[g.rPrStart:g.rPrEnd])
		}
		attrs := g.tAttrs
		runTag := string(data[g.tagStart:g.tagEnd])
		isLast := g == lastAffected

		if before != "" {
			out.WriteString(runTag)
			out.WriteString(rPrRaw)
			out.WriteString(textElement("a:t", before, attrs))
			out.WriteString("</a:r>")
		}
		// 替换文本挂在最后受影响 run（继承其 rPr）
		if isLast && !insInserted {
			if replacement != "" {
				out.WriteString(runTag)
				out.WriteString(rPrRaw)
				out.WriteString(textElement("a:t", replacement, attrs))
				out.WriteString("</a:r>")
			}
			if after != "" {
				out.WriteString(runTag)
				out.WriteString(rPrRaw)
				out.WriteString(textElement("a:t", after, attrs))
				out.WriteString("</a:r>")
			}
			insInserted = true
		} else if after != "" {
			// 中间受影响 run 的尾部残留（非 last 却有 after：不存在——last 之后
			// 的 run 不受影响；这里守卫性保留）
			out.WriteString(runTag)
			out.WriteString(rPrRaw)
			out.WriteString(textElement("a:t", after, attrs))
			out.WriteString("</a:r>")
		}
		pos = g.end
	}
	out.Write(data[pos:p.end])
	return out.Bytes(), nil
}

// textElement 生成 <a:t 属性>文本</a:t>；文本 XML 转义，首尾空白补 xml:space="preserve"。
func textElement(elem, text, attrs string) string {
	esc := xmlEscape(text)
	if (strings.HasPrefix(text, " ") || strings.HasSuffix(text, " ")) && !strings.Contains(attrs, "xml:space") {
		if attrs != "" {
			attrs += ` xml:space="preserve"`
		} else {
			attrs = `xml:space="preserve"`
		}
	}
	if attrs != "" {
		return "<" + elem + " " + attrs + ">" + esc + "</" + elem + ">"
	}
	return "<" + elem + ">" + esc + "</" + elem + ">"
}

func xmlEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&#34;")
		case '\'':
			b.WriteString("&#39;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
