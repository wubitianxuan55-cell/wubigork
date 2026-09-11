// Package docxedit 提供 docx 原地编辑能力：把用户选中的文本替换为 AI 生成的新文本，
// 以 Word 修订模式（w:del + w:ins）写入，文档其余字节保持不变。
//
// 设计取舍：不解析/重建整份 OOXML，只对 document.xml 中命中的段落做字节级手术，
// 保证「框选即改、其余内容与版式零扰动」，与 gaea「AI 不啃完整 OOXML」的原则一致。
package docxedit

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const wmlNS = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"

// ApplyTrackedReplace 在 docx 正文中定位 targetText，将其替换为 replacement，
// 以修订模式写入（删除原文 + 插入新文），其余内容原样保留。
// author 为修订作者名（默认 "gaea AI"）。targetText 为空或未命中时返回错误。
func ApplyTrackedReplace(path, targetText, replacement, author string) error {
	if targetText == "" {
		return fmt.Errorf("选中文本为空")
	}
	if author == "" {
		author = "gaea AI"
	}

	doc, err := readDocx(path)
	if err != nil {
		return err
	}
	patched, ok, err := patchDocumentXML(doc.documentXML, targetText, replacement, author)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("未能在文档正文中定位到所选文本（可能跨表格/页眉页脚或文本被格式拆分），请整段重选后重试")
	}
	doc.documentXML = patched
	doc.files["word/document.xml"] = patched
	return writeDocx(path, doc)
}

// AcceptChanges 接受指定作者的全部修订：删除 w:del（原文移除），保留 w:ins（新文生效）。
// 没有该作者的待处理修订时返回错误。
func AcceptChanges(path, author string) error {
	return flattenChanges(path, author, true)
}

// RejectChanges 拒绝指定作者的全部修订：删除 w:ins（新文移除），还原 w:del（原文恢复）。
func RejectChanges(path, author string) error {
	return flattenChanges(path, author, false)
}

// flattenChanges 按作者扁平化 document.xml 中的修订（w:del / w:ins）。
func flattenChanges(path, author string, accept bool) error {
	if author == "" {
		author = "gaea AI"
	}
	doc, err := readDocx(path)
	if err != nil {
		return err
	}
	patched, changed, err := flattenDocumentXML(doc.documentXML, author, accept)
	if err != nil {
		return err
	}
	if !changed {
		return fmt.Errorf("没有 %s 的待处理修订", author)
	}
	doc.documentXML = patched
	doc.files["word/document.xml"] = patched
	return writeDocx(path, doc)
}

// ── zip 读写 ────────────────────────────────────────────────

type docxFile struct {
	files       map[string][]byte
	order       []string
	documentXML []byte
}

func readDocx(path string) (*docxFile, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("打开 docx 失败: %w", err)
	}
	defer r.Close()

	doc := &docxFile{files: map[string][]byte{}}
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
		if f.Name == "word/document.xml" {
			doc.documentXML = b
		}
	}
	if doc.documentXML == nil {
		return nil, fmt.Errorf("docx 缺少 word/document.xml")
	}
	return doc, nil
}

