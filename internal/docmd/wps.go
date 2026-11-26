package docmd

// wps.go — PDF 文字层检出乱码时改走本机 WPS 转换（2026-11-24）。
//
// 为什么需要：pdf.go 的自写提取只按 UTF-16BE / Latin-1 猜测解码，没有字体映射这一层。
// Word / WPS 导出的中文 PDF 把文字存成「子集字体字形号 + 每个字体的 /ToUnicode 映射
// 表」，自写提取把字形号当码点硬解，整本输出成乱码脚本（2026-11-24 实弹：风险评估
// 报告 233 页，汉字只剩 1589 个，还带非法字节与 NUL；用户语料 86 份 PDF 里 57 份属
// 于这一类）。同一原因让页对象计数失真——页对象在压缩对象流里，233 页被数成 1 页。
//
// 做法：文字层检出乱码的 PDF 走本机 WPS Office 自带的命令行 wpscli（面向 AI agent 的
// 官方入口，`<安装目录>\clitool\wpscli.exe`）：
//
//	pdf2word → docx → markitdown（与 docx 转换同一条链，图片以 data URI 保留）→ Markdown
//	→ 合并跨页断表
//
// 同一份报告实测：文本层 2015 处连续汉字段 100% 逐字命中、165 张真表格、108 张图片、
// 42 秒；pdfinfo 本地读页数 0.5 秒。
//
// 口径与边界（2026-11-26 用户拍板「转换解析优先用 WPS」）：
//   - 提取不到文字的 PDF（扫描件 / 字形号文字层）先走 WPS 在线 OCR 引擎
//     （wpsScannedPDFMarkdown：--scanned true），WPS 不可用才落本地 OCR；
//   - 文字型 PDF：format_convert 也先经 WPS——表格是真表格、图片保留，自写提取只有
//     平铺文字；检索索引 / @引用 / 预览等快速通道仍只认本地提取，乱码的才上 WPS。
//   - 结果按「文件内容指纹 + 页范围 + 引擎版本」进进程内缓存，同一份只转一次。
//   - wpscli 的「成功」不可信：pdf2md 实测 4 次全部 1 秒返回 success、输出 0 字节。
//     这里一律核对产物（docx 存在且非空、Markdown 化后有文字），不达标即判失败。
//   - WPS 是云转换（需登录 + 会员，有每日配额）。用户 2026-11-24 明示可用
//     （「我是 WPS 会员，这些方案去云端没有任何风险」）。GAEA_WPS_PDF=0 可整条关掉；
//     GAEA_WPSCLI 可指定 wpscli 路径。
//   - 失败不缓存成功态，但同一份 10 分钟内不重试（防配额与进程反复拉起）。

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/gaea/proc"
)

const (
	// wpsPDFEngineVersion 缓存键的引擎版本：转换链行为变化（合并规则、docx→md 路径）时递增。
	// wps2：表格纯数字单元格里被 OCR 拆开的小数并回（rejoinSplitDecimals）。
	wpsPDFEngineVersion = "wps2"
	// wpsPDFTimeout 单次 wpscli 调用上限（233 页实测 42 秒；留足大文档与网络余量）。
	wpsPDFTimeout = 15 * time.Minute
	// wpsMarkItDownTimeout WPS 转出的 docx 走 markitdown 的上限（233 页、108 张图实测 32 秒）。
	wpsMarkItDownTimeout = 5 * time.Minute
	// wpsFailCooldown 同一份转换失败后的静默期。
	wpsFailCooldown = 10 * time.Minute
)

