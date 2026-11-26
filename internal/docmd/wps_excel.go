package docmd

// wps_excel.go — PDF / 图片转成 Excel 表格，走本机 wpscli（2026-11-26）。
//
// format_convert 的 to=xlsx 用。清单、定额表、报价单这类以表为主的文档，转成 Markdown
// 只是「看起来像表」——数字混在竖线里，模型得自己抠；转成 xlsx 是真表格：金额是真数值、
// 12 位定额编码带前导零存成文本、合并单元格保留。
//
// 三条路线（底数见 C:\AI\re\wps-12.1.0.29690\docs\WPSCLI-CAPABILITIES.md §2A）：
//   - 文字型 PDF：pdf2excel 本地引擎（约 0.6 秒/页，22 页对文字层数字召回 1198/1198）；
//   - 扫描件 / 照片版 PDF：pdf2excel --scanned true 走 WPS 在线识别。纯在线识别实测
//     392 个数字格丢改 18 个（金额末位被截、工程量为 1 的格子整格丢），开 --ai-fix true
//     后丢改 0、耗时约 8 倍（14 秒/页）——取数场景默认开；
//   - 图片：photo2excel（同一套在线识别，没有 --ai-fix，工程量为 1 的格子同样会丢）。
//
// 口径：
//   - 「success」不可信：一律核对产物（文件存在且非空、工作簿里至少有一个非空格子）；
//   - 产物路径以 wpscli 终态行的 output 为准——它不覆盖已有文件，会另存成 <名>_1；
//   - 一次调用只出一个文件：在线识别按页数设单次上限，超限报错指路 pages 分段。不在
//     内部切段再拼——每段产物都是一个带样式的整本工作簿，拼接会丢格式；
//   - 本机没有 WPS / 非会员：如实报错，没有本地兜底（本地 OCR 只出纯文本，出不了表格）。

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ── 参数与回执 ───────────────────────────────────────────────────────

// ExcelOptions ConvertToExcel 的参数。
type ExcelOptions struct {
	// Pages PDF 页范围规格（"1-5" / "7"），只收一段连续页；空=整本。
	Pages string
	// Scanned 怎么看待输入的 PDF：空 / "auto"=自动判断；"true"=按扫描件走 WPS 在线识别；
	// "false"=按文字型走本地引擎。
	Scanned string
	// AIFix 在线识别路线是否开 AI 修复（数字保真，慢约 8 倍）。
	AIFix bool
	// PerPageSheet 一页一个工作表；默认全部页堆进一个工作表。
	PerPageSheet bool
}

// ExcelReceipt Excel 转换的产物核对结果（回执用：让调用方一眼判断产物可用）。
type ExcelReceipt struct {
	Sheets     int // 工作表数
	Rows       int // 各工作表行数之和
	MaxCols    int // 最宽工作表的列数
	Merged     int // 合并单元格数
	Cells      int // 非空格子数
	Numbers    int // 数值格子数
	NumberText int // 整格是数字却存成文本的格子数（定额编码、多级序号）
	// Suspects 疑似缺格的数据行位置（"Sheet3!13"），最多列 xlsxSuspectSamples 条；
	// SuspectCount 是总行数。
	Suspects     []string
	SuspectCount int
	// Pages PDF 页数（图片输入=图片张数；读不到页数时为 0）。
	Pages int
	// OnlineOCR 走了 WPS 在线识别。
	OnlineOCR bool
	// Elapsed wpscli 耗时。
	Elapsed time.Duration
}

const (
	// wpsExcelMaxLocalPages 文字型一次转换的页数上限（约 0.6 秒/页，2000 页约 20 分钟）。
	wpsExcelMaxLocalPages = 2000
	// wpsExcelMaxScanPages 在线识别（不开 AI 修复，约 1.7 秒/页）一次转换的页数上限。
	wpsExcelMaxScanPages = 100
	// wpsExcelMaxScanPagesAI 在线识别 + AI 修复（约 14 秒/页）一次转换的页数上限：20 页约 5 分钟。
	wpsExcelMaxScanPagesAI = 20
	// wpsExcelMinTimeout 传给 wpscli 的 --timeout 下限（也是 wpscli 自己的默认值）。
	wpsExcelMinTimeout = 300 * time.Second
	// wpsExcelGrace gaea 侧上限比 wpscli 的 --timeout 多留的余量。
	wpsExcelGrace = 30 * time.Second
)

