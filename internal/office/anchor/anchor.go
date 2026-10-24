// Package anchor — 跨格式统一文档锚点语言（A1 蒸馏刀，语法层 v1）。
//
// 移植自 zai-org/ZCode office-runtime/src/domain/anchor.ts（Apache-2.0）：
// 锚点=面板、服务端与模型三方共享的地址 `<workspace-relative path>#<locator>`。
// 对源实证结论（蒸馏对话档第 5/6 轮）：docx 无持久段落 ID——p:N 是读时位序，
// 锚点只在同一 revision 内可靠；跨编辑稳定性由 revision 门（内容哈希）承担，
// 不做内容指纹持久锚（量级差一个数量级，v2 再议）。
//
// 本包是纯语法层：解析/格式化/校验/往返，不读文档（check() 归一化回真名
// 需要文档模型，属预览接线相位）。1-based。
package anchor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// DocKind 文档格式。
type DocKind string

const (
	KindDocx DocKind = "docx"
	KindPptx DocKind = "pptx"
	KindPdf  DocKind = "pdf"
	KindXlsx DocKind = "xlsx"
)

// CellRef 单元格（1-based）。
type CellRef struct {
	Row int
	Col int
}

// Locator 类型化定位：各格式一种形态。
type Locator struct {
	Kind DocKind
	// docx: 段落区间 [From,To]（p:N / p:N-M）
	From, To int
	// docx 表格: Table 序号 + 可选单元格（t:N / t:N/rRcC）
	Table int
	Cell  *CellRef
	// pptx: 幻灯片区间 + 可选 shape 对象 id（slide:N / slide:N-M / slide:N/shape:ID）
	Shape string
	// pdf: 页码区间（page:N / page:N-M）
	// xlsx: 工作表 + 可选区域（Sheet / Sheet!A1 / Sheet!A1:C9）
	Sheet     string
	RangeFrom *CellRef
	RangeTo   *CellRef
}

// Span 返回区间语义的主档（docx 段落/pptx 幻灯片/pdf 页 共用 From/To）。
func (l Locator) Span() (from, to int) { return l.From, l.To }

// Rev1 计算文件 revision（sha256 前 16 hex，ZCode 同口径）：编辑落盘后
// revision 变化，旧锚点失效须重定位——revision 门的原语。
func Rev1(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:16]
}

// ── 列名换算（xlsx）──

// ColumnName 1 → "A", 27 → "AA"。
func ColumnName(col int) string {
	name := ""
	for n := col; n > 0; n = (n - 1) / 26 {
		name = string(rune('A'+(n-1)%26)) + name
	}
	return name
}

var reCell = regexp.MustCompile(`^\$?([A-Za-z]{1,3})\$?(\d{1,7})$`)

// ParseCell "B2" → {2,2}；不合法返回 nil。
func ParseCell(text string) *CellRef {
	m := reCell.FindStringSubmatch(strings.TrimSpace(text))
	if m == nil {
		return nil
	}
	col := 0
	for _, ch := range strings.ToUpper(m[1]) {
		if ch < 'A' || ch > 'Z' {
			return nil
		}
		col = col*26 + int(ch-'A'+1)
	}
	row, err := strconv.Atoi(m[2])
	if err != nil || row < 1 || row > 1_048_576 || col > 16_384 {
		return nil
	}
	return &CellRef{Row: row, Col: col}
}

func (c CellRef) String() string { return ColumnName(c.Col) + strconv.Itoa(c.Row) }

// ── 解析 ──

const maxIndex = 1_000_000

func spanFrom(text, fromS, toS string) (int, int, error) {
	from, err1 := strconv.Atoi(fromS)
	to := from
	if err1 != nil {
		return 0, 0, fmt.Errorf("cannot read location %q: indices must be 1-based integers", text)
	}
	if toS != "" {
		var err2 error
		to, err2 = strconv.Atoi(toS)
		if err2 != nil {
			return 0, 0, fmt.Errorf("cannot read location %q: range end must be an integer", text)
		}
	}
	if from < 1 || to < from || to > maxIndex {
		return 0, 0, fmt.Errorf("cannot read location %q: need 1 ≤ from ≤ to ≤ %d", text, maxIndex)
	}
	return from, to, nil
}

var (
	reDocxPara  = regexp.MustCompile(`^p:(\d+)(?:-(\d+))?$`)
	reDocxTable = regexp.MustCompile(`^t:(\d+)(?:/r(\d+)c(\d+))?$`)
	rePptxSlide = regexp.MustCompile(`^slide:(\d+)(?:-(\d+))?(?:/shape:(.+))?$`)
	rePdfPage   = regexp.MustCompile(`^page:(\d+)(?:-(\d+))?$`)
)