var (
	// wpsCLILookup 查找 seam（测试堵住路径，不得真调 WPS）。
	wpsCLILookup = findWpsCLI
	// wpsPDFMarkdownFn 乱码 PDF 的 WPS 转换 seam（pdf.go 调用；测试打桩）。
	wpsPDFMarkdownFn = wpsPDFMarkdown
	// wpsScannedPDFMarkdownFn 「提取不到文字」PDF 的 WPS 在线 OCR seam（同上）。
	wpsScannedPDFMarkdownFn = wpsScannedPDFMarkdown
	// wpsScannedConvertFn 在线 OCR 单段转换 seam（分段编排的单测打桩）。
	wpsScannedConvertFn = wpsConvertScannedRange
	wpsCLIMu            sync.Mutex // wpscli 一次只跑一个：WPS 自动化不支持并发
	wpsFailMu           sync.Mutex
	wpsFail             = map[string]time.Time{}
	// wpsSuspendedUntil 账号级失败（未登录 / 无会员 / 配额）后的全局静默期截止时刻：
	// 这类失败换哪份文件都一样，静默期内不再每份 PDF 都去撞一次。
	wpsSuspendMu      sync.Mutex
	wpsSuspendedUntil time.Time
)

// wpsPDFEnabled 本机 WPS 通道是否开启（默认开；GAEA_WPS_PDF=0 关）。
func wpsPDFEnabled() bool {
	return !strings.EqualFold(strings.TrimSpace(os.Getenv("GAEA_WPS_PDF")), "0")
}

// findWpsCLI 定位 WPS Office 自带的 wpscli：GAEA_WPSCLI → PATH → 各安装根下版本目录
// （取最高版本）。找不到返回空串，调用方据此走原有兜底。
func findWpsCLI() string {
	if p := strings.TrimSpace(os.Getenv("GAEA_WPSCLI")); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	exe := "wpscli"
	if runtime.GOOS == "windows" {
		exe = "wpscli.exe"
	}
	if p, err := exec.LookPath(exe); err == nil {
		return p
	}
	roots := []string{
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Kingsoft", "WPS Office"),
		filepath.Join(os.Getenv("ProgramFiles"), "Kingsoft", "WPS Office"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Kingsoft", "WPS Office"),
	}
	var best string
	for _, root := range roots {
		if root == "" {
			continue
		}
		matches, _ := filepath.Glob(filepath.Join(root, "*", "clitool", exe))
		sort.Strings(matches)
		for _, m := range matches {
			if wpsVersionAtLeast(m, best) {
				best = m
			}
		}
	}
	return best
}

// wpsVersionAtLeast 比较 …\WPS Office\<版本>\clitool\wpscli.exe 路径里的版本号。
func wpsVersionAtLeast(a, b string) bool {
	if b == "" {
		return true
	}
	va, vb := wpsVersionOf(a), wpsVersionOf(b)
	for i := 0; i < len(va) || i < len(vb); i++ {
		var x, y int
		if i < len(va) {
			x = va[i]
		}
		if i < len(vb) {
			y = vb[i]
		}
		if x != y {
			return x > y
		}
	}
	return true
}

func wpsVersionOf(path string) []int {
	// …/WPS Office/<version>/clitool/wpscli.exe
	parts := strings.Split(filepath.ToSlash(path), "/")
	if len(parts) < 3 {
		return nil
	}
	ver := parts[len(parts)-3]
	var out []int
	for _, seg := range strings.Split(ver, ".") {
		n, err := strconv.Atoi(seg)
		if err != nil {
			return out
		}
		out = append(out, n)
	}
	return out
}

// ── 文字层乱码判定 ───────────────────────────────────────────────────

// pdfTextGarbled 判断自写提取出的文字层是否已是乱码。
//
// hard（确定乱码，不能当正文返回）：非法 UTF-8 占比高，或控制字符（不含换行/制表）
// 占比高——正常文档的文字层不会这样，只有字形号被硬解才会。
// soft（疑似乱码，优先换引擎）：非预期文字区的字符占比高——字形号被当码点后，字符
// 近似均匀撒满整个基本平面，汉字只占三成、其余散落在几十种文字里；正常中文/英文
// 文档几乎全部落在常用区。泰文、阿拉伯文等小语种文档会落入 soft：有 WPS 就换 WPS
// （结果只会更好），没有 WPS 时原样返回，不比现在差。
func pdfTextGarbled(s string) (hard, soft bool) {
	invalid, ctrl, total, unexpected := 0, 0, 0, 0
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			invalid++
			i++
			continue
		}
		i += size
		if unicode.IsSpace(r) {
			continue
		}
		total++
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			ctrl++
			continue
		}
		if !pdfTextExpectedRune(r) {
			unexpected++
		}
	}
	if len(s) > 0 && float64(invalid)/float64(len(s)) > 0.005 {
		hard = true
	}
	if total > 0 {
		if float64(ctrl)/float64(total) > 0.01 {
			hard = true
		}
		if float64(unexpected)/float64(total) > 0.10 {
			soft = true
		}
	}
	return hard, soft
}

