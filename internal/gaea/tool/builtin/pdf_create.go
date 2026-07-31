package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(pdfCreate{}) }

type pdfCreate struct{}

func (pdfCreate) Name() string { return "pdf_create" }

func (pdfCreate) Description() string {
	return "创建 PDF 文件：输入 Markdown 文本内容，生成简单的 PDF 文档（标题+段落）。零外部依赖。"
}

func (pdfCreate) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"输出文件路径（.pdf）"},
  "title":{"type":"string","description":"文档标题（页面顶部居中显示）"},
  "content":{"type":"string","description":"文档正文（多行文本，#/##/### 标题，空行分段）"},
  "footer":{"type":"string","description":"页脚文本（可选，默认留空）"}
},
"required":["path","content"]
}`)
}

func (pdfCreate) ReadOnly() bool { return false }

func (pdfCreate) CompactDescription() string     { return compactDesc["pdf_create"] }
func (pdfCreate) CompactSchema() json.RawMessage { return compactSchema["pdf_create"] }

func (pdfCreate) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Path    string `json:"path"`
		Title   string `json:"title,omitempty"`
		Content string `json:"content"`
		Footer  string `json:"footer,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if p.Path == "" || p.Content == "" {
		return "", fmt.Errorf("path 和 content 不能为空")
	}
	if !strings.HasSuffix(strings.ToLower(p.Path), ".pdf") {
		return "", fmt.Errorf("path 必须以 .pdf 结尾")
	}

	buf := buildPDF(p.Title, p.Content, p.Footer)
	if err := os.WriteFile(p.Path, buf, 0644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}
	return tool.WrapText(fmt.Sprintf("已创建 PDF 文件：%s（%d 字节）", p.Path, len(buf))), nil
}

// --- PDF 构建 ---

const (
	a4W = 595.28
	a4H = 841.89
	lm  = 56.69 // 左边距 2cm
)

// pdfWriter 逐步构造 PDF
type pdfWriter struct {
	buf     bytes.Buffer
	objects int
	offsets []int
}

func newPDF() *pdfWriter {
	w := &pdfWriter{offsets: make([]int, 0, 16)}
	w.w("%%PDF-1.4\n")
	return w
}

func (w *pdfWriter) w(f string, a ...interface{}) { w.buf.WriteString(fmt.Sprintf(f, a...)) }

func (w *pdfWriter) obj(body func()) int {
	w.objects++
	w.offsets = append(w.offsets, w.buf.Len())
	w.w("%d 0 obj\n", w.objects)
	body()
	w.w("endobj\n")
	return w.objects
}

func (w *pdfWriter) finish() []byte {
	xref := w.buf.Len()
	w.w("xref\n0 %d\n", w.objects+1)
	w.w("0000000000 65535 f \n")
	for _, off := range w.offsets {
		w.w("%010d 00000 n \n", off)
	}
	w.w("trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", w.objects+1, xref)
	return w.buf.Bytes()
}

// escPDF 转义 PDF 字符串中的特殊字符
func escPDF(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}

// splitLines 根据字体大小近似换行
func splitLines(text string, maxW, fontSize float64) []string {
	cw := fontSize * 0.5
	max := int(maxW / cw)
	if max < 1 {
		max = 1
	}
	runes := []rune(text)
	if len(runes) <= max {
		return []string{text}
	}
	var out []string
	for len(runes) > 0 {
		if len(runes) <= max {
			out = append(out, string(runes))
			break
		}
		cut := max
		for i := max; i > max/2; i-- {
			if runes[i] == ' ' {
				cut = i
				break
			}
		}
		out = append(out, string(runes[:cut]))
		runes = runes[cut:]
		for len(runes) > 0 && runes[0] == ' ' {
			runes = runes[1:]
		}
	}
	return out
}

func buildPDF(title, content, footer string) []byte {
	w := newPDF()

	// Obj 1: Catalog
	w.obj(func() { w.w("<< /Type /Catalog /Pages 2 0 R >>\n") })
	// Obj 2: Pages
	w.obj(func() { w.w("<< /Type /Pages /Kids [3 0 R] /Count 1 >>\n") })

	// 预留 Obj 3 (Page)，先写 Obj 4 (Font)
	_ = w.obj(func() {
		w.w("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\n")
	})

	// 生成页面内容流
	var stream bytes.Buffer
	y := a4H - lm

	text := func(s string, size, x, y float64) {
		stream.WriteString(fmt.Sprintf("BT\n/F1 %.2f Tf\n%.2f %.2f Td\n(%s) Tj\nET\n",
			size, x, y, escPDF(s)))
	}

	newPage := func() {
		stream.WriteString("showpage\n")
		y = a4H - lm
	}

	draw := func(s string, size float64, bold, center bool) {
		lines := splitLines(s, a4W-2*lm, size)
		for _, line := range lines {
			if y-size < lm {
				newPage()
			}
			y -= size + 3
			x := lm
			if center {
				x = (a4W - float64(len([]rune(line)))*size*0.5) / 2
			}
			text(line, size, x, y)
		}
	}

	// 标题
	if title != "" {
		draw(title, 22, true, true)
		y -= 14
	}

	// 正文段落
	paras := parsePara(content)
	for _, p := range paras {
		if p == "" {
			y -= 4
			continue
		}
		var size float64
		var space float64
		switch {
		case strings.HasPrefix(p, "### "):
			p = strings.TrimPrefix(p, "### ")
			size, space = 12, 5
		case strings.HasPrefix(p, "## "):
			p = strings.TrimPrefix(p, "## ")
			size, space = 14, 6
		case strings.HasPrefix(p, "# "):
			p = strings.TrimPrefix(p, "# ")
			size, space = 16, 8
		default:
			size, space = 11, 14
		}
		lines := splitLines(p, a4W-2*lm, size)
		for i, line := range lines {
			if y-size < lm {
				newPage()
			}
			sp := space
			if i > 0 {
				sp = size + 3
			}
			y -= sp
			text(line, size, lm, y)
		}
		y -= 6
	}

	// 页脚
	if footer == "" {
		footer = " "
	}
	text(footer, 8, lm, lm+16)

	streamData := stream.Bytes()

	// Obj 5: Content Stream
	contentObj := w.obj(func() {
		w.w("<< /Length %d >>\nstream\n", len(streamData))
		w.buf.Write(streamData)
		w.w("\nendstream\n")
	})

	// Obj 3: Page（现在才能写，因为需要知道 Contents obj 号）
	w.obj(func() {
		w.w("<< /Type /Page /Parent 2 0 R\n")
		w.w("   /MediaBox [0 0 %.2f %.2f]\n", a4W, a4H)
		w.w("   /Resources << /Font << /F1 4 0 R >> >>\n")
		w.w("   /Contents %d 0 R\n", contentObj)
		w.w(">>\n")
	})

	return w.finish()
}

func parsePara(content string) []string {
	lines := strings.Split(content, "\n")
	var out []string
	var cur strings.Builder
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t == "" {
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
			continue
		}
		// 标题行单独成段
		if strings.HasPrefix(t, "#") {
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
			out = append(out, t)
			continue
		}
		if cur.Len() > 0 {
			cur.WriteString(" ")
		}
		cur.WriteString(t)
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}
