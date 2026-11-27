package docmd

// wps_docx.go — PDF 转成 Word（docx），走本机 wpscli 的 pdf2word（2026-11-27）。
//
// format_convert 的 to=docx 用。录方案库的 PDF 源不再直接转 Markdown——直转的表格
// 散架（包1技术标复盘实测：合并单元格全丢、单元格内折行变碎片、跨页断表，列数矛盾
// 42 块）。改为先把 PDF 转成 Word 交给用户修正格式，修正后的 docx 再走 docx→md 录入。
//
// 口径（与 wps_excel.go 同纪律）：
//   - 「success」不可信：一律核对产物（docx 存在且非空）；
//   - 产物路径以 wpscli 终态行的 output 为准——它不覆盖已有文件，会另存成 <名>_1；
//   - pages 只收一段连续页（多段会转出多个 docx，拼不成一个）；
//   - 扫描件路线（--scanned true，走 WPS 在线识别）一次最多 100 页，超限指路 pages；
//   - 本机没有 WPS / 非会员：如实报错，没有本地兜底（本地提取出不了 docx）。
//
// 机器转出的 Word 必然带着 PDF 无表格语义导致的三种结构性缺陷（见复盘 §2.1）：
// 单元格内每个视觉折行都是独立段落（连词被切断）、垂直居中夹空段落（成片空白格）、
// 目录域与章节目录表未标注为目录（混在正文里）。回执必须提醒修正这三处，不能让
// 调用方拿「转出成功」当「格式可录」。

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ── 参数与回执 ───────────────────────────────────────────────────────

// DocxOptions ConvertToDocx 的参数。
type DocxOptions struct {
	// Pages PDF 页范围规格（"1-5" / "7"），只收一段连续页；空=整本。
	Pages string
	// Scanned 怎么看待输入的 PDF：空 / "auto"=自动判断；"true"=按扫描件走 WPS 在线识别；
	// "false"=按文字型走本地引擎。
	Scanned string
}

// DocxReceipt PDF→Word 转换的回执（产物核对结果）。
type DocxReceipt struct {
	// Pages 转换的页数（读不到页数时为 0）。
	Pages int
	// OnlineOCR 走了 WPS 在线识别（扫描件路线）。
	OnlineOCR bool
	// Elapsed wpscli 耗时。
	Elapsed time.Duration
}

const (
	// wpsDocxMaxLocalPages 文字型一次转换的页数上限（pdf2word 实测 233 页约 42 秒，
	// 上限留足余量）。
	wpsDocxMaxLocalPages = 2000
	// wpsDocxMaxScanPages 扫描件路线（在线识别）一次转换的页数上限：与转 Excel 同口径，
	// 超限指路 pages 分段——单次产物必须是一个完整 docx，不能切段再拼。
	wpsDocxMaxScanPages = 100
)

// ── 入口 ─────────────────────────────────────────────────────────────