func writeDocx(path string, doc *docxFile) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".gaea-docxedit-*.docx")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	zw := zip.NewWriter(tmp)
	for _, name := range doc.order {
		b := doc.files[name]
		w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate})
		if err != nil {
			zw.Close()
			tmp.Close()
			return fmt.Errorf("写回 %s 失败: %w", name, err)
		}
		if _, err := w.Write(b); err != nil {
			zw.Close()
			tmp.Close()
			return fmt.Errorf("写回 %s 失败: %w", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		tmp.Close()
		return fmt.Errorf("打包 docx 失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("关闭临时文件失败: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("替换原文件失败: %w", err)
	}
	return nil
}

// ── document.xml 字节级手术 ─────────────────────────────────

// textSeg 是段落中一个可编辑文本单元（w:t 的文本内容）。
type textSeg struct {
	text string
	// runTagStart/runTagEnd：w:r 起始标签的原始字节区间（如 <w:r w:rsidRPr="..">）
	runTagStart, runTagEnd int
	// runEnd：w:r 结束标签结束位置（含 </w:r>）
	runEnd int
	// rPrStart/rPrEnd：该 run 内 <w:rPr>...</w:rPr> 原始字节区间；无 rPr 时为 -1
	rPrStart, rPrEnd int
	// tAttrs：w:t 起始标签中的属性子串（不含 <w:t 与 >）
	tAttrs string
	// link：该文本是否位于 w:hyperlink 内（TOC 目录项等）
	link bool
}

type blocker struct {
	pos int // 在段落拼接文本中的字符位置（用于检测选区是否覆盖特殊元素）
}

type paragraph struct {
	start, end int // <w:p ...> ... </w:p> 原始字节区间
	segs       []textSeg
	blockers   []blocker
	text       string
}

// xmlTokenRange 记录一次 Token 的原始字节区间。
type xmlTokenRange struct {
	start, end int
}

// patchDocumentXML 在 document.xml 字节流上执行修订式替换。
func patchDocumentXML(data []byte, target, replacement, author string) ([]byte, bool, error) {
	paras, err := parseParagraphs(data)
	if err != nil {
		return nil, false, err
	}

	type candidate struct {
		p    paragraph
		s, e int
		link bool
	}
	var first *candidate
	var nonLink *candidate
	for _, p := range paras {
		s, e, ok := locateSpan(p, target)
		if !ok {
			continue
		}
		c := &candidate{p: p, s: s, e: e, link: spanCoversLink(p, s, e)}
		if first == nil {
			first = c
		}
		if !c.link && nonLink == nil {
			nonLink = c
		}
	}
	chosen := nonLink
	if chosen == nil {
		chosen = first
	}
	if chosen != nil {
		p, s, e := chosen.p, chosen.s, chosen.e
		// 选区覆盖特殊元素（图片/制表符/换行等）时拒绝，避免破坏版式。
		for _, b := range p.blockers {
			if b.pos >= s && b.pos < e {
				return nil, false, fmt.Errorf("选区包含特殊格式（图片/制表符/换行等），暂不支持，请缩小选区后重试")
			}
		}
		patched, err := rebuildParagraph(data, p, s, e, replacement, author)
		if err != nil {
			return nil, false, err
		}
		out := make([]byte, 0, len(data)+len(patched))
		out = append(out, data[:p.start]...)
		out = append(out, patched...)
		out = append(out, data[p.end:]...)
		return out, true, nil
	}
	return nil, false, nil
}

// changeSpan 是待扁平化修订元素的原始区间。
type changeSpan struct {
	kind        string // "del" | "ins"
	start, end  int    // 元素原始字节区间
	startTagEnd int    // 起始标签结束位置（含 >）
	author      string
	selfClose   bool
}

// flattenDocumentXML 把指定作者的 w:del/w:ins 扁平化：
//   accept=true  → 接受：删 w:del、保留 w:ins 内容；
//   accept=false → 拒绝：删 w:ins、还原 w:del（delText → t）。
func flattenDocumentXML(data []byte, author string, accept bool) ([]byte, bool, error) {
	spans, err := parseChangeSpans(data)
	if err != nil {
		return nil, false, err
	}
	type edit struct {
		start, end int
		raw        []byte // nil = 删除该区间；非 nil = 替换为该内容
	}
	var edits []edit
	for _, s := range spans {
		if s.author != author {
			continue
		}
		if s.selfClose {
			// 自闭合（如段落标记删除标记）：移除元素即恢复原状
			edits = append(edits, edit{start: s.start, end: s.end})
			continue
		}
		inner := unwrapInner(data, s)
		if inner == nil {
			continue // 结构异常，跳过
		}
		if accept && s.kind == "del" {
			edits = append(edits, edit{start: s.start, end: s.end})
		} else if accept && s.kind == "ins" {
			edits = append(edits, edit{start: s.start, end: s.end, raw: inner})
		} else if !accept && s.kind == "ins" {
			edits = append(edits, edit{start: s.start, end: s.end})
		} else if !accept && s.kind == "del" {
			edits = append(edits, edit{start: s.start, end: s.end, raw: renameDelText(inner)})
		}
	}
	if len(edits) == 0 {
		return nil, false, nil
	}
	sort.Slice(edits, func(i, j int) bool { return edits[i].start < edits[j].start })
	var out bytes.Buffer
	pos := 0
	for _, e := range edits {
		if e.start < pos {
			continue // 与已处理区间重叠（嵌套等），跳过
		}
		out.Write(data[pos:e.start])
		if e.raw != nil {
			out.Write(e.raw)
		}
		pos = e.end
	}
	out.Write(data[pos:])
	return out.Bytes(), true, nil
}

// parseChangeSpans 收集 document.xml 中 w:del / w:ins 元素的原始区间与作者。
func parseChangeSpans(data []byte) ([]changeSpan, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	type stackItem struct {
		kind        string
		start       int
		startTagEnd int
		author      string
	}
	var stack []stackItem
	var spans []changeSpan
	prevEnd := 0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("解析 document.xml 失败: %w", err)
		}
		curEnd := int(dec.InputOffset())
		switch t := tok.(type) {
		case xml.StartElement:
			if isWML(t.Name, "del") || isWML(t.Name, "ins") {
				stack = append(stack, stackItem{
					kind: t.Name.Local, start: prevEnd, startTagEnd: curEnd,
					author: changeAuthor(data[prevEnd:curEnd]),
				})
			}
		case xml.EndElement:
			if isWML(t.Name, "del") || isWML(t.Name, "ins") {
				if len(stack) > 0 {
					item := stack[len(stack)-1]
					stack = stack[:len(stack)-1]
					if item.kind == t.Name.Local {
						spans = append(spans, changeSpan{
							kind: item.kind, start: item.start, end: curEnd,
							startTagEnd: item.startTagEnd, author: item.author,
							selfClose: prevEnd == item.startTagEnd,
						})
					}
				}
			}
		}
		prevEnd = curEnd
	}
	return spans, nil
}