// Parse 按格式解析 locator 文本。hint 是失败信息里的语法提示。
func Parse(kind DocKind, locator string) (Locator, error) {
	loc := Locator{Kind: kind}
	text := strings.TrimSpace(locator)
	switch kind {
	case KindDocx:
		if m := reDocxPara.FindStringSubmatch(text); m != nil {
			from, to, err := spanFrom(text, m[1], m[2])
			if err != nil {
				return loc, err
			}
			loc.From, loc.To = from, to
			return loc, nil
		}
		if m := reDocxTable.FindStringSubmatch(text); m != nil {
			from, to, err := spanFrom(text, m[1], "")
			if err != nil {
				return loc, err
			}
			loc.Table = from
			loc.From, loc.To = from, to
			if m[2] != "" {
				r, err1 := strconv.Atoi(m[2])
				c, err2 := strconv.Atoi(m[3])
				if err1 != nil || err2 != nil || r < 1 || c < 1 {
					return loc, fmt.Errorf("cannot read location %q: table cell must be r≥1c≥1", text)
				}
				loc.Cell = &CellRef{Row: r, Col: c}
			}
			return loc, nil
		}
		return loc, fmt.Errorf("cannot read location %q: use p:N, p:N-M, t:N or t:N/rRcC", text)
	case KindPptx:
		if m := rePptxSlide.FindStringSubmatch(text); m != nil {
			from, to, err := spanFrom(text, m[1], m[2])
			if err != nil {
				return loc, err
			}
			loc.From, loc.To = from, to
			loc.Shape = strings.TrimSpace(m[3])
			return loc, nil
		}
		return loc, fmt.Errorf("cannot read location %q: use slide:N, slide:N-M or slide:N/shape:ID", text)
	case KindPdf:
		if m := rePdfPage.FindStringSubmatch(text); m != nil {
			from, to, err := spanFrom(text, m[1], m[2])
			if err != nil {
				return loc, err
			}
			loc.From, loc.To = from, to
			return loc, nil
		}
		return loc, fmt.Errorf("cannot read location %q: use page:N or page:N-M", text)
	case KindXlsx:
		return parseXlsx(text)
	}
	return loc, fmt.Errorf("unknown document kind %q", kind)
}

// parseXlsx Sheet / Sheet!A1 / Sheet!A1:C9 / 'My sheet'!A1（带空格表名加引号）。
func parseXlsx(text string) (Locator, error) {
	loc := Locator{Kind: KindXlsx}
	bang := strings.LastIndexByte(text, '!')
	var sheet string
	if bang < 0 {
		sheet = text
		if sheet == "" {
			return loc, fmt.Errorf("cannot read location %q: sheet name is empty", text)
		}
		loc.Sheet = unquoteSheet(sheet)
		return loc, nil
	}
	sheet = strings.TrimSpace(text[:bang])
	if sheet == "" {
		return loc, fmt.Errorf("cannot read location %q: sheet name is empty", text)
	}
	loc.Sheet = unquoteSheet(sheet)
	rangePart := strings.TrimSpace(text[bang+1:])
	if rangePart == "" {
		return loc, fmt.Errorf("cannot read location %q: cell reference missing after '!'", text)
	}
	parts := strings.SplitN(rangePart, ":", 2)
	from := ParseCell(parts[0])
	if from == nil {
		return loc, fmt.Errorf("cannot read location %q: %q is not a cell like A1", text, parts[0])
	}
	loc.RangeFrom = from
	if len(parts) == 2 {
		to := ParseCell(parts[1])
		if to == nil {
			return loc, fmt.Errorf("cannot read location %q: %q is not a cell like A1", text, parts[1])
		}
		loc.RangeTo = to
	}
	return loc, nil
}

func unquoteSheet(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
		return s[1 : len(s)-1]
	}
	return s
}

// ── 格式化（往返规范形）──

// Format 输出规范形（roundtrip：Parse(Format(x)) 恒等）。
func (l Locator) Format() string {
	switch l.Kind {
	case KindDocx:
		if l.Table > 0 {
			s := "t:" + strconv.Itoa(l.Table)
			if l.Cell != nil {
				s += fmt.Sprintf("/r%dc%d", l.Cell.Row, l.Cell.Col)
			}
			return s
		}
		if l.To != l.From {
			return fmt.Sprintf("p:%d-%d", l.From, l.To)
		}
		return "p:" + strconv.Itoa(l.From)
	case KindPptx:
		s := "slide:" + strconv.Itoa(l.From)
		if l.To != l.From {
			s += "-" + strconv.Itoa(l.To)
		}
		if l.Shape != "" {
			s += "/shape:" + l.Shape
		}
		return s
	case KindPdf:
		if l.To != l.From {
			return fmt.Sprintf("page:%d-%d", l.From, l.To)
		}
		return "page:" + strconv.Itoa(l.From)
	case KindXlsx:
		needQuote := strings.ContainsAny(l.Sheet, " !'[]\\")
		sheet := l.Sheet
		if needQuote {
			sheet = "'" + strings.ReplaceAll(sheet, "'", "''") + "'"
		}
		if l.RangeFrom == nil {
			return sheet
		}
		if l.RangeTo == nil {
			return sheet + "!" + l.RangeFrom.String()
		}
		return sheet + "!" + l.RangeFrom.String() + ":" + l.RangeTo.String()
	}
	return ""
}
