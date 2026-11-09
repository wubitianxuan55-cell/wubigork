package app

// ── 原罪附件引用解析（@路径 → 本轮可读内容块）──
//
// Composer 提交时把附件统一注入为 `@绝对路径` 文本；办公线的 @引用由
// control.Controller.ResolveRefs 解析（含 MCP 资源/工作区面）。原罪按硬隔离
// 不接 Controller（sin 设计档 §8），本文件做**原罪域内的最小同构**：
//
//   - 只认「真实存在的本地文件」——正文里的 @字样（@角色、邮箱、提及）只要
//     不是存在的路径就原样保留，零误伤；
//   - 文本文件注入头部（64KB 封顶，与办公 maxFileRefBytes 同档）；二进制
//     （头 8KiB 含 NUL）只提示不倾倒；目录如实说明不注入；
//   - 图片走识图（vision.RecognizeImage，包级配置），失败回退占位——与办公
//     appendImageBlock 同语义；
//   - 引用块只进**本轮提示**，不落库：消息仍存 @路径 原文（与办公同口径，
//     历史不膨胀）。
//
// 实现独立于 control（控制面零耦合）；语义漂移靠两边各自的测试锚定。

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/docmd"
	"github.com/gaea/gaea/internal/gaea/vision"
)

const (
	// sinFileRefMaxBytes 单个附件注入头部上限（与办公 maxFileRefBytes 同档：
	// "@somehuge.log" 不能撑爆上下文）。
	sinFileRefMaxBytes = 64 * 1024
	// sinFileRefMaxVisionBytes 识图文本注入上限。
	sinFileRefMaxVisionBytes = 1200
	// sinFileRefMaxOCRBytes 全文 OCR 文本注入上限；超限部分让 AI 用 ocr 工具补读
	// （与办公 maxVisionOCRBytes 同档）。
	sinFileRefMaxOCRBytes = 8 * 1024
	// sinFileRefBinaryScan 二进制判定扫描窗口：头 8KiB 含 NUL 即按二进制处理。
	sinFileRefBinaryScan = 8 * 1024
)

// sinRefTokenRe @引用 token：@ + 非空白串，或 @"带引号路径"（含空格的
// Windows 路径——Composer 对含空格路径注入 @"..." 形式；裸 @token 在第一个
// 空格处截断，历史实现在这类路径上静默失效且不报错）。
var sinRefTokenRe = regexp.MustCompile(`@"([^"]+)"|@([^\s]+)`)

// sinVisionRecognize 识图入口（可注入以便测试）。
var sinVisionRecognize = vision.RecognizeImage

// sinOCRImageText 全文 OCR 入口（可注入以便测试）。与识图腿并联：识图管
// "看懂图"，OCR 腿走 docmd 本地链（auto = uocr2→ovis→tesseract）管图中文字
// 全文（与办公线 ocrImageText 同语义，2026-11-09 同构双注入）。测试必须注入
// 桩，否则会真拉起本地 uocr2 服务（测试语义管线地雷形态）。
var sinOCRImageText = docmd.OCRImageText

// sinVisionOCRBudget 全文 OCR 腿等待预算：引擎冷启动可达分钟级，超预算放弃
// 本轮注入（附提示行），后台拉起不受影响（与办公线 visionOCRBudget 同档；
// 硬隔离两线各自锚定，语义漂移靠两边测试）。包级变量以便测试收缩。
var sinVisionOCRBudget = 15 * time.Second

// sinVisionConcurrency 图片附件识图的并发上限：识图是分钟级慢操作（每张至多
// 90s），串行时 5 张图 = 7.5 分钟首帧前干等；并发 2 既压缩等待又不至于把本地
// 视觉模型打满（与 CPU/GPU 占用的折中，进度提示照旧）。
const sinVisionConcurrency = 2

