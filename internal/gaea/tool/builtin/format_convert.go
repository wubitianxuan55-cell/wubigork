// Package builtin — 文档格式转换工具（docx/xlsx/pptx/pdf → Markdown；PDF / 图片 → Excel）。
package builtin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/docmd"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(formatConvert{}) }

type formatConvert struct{ workDir string }

func (formatConvert) Name() string { return "format_convert" }

func (formatConvert) Description() string {
	return "文档格式转换：将 docx/xlsx/pptx/pdf 文件转换为 Markdown。docx→md 保留标题层级和表格；xlsx→md 生成表格；pptx→md 提取每页文本与备注。PDF 优先经本机 WPS 转换（需已登录 WPS 会员）：文字型 PDF 表格成真表格、图片保留；扫描件走 WPS 在线 OCR（174 页约 1 分钟），照片版 PDF 也能出真表格。WPS 不可用或失败时退回本地提取 / 本地 OCR（本地 OCR 一次最多 30 页，超限报错——用 pages 分段）。preserve_images=true 时保留内嵌图片到 Markdown 同目录的 -assets 子目录并改相对引用（必须同时给 output）；录方案库时建议开启。回执附标题/表格/图片计数，三项全 0 说明产物不可用。耗时数秒到数分钟，取决于页数（233 页经 WPS 约 1 分半）；不消耗主模型 token。to=xlsx：把 PDF（清单 / 定额表 / 报价单）或图片、装着图片的目录转成 Excel 真表格（金额是真数值，定额编码带前导零存文本），必须给 output（.xlsx）。文字型 PDF 走本地引擎（约 0.6 秒/页）；扫描件 / 照片走 WPS 在线识别，默认开 ai_fix（数字零丢改，约 14 秒/页，一次最多 20 页，超出用 pages 分一段连续页）；ai_fix=false 快 8 倍但实测约 5% 数字格会丢改，只能要快不能取数。scanned=true/false 强制路线；per_page_sheet=true 一页一个工作表。回执列出疑似缺格的行（在线识别实测会漏掉工程量为 1 的格子），须对照原件抽检。"
}

