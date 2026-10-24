// Package cad 提供 CAD 图纸（DWG/DXF）的只读解析、图框识别、分幅光栅化与
// 文本提取（办公「CAD 图纸」卡一期内核；PDF 走 pdf.go 的 poppler 页图通道）。
//
// 选型与红线（docs/gaea-cad-tool-master-plan-2026-10.md / PoC 报告）：
//   - 解析/渲染用 go-cad（Apache-2.0 纯 Go）；渲染必须显式 FontPath（Windows
//     自动探测不命中会画占位条——PoC §3）；图框识别契约=闭合 LWPOLYLINE 块
//     图框，散线图框走「整图」兜底。
//   - ASCII DXF（R12-R2004，天正 T3 主力形态）的 `\U+XXXX` 中文转义 go-cad
//     不解码——本包 DecodeUnicodeEscape 自修（PoC §2 缺陷①）。
//   - 只读零记账：不触碰引擎/模型；产物由调用方落 .gaea/cad/ 缓存。
package cad

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf16"

	"github.com/gaea/gaea/internal/gaea/fileutil"

	gocad "github.com/unitedrhino/go-cad"
)

// 图纸类型（Sniff 判型结果）。
const (
	KindDWG = "dwg"
	KindDXF = "dxf"
	KindPDF = "pdf"
)

// RenderWidthDefault 分幅渲染默认像素宽（PoC：真实 A3 图幅 2400px 下小字不足，
// 6000px 达标；默认取 3000 折中首屏，前端可按需传更宽重渲）。
const RenderWidthDefault = 3000

const (
	// MaxDWGOrDXFBytes bounds DWG/DXF input before parsing.
	MaxDWGOrDXFBytes int64 = 32 << 20
	// MaxPDFBytes bounds PDF input before invoking poppler.
	MaxPDFBytes int64 = 128 << 20
	// MaxSheets bounds frames/pages rendered by one open request.
	MaxSheets = 100
	// MaxRenderedPNGTotal bounds combined PNG bytes produced by one render pass.
	MaxRenderedPNGTotal int64 = 512 << 20
	// RenderWidthMin/RenderWidthMax keep raster dimensions within a safe budget.
	RenderWidthMin = 64
	RenderWidthMax = 8000
)

// Sheet 图框/页（DWG/DXF=DetectSheets 识别的图框；PDF=页，Box 全 0）。
type Sheet struct {
	Name     string     `json:"name"`     // 图框名（图号-图名 > 图号 > 兜底「图 N」）
	Box      [4]float64 `json:"box"`      // 世界坐标 {minX, minY, maxX, maxY}
	PassRate float64    `json:"passRate"` // 文字清晰达标率 0~1（PoC：真图幅 0.77+，误判巨框 <0.1；PDF 无此口径为 -1）
	Texts    int        `json:"texts"`    // 归属该图框的文本条数
}

