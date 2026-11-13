package comhelper

// format.go — helper 回执结构体与格式化（上游 src/format.ts 的 Go 直译，
// 文本逐句对齐：这些行是模型逐 token 付费的上下文，保持 terse）。

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// AppKind 三应用标识（helper 协议原词）。
type AppKind = string

const (
	KindWord  AppKind = "word"
	KindExcel AppKind = "excel"
	KindPPT   AppKind = "ppt"
)

var appNames = map[string]string{KindWord: "Word", KindExcel: "Excel", KindPPT: "PowerPoint"}

// appName 应用在回执里的称呼（WPS 与微软 Office 工具操作一致，名称跟随
// helper 回执里的 suite 字段）。
func AppDisplayName(app, suite string) string {
	if suite == "wps" {
		switch app {
		case KindWord:
			return "WPS Writer"
		case KindExcel:
			return "WPS Spreadsheets"
		case KindPPT:
			return "WPS Presentation"
		}
	}
	return appNames[app]
}

// DocInfo status/open 回执里的单文档信息（path 可为 null=从未保存过）。
type DocInfo struct {
	App        string  `json:"app"`
	Suite      string  `json:"suite,omitempty"`
	Name       string  `json:"name"`
	Path       *string `json:"path"`
	Saved      bool    `json:"saved"`
	Active     bool    `json:"active"`
	ReadOnly   bool    `json:"readOnly,omitempty"`
	How        string  `json:"how,omitempty"`        // attached | opened | created
	Background bool    `json:"background,omitempty"` // 静默模式的无窗口文档
}

// AppStatus status 回执的单应用条目。
type AppStatus struct {
	App       string    `json:"app"`
	Suite     string    `json:"suite,omitempty"`
	Installed bool      `json:"installed"`
	Running   bool      `json:"running"`
	Version   string    `json:"version,omitempty"`
	Error     string    `json:"error,omitempty"`
	Documents []DocInfo `json:"documents"`
}

// WordItem Word 读取的段/表条目（缺省字段用指针区分 null 与零值）。
type WordItem struct {
	I       int     `json:"i"`
	Text    *string `json:"text,omitempty"`
	Style   *string `json:"style,omitempty"`
	Level   *int    `json:"level,omitempty"`
	To      *int    `json:"to,omitempty"`
	Table   *int    `json:"table,omitempty"`
	Size    *string `json:"size,omitempty"`
	Open    bool    `json:"open,omitempty"`
	InTable bool    `json:"inTable,omitempty"`
}

// WordFootnote 原生脚注条目（i=集合序号，非显示编号）。
type WordFootnote struct {
	I           int     `json:"i"`
	Label       *string `json:"label"`
	NumberStyle int     `json:"numberStyle"`
	Para        int     `json:"para"`
	Offset      int     `json:"offset"`
	Page        int     `json:"page"`
	Section     int     `json:"section"`
	Text        string  `json:"text"`
}

// WordRead Word 文档读取回执。
type WordRead struct {
	Paragraphs int            `json:"paragraphs"`
	Tables     int            `json:"tables"`
	Pages      *int           `json:"pages,omitempty"`
	From       *int           `json:"from,omitempty"`
	Items      []WordItem     `json:"items,omitempty"`
	Footnotes  []WordFootnote `json:"footnotes,omitempty"`
	More       string         `json:"more,omitempty"`
	Table      *int           `json:"table,omitempty"`
	Cells      [][]*string    `json:"cells,omitempty"`
}

// ExcelSheet 工作表概况。
type ExcelSheet struct {
	Name   string `json:"name"`
	Used   string `json:"used"`
	Charts *int   `json:"charts,omitempty"`
}

// ExcelFormula 单元格公式。
type ExcelFormula struct {
	Cell    string `json:"cell"`
	Formula string `json:"formula"`
}