// sinFileRefBlock 解析 line 里的 @文件引用，返回拼装好的内容块与逐条失败原因。
// 块为空 = 没有可解析的引用。 errs 非空的条目也已如实写入块内（模型看得见
// 失败原因），调用方无需重复上报。图片附件按 sinVisionConcurrency 有界并发
// 识图，块序保持 token 出现序。
func sinFileRefBlock(ctx context.Context, line string) (string, []string) {
	toks := sinRefTokens(line)
	if len(toks) == 0 {
		return "", nil
	}
	type refItem struct {
		block string // 空串 = 该 token 不是引用（stat 不中，零误伤跳过）
		err   string
	}
	items := make([]refItem, len(toks))
	var (
		wg   sync.WaitGroup
		sem  = make(chan struct{}, sinVisionConcurrency)
		b    strings.Builder
		errs []string
	)
	for i, tok := range toks {
		path := filepath.FromSlash(tok)
		info, err := os.Stat(path)
		if err != nil {
			continue // 不存在的路径不是引用（@提及/正文 @字样零误伤，与办公同口径）
		}
		if info.IsDir() {
			items[i] = refItem{
				block: sinRefBlockText("dir", `path="`+path+`"`, "目录不注入内容。"),
				err:   "@" + tok + " — 目录不注入附件内容（请引用具体文件）",
			}
			continue
		}
		if sinIsImagePath(path) {
			wg.Add(1)
			sem <- struct{}{}
			go func(i int, path, tok string) {
				defer wg.Done()
				defer func() { <-sem }()
				items[i] = refItem{block: sinImageBlock(ctx, path, tok)}
			}(i, path, tok)
			continue
		}
		content, err := sinReadFileRef(path)
		if err != nil {
			items[i] = refItem{
				block: sinRefBlockText("file", `path="`+path+`"`, "[读取失败："+err.Error()+"]"),
				err:   "@" + tok + " — " + err.Error(),
			}
			continue
		}
		items[i] = refItem{block: sinRefBlockText("file", `path="`+path+`"`, content)}
	}
	wg.Wait()
	for _, it := range items {
		if it.block == "" && it.err == "" {
			continue
		}
		if it.err != "" {
			errs = append(errs, it.err)
		}
		sinAppendRefText(&b, it.block)
	}
	return b.String(), errs
}

// sinRefTokens 提取去重、去尾标的 @token（与办公 parseRefTokens 同语义；
// 引号形式取引号内原文，不做尾标剔除——引号本身就是边界）。
func sinRefTokens(line string) []string {
	var toks []string
	seen := map[string]bool{}
	for _, g := range sinRefTokenRe.FindAllStringSubmatch(line, -1) {
		quoted, bare := g[1], g[2]
		if quoted != "" {
			if !seen[quoted] {
				seen[quoted] = true
				toks = append(toks, quoted)
			}
			continue
		}
		t := strings.TrimRight(bare, ".,;!?)]}")
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		toks = append(toks, t)
	}
	return toks
}

// sinReadFileRef 读附件头部：截断封顶 + 二进制判定（头 8KiB 含 NUL）。
func sinReadFileRef(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	buf := make([]byte, sinFileRefMaxBytes+1)
	// io.ReadFull 读满：单次 f.Read 允许短读（历史实现大文件只注入半截且无
	// 截断标记）；EOF 且零字节 = 空文件，按空内容处理（不当读取失败报）。
	n, err := io.ReadFull(f, buf)
	if err != nil && n == 0 {
		if errors.Is(err, io.EOF) {
			return "", nil
		}
		return "", err
	}
	content := sinTrimPartialUTF8(buf[:n])
	head := buf[:n]
	if len(head) > sinFileRefBinaryScan {
		head = head[:sinFileRefBinaryScan]
	}
	if strings.IndexByte(string(head), 0) >= 0 {
		return "[二进制文件，不注入内容]", nil
	}
	if n > sinFileRefMaxBytes {
		content = append(content, []byte("\n\n[已截断：仅注入前 64KB]")...)
	}
	return string(content), nil
}

// sinTrimPartialUTF8 把字节切片尾部不完整的多字节字符削掉（截断在字节边界
// 时防止半个 UTF-8 序列变成乱码尾巴；完整序列原样保留）。
func sinTrimPartialUTF8(b []byte) []byte {
	for i := 0; i < 3 && len(b) > 0; i++ {
		last := b[len(b)-1]
		if last < 0x80 {
			break // ASCII 尾字节：完整
		}
		// 从尾往前找本字符的首字节，核对期望长度
		start := len(b) - 1
		for start > 0 && b[start]&0xC0 == 0x80 {
			start--
		}
		if first := b[start]; first&0xC0 == 0xC0 {
			expect := 0
			switch {
			case first&0xE0 == 0xC0:
				expect = 2
			case first&0xF0 == 0xE0:
				expect = 3
			case first&0xF8 == 0xF0:
				expect = 4
			default:
				return b[:start] // 非法首字节：削掉
			}
			if len(b)-start < expect {
				b = b[:start]
				continue
			}
		}
		break
	}
	return b
}