// pdfTextExpectedRune 正常文档文字层会落到的字符区。
func pdfTextExpectedRune(r rune) bool {
	switch {
	case r >= 0x20 && r < 0x7f: // ASCII
		return true
	case r >= 0xa0 && r <= 0x24f: // Latin-1、拉丁扩展（声调、越南语）
		return true
	case r >= 0x300 && r <= 0x36f: // 组合附加符号
		return true
	case r >= 0x370 && r <= 0x52f: // 希腊、西里尔
		return true
	case r >= 0x2000 && r <= 0x27bf: // 常用标点、货币、数学、箭头、技术符号、几何图形
		return true
	case r >= 0x3000 && r <= 0x30ff: // CJK 符号与标点、假名
		return true
	case r >= 0x3400 && r <= 0x4dbf: // CJK 扩展 A
		return true
	case r >= 0x4e00 && r <= 0x9fff: // CJK 统一汉字
		return true
	case r >= 0xf900 && r <= 0xfaff: // CJK 兼容汉字
		return true
	case r >= 0xfe30 && r <= 0xfe4f: // CJK 兼容形式
		return true
	case r >= 0xff00 && r <= 0xffef: // 全角、半角形式
		return true
	}
	return false
}

// ── 压缩对象流里的页对象 ─────────────────────────────────────────────

// countObjStmPages 数压缩对象流（字典含 /ObjStm 的 stream）里的页对象。入参是
// decodeFlateStreams 之后的视图：对象流已原位解压，页对象以明文出现在流体内。
// 只看对象流——正文文字流里出现 /Type /Page 字样不算页。
func countObjStmPages(decoded string) int {
	n := 0
	sc := newStreamScanner(decoded, isPDFSpace)
	for {
		span, ok := sc.next()
		if !ok {
			break
		}
		if dictStart := findDictStart(decoded, span.dictEnd); dictStart >= 0 &&
			strings.Contains(decoded[dictStart:span.dictEnd], "/ObjStm") {
			n += countPDFPages(decoded[span.bodyStart:span.end])
		}
	}
	return n
}

// ── wpscli 调用 ─────────────────────────────────────────────────────

// wpsCompleted wpscli --json 输出里的终态行。
type wpsCompleted struct {
	Type      string `json:"type"`
	Status    string `json:"status"`
	Output    string `json:"output"`
	PageCount int    `json:"page_count"`
	Scan      bool   `json:"is_scan_document"`
	ErrorMsg  string `json:"error_msg"`
}

// wpsRun 跑一次 wpscli，返回退出码与终态行（解析不出终态行时为零值）。
func wpsRun(cli string, args ...string) (int, wpsCompleted, string, error) {
	return wpsRunTimed(wpsPDFTimeout, cli, args...)
}