// ConvertToDocx 把一份 PDF 转成 Word（docx），走本机 WPS 的 wpscli pdf2word。
//
// 产物先落在临时目录，核对无误后由 save 搬到最终位置（save=nil 时只核对不落盘）。
// save 里报的错原样返回——调用方在 save 里看取消信号，用户按停止后不落盘。
func ConvertToDocx(src string, opt DocxOptions, save func(prod string) error) (DocxReceipt, error) {
	if !strings.EqualFold(filepath.Ext(src), ".pdf") {
		return DocxReceipt{}, fmt.Errorf("不支持的文件格式: %s（to=docx 只收 .pdf；docx/pptx 源直接走转 Markdown）", filepath.Ext(src))
	}
	if _, err := os.Stat(src); err != nil {
		return DocxReceipt{}, err
	}
	if !wpsPDFEnabled() {
		return DocxReceipt{}, errors.New("本机 WPS 通道已关闭（GAEA_WPS_PDF=0），无法转 Word")
	}
	if wpsSuspended() {
		return DocxReceipt{}, errors.New("WPS 账号级失败后的静默期内（未登录 / 无会员 / 超配额），暂不重试")
	}
	cli := wpsCLILookup()
	if cli == "" {
		return DocxReceipt{}, errors.New("本机未找到 WPS（wpscli）：转 Word 需要 WPS Office（登录会员）；也可在 WPS 里手动「PDF 转 Word」后把 docx 给回来")
	}
	dir, err := os.MkdirTemp("", "gaea-wpsdocx-")
	if err != nil {
		return DocxReceipt{}, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	out := filepath.Join(dir, "out.docx")
	plan, err := wpsDocxPlan(src, opt, cli, out)
	if err != nil {
		return DocxReceipt{}, err
	}
	start := time.Now()
	code, last, tail, err := wpsRunTimed(plan.timeout+wpsExcelGrace, cli, plan.args...)
	if err != nil {
		return DocxReceipt{}, fmt.Errorf("wpscli pdf2word 拉起失败: %w", err)
	}
	if code != 0 {
		if code == 100 || code == 101 || code == 429 { // 账号级失败：进全局静默期
			wpsSuspend(wpsFailCooldown)
		}
		return DocxReceipt{}, wpsError("pdf2word", code, tail)
	}
	// 产物路径以终态行为准：wpscli 不覆盖已有文件，会另存成 <名>_1。只认临时目录里的
	// 路径——别的位置一律不信（下面要把它搬走）。
	prod := out
	if reported := strings.TrimSpace(last.Output); reported != "" {
		if p := filepath.Clean(filepath.FromSlash(reported)); filepath.Dir(p) == dir {
			prod = p
		}
	}
	if st, statErr := os.Stat(prod); statErr != nil || st.Size() == 0 {
		return DocxReceipt{}, errors.New("wpscli 报成功但没有产出 Word（产物缺失或为空）")
	}
	rec := DocxReceipt{Pages: plan.pages, OnlineOCR: plan.online, Elapsed: time.Since(start)}
	if save != nil {
		if err := save(prod); err != nil {
			return DocxReceipt{}, err
		}
	}
	return rec, nil
}

// ── 执行计划 ─────────────────────────────────────────────────────────

// wpsDocxCall 一次 pdf2word 的执行计划。
type wpsDocxCall struct {
	args    []string      // 交给 wpscli 的完整参数
	pages   int           // 页数（读不到页数时为 0）
	online  bool          // 走 WPS 在线识别（扫描件路线）
	timeout time.Duration // 传给 wpscli 的 --timeout
}

// wpsDocxPlan 决定路线与参数。只读文件与调用 pdfinfo（本地引擎，不到 1 秒），不消耗
// 会员额度——超限、越界这些错都在这里就报出去。路线判定与 to=xlsx 同一套：
// pdfinfo 判是扫描件、或自写提取读不出可信文字层，就走 --scanned true 在线识别。
func wpsDocxPlan(src string, opt DocxOptions, cli, out string) (wpsDocxCall, error) {
	switch opt.Scanned {
	case "", "auto", "true", "false":
	default:
		return wpsDocxCall{}, fmt.Errorf("scanned 取值无效（%q）：只收 auto / true / false", opt.Scanned)
	}
	var p wpsDocxCall
	total, scan := wpsExcelPageInfo(cli, src)
	spans := pageSpans(opt.Pages, total)
	if len(spans) == 0 {
		return wpsDocxCall{}, fmt.Errorf("页范围 %q 超出这份 PDF 的页数（共 %d 页）", opt.Pages, total)
	}
	if len(spans) > 1 {
		return wpsDocxCall{}, fmt.Errorf("to=docx 的 pages 只收一段连续页（如 1-100），收到 %q", opt.Pages)
	}
	first, last := spans[0][0], spans[0][1] // last=0：到文件尾
	if last <= 0 {
		last = total
	}
	online := scan
	switch opt.Scanned {
	case "false":
		online = false
	case "true":
		online = true
	default: // auto：WPS 判是扫描件，或自写提取读不出可信文字层
		if !online {
			online = !pdfTextLayerReadable(src, first, last, total)
		}
	}
	if online && last <= 0 {
		return wpsDocxCall{}, errors.New("读不到 PDF 页数，无法指定 WPS 在线识别的页范围")
	}
	if last > 0 {
		p.pages = last - first + 1
	}
	limit, route := wpsDocxMaxLocalPages, "文字型转 Word"
	if online {
		route, limit = "WPS 在线识别（扫描件）", wpsDocxMaxScanPages
	}
	if p.pages > limit {
		return wpsDocxCall{}, fmt.Errorf("%s一次最多转 %d 页，这次是 %d 页：请用 pages 分段转换（如 1-%d），每段各出一份 docx",
			route, limit, p.pages, limit)
	}
	p.args = []string{"pdf2word", src, "-o", out, "--json"}
	if online {
		// 在线识别引擎要显式页范围
		p.args = append(p.args, "--scanned", "true")
	}
	if online || opt.Pages != "" {
		// 整本文字型不带 --range：不把页数读数写进去，免得读数有偏差时截掉末页
		p.args = append(p.args, "--range", wpsExcelRange(first, last))
	}
	// 超时沿用转 Excel 的每页余量（文字型 0.6 秒/页、在线 1.7 秒/页，下限 300 秒）——
	// pdf2word 实测更快（233 页 42 秒），余量只多不少。
	p.timeout = wpsExcelTimeout(p.pages, online, false)
	p.args = append(p.args, "--timeout", strconv.Itoa(int(p.timeout/time.Second)))
	return p, nil
}