// sinImageRefCount 统计 line 里形如图片扩展名的 @引用数（识图等待提示用；
// 只按扩展名计，存在性判定仍由 sinFileRefBlock 逐个做）。
func sinImageRefCount(line string) int {
	n := 0
	for _, tok := range sinRefTokens(line) {
		if sinIsImagePath(tok) {
			n++
		}
	}
	return n
}

// sinIsImagePath 按扩展名判断图片文件（与办公 isImagePath 同表）。
func sinIsImagePath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".tiff", ".tif":
		return true
	}
	return false
}

// sinImageBlock 识图 + 全文 OCR 双腿并联注入块（与办公 appendImageBlock 同
// 语义）：识图失败/空结果回退占位（图片引用始终可进上下文），OCR 腿失败/
// 超时/空结果静默降级；识图腿倒了但 OCR 腿活着时占位块照样带全文。
func sinImageBlock(ctx context.Context, path, raw string) string {
	ocrCh := make(chan sinOCRResult, 1)
	go func() {
		text, ocrErr := sinOCRImageText(path)
		ocrCh <- sinOCRResult{text: text, err: ocrErr}
	}()

	desc, err := sinVisionRecognize(ctx, path, "")
	ocr := <-sinWaitOCR(ocrCh)

	var body string
	switch {
	case err != nil:
		body = "[图片附件识图失败：" + err.Error() + "]"
	case strings.TrimSpace(desc) == "":
		body = "[图片附件 @" + raw + " 已就位；如需视觉理解请用识图工具]"
	default:
		body = "【画面理解】\n" + sinTruncateVision(desc)
	}
	if ocrBody := sinOCRBlockText(ocr); ocrBody != "" {
		body += "\n\n" + ocrBody
	}
	return sinRefBlockText("image", `path="`+path+`"`, body)
}

// sinTruncateVision 截断识图文本（按 rune，不劈多字节字符；识别文本是中文为
// 主的多字节内容，字节硬切劈字符）。
func sinTruncateVision(s string) string {
	if len(s) <= sinFileRefMaxVisionBytes {
		return s
	}
	r := []rune(s)
	return string(r[:sinFileRefMaxVisionBytes]) + "…[已截断]"
}

// sinOCRResult 全文 OCR 腿的产出。
type sinOCRResult struct {
	text     string
	err      error
	timedOut bool
}

// sinWaitOCR 等待 OCR 腿产出，最多等 sinVisionOCRBudget，超时返回 timedOut
// 结果（放弃的 goroutine 继续在后台拉起服务，就绪后下一轮秒回）。
func sinWaitOCR(ch <-chan sinOCRResult) <-chan sinOCRResult {
	out := make(chan sinOCRResult, 1)
	timer := time.NewTimer(sinVisionOCRBudget)
	defer timer.Stop()
	select {
	case r := <-ch:
		out <- r
	case <-timer.C:
		out <- sinOCRResult{timedOut: true}
	}
	return out
}

// sinOCRBlockText 把 OCR 腿产出整理成注入块（与办公 ocrBlockText 同语义，
// 硬隔离两线各自锚定）：空/噪声/失败不注入，超时给提示行，成功给全文
// （截到 sinFileRefMaxOCRBytes）。返回空串表示该腿没有可注入内容。
func sinOCRBlockText(r sinOCRResult) string {
	if r.timedOut {
		return "【全文 OCR】未就绪（引擎冷启动中），可调用 ocr 工具提取文字全文"
	}
	text := strings.TrimSpace(r.text)
	if r.err != nil || utf8.RuneCountInString(text) < 12 {
		return ""
	}
	if len(text) > sinFileRefMaxOCRBytes {
		text = string(sinTrimPartialUTF8([]byte(text)[:sinFileRefMaxOCRBytes])) + "…[已截断；全文可调用 ocr 工具提取]"
	}
	return "【全文 OCR】\n" + text
}

// sinRefBlockText 单个引用块文本（<tag attr>\nbody\n</tag>，与办公 appendRefBlock 同形）。
func sinRefBlockText(tag, attr, body string) string {
	return fmt.Sprintf("<%s %s>\n%s\n</%s>", tag, attr, body, tag)
}

// sinAppendRefText 块拼装（多块之间以空行分隔）。
func sinAppendRefText(b *strings.Builder, block string) {
	if b.Len() > 0 {
		b.WriteString("\n\n")
	}
	b.WriteString(block)
}