// wpsRunTimed 是 wpsRun 的可调上限形态（to=xlsx 按页数放宽，见 wps_excel.go）。limit
// 是这一次调用的上限，要比传给 wpscli 的 --timeout 留出余量：wpscli 自己到点会以
// 退出码 3 收尾，能翻译成「WPS 转换超时」；这里先到点只能杀进程。
func wpsRunTimed(limit time.Duration, cli string, args ...string) (int, wpsCompleted, string, error) {
	wpsCLIMu.Lock()
	defer wpsCLIMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	cmd := exec.CommandContext(ctx, cli, args...)
	proc.HideWindow(cmd) // Windows: 防止弹出黑框
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &outBuf, &errBuf
	runErr := cmd.Run()
	tail := strings.TrimSpace(errBuf.String() + "\n" + stripWpsProgress(outBuf.String()))
	if ctx.Err() != nil && runErr != nil {
		// gaea 侧上限先到：被杀的 wpscli 在 Windows 上是退出码 1，按码翻译会误报成
		// 「转换失败」。如实报成超时。
		action := ""
		if len(args) > 0 {
			action = args[0]
		}
		return -1, wpsCompleted{}, tail, fmt.Errorf("WPS 转换超过 %s 仍未结束，已终止（wpscli %s）: %w",
			limit.Round(time.Second), action, context.DeadlineExceeded)
	}
	code := 0
	if runErr != nil {
		var ee *exec.ExitError
		if errors.As(runErr, &ee) {
			code = ee.ExitCode()
		} else {
			return -1, wpsCompleted{}, tail, runErr
		}
	}
	var last wpsCompleted
	for _, ln := range strings.Split(outBuf.String(), "\n") {
		var c wpsCompleted
		if json.Unmarshal([]byte(strings.TrimSpace(ln)), &c) == nil && c.Type != "" {
			last = c
		}
	}
	return code, last, tail, nil
}

// stripWpsProgress 去掉 JSON 行，只留人读文本（进错误信息的那部分）。
func stripWpsProgress(s string) string {
	var keep []string
	for _, ln := range strings.Split(s, "\n") {
		t := strings.TrimSpace(ln)
		if t == "" || strings.HasPrefix(t, "{") {
			continue
		}
		keep = append(keep, t)
	}
	return strings.Join(keep, "\n")
}

// wpsError 把 wpscli 的失败翻译成可执行的原因（模型按文案路由恢复，不用猜）。
func wpsError(action string, code int, tail string) error {
	detail := truncRunes(strings.TrimSpace(tail), 200)
	switch code {
	case 100:
		return errors.New("WPS 未登录账号（wpscli 退出码 100；在 WPS 客户端登录后重试）")
	case 101:
		return errors.New("当前 WPS 账号没有会员权限（wpscli 退出码 101）")
	case 429:
		return errors.New("WPS 转换超出当日配额或过于频繁（wpscli 退出码 429）")
	case 3:
		return errors.New("WPS 转换超时（wpscli 退出码 3）")
	case 207:
		return errors.New("PDF 超过 200MB，WPS 拒绝转换（退出码 207）")
	case 209, 210:
		return errors.New("PDF 带打开密码，WPS 无法转换（退出码 209/210）")
	}
	if detail != "" {
		return fmt.Errorf("wpscli %s 失败（退出码 %d）：%s", action, code, detail)
	}
	return fmt.Errorf("wpscli %s 失败（退出码 %d）", action, code)
}

// wpsPageCount 用 pdfinfo 读 PDF 页数（WPS 本地引擎，0.5 秒）。读不到返回 0。
func wpsPageCount(cli, path string) int {
	if cli == "" {
		return 0
	}
	code, last, tail, err := wpsRun(cli, "pdfinfo", path, "--json")
	if err != nil || code != 0 {
		slog.Debug("docmd: wpscli pdfinfo 失败", "code", code, "detail", truncRunes(tail, 160))
		return 0
	}
	return last.PageCount
}