func (formatConvert) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"源文件路径（支持 .docx/.xlsx/.pptx/.pdf）"},
  "output":{"type":"string","description":"输出文件路径：转 Markdown 时可省略（不指定则返回文本）；to=xlsx 时必填（.xlsx）"},
  "pages":{"type":"string","description":"PDF页码范围，如\"1-5\"或\"1,3,5\"（仅PDF有效）；to=xlsx 时只收一段连续页"},
  "preserve_images":{"type":"boolean","description":"保留 DOCX/PPTX 内嵌图片：要求同时给 output，图片写入同名 -assets 目录，Markdown 改相对引用。录方案库时建议开启"},
  "to":{"type":"string","enum":["md","xlsx"],"description":"md（默认）=转 Markdown；xlsx=PDF/图片转 Excel 表格（走本机 WPS，需登录会员）"},
  "scanned":{"type":"string","enum":["auto","true","false"],"description":"仅 to=xlsx 且输入是 PDF：auto=自动判断（默认）；true=按扫描件走 WPS 在线识别；false=按文字型走本地引擎"},
  "ai_fix":{"type":"boolean","description":"仅 to=xlsx 的在线识别路线：AI 修复，默认开（数字零丢改，约 14 秒/页）；false 快 8 倍但约 5% 数字格会丢改"},
  "per_page_sheet":{"type":"boolean","description":"仅 to=xlsx：一页一个工作表（默认全部页堆进一个工作表）"}
},
"required":["path"]
}`)
}

// ReadOnly is false because the optional output path writes Markdown to disk.
func (formatConvert) ReadOnly() bool { return false }

func (f formatConvert) ReadTarget(args json.RawMessage) (string, bool) {
	var p fcInput
	if err := json.Unmarshal(args, &p); err != nil || strings.TrimSpace(p.Path) == "" {
		return "", false
	}
	return resolveIn(f.workDir, p.Path), true
}

func (f formatConvert) WritePaths(args json.RawMessage) ([]string, error) {
	var p fcInput
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.Output) == "" {
		return nil, nil
	}
	out := resolveIn(f.workDir, p.Output)
	if p.To == "xlsx" { // Excel 产物只有一个文件，没有图片 sidecar
		return []string{out}, nil
	}
	if p.PreserveImages {
		return []string{out, formatConvertAssetDir(out)}, nil
	}
	return []string{out}, nil
}

func (formatConvert) CompactDescription() string     { return compactDesc["format_convert"] }
func (formatConvert) CompactSchema() json.RawMessage { return compactSchema["format_convert"] }

type fcInput struct {
	Path           string `json:"path"`
	Output         string `json:"output,omitempty"`
	Pages          string `json:"pages,omitempty"`
	PreserveImages bool   `json:"preserve_images,omitempty"`
	// To："xlsx" 走 to=xlsx（PDF / 图片 → Excel）；空 / "md" 是默认的转 Markdown。
	To string `json:"to,omitempty"`
	// Scanned 仅 to=xlsx 且输入是 PDF：空 / "auto"（默认）/ "true" / "false"。
	Scanned string `json:"scanned,omitempty"`
	// AIFix 仅 to=xlsx 的在线识别路线；nil = 默认开。
	AIFix *bool `json:"ai_fix,omitempty"`
	// PerPageSheet 仅 to=xlsx：一页一个工作表。
	PerPageSheet bool `json:"per_page_sheet,omitempty"`
}

// convertPreservingImages 转换 seam（测试打桩；单测不得真调 WPS / OCR）。
var convertPreservingImages = docmd.ConvertPreservingImages

// convertToExcel to=xlsx 的转换 seam（同上）。
var convertToExcel = docmd.ConvertToExcel

// fcOutcome 一次转换的结果（后台 goroutine 经管道回传）。
type fcOutcome struct {
	md        string
	total     int
	truncated bool
	images    int
	err       error
}

func (f formatConvert) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p fcInput
	if err := json.Unmarshal(args, &p); err != nil {
		return "", codedError("FORMAT_INVALID_ARGS", "参数无效: %v", err)
	}
	if p.Path == "" {
		return "", codedError("FORMAT_INVALID_ARGS", "path 不能为空")
	}
	switch p.To {
	case "", "md", "xlsx":
	default:
		return "", codedError("FORMAT_INVALID_ARGS", "to 取值无效（%s）：只收 md / xlsx", p.To)
	}
	if p.PreserveImages && strings.TrimSpace(p.Output) == "" {
		return "", codedError("FORMAT_INVALID_ARGS", "preserve_images=true 需要同时给 output 路径")
	}
	p.Path = resolveIn(f.workDir, p.Path)
	if p.Output != "" {
		p.Output = resolveIn(f.workDir, p.Output)
	}
	if p.To == "xlsx" {
		return f.toExcel(ctx, p)
	}

	// 转换链不看取消信号（pdftoppm / wpscli / OCR 引擎请求都是整段同步的）：用户按
	// 停止时立即返回，让本轮收尾——否则 executeBatch 死等这个工具，下一条消息一直
	// 「排队中」（2026-11-26 实弹：174 页扫描件 OCR 堵到重启）。转换在后台跑完即弃，
	// 结果仍进 docmd 的进程内缓存，重试同一份可秒回；落盘在下面，取消后不会写文件。
	ch := make(chan fcOutcome, 1)
	go func() {
		md, total, truncated, images, err := convertPreservingImages(p.Path, p.Pages, docmd.DefaultMaxPDFPages)
		ch <- fcOutcome{md: md, total: total, truncated: truncated, images: images, err: err}
	}()
	var r fcOutcome
	select {
	case r = <-ch:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	md, total, truncated, images, err := r.md, r.total, r.truncated, r.images, r.err
	if err != nil {
		// U1 错误码：模型按 code 路由恢复（换路径/如实告知不支持），不解析散文
		switch {
		case strings.Contains(err.Error(), "不支持的文件格式"):
			return "", codedError("FORMAT_UNSUPPORTED", "%v", err)
		case errors.Is(err, os.ErrNotExist) || os.IsNotExist(err):
			return "", codedError("FORMAT_SOURCE_MISSING", "源文件不存在: %s（%v）", p.Path, err)
		case strings.Contains(err.Error(), "文字层无法解码"):
			// 自写提取只认得到乱码（子集字体缺字体映射），本机 WPS 与 OCR 都没接住：
			// 如实报失败，不把乱码落盘（2026-11-24 实弹：233 页报告转出 17 万替换字符却报成功）。
			return "", codedError("FORMAT_TEXT_UNDECODABLE", "%v", err)
		default:
			return "", codedError("FORMAT_CONVERT_FAILED", "%v", err)
		}
	}
	if truncated {
		md += fmt.Sprintf("\n\n> 转换已截断：PDF 共 %d 页，本次仅处理前 %d 页。可指定 pages 参数分段转换，或使用 summarize_file 获取全文摘要。",
			total, docmd.DefaultMaxPDFPages)
	}
	md = fmt.Sprintf("# 文档转换: %s\n\n%s\n\n---\n*由 gaea format_convert 转换*", filepath.Base(p.Path), md)

	if p.Output != "" {
		// 自动创建输出文件的父目录，与 write_file 行为一致；否则输出路径
		// 指向不存在的目录时 os.WriteFile 会失败，导致“没有生成文件”。
		if dir := filepath.Dir(p.Output); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return "", codedError("FORMAT_OUTPUT_WRITE_FAILED", "创建输出目录失败: %v", err)
			}
		}
		assetDir := ""
		if p.PreserveImages {
			assetDir = formatConvertAssetDir(p.Output)
			var externalized int
			md, externalized, err = docmd.ExternalizeDataImages(md, filepath.Dir(p.Output), assetDir)
			if err != nil {
				return "", codedError("FORMAT_OUTPUT_WRITE_FAILED", "保存文档图片失败: %v", err)
			}
			images = externalized
		}
		if err := os.WriteFile(p.Output, []byte(md), 0o644); err != nil {
			return "", codedError("FORMAT_OUTPUT_WRITE_FAILED", "写入输出文件失败: %v", err)
		}
		// 回执带结构计数：调用方一眼判断产物可用（乱码/空转换的特征是三项全 0）。
		st := docmd.CountStructure(md)
		detail := fmt.Sprintf("已转换并保存为 %s（%d 字）", p.Output, utf8.RuneCountInString(md))
		detail += fmt.Sprintf("；标题 %d / 表格 %d / 图片 %d", st.Headings, st.Tables, st.Images)
		switch {
		case images > 0 && p.PreserveImages:
			detail += fmt.Sprintf("（保留图片 %d 张到 %s）", images, assetDir)
		case images > 0:
			detail += "（图片以 data URI 内嵌在 Markdown 里；要落成图片文件请设 preserve_images=true）"
		case p.PreserveImages && strings.EqualFold(filepath.Ext(p.Path), ".pdf"):
			detail += "（未抽取到图片：该 PDF 没有内嵌图，或本机没有 WPS、只做了纯文本提取）"
		}
		return tool.WrapText(detail), nil
	}
	// 不落盘时图片没法交付：去掉 data URI 再回（几百页的 PDF 经 WPS 转换带 108 张图，
	// 原样回给模型是几 MB 的 base64），并在回执里说明怎么要图片。
	if strings.Contains(md, "data:image/") {
		n := strings.Count(md, "](data:image/")
		md = strings.TrimSpace(docmd.StripDataImages(md)) +
			fmt.Sprintf("\n\n---\n*内嵌图片 %d 张未随文本返回；需要图片请给 output 与 preserve_images=true 落成文件*", n)
	}
	return tool.WrapText(md), nil
}

func formatConvertAssetDir(output string) string {
	ext := filepath.Ext(output)
	stem := strings.TrimSuffix(filepath.Base(output), ext)
	return filepath.Join(filepath.Dir(output), stem+"-assets")
}

// ── to=xlsx：PDF / 图片 → Excel ─────────────────────────────────────

// errExcelOutputWrite Excel 产物搬到目标位置失败（与转换失败分开报错误码）。
var errExcelOutputWrite = errors.New("Excel 产物写入目标位置失败")

// toExcel to=xlsx：PDF / 图片转 Excel 真表格，走本机 WPS（见 docmd/wps_excel.go）。
//
// 取消语义与转 Markdown 一致：用户按停止立即返回。落盘在 save 回调里、先看取消信号，
// 所以取消后不写文件。目标目录在拉起 WPS 之前就建好——转换动辄几分钟，不能到最后才
// 因为目录不存在失败。
func (f formatConvert) toExcel(ctx context.Context, p fcInput) (string, error) {
	if strings.TrimSpace(p.Output) == "" {
		return "", codedError("FORMAT_INVALID_ARGS", "to=xlsx 需要同时给 output（.xlsx 文件路径）")
	}
	if !strings.EqualFold(filepath.Ext(p.Output), ".xlsx") {
		return "", codedError("FORMAT_INVALID_ARGS", "to=xlsx 的 output 必须是 .xlsx 文件，收到 %s", p.Output)
	}
	if p.PreserveImages {
		return "", codedError("FORMAT_INVALID_ARGS", "preserve_images 只用于转 Markdown，to=xlsx 不需要它")
	}
	st, statErr := os.Stat(p.Path)
	if statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			return "", codedError("FORMAT_SOURCE_MISSING", "源文件不存在: %s（%v）", p.Path, statErr)
		}
		return "", codedError("FORMAT_CONVERT_FAILED", "读取源文件失败: %v", statErr)
	}
	fromImages := true // 目录 / 图片走 photo2excel；回执里的「页」是图片张数
	switch {
	case st.IsDir():
	case strings.EqualFold(filepath.Ext(p.Path), ".pdf"):
		fromImages = false
	case docmd.IsExcelSourceExt(filepath.Ext(p.Path)):
	default:
		return "", codedError("FORMAT_UNSUPPORTED",
			"to=xlsx 支持 .pdf、图片（jpg/jpeg/png/bmp/gif/tif/tiff）或装着图片的目录，收到 %s", p.Path)
	}
	if dir := filepath.Dir(p.Output); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", codedError("FORMAT_OUTPUT_WRITE_FAILED", "创建输出目录失败: %v", err)
		}
	}
	aiFix := p.AIFix == nil || *p.AIFix // 默认开：数字保真优先，要快由调用方显式关
	type outcome struct {
		rec docmd.ExcelReceipt
		err error
	}
	ch := make(chan outcome, 1)
	go func() {
		rec, err := convertToExcel(p.Path, docmd.ExcelOptions{
			Pages: p.Pages, Scanned: p.Scanned, AIFix: aiFix, PerPageSheet: p.PerPageSheet,
		}, func(prod string) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := docmd.MoveOrCopyFile(prod, p.Output); err != nil {
				return fmt.Errorf("%w（目标被 Excel / WPS 占用时先关闭它）: %v", errExcelOutputWrite, err)
			}
			return nil
		})
		ch <- outcome{rec, err}
	}()
	var o outcome
	select {
	case o = <-ch:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	if o.err != nil {
		switch {
		case errors.Is(o.err, context.Canceled), errors.Is(o.err, context.DeadlineExceeded):
			return "", o.err
		case errors.Is(o.err, errExcelOutputWrite):
			return "", codedError("FORMAT_OUTPUT_WRITE_FAILED", "%v", o.err)
		case strings.Contains(o.err.Error(), "不支持的文件格式"), strings.Contains(o.err.Error(), "目录里没有图片"):
			// 输入不是 to=xlsx 认的类型（含空图片目录）：如实告知不支持，不落盘
			return "", codedError("FORMAT_UNSUPPORTED", "%v", o.err)
		default:
			return "", codedError("FORMAT_CONVERT_FAILED", "%v", o.err)
		}
	}
	return formatExcelReceipt(p.Output, fromImages, aiFix, o.rec), nil
}

// formatExcelReceipt to=xlsx 的回执：产物位置、结构计数，以及该抽检的地方。
func formatExcelReceipt(output string, images, aiFix bool, r docmd.ExcelReceipt) string {
	var b strings.Builder
	fmt.Fprintf(&b, "已转换为 Excel：%s（", output)
	switch {
	case images && r.Pages > 1:
		fmt.Fprintf(&b, "%d 张图片，", r.Pages)
	case !images && r.Pages > 0:
		fmt.Fprintf(&b, "%d 页，", r.Pages)
	}
	fmt.Fprintf(&b, "耗时 %s）\n工作表 %d 个，共 %d 行、最宽 %d 列；非空格子 %d（数字格 %d",
		r.Elapsed.Round(time.Second), r.Sheets, r.Rows, r.MaxCols, r.Cells, r.Numbers)
	if r.NumberText > 0 {
		fmt.Fprintf(&b, "，另有 %d 格数字以文本保存——定额编码、多级序号，前导零保留", r.NumberText)
	}
	b.WriteString("）")
	if r.OnlineOCR {
		if aiFix {
			b.WriteString("\n经 WPS 在线识别（AI 修复开）：数字须对照原件抽检")
		} else {
			b.WriteString("\n经 WPS 在线识别（未开 AI 修复，实测约 5% 数字格会丢改）：只能要快，不能直接取数，数字须对照原件逐个核对")
		}
	}
	if r.SuspectCount > 0 {
		fmt.Fprintf(&b, "\n疑似缺格 %d 行（%s", r.SuspectCount, strings.Join(r.Suspects, "、"))
		if r.SuspectCount > len(r.Suspects) {
			fmt.Fprintf(&b, " 等")
		}
		b.WriteString("）：这些行的数字列是空的——在线识别实测会漏掉工程量为 1 的格子，请对照原件核对")
	}
	return tool.WrapText(b.String())
}