// ExcelRead 工作簿区域读取回执。
type ExcelRead struct {
	Sheets   []ExcelSheet   `json:"sheets"`
	Sheet    string         `json:"sheet"`
	Range    string         `json:"range"`
	Values   [][]any        `json:"values"`
	Formulas []ExcelFormula `json:"formulas,omitempty"`
	Clipped  string         `json:"clipped,omitempty"`
	Objects  []string       `json:"objects,omitempty"`
	Profile  []string       `json:"profile,omitempty"`
}

// PptShape 幻灯片形状。
type PptShape struct {
	Name string    `json:"name"`
	ID   *int      `json:"id,omitempty"`
	Type string    `json:"type"`
	Box  []float64 `json:"box"`
	Text *string   `json:"text,omitempty"`
}

// PptItem 单页读取条目。
type PptItem struct {
	Slide  int        `json:"slide"`
	Layout *string    `json:"layout,omitempty"`
	Plain  *int       `json:"plain,omitempty"`
	Shapes []PptShape `json:"shapes"`
}

// PptRead 演示文稿读取回执。
type PptRead struct {
	Slides int       `json:"slides"`
	Width  float64   `json:"width"`
	Height float64   `json:"height"`
	Items  []PptItem `json:"items"`
}

// EditFailed office_edit 批量操作的中止点。
type EditFailed struct {
	Index int    `json:"index"` // 0 基
	Op    string `json:"op"`
	Error string `json:"error"`
	Code  string `json:"code"`
}

// EditResult office_edit 回执。
type EditResult struct {
	Done   []string    `json:"done"`
	Total  int         `json:"total"`
	Failed *EditFailed `json:"failed,omitempty"`
}