// wpsPDFMarkdown 用 WPS 把 PDF 的指定页范围转成 Markdown（带 data URI 图片）。
// effSpec 为空=整本；可含多段，逐段转换后拼接。total 是已知的总页数（0=未知），调用方
// 读过就传进来，不重复拉起 pdfinfo。cli 为空、页范围全部越界或任何一段失败都返回 error。
func wpsPDFMarkdown(cli, path, effSpec string, total int) (string, error) {
	if cli == "" {
		return "", errors.New("本机未找到 WPS 的 wpscli")
	}
	fp, err := ocrFileFingerprint(path)
	if err != nil {
		return "", err
	}
	spans := pageSpans(effSpec, total)
	if len(spans) == 0 {
		return "", fmt.Errorf("页范围 %q 超出这份 PDF 的页数（共 %d 页）", effSpec, total)
	}
	var parts []string
	for _, sp := range spans {
		key := "wpsmd:" + fp + ":" + wpsPDFEngineVersion + fmt.Sprintf(":%d-%d", sp[0], sp[1])
		if err := wpsRecentlyFailedError(key); err != nil {
			return "", err
		}
		md, err := ocrCached(key, func() (string, error) {
			return wpsConvertRange(cli, path, sp[0], sp[1])
		})
		if err != nil {
			wpsMarkFailed(key)
			return "", err
		}
		parts = append(parts, md)
	}
	return strings.Join(parts, "\n\n"), nil
}

func wpsRecentlyFailed(key string) (time.Time, bool) {
	wpsFailMu.Lock()
	defer wpsFailMu.Unlock()
	at, ok := wpsFail[key]
	if !ok {
		return time.Time{}, false
	}
	if time.Since(at) > wpsFailCooldown {
		delete(wpsFail, key)
		return time.Time{}, false
	}
	return at, true
}

func wpsMarkFailed(key string) {
	wpsFailMu.Lock()
	defer wpsFailMu.Unlock()
	wpsFail[key] = time.Now()
}

// wpsRecentlyFailedError 这一份刚失败过时返回给调用方的原因（静默期内不重试）。
func wpsRecentlyFailedError(key string) error {
	failedAt, ok := wpsRecentlyFailed(key)
	if !ok {
		return nil
	}
	return fmt.Errorf("这一份 %d 分钟前经 WPS 转换失败过（%s），暂不重试", int(wpsFailCooldown.Minutes()), failedAt.Format("15:04:05"))
}

// wpsSuspend 记一次账号级失败后的全局静默期。
func wpsSuspend(d time.Duration) {
	wpsSuspendMu.Lock()
	wpsSuspendedUntil = time.Now().Add(d)
	wpsSuspendMu.Unlock()
}

// wpsSuspended 是否仍在账号级失败的静默期内。
func wpsSuspended() bool {
	wpsSuspendMu.Lock()
	defer wpsSuspendMu.Unlock()
	return time.Now().Before(wpsSuspendedUntil)
}

// wpsScannedChunk 扫描件经在线 OCR 引擎的分段页数：--scanned true 必须带显式页范围，
// 整本按此分段串行转换（174 页实测一次 68 秒；分段让单次调用远离 wpscli 默认 300 秒
// 超时，失败也只损一段）。
const wpsScannedChunk = 100