var changeAuthorRe = regexp.MustCompile(`w:author="([^"]*)"`)

func changeAuthor(raw []byte) string {
	if m := changeAuthorRe.FindSubmatch(raw); m != nil {
		return string(m[1])
	}
	return ""
}

// unwrapInner 返回元素内层原始字节（去除起始标签与结束标签）。
func unwrapInner(data []byte, s changeSpan) []byte {
	endTag := []byte("</w:" + s.kind + ">")
	idx := bytes.LastIndex(data[s.startTagEnd:s.end], endTag)
	if idx < 0 {
		return nil
	}
	return data[s.startTagEnd : s.startTagEnd+idx]
}

// renameDelText 拒绝删除时把 delText 还原为 t（delInstrText → instrText）。
func renameDelText(inner []byte) []byte {
	s := string(inner)
	s = strings.ReplaceAll(s, "<w:delInstrText", "<w:instrText")
	s = strings.ReplaceAll(s, "</w:delInstrText>", "</w:instrText>")
	s = strings.ReplaceAll(s, "<w:delText", "<w:t")
	s = strings.ReplaceAll(s, "</w:delText>", "</w:t>")
	return []byte(s)
}

// spanCoversLink 判断选区 [s,e) 是否覆盖超链接（TOC 目录项）内的文本。
func spanCoversLink(p paragraph, s, e int) bool {
	cursor := 0
	for _, seg := range p.segs {
		runes := []rune(seg.text)
		n := len(runes)
		if seg.link && cursor < e && cursor+n > s {
			return true
		}
		cursor += n
	}
	return false
}