// Text 图纸文本（世界坐标 + 图框归属）。
type Text struct {
	Text   string  `json:"text"`
	Layer  string  `json:"layer"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Height float64 `json:"h"`
	Sheet  int     `json:"sheet"` // 1 基图框序号；0=不在任何图框内
}

// Summary 解析摘要（OpenResult 内嵌）。
type Summary struct {
	Kind        string   `json:"kind"`
	Version     string   `json:"version"` // DWG 版本串（如 AC1027）；DXF/PDF 视解析器
	EntityCount int      `json:"entityCount"`
	Skipped     int      `json:"skipped"`
	TextCount   int      `json:"textCount"`
	Layers      []string `json:"layers"`
	Warnings    []string `json:"warnings"`
}

// SheetOut 图框 + 渲染产物路径（GaeaCadOpen/CadRaster 载荷单元）。
type SheetOut struct {
	Sheet
	PNG string `json:"png"` // 分幅 PNG 落盘路径（空=未渲染）
}

// OpenResult GaeaCadOpen 的完整载荷。
type OpenResult struct {
	Summary
	Sheets []SheetOut `json:"sheets"`
}

// Drawing 解析后的 DWG/DXF（只读；PDF 不走此结构）。
type Drawing struct {
	Path    string
	Kind    string
	Version string

	doc    *gocad.Document
	sheets []gocad.Sheet
	texts  []Text // 已解码转义
	layers []string

	entityCount int
	skipped     int
	warnings    []string
}

// Sniff 按扩展名 + DWG 魔数判型；未知返回空串。
func Sniff(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".dwg":
		return KindDWG
	case ".dxf":
		return KindDXF
	case ".pdf":
		return KindPDF
	}
	return ""
}

// ValidateInput checks the file kind, regular-file shape, and size budget before
// any parser or renderer sees the bytes.
func ValidateInput(path string) (string, os.FileInfo, error) {
	kind := Sniff(path)
	if kind == "" {
		return "", nil, fmt.Errorf("仅支持 .dwg/.dxf/.pdf 图纸文件")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", nil, fmt.Errorf("读取图纸失败: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", nil, fmt.Errorf("图纸必须是普通文件: %s", path)
	}
	limit := MaxDWGOrDXFBytes
	label := "32 MiB"
	if kind == KindPDF {
		limit = MaxPDFBytes
		label = "128 MiB"
	}
	if info.Size() > limit {
		return "", nil, fmt.Errorf("%s 图纸超过 %s 上限（当前 %d 字节）", strings.ToUpper(kind), label, info.Size())
	}
	return kind, info, nil
}

// Open 解析 DWG/DXF：图框识别 + 文本收集（含 \U+ 转义解码）+ 天正边界警告。
// 解析零渲染——分幅 PNG 由 RenderSheets/RenderSheet 按需产出。
func Open(path string) (*Drawing, error) {
	return OpenContext(context.Background(), path)
}

// OpenContext is Open with cooperative cancellation before/after parsing and a
// bounded read after stat.
func OpenContext(ctx context.Context, path string) (*Drawing, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	kind, info, err := ValidateInput(path)
	if err != nil {
		return nil, err
	}
	if kind == KindPDF {
		return nil, fmt.Errorf("PDF 由 OpenPDF 处理")
	}
	limit := MaxDWGOrDXFBytes
	data, err := readFileBounded(path, info.Size(), limit)
	if err != nil {
		return nil, fmt.Errorf("读取图纸失败: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var doc *gocad.Document
	switch {
	case kind == KindDWG || (len(data) > 2 && data[0] == 'A' && data[1] == 'C'):
		doc, err = gocad.Parse(data)
		if err != nil {
			return nil, fmt.Errorf("解析 DWG 失败: %w", err)
		}
		kind = KindDWG
	default:
		doc, err = gocad.ParseDXF(data)
		if err != nil {
			return nil, fmt.Errorf("解析 DXF 失败: %w", err)
		}
		kind = KindDXF
	}
	d := &Drawing{Path: path, Kind: kind, Version: doc.Version(), doc: doc}
	d.entityCount = doc.EntityCount()
	d.skipped = doc.Skipped()

	sheets := gocad.DetectSheets(doc)
	if len(sheets) > MaxSheets {
		return nil, fmt.Errorf("图纸识别出 %d 个图框，超过单次 %d 个上限；请拆分为多张图纸后重试", len(sheets), MaxSheets)
	}
	d.sheets = sheets
	layerSet := map[string]bool{}
	perSheet := map[int]int{}
	for _, t := range gocad.SheetTexts(doc) {
		txt := Text{
			Text: DecodeUnicodeEscape(t.Text), Layer: DecodeUnicodeEscape(t.Layer),
			X: t.X, Y: t.Y, Height: t.Height, Sheet: t.Sheet,
		}
		d.texts = append(d.texts, txt)
		if txt.Layer != "" {
			layerSet[txt.Layer] = true
		}
		if t.Sheet > 0 {
			perSheet[t.Sheet]++
		}
	}
	for l := range layerSet {
		d.layers = append(d.layers, l)
	}
	sort.Strings(d.layers)
	for i, s := range sheets { // 图框名同样过转义解码（图名来自图上文字）
		sheets[i].Name = DecodeUnicodeEscape(s.Name)
	}

	// 天正边界（诚实口径：PoC 真实样本未触发代理实体场景，以解析信号兜底提示）。
	if d.entityCount == 0 {
		d.warn("未解析出任何图形对象：图纸可能全部由天正等自定义对象（代理实体）构成，建议在天正中「图形导出(TXDC)」为 T3 或另存标准 DWG/DXF 后重试")
	} else if d.skipped > 0 {
		d.warn(fmt.Sprintf("有 %d 个对象未能解析：图纸可能含天正等自定义对象，建议导出 T3 后重试（已解析内容不受影响）", d.skipped))
	}
	if len(sheets) == 0 {
		d.warn("未识别到图框：将按整图渲染")
	}
	return d, nil
}

func readFileBounded(path string, statSize, limit int64) ([]byte, error) {
	if statSize > limit {
		return nil, fmt.Errorf("文件超过 %d 字节上限", limit)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("文件在读取期间超过 %d 字节上限", limit)
	}
	return data, nil
}

// Sheets 图框清单（含文本归属计数；PassRate 渲染前为 0）。
func (d *Drawing) Sheets() []Sheet {
	out := make([]Sheet, 0, len(d.sheets))
	for i, s := range d.sheets {
		out = append(out, Sheet{Name: s.Name, Box: s.Box, Texts: d.sheetTextCount(i + 1)})
	}
	return out
}

func (d *Drawing) sheetTextCount(sheet int) int {
	n := 0
	for _, t := range d.texts {
		if t.Sheet == sheet {
			n++
		}
	}
	return n
}

// Texts 文本清单（cap>0 时截断并置 Truncated）。
func (d *Drawing) Texts(cap int) (texts []Text, truncated bool) {
	if cap <= 0 || len(d.texts) <= cap {
		return d.texts, false
	}
	return d.texts[:cap], true
}

// Summary 摘要（Sheets 由调用方合成——渲染后才有 PassRate/PNG）。
func (d *Drawing) Summary() Summary {
	return Summary{
		Kind: d.Kind, Version: d.Version,
		EntityCount: d.entityCount, Skipped: d.skipped,
		TextCount: len(d.texts), Layers: d.layers,
		Warnings: d.warnings,
	}
}

func (d *Drawing) warn(msg string) { d.warnings = append(d.warnings, msg) }

// fontCandidates Windows 中文字体探测序列（PoC §3 红线：渲染必须显式指定；
// 自动探测不命中时 go-cad 会把中文画成占位条）。
var fontCandidates = []string{
	`simhei.ttf`, `msyh.ttc`, `msyh.ttf`, `simsun.ttc`,
}

// FontPath 显式中文字体路径（环境变量 CAD_FONT 优先；未命中返回空串交由
// go-cad 自行回退——此时 PNG 中文可能为占位条，问答链路由前端提示）。
func FontPath() string {
	if p := os.Getenv("CAD_FONT"); p != "" {
		return p
	}
	for _, c := range fontCandidates {
		for _, root := range []string{os.Getenv("WINDIR"), os.Getenv("SystemRoot"), `C:\Windows`} {
			if root == "" {
				continue
			}
			p := filepath.Join(root, "Fonts", c)
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p
			}
		}
	}
	return ""
}

// RenderSheets 渲染全部图框为 PNG 落盘（文件名 sheet-<N>.png，N 为 1 基序号；
// width<=0 用 RenderWidthDefault）。PassRate 回填进返回值。
func (d *Drawing) RenderSheets(dir string, width int) ([]SheetOut, error) {
	width = clampRenderWidth(width)
	if len(d.sheets) > MaxSheets {
		return nil, fmt.Errorf("图框数量 %d 超过 %d 上限", len(d.sheets), MaxSheets)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("创建缓存目录失败: %w", err)
	}
	results, err := gocad.RenderAllSheets(d.doc, gocad.RenderOptions{Width: width, FontPath: FontPath()}, "png")
	if err != nil {
		return nil, fmt.Errorf("渲染图框失败: %w", err)
	}
	if len(results) > MaxSheets {
		return nil, fmt.Errorf("渲染结果 %d 个图框，超过 %d 上限", len(results), MaxSheets)
	}
	var totalBytes int64
	for _, r := range results {
		totalBytes += int64(len(r.Data))
		if totalBytes > MaxRenderedPNGTotal {
			return nil, fmt.Errorf("分幅 PNG 总计超过 %d MiB 输出上限", MaxRenderedPNGTotal>>20)
		}
	}
	sheets := d.Sheets()
	out := make([]SheetOut, 0, len(results))
	for i, r := range results {
		so := SheetOut{PNG: filepath.Join(dir, fmt.Sprintf("sheet-%d.png", i+1))}
		if i < len(sheets) {
			so.Sheet = sheets[i]
		} else {
			so.Sheet.Name = fmt.Sprintf("图 %d", i+1)
		}
		so.Sheet.PassRate = r.PassRate
		if err := fileutil.AtomicWrite(so.PNG, r.Data, 0o644); err != nil {
			return nil, fmt.Errorf("落盘分幅 PNG 失败: %w", err)
		}
		out = append(out, so)
	}
	return out, nil
}

// RenderSheet 渲染第 idx 个图框（0 基）为 PNG 落盘并返回路径（问图纸/单页
// 重渲入口；width<=0 用 RenderWidthDefault）。PDF 不走此路径。
func (d *Drawing) RenderSheet(idx int, dir string, width int) (string, error) {
	if idx < 0 || idx >= len(d.sheets) {
		return "", fmt.Errorf("图框序号越界：%d（共 %d 个）", idx, len(d.sheets))
	}
	width = clampRenderWidth(width)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("创建缓存目录失败: %w", err)
	}
	png, err := gocad.RenderSheetPNG(d.doc, d.sheets[idx], gocad.RenderOptions{Width: width, FontPath: FontPath()})
	if err != nil {
		return "", fmt.Errorf("渲染图框失败: %w", err)
	}
	p := filepath.Join(dir, fmt.Sprintf("sheet-%d-w%d.png", idx+1, width))
	if err := fileutil.AtomicWrite(p, png, 0o644); err != nil {
		return "", fmt.Errorf("落盘分幅 PNG 失败: %w", err)
	}
	return p, nil
}

// OpenAndRender DWG/DXF 打开+全图框渲染一步出（GaeaCadOpen 主路径）。
func OpenAndRender(path, cacheDir string, width int) (*OpenResult, error) {
	return OpenAndRenderContext(context.Background(), path, cacheDir, width)
}

// OpenAndRenderContext is OpenAndRender with parser cancellation.
func OpenAndRenderContext(ctx context.Context, path, cacheDir string, width int) (*OpenResult, error) {
	d, err := OpenContext(ctx, path)
	if err != nil {
		return nil, err
	}
	sheets, err := d.RenderSheets(cacheDir, width)
	if err != nil {
		return nil, err
	}
	res := &OpenResult{Summary: d.Summary(), Sheets: sheets}
	if res.Layers == nil {
		res.Layers = []string{}
	}
	if res.Warnings == nil {
		res.Warnings = []string{}
	}
	return res, nil
}

func clampRenderWidth(width int) int {
	if width <= 0 {
		return RenderWidthDefault
	}
	if width < RenderWidthMin {
		return RenderWidthMin
	}
	if width > RenderWidthMax {
		return RenderWidthMax
	}
	return width
}

// DecodeUnicodeEscape 解码 DXF ASCII 的 `\U+XXXX` 中文转义（GB 规范存储形态；
// go-cad 读入不做解码——PoC §2 缺陷①的 gaea 侧自修。仅 BMP 逐码点替换，
// 代理对按 UTF-16 规则合成；无转义串时原样返回（零拷贝快路径）。
func DecodeUnicodeEscape(s string) string {
	const marker = `\U+`
	if !strings.Contains(s, marker) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if strings.HasPrefix(s[i:], marker) && i+6 <= len(s) {
			if r, ok := hex4(s[i+3 : i+7]); ok {
				i += 7
				if utf16.IsSurrogate(r) { // 高代理后跟低代理则合成码点
					if i+7 <= len(s) && strings.HasPrefix(s[i:], marker) {
						if r2, ok2 := hex4(s[i+3 : i+7]); ok2 {
							if dec := utf16.DecodeRune(r, r2); dec != 0xFFFD {
								i += 7
								b.WriteRune(dec)
								continue
							}
						}
					}
					b.WriteRune(0xFFFD) // 孤立代理：替换符占位
					continue
				}
				b.WriteRune(r)
				continue
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func hex4(s string) (rune, bool) {
	var v rune
	for i := 0; i < 4; i++ {
		c := s[i]
		var d rune
		switch {
		case c >= '0' && c <= '9':
			d = rune(c - '0')
		case c >= 'a' && c <= 'f':
			d = rune(c-'a') + 10
		case c >= 'A' && c <= 'F':
			d = rune(c-'A') + 10
		default:
			return 0, false
		}
		v = v*16 + d
	}
	return v, true
}