// wpsScannedPDFMarkdown 用 WPS 在线 OCR 引擎转「提取不到文字的 PDF」。effSpec 为空=
// 整本；可含多段，每段再按 wpsScannedChunk 分段，逐段转换后拼接。total<=0 时先用
// pdfinfo 读页数（在线引擎要显式页范围，读不到即报错）。
func wpsScannedPDFMarkdown(cli, path, effSpec string, total int) (string, error) {
	if cli == "" {
		return "", errors.New("本机未找到 WPS 的 wpscli")
	}
	if total <= 0 {
		total = wpsPageCount(cli, path)
	}
	if total <= 0 {
		return "", errors.New("读不到 PDF 页数，无法指定在线 OCR 的页范围")
	}
	fp, err := ocrFileFingerprint(path)
	if err != nil {
		return "", err
	}
	spans := pageSpans(effSpec, total)
	if len(spans) == 0 {
		return "", fmt.Errorf("页范围 %q 超出这份 PDF 的页数（共 %d 页）", effSpec, total)
	}
	var parts []string
	for _, sp := range spans {
		last := sp[1]
		if last <= 0 { // pageSpans 的整本记法：到文件尾
			last = total
		}
		for first := sp[0]; first <= last; first += wpsScannedChunk {
			end := first + wpsScannedChunk - 1
			if end > last {
				end = last
			}
			key := "wpsscan:" + fp + ":" + wpsPDFEngineVersion + fmt.Sprintf(":%d-%d", first, end)
			if err := wpsRecentlyFailedError(key); err != nil {
				return "", err
			}
			md, err := ocrCached(key, func() (string, error) {
				return wpsScannedConvertFn(cli, path, first, end)
			})
			if err != nil {
				wpsMarkFailed(key)
				return "", fmt.Errorf("WPS 在线 OCR 失败: %w", err)
			}
			parts = append(parts, md)
		}
	}
	return strings.Join(parts, "\n\n"), nil
}

// pageSpans 把页规格拆成连续段 [(first,last)]；规格为空=整本（{1,0}，last=0 表示到
// 文件尾——不把页数写进 --range，免得页数读数有偏差时截掉末页）。total<=0 表示页数未知。
func pageSpans(spec string, total int) [][2]int {
	if strings.TrimSpace(spec) == "" {
		return [][2]int{{1, 0}}
	}
	var out [][2]int
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if f, l, ok := parsePageSpecBounds(part); ok {
			if total > 0 && l > total {
				l = total
			}
			if f >= 1 && f <= l {
				out = append(out, [2]int{f, l})
			}
		}
	}
	// 全部越界 → nil：调用方报错。不能回退成「整本」——那会把整本内容当成
	// 用户要的那几页返回。
	return out
}

// wpsConvertRange 单段的实际转换：pdf2word → docx → markitdown → 合并断表。
func wpsConvertRange(cli, path string, first, last int) (string, error) {
	return wpsConvert(cli, path, first, last, false)
}

// wpsConvertScannedRange 同上，但走 WPS 在线 OCR 引擎（扫描件 / 字形号文字层）。
func wpsConvertScannedRange(cli, path string, first, last int) (string, error) {
	if last < first {
		return "", fmt.Errorf("WPS 在线 OCR 需要显式页范围（收到 %d-%d）", first, last)
	}
	return wpsConvert(cli, path, first, last, true)
}

// wpsConvertArgs 组一次 pdf2word 的参数。非 scanned 整本（last=0）不带 --range——
// 不把页数读数写进去，免得读数有偏差时截掉末页；scanned 必须带显式页范围
// （在线 OCR 引擎的要求）。
func wpsConvertArgs(path, docx string, first, last int, scanned bool) []string {
	args := []string{"pdf2word", path, "-o", docx, "--json"}
	if scanned {
		args = append(args, "--scanned", "true")
	}
	if scanned || last > 0 {
		args = append(args, "--range", fmt.Sprintf("%d-%d", first, last))
	}
	return args
}

func wpsConvert(cli, path string, first, last int, scanned bool) (string, error) {
	dir, err := os.MkdirTemp("", "gaea-wpspdf-")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	docx := filepath.Join(dir, "wps.pdf2word.docx")
	code, _, tail, err := wpsRun(cli, wpsConvertArgs(path, docx, first, last, scanned)...)
	if err != nil {
		return "", fmt.Errorf("wpscli 拉起失败: %w", err)
	}
	if code != 0 {
		if code == 100 || code == 101 || code == 429 { // 账号级失败：进全局静默期
			wpsSuspend(wpsFailCooldown)
		}
		return "", wpsError("pdf2word", code, tail)
	}
	// 「成功」不可信：核对产物（pdf2md 实测会 success + 0 字节）。
	if st, statErr := os.Stat(docx); statErr != nil || st.Size() == 0 {
		return "", errors.New("wpscli pdf2word 报成功但没有产出 docx（产物缺失或为空）")
	}
	md, convErr := convertViaMarkItDownTimed(docx, wpsMarkItDownTimeout, "--keep-data-uris")
	if convErr != nil {
		if hasOfficeMedia(docx) {
			return "", fmt.Errorf("WPS 已转出 docx，但 Markdown 化失败（需要 markitdown）: %w", convErr)
		}
		if md, convErr = docxToMarkdown(docx); convErr != nil {
			return "", fmt.Errorf("WPS 已转出 docx，但 Markdown 化失败: %w", convErr)
		}
	}
	if strings.TrimSpace(stripDataImages(md)) == "" {
		return "", errors.New("WPS 转换结果没有文字（产物为空）")
	}
	return mergeSplitTables(rejoinSplitDecimals(md)), nil
}