// parseParagraphs 用 xml.Decoder 的 InputOffset 精确记录各 token 的原始字节区间，
// 收集正文段落、run、w:t 文本与阻断元素。
func parseParagraphs(data []byte) ([]paragraph, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var paras []paragraph
	var cur *paragraph
	var runStack []xmlTokenRange // w:r 起始标签区间栈
	var curRunPr xmlTokenRange
	var curRun xmlTokenRange
	var curT xmlTokenRange
	var curTText string
	linkDepth := 0
	prevEnd := 0

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("解析 document.xml 失败: %w", err)
		}
		curEnd := int(dec.InputOffset())
		raw := xmlTokenRange{start: prevEnd, end: curEnd}
		prevEnd = curEnd

		switch t := tok.(type) {
		case xml.StartElement:
			if isWML(t.Name, "p") {
				paras = append(paras, paragraph{start: raw.start})
				cur = &paras[len(paras)-1]
				continue
			}
			if cur == nil {
				continue
			}
			switch {
			case isWML(t.Name, "hyperlink"):
				linkDepth++
			case isWML(t.Name, "r"):
				runStack = append(runStack, raw)
				curRun = raw
				curRunPr = xmlTokenRange{start: -1}
			case isWML(t.Name, "rPr") && len(runStack) > 0:
				curRunPr = raw
			case isWML(t.Name, "t") && len(runStack) > 0:
				curT = raw
				curTText = ""
			case isWML(t.Name, "tab"), isWML(t.Name, "br"), isWML(t.Name, "cr"),
				isWML(t.Name, "noBreakHyphen"), isWML(t.Name, "softHyphen"),
				isWML(t.Name, "drawing"), isWML(t.Name, "pict"), isWML(t.Name, "object"),
				isWML(t.Name, "fldChar"), isWML(t.Name, "instrText"), isWML(t.Name, "delText"),
				isWML(t.Name, "delInstrText"):
				if len(runStack) > 0 {
					cur.blockers = append(cur.blockers, blocker{pos: len([]rune(cur.text))})
				}
			}
		case xml.EndElement:
			if cur == nil {
				continue
			}
			if isWML(t.Name, "t") {
				if curT.start >= 0 {
					cur.segs = append(cur.segs, textSeg{
						text:        curTText,
						runTagStart: curRun.start, runTagEnd: curRun.end, runEnd: curRun.end,
						rPrStart: curRunPr.start, rPrEnd: curRunPr.end,
						tAttrs: extractAttrs(data[curT.start:curT.end]),
						link:   linkDepth > 0,
					})
					cur.text += curTText
				}
				curT = xmlTokenRange{start: -1}
				continue
			}
			if isWML(t.Name, "rPr") {
				curRunPr.end = curEnd
				continue
			}
			if isWML(t.Name, "r") {
				if len(runStack) > 0 {
					last := runStack[len(runStack)-1]
					// 回填刚结束的 run 的结束位置
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
			if isWML(t.Name, "p") {
				cur.end = curEnd
				// 空 run 标签填充（只有 start tag，无结束标签的情况不存在）
				cur = nil
			}
			if isWML(t.Name, "hyperlink") && linkDepth > 0 {
				linkDepth--
			}
		case xml.CharData:
			if cur != nil && curT.start >= 0 {
				curTText += string(t)
			}
		}
	}
	return paras, nil
}

func isWML(name xml.Name, local string) bool {
	return name.Local == local && (name.Space == wmlNS || strings.Contains(name.Space, "wordprocessingml"))
}