// RenderResult render 回执（what 形如 "Word page 3 of 12"；overview 时带 total）。
type RenderResult struct {
	What   string  `json:"what"`
	Total  int     `json:"total,omitempty"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// docState 文档状态短语（与上游 state() 一致）。
func docState(doc DocInfo) string {
	parts := []string{map[bool]string{true: "saved", false: "unsaved changes"}[doc.Saved]}
	if doc.ReadOnly {
		parts = append(parts, "read-only")
	}
	if doc.Active {
		parts = append(parts, "active")
	}
	if doc.Background {
		parts = append(parts, "in the background, no window")
	}
	return strings.Join(parts, ", ")
}

// FormatStatus office_status 回执文本。
func FormatStatus(apps []AppStatus) string {
	var lines []string
	for _, app := range apps {
		name := AppDisplayName(app.App, app.Suite)
		if !app.Installed {
			lines = append(lines, name+": not installed")
			continue
		}
		if !app.Running && len(app.Documents) == 0 {
			lines = append(lines, name+": installed, not running (office_open starts it)")
			continue
		}
		if app.Error != "" {
			lines = append(lines, name+": "+app.Error)
			continue
		}
		if len(app.Documents) == 0 {
			lines = append(lines, name+": running, nothing open")
			continue
		}
		lines = append(lines, name+":")
		for _, doc := range app.Documents {
			docPath := doc.Name + " (never saved)"
			if doc.Path != nil {
				docPath = *doc.Path
			}
			lines = append(lines, "  "+docPath+" — "+docState(doc))
		}
	}
	return strings.Join(lines, "\n")
}

// FormatOpened office_open 回执文本（含 WPS 演示公式降级的如实提示）。
func FormatOpened(doc DocInfo) string {
	how := "Already open, attached to"
	switch doc.How {
	case "created":
		how = "Created"
	case "opened":
		how = "Opened"
	}
	docPath := doc.Name
	if doc.Path != nil {
		docPath = *doc.Path
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s in %s (%s).", how, docPath, AppDisplayName(doc.App, doc.Suite), docState(doc))
	if doc.Suite == "wps" {
		b.WriteString(" This is WPS Office: the same tools and operations apply")
		if doc.App == KindPPT {
			b.WriteString(", except that a formula ($...$) on a slide is a picture laid over blanks kept for it in the text, not an equation the user can edit: after you move, resize or rewrite a text shape that has formulas, write its text again (set_text) so they are placed anew, and look at the slide (office_render)")
		}
		b.WriteString(".")
	}
	if doc.ReadOnly {
		b.WriteString(" It is read-only: edits cannot be saved to this file.")
	}
	return b.String()
}

var bodyStyles = map[string]bool{"normal": true, "正文": true, "body text": true}

// FormatWord office_read 的 Word 回执文本。
func FormatWord(read WordRead) string {
	var notes []string
	if len(read.Footnotes) > 0 {
		notes = append(notes, "Footnotes (i is the collection index, not the displayed number; edits need expect from the note text):")
		for _, note := range read.Footnotes {
			label := fmt.Sprintf("unavailable, number style %d", note.NumberStyle)
			if note.Label != nil {
				label = "display " + *note.Label
			}
			notes = append(notes, fmt.Sprintf("footnote %d [%s; para %d, offset %d, page %d, section %d] %s",
				note.I, label, note.Para, note.Offset, note.Page, note.Section, note.Text))
		}
	}
	if read.Cells != nil {
		lines := []string{fmt.Sprintf("Table %d (%d rows):", *read.Table, len(read.Cells))}
		for index, row := range read.Cells {
			parts := make([]string, len(row))
			for i, cell := range row {
				if cell == nil {
					parts[i] = "(merged)"
				} else {
					parts[i] = *cell
				}
			}
			lines = append(lines, fmt.Sprintf("%d: %s", index+1, strings.Join(parts, " | ")))
		}
		lines = append(lines, notes...)
		return strings.Join(lines, "\n")
	}
	head := fmt.Sprintf("%d paragraphs, %d table(s)", read.Paragraphs, read.Tables)
	if read.Pages != nil {
		head += fmt.Sprintf(", %d page(s)", *read.Pages)
	}
	head += "."
	var lines []string
	for _, item := range read.Items {
		if item.Size != nil {
			name := "table inside a cell"
			if item.Table != nil {
				name = fmt.Sprintf("table %d", *item.Table)
			}
			line := fmt.Sprintf("%d–%d [%s, %s]", item.I, *item.To, name, *item.Size)
			if item.Open {
				line += " holds the text below; write inside it by paragraph number:"
			}
			lines = append(lines, line)
			continue
		}
		tag := ""
		if item.Level != nil {
			tag = fmt.Sprintf("H%d", *item.Level)
		} else if item.Style != nil && !bodyStyles[strings.ToLower(*item.Style)] {
			tag = *item.Style
		}
		text := "(empty)"
		if item.Text != nil && *item.Text != "" {
			text = *item.Text
		}
		indent := ""
		if item.InTable {
			indent = "  "
		}
		if tag != "" {
			lines = append(lines, fmt.Sprintf("%s%d [%s] %s", indent, item.I, tag, text))
		} else {
			lines = append(lines, fmt.Sprintf("%s%d %s", indent, item.I, text))
		}
	}
	out := append([]string{head}, lines...)
	if read.More != "" {
		out = append(out, read.More)
	}
	return strings.Join(append(out, notes...), "\n")
}

// FormatExcel office_read 的 Excel 回执文本。
func FormatExcel(read ExcelRead) string {
	sheetParts := make([]string, len(read.Sheets))
	for i, sheet := range read.Sheets {
		part := fmt.Sprintf("%s (%s)", sheet.Name, sheet.Used)
		if sheet.Charts != nil && *sheet.Charts > 0 {
			part += fmt.Sprintf(", %d chart(s)", *sheet.Charts)
		}
		sheetParts[i] = part
	}
	first := 1
	if m := firstDigitRun.FindString(read.Range); m != "" {
		if v, aerr := strconv.Atoi(m); aerr == nil {
			first = v
		}
	}
	rows := make([]string, len(read.Values))
	for index, row := range read.Values {
		parts := make([]string, len(row))
		for i, value := range row {
			if value == nil {
				parts[i] = ""
				continue
			}
			parts[i] = fmt.Sprintf("%v", value)
		}
		rows[index] = fmt.Sprintf("%d: %s", first+index, strings.Join(parts, " | "))
	}
	var lines []string
	if len(read.Formulas) > 0 {
		parts := make([]string, len(read.Formulas))
		for i, f := range read.Formulas {
			parts[i] = f.Cell + " " + f.Formula
		}
		lines = append(lines, "Formulas: "+strings.Join(parts, "; "))
	}
	if read.Clipped != "" {
		lines = append(lines, read.Clipped)
	}
	if len(read.Objects) > 0 {
		lines = append(lines, "On this sheet: "+strings.Join(read.Objects, "; ")+".")
	}
	if len(read.Profile) > 0 {
		lines = append(lines, "What the columns hold (the whole block, not only the rows shown):")
		for _, line := range read.Profile {
			lines = append(lines, "  "+line)
		}
	}
	out := append([]string{
		"Sheets: " + strings.Join(sheetParts, ", ") + ".",
		fmt.Sprintf("%s!%s (values; one line per row):", read.Sheet, read.Range),
	}, rows...)
	return strings.Join(append(out, lines...), "\n")
}

var firstDigitRun = regexp.MustCompile(`\d+`)

// FormatPpt office_read 的 PowerPoint 回执文本。
func FormatPpt(read PptRead) string {
	lines := []string{fmt.Sprintf("%d slide(s), %g×%g pt.", read.Slides, read.Width, read.Height)}
	for _, slide := range read.Items {
		head := fmt.Sprintf("Slide %d", slide.Slide)
		if slide.Layout != nil {
			head += fmt.Sprintf(" (%s)", *slide.Layout)
		}
		lines = append(lines, head)
		if len(slide.Shapes) == 0 {
			lines = append(lines, "  (empty)")
		}
		for _, shape := range slide.Shapes {
			box := make([]string, len(shape.Box))
			for i, v := range shape.Box {
				box[i] = fmt.Sprintf("%g", v)
			}
			line := fmt.Sprintf("  \"%s\"", shape.Name)
			if shape.ID != nil {
				line += fmt.Sprintf(" #%d", *shape.ID)
			}
			line += " " + shape.Type + " [" + strings.Join(box, ", ") + "]"
			if shape.Text != nil {
				line += ": " + strings.ReplaceAll(*shape.Text, "\n", " / ")
			}
			lines = append(lines, line)
		}
		if slide.Plain != nil && *slide.Plain > 0 {
			lines = append(lines, fmt.Sprintf("  (+ %d shape(s) without text: decoration; full: true lists them)", *slide.Plain))
		}
	}
	return strings.Join(lines, "\n")
}

// precheckedCodes 在动文档之前就失败的操作码（无需提醒重读）。
var precheckedCodes = map[string]bool{
	"BAD_ARGS": true, "ANCHOR_MISSING": true, "ANCHOR_MISMATCH": true,
	"NOT_FOUND": true, "STYLE_MISSING": true, "BUSY": true,
}

// FormatEdit office_edit 回执文本（中止点与善后指引）。
func FormatEdit(result EditResult) string {
	done := make([]string, len(result.Done))
	for i, line := range result.Done {
		done[i] = fmt.Sprintf("%d. %s", i+1, line)
	}
	if result.Failed == nil {
		out := append([]string{fmt.Sprintf("All %d operation(s) applied (not saved yet):", result.Total)}, done...)
		return strings.Join(append(out, "Before you finish, look at the pages you changed with office_render and fix what is off."), "\n")
	}
	f := result.Failed
	var out []string
	out = append(out, fmt.Sprintf("Stopped at operation %d (%s): %s", f.Index+1, f.Op, f.Error))
	if len(done) > 0 {
		out = append(out, "Applied before it:")
		out = append(out, done...)
	} else {
		out = append(out, "Nothing before it was applied.")
	}
	if !precheckedCodes[f.Code] {
		out = append(out, "The failed operation itself may have changed part of the document: read it before you retry.")
	}
	if result.Total-f.Index-1 > 0 {
		out = append(out, fmt.Sprintf("The %d operation(s) after it did not run.", result.Total-f.Index-1))
	}
	return strings.Join(out, "\n")
}