// ── Markdown 后处理 ─────────────────────────────────────────────────

var dataImageRe = regexp.MustCompile(`!\[[^\]]*\]\(\s*data:image/[^)]*\)`)

// stripDataImages 去掉 data URI 图片（纯文本消费方用：检索索引、摘要不该吃 base64）。
func stripDataImages(md string) string {
	return dataImageRe.ReplaceAllString(md, "")
}

// StripDataImages 是 stripDataImages 的导出形态：工具面不落盘返回 Markdown 时用——
// 图片没法随文本交付，几 MB 的 base64 不能回给模型。
func StripDataImages(md string) string { return stripDataImages(md) }

// mergeSplitTables 合并跨页断表：相邻两张表（中间只有空行）列数相同、且后一张的表头
// 与前一张逐字相同（Word/WPS 导出 PDF 时续页重复表头），判为同一张表被页断开，去掉
// 后一张的表头与分隔行后并进前一张。
//
// 规则刻意从严：真实报告里相邻的两张不同表之间隔着表题；列数或表头对不上的（样本
// 63 处相邻里 61 处符合）一律不动，宁可留断表也不误并。
func mergeSplitTables(md string) string {
	lines := strings.Split(md, "\n")
	// 表块：以 | 开头的连续行；blocks[i] = {start, end}（含）。
	type block struct{ start, end int }
	var blocks []block
	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "|") {
			j := i
			for j < len(lines) && strings.HasPrefix(lines[j], "|") {
				j++
			}
			if j-i >= 3 { // 表头 + 分隔行 + 至少一行数据
				blocks = append(blocks, block{i, j - 1})
			}
			i = j
		}
	}
	if len(blocks) < 2 {
		return md
	}
	keep := make([]bool, len(lines))
	for i := range keep {
		keep[i] = true
	}
	for k := 0; k+1 < len(blocks); k++ {
		a, b := blocks[k], blocks[k+1]
		if mdTableCols(lines[a.start]) == 0 || mdTableCols(lines[a.start]) != mdTableCols(lines[b.start]) {
			continue
		}
		if strings.TrimSpace(lines[a.start]) != strings.TrimSpace(lines[b.start]) {
			continue
		}
		blank := true
		for i := a.end + 1; i < b.start; i++ {
			if strings.TrimSpace(lines[i]) != "" {
				blank = false
				break
			}
		}
		if !blank {
			continue
		}
		for i := a.end + 1; i < b.start; i++ {
			keep[i] = false // 两张表之间的空行一并去掉，否则合并后的表会被空行拆开
		}
		keep[b.start], keep[b.start+1] = false, false // 去掉续页的重复表头与分隔行
		blocks[k+1] = block{a.start, b.end}           // 并入前一张，支持三页以上的长表连续合并
	}
	var out []string
	for i, ln := range lines {
		if keep[i] {
			out = append(out, ln)
		}
	}
	return strings.Join(out, "\n")
}