// wpsExcelImageExts photo2excel 认的图片格式。
var wpsExcelImageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".bmp": true, ".gif": true, ".tif": true, ".tiff": true,
}

// IsExcelSourceExt to=xlsx 认不认这个扩展名：PDF 与 photo2excel 支持的图片。
func IsExcelSourceExt(ext string) bool {
	ext = strings.ToLower(ext)
	return ext == ".pdf" || wpsExcelImageExts[ext]
}

// ── 入口 ─────────────────────────────────────────────────────────────

// ConvertToExcel 把 src 转成一个 xlsx，走本机 WPS 的 wpscli。src 可以是一份 PDF、一张
// 图片，或一个装着图片的目录（多张照片合成一个工作簿，一图一表）。
//
// 产物先落在临时目录，核对无误后由 save 搬到最终位置（save=nil 时只核对不落盘）。
// save 里报的错原样返回——调用方在 save 里看取消信号，用户按停止后不落盘。
func ConvertToExcel(src string, opt ExcelOptions, save func(prod string) error) (ExcelReceipt, error) {
	inputs, images, err := wpsExcelInputs(src)
	if err != nil {
		return ExcelReceipt{}, err
	}
	if !wpsPDFEnabled() {
		return ExcelReceipt{}, errors.New("本机 WPS 通道已关闭（GAEA_WPS_PDF=0），无法转 Excel")
	}
	if wpsSuspended() {
		return ExcelReceipt{}, errors.New("WPS 账号级失败后的静默期内（未登录 / 无会员 / 超配额），暂不重试")
	}
	cli := wpsCLILookup()
	if cli == "" {
		return ExcelReceipt{}, errors.New("本机未找到 WPS（wpscli）：转 Excel 需要 WPS Office（登录会员）")
	}
	dir, err := os.MkdirTemp("", "gaea-wpsexcel-")
	if err != nil {
		return ExcelReceipt{}, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	out := filepath.Join(dir, "out.xlsx")
	plan, err := wpsExcelPlan(src, inputs, images, opt, cli, out)
	if err != nil {
		return ExcelReceipt{}, err
	}
	start := time.Now()
	code, last, tail, err := wpsRunTimed(plan.timeout+wpsExcelGrace, cli, plan.args...)
	if err != nil {
		return ExcelReceipt{}, fmt.Errorf("wpscli %s 拉起失败: %w", plan.action, err)
	}
	if code != 0 {
		if code == 100 || code == 101 || code == 429 { // 账号级失败：进全局静默期
			wpsSuspend(wpsFailCooldown)
		}
		return ExcelReceipt{}, wpsError(plan.action, code, tail)
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
		return ExcelReceipt{}, errors.New("wpscli 报成功但没有产出 Excel（产物缺失或为空）")
	}
	rec, err := xlsxInspect(prod)
	if err != nil {
		return ExcelReceipt{}, err
	}
	rec.Pages, rec.OnlineOCR, rec.Elapsed = plan.pages, plan.online, time.Since(start)
	if save != nil {
		if err := save(prod); err != nil {
			return ExcelReceipt{}, err
		}
	}
	return rec, nil
}

// ── 输入 ─────────────────────────────────────────────────────────────

// wpsExcelInputs 列出要交给 wpscli 的输入文件，以及它是不是图片。目录按文件名自然序
// （IMG_2 排在 IMG_10 前面），只收图片。
func wpsExcelInputs(src string) (inputs []string, images bool, err error) {
	st, err := os.Stat(src)
	if err != nil {
		return nil, false, err
	}
	if !st.IsDir() {
		if !IsExcelSourceExt(filepath.Ext(src)) {
			return nil, false, fmt.Errorf("不支持的文件格式: %s（转 Excel 支持 .pdf 与图片 jpg/jpeg/png/bmp/gif/tif/tiff）",
				filepath.Ext(src))
		}
		return []string{src}, !strings.EqualFold(filepath.Ext(src), ".pdf"), nil
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return nil, false, err
	}
	for _, e := range entries {
		if !e.IsDir() && wpsExcelImageExts[strings.ToLower(filepath.Ext(e.Name()))] {
			inputs = append(inputs, filepath.Join(src, e.Name()))
		}
	}
	if len(inputs) == 0 {
		return nil, false, fmt.Errorf("目录里没有图片（支持 jpg/jpeg/png/bmp/gif/tif/tiff）: %s", src)
	}
	sort.SliceStable(inputs, func(i, j int) bool { return naturalLess(inputs[i], inputs[j]) })
	return inputs, true, nil
}

// naturalLess 字符串的自然序：相邻数字按数值比（忽略前导零），其余按字节序。
func naturalLess(a, b string) bool {
	ai, bi := 0, 0
	for ai < len(a) && bi < len(b) {
		ca, cb := a[ai], b[bi]
		if isASCIIDigit(ca) && isASCIIDigit(cb) {
			ea, eb := ai, bi
			for ea < len(a) && isASCIIDigit(a[ea]) {
				ea++
			}
			for eb < len(b) && isASCIIDigit(b[eb]) {
				eb++
			}
			na, nb := strings.TrimLeft(a[ai:ea], "0"), strings.TrimLeft(b[bi:eb], "0")
			if len(na) != len(nb) {
				return len(na) < len(nb)
			}
			if na != nb {
				return na < nb
			}
			ai, bi = ea, eb
			continue
		}
		if ca != cb {
			return ca < cb
		}
		ai++
		bi++
	}
	return len(a)-ai < len(b)-bi
}

func isASCIIDigit(c byte) bool { return c >= '0' && c <= '9' }

// ── 执行计划 ─────────────────────────────────────────────────────────

// wpsExcelCall 一次转换的执行计划。
type wpsExcelCall struct {
	action  string        // 子命令名（报错用）
	args    []string      // 交给 wpscli 的完整参数
	pages   int           // 页数 / 图片张数（读不到页数时为 0）
	online  bool          // 走 WPS 在线识别
	timeout time.Duration // 传给 wpscli 的 --timeout
}

// wpsExcelPlan 决定走哪条路线、带哪些参数。只读文件与调用 pdfinfo（本地引擎，不到
// 1 秒），不消耗会员额度——超限、越界这些错都在这里就报出去。
func wpsExcelPlan(src string, inputs []string, images bool, opt ExcelOptions, cli, out string) (wpsExcelCall, error) {
	switch opt.Scanned {
	case "", "auto", "true", "false":
	default:
		return wpsExcelCall{}, fmt.Errorf("scanned 取值无效（%q）：只收 auto / true / false", opt.Scanned)
	}
	var p wpsExcelCall
	if images {
		p.action, p.pages, p.online = "photo2excel", len(inputs), true
		p.args = append([]string{"photo2excel"}, inputs...)
		p.args = append(p.args, "-o", out, "--json")
		if len(inputs) > 1 {
			p.args = append(p.args, "--merge", "true") // 合成一个工作簿，一图一表
		}
		p.timeout = wpsExcelTimeout(len(inputs), false, false)
	} else {
		total, scan := wpsExcelPageInfo(cli, src)
		spans := pageSpans(opt.Pages, total)
		if len(spans) == 0 {
			return wpsExcelCall{}, fmt.Errorf("页范围 %q 超出这份 PDF 的页数（共 %d 页）", opt.Pages, total)
		}
		if len(spans) > 1 {
			return wpsExcelCall{}, fmt.Errorf("转 Excel 的 pages 只收一段连续页（如 1-20），收到 %q", opt.Pages)
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
			return wpsExcelCall{}, errors.New("读不到 PDF 页数，无法指定 WPS 在线识别的页范围")
		}
		if last > 0 {
			p.pages = last - first + 1
		}
		limit, route := wpsExcelMaxLocalPages, "文字型转 Excel"
		if online {
			route = "WPS 在线识别"
			limit = wpsExcelMaxScanPages
			if opt.AIFix {
				route, limit = "WPS 在线识别（AI 修复）", wpsExcelMaxScanPagesAI
			}
		}
		if p.pages > limit {
			return wpsExcelCall{}, fmt.Errorf("%s一次最多转 %d 页，这次是 %d 页：请用 pages 分段转换（如 1-%d）",
				route, limit, p.pages, limit)
		}
		p.action, p.online = "pdf2excel", online
		p.args = []string{"pdf2excel", src, "-o", out, "--json"}
		if online {
			// 在线识别引擎要显式页范围；--ai-fix 只在 --scanned 时有效
			p.args = append(p.args, "--scanned", "true")
			if opt.AIFix {
				p.args = append(p.args, "--ai-fix", "true")
			}
		}
		if online || opt.Pages != "" {
			// 整本文字型不带 --range：不把页数读数写进去，免得读数有偏差时截掉末页
			p.args = append(p.args, "--range", wpsExcelRange(first, last))
		}
		if opt.PerPageSheet {
			p.args = append(p.args, "--multi-sheet", "true")
		}
		p.timeout = wpsExcelTimeout(p.pages, online, opt.AIFix)
	}
	p.args = append(p.args, "--timeout", strconv.Itoa(int(p.timeout/time.Second)))
	return p, nil
}

// wpsExcelPageInfo 用 pdfinfo 读 PDF 页数与 WPS 自己的扫描件判定（本地引擎）。读不到
// 时两个都返回零值。
func wpsExcelPageInfo(cli, path string) (pages int, scan bool) {
	code, last, _, err := wpsRun(cli, "pdfinfo", path, "--json")
	if err != nil || code != 0 {
		return 0, false
	}
	return last.PageCount, last.Scan
}

// wpsExcelRange wpscli 的 --range 写法（单页不带短横）。
func wpsExcelRange(first, last int) string {
	if first == last {
		return strconv.Itoa(first)
	}
	return strconv.Itoa(first) + "-" + strconv.Itoa(last)
}

// wpsExcelTimeout 这次调用的超时：按路线给每页留余量（实测 文字型 0.6 秒/页、在线识别
// 1.7 秒/页、AI 修复 14 秒/页），下限 wpsExcelMinTimeout。默认的 300 秒不够用——839 页
// 文字型实测 495 秒。
func wpsExcelTimeout(pages int, online, aiFix bool) time.Duration {
	perPage := 3 * time.Second
	if online {
		perPage = 6 * time.Second
		if aiFix {
			perPage = 30 * time.Second
		}
	}
	t := time.Duration(pages)*perPage + 2*time.Minute
	if t < wpsExcelMinTimeout {
		t = wpsExcelMinTimeout
	}
	return t
}

// pdfTextLayerReadable 自写提取在 [first,last] 里是否读得出可信的文字层（不乱码、不是
// 只有页码戳）。只读文件、不拉起任何引擎——to=xlsx 靠它决定走本地引擎还是 WPS 在线
// 识别。过半的页读得出就算有：扫描件常带一页数字排版的封面，不该因此把整本当文字型；
// 反过来文字型文档里夹两页图片，也不该为了它们把整本送去在线识别（那两页会转空，
// 要它们就传 scanned=true）。
//
// 页对象在压缩对象流里的 PDF（Word 2016 起导出的默认形态）分不清哪段文字属于哪一页，
// 只能整档判断。
func pdfTextLayerReadable(path string, first, last, total int) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	decoded := decodeFlateStreams(string(data))
	content := stripNonTextStreams(decoded)
	trust := func(s string) bool {
		if strings.TrimSpace(s) == "" || stampNoiseText(s) {
			return false
		}
		hard, soft := pdfTextGarbled(s)
		return !hard && !soft
	}
	texts := pdfPageTexts(content)
	if countObjStmPages(decoded) > 0 {
		return trust(strings.Join(texts, "\n"))
	}
	readable := 0
	for i, text := range texts {
		page := i + 1
		if page < first || last > 0 && page > last {
			continue
		}
		if trust(text) {
			readable++
		}
	}
	// 分母取真实页数：没有文字的页不会出现在 texts 里（末尾的页整个缺席）。
	pages := len(texts) // 页数未知时的分母
	if last > 0 && last < pages {
		pages = last
	}
	if total > 0 {
		pages = last - first + 1
	}
	return readable > 0 && readable*2 >= pages
}