// extractAttrs 从 w:t 起始标签原始字节提取属性子串（不含标签名与 >）。
func extractAttrs(raw []byte) string {
	s := string(raw)
	// 找到元素名结束位置（空格或 >）
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

// locateSpan 返回 target 在段落拼接文本中的字符区间 [s,e)；未命中返回 false。
// 优先精确匹配；失败时折叠空白后模糊匹配（还原到原始区间）。
func locateSpan(p paragraph, target string) (int, int, bool) {
	if idx := strings.Index(p.text, target); idx >= 0 {
		// 统一使用 rune 偏移（重建阶段按 rune 切分）
		s := len([]rune(p.text[:idx]))
		return s, s + len([]rune(target)), true
	}
	// 折叠空白兜底：需要 rune 级别映射
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
	// 同样折叠 target
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
	// normText 中的空白已被折叠为单个空格
	ni := indexRunes(normText, tNorm)
	if ni < 0 {
		return 0, 0, false
	}
	s := norm[ni]
	e := norm[ni+len(tNorm)-1] + 1
	// 去掉首尾纯空白，避免吃掉相邻空白
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

// rebuildParagraph 重建命中段落的 run 序列：被选区覆盖的文本包进 w:del，
// 之后插入 w:ins（新文本），其余 run 原样保留。
func rebuildParagraph(data []byte, p paragraph, s, e int, replacement, author string) ([]byte, error) {
	// 按 run 分组：runTagStart → segs
	type runGroup struct {
		tagStart, tagEnd, end int
		rPrStart, rPrEnd     int
		tAttrs               string
		segs                 []textSeg
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

	// 计算每个 run 在段落文本中的字符区间，找出被选区覆盖的 run。
	type runSpan struct {
		g       *runGroup
		start   int // run 文本起点（rune 偏移，基于 p.text）
		end     int // run 文本终点
		delFrom int // run 内删除起点（run 内 rune 偏移）
		delTo   int // run 内删除终点
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
		// 与 [s,e) 求交
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
	lastAffectedGroup := affected[len(affected)-1].g

	maxID := maxWID(data)
	delID := maxID + 1
	insID := maxID + 2
	date := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	// 生成新的段落字节：逐 run 重建（w:del/w:ins 与 w:r 平级，不嵌套在 run 内）。
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
		// 拷贝 run 之前的内容（含未分组的 run：图片等原样保留）
		out.Write(data[pos:g.tagStart])
		if !isAffected {
			// 原样拷贝
			out.Write(data[g.tagStart:g.end])
			pos = g.end
			continue
		}

		before := string(runRunes[:delFrom])
		deleted := string(runRunes[delFrom:delTo])
		after := string(runRunes[delTo:])
		rPrRaw := ""
		if g.rPrStart >= 0 && g.rPrEnd > g.rPrStart {
			rPrRaw = string(data[g.rPrStart:g.rPrEnd])
		}
		attrs := g.tAttrs
		runTag := string(data[g.tagStart:g.tagEnd])

		// 删除前的剩余文本：独立 run
		if before != "" {
			out.WriteString(runTag)
			out.WriteString(rPrRaw)
			out.WriteString(textElement("w:t", before, attrs))
			out.WriteString("</w:r>")
		}
		// 删除：w:del 包一个 run
		if deleted != "" {
			out.WriteString("<w:del w:id=\"" + strconv.Itoa(delID) + "\" w:author=\"" + xmlAttrEscape(author) +
				"\" w:date=\"" + date + "\"><w:r>")
			out.WriteString(rPrRaw)
			out.WriteString(textElement("w:delText", deleted, attrs))
			out.WriteString("</w:r></w:del>")
		}
		// 新文本插入在删除之后、后续内容之前；只插一次（最后一个受影响 run 处）。
		if g == lastAffectedGroup && !insInserted && replacement != "" {
			out.WriteString("<w:ins w:id=\"" + strconv.Itoa(insID) + "\" w:author=\"" + xmlAttrEscape(author) +
				"\" w:date=\"" + date + "\"><w:r>")
			out.WriteString(rPrRaw)
			out.WriteString(textElement("w:t", replacement, attrs))
			out.WriteString("</w:r></w:ins>")
			insInserted = true
		}
		// 删除后的剩余文本：独立 run
		if after != "" {
			out.WriteString(runTag)
			out.WriteString(rPrRaw)
			out.WriteString(textElement("w:t", after, attrs))
			out.WriteString("</w:r>")
		}
		pos = g.end
	}
	out.Write(data[pos:p.end])
	return out.Bytes(), nil
}

// textElement 生成 <w:t 属性>文本</w:t>；文本自动 XML 转义，
// 首尾含空白时补 xml:space="preserve"。
func textElement(elem, text, attrs string) string {
	esc := xmlEscape(text)
	if (strings.HasPrefix(text, " ") || strings.HasSuffix(text, " ") ||
		strings.HasPrefix(text, "\t") || strings.HasSuffix(text, "\t")) &&
		!strings.Contains(attrs, "xml:space") {
		attrs = strings.TrimSpace(attrs)
		if attrs != "" {
			attrs += " "
		}
		attrs += `xml:space="preserve"`
	}
	if attrs != "" {
		attrs = " " + attrs
	}
	return "<" + elem + attrs + ">" + esc + "</" + elem + ">"
}

func xmlEscape(s string) string {
	var b bytes.Buffer
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func xmlAttrEscape(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "&", "&amp;"), "\"", "&quot;")
}

var widRe = regexp.MustCompile(`(?i)\bw:id="(\d+)"`)

// maxWID 返回 document.xml 中现存最大 w:id（用于保证新修订 id 唯一）。
func maxWID(data []byte) int {
	max := 0
	for _, m := range widRe.FindAllSubmatch(data, -1) {
		if n, err := strconv.Atoi(string(m[1])); err == nil && n > max {
			max = n
		}
	}
	return max
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