// splitDecimalCellRe 表格单元格里被 OCR 拆开的小数。WPS 在线 OCR 实测把 3.1 写成
// 「3.」+空格+「1」（2026-11-26 实弹 174 页扫描件：2624 个单元格里 21 个，docx 原件
// 即如此），规则取数会读错。只认两种形态：整个单元格就是数字，或数字后面直接贴着
// 拉丁字母单位（115953. 12t）——带文字、加粗、中文单位的格子一律不动（「H/3, 1次」
// 这种并了就错），正文里的数字也不动。
var splitDecimalCellRe = regexp.MustCompile(`^(\d+(?:[.,]\d+)*[.,])[ \t\x{00a0}\x{3000}]+(\d{1,4})([A-Za-z\x{00b2}\x{00b3}\x{00b5}]+)?$`)

// rejoinSplitDecimals 把表格行里被拆开的小数并回去（只动纯数字单元格）。
func rejoinSplitDecimals(md string) string {
	lines := strings.Split(md, "\n")
	for i, ln := range lines {
		if !strings.HasPrefix(ln, "|") {
			continue
		}
		cells := splitMDTableRow(ln)
		changed := false
		for j := 1; j+1 < len(cells); j++ { // 首尾是管道外的空串
			if m := splitDecimalCellRe.FindStringSubmatch(strings.TrimSpace(cells[j])); m != nil {
				cells[j] = " " + m[1] + m[2] + m[3] + " "
				changed = true
			}
		}
		if changed {
			lines[i] = strings.Join(cells, "|")
		}
	}
	return strings.Join(lines, "\n")
}

// splitMDTableRow 按未转义的 | 切分表格行；返回的切片首尾是管道外的空串。
func splitMDTableRow(line string) []string {
	var cells []string
	start := 0
	for i := 0; i < len(line); i++ {
		if line[i] == '|' && (i == 0 || line[i-1] != '\\') {
			cells = append(cells, line[start:i])
			start = i + 1
		}
	}
	return append(cells, line[start:])
}

// mdTableCols 一行 Markdown 表格的列数（反斜杠转义的 \| 不算分隔符）。不是表格行返回 0。
func mdTableCols(line string) int {
	if !strings.HasPrefix(line, "|") {
		return 0
	}
	cols := 0
	for i := 0; i < len(line); i++ {
		if line[i] == '|' && (i == 0 || line[i-1] != '\\') {
			cols++
		}
	}
	if cols < 2 {
		return 0
	}
	return cols - 1
}

// Outline 文档结构计数（转换回执用：让调用方一眼判断产物可用）。
type Outline struct {
	Headings int // 标题行（# ~ ######）
	Tables   int // 表格（按分隔行计）
	Images   int // 图片引用（data URI 或路径）
}

// CountStructure 统计 Markdown 的标题 / 表格 / 图片数。
func CountStructure(md string) Outline {
	var o Outline
	for _, ln := range strings.Split(md, "\n") {
		if mdHeadingLevel(ln) > 0 {
			o.Headings++
		}
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, "|") && isMDTableSeparator(t) {
			o.Tables++
		}
	}
	o.Images = strings.Count(md, "![")
	return o
}

// mdHeadingLevel 行首 ATX 标题的级别（1~6）；不是标题行（含缩进代码块里的 #）返回 0。
func mdHeadingLevel(line string) int {
	n := 0
	for n < len(line) && line[n] == '#' {
		n++
	}
	if n >= 1 && n <= 6 && n < len(line) && (line[n] == ' ' || line[n] == '\t') {
		return n
	}
	return 0
}

// mdTableSeparatorRe Markdown 表格的分隔行（| --- | :---: |）。每格至少三个短横：
// 单元格内容是「-」的数据行不算（2026-11-24 实弹：把 104 张表数成了 207）。
var mdTableSeparatorRe = regexp.MustCompile(`^\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)*\|?$`)

// isMDTableSeparator 判定表格分隔行。
func isMDTableSeparator(t string) bool {
	return mdTableSeparatorRe.MatchString(t)
}

func truncRunes(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}
