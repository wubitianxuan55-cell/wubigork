package control

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/docmd"
	"github.com/gaea/gaea/internal/gaea/vision"
)

// maxFileRefBytes caps how much of an @-referenced file is injected into a
// message, so "@somehuge.log" can't blow the context window. The head is kept
// and the rest noted as truncated.
const maxFileRefBytes = 64 * 1024

// refOfficeMaxPages caps how many PDF pages an @-referenced office document
// injects (head-only; the rest is reachable via summarize_file / format_convert
// with a pages range). Kept small so @-ing a 1000-page PDF stays instant.
const refOfficeMaxPages = 20

// visionRecognize 图片识图入口（可注入以便测试）。识别失败时回退为
// 原有占位提示，保证 @图片 引用始终能进入上下文。
var visionRecognize = vision.RecognizeImage

// ocrImageText 全文 OCR 入口（可注入以便测试）。与识图腿并联：识图管"看懂图"，
// OCR 腿走 docmd 本地链（auto = uocr2→ovis→tesseract）管图中文字全文。
// 测试必须注入桩，否则会真拉起本地 uocr2 服务（同 internal/app 测试语义
// 管线地雷形态：单测静默连真实引擎）。
var ocrImageText = docmd.OCRImageText

// convertOfficeDoc 办公文档转换 seam（可注入以便测试）。docmd 内置扫描件
// OCR 兜底：文本层为空/只是页码戳噪声时自动 pdftoppm+本地 OCR 链。
var convertOfficeDoc = docmd.ConvertLimit

// officeConvBudget @引用办公文档转换预算：文本 PDF 毫秒级，扫描件 OCR 分钟级
// （GPU 6s/页、CPU 20s/页起），消息发送路径不能干等——超预算放弃本轮注入
// （附指引行），后台转换继续跑完不浪费引擎预热。包级变量以便测试收缩。
var officeConvBudget = 30 * time.Second

// maxVisionTextBytes 限制注入的识图文本长度，避免撑爆上下文。
const maxVisionTextBytes = 1200

// maxVisionOCRBytes 限制注入的全文 OCR 文本长度；超限部分让 AI 用 ocr 工具补读。
const maxVisionOCRBytes = 8 * 1024

// visionOCRBudget 全文 OCR 腿的等待预算：OCR 引擎按需拉起（冷启动可达分钟级），
// 超预算即放弃本轮注入（附提示行），后台拉起不受影响，服务就绪后下一轮秒回。
// 包级变量以便测试收缩。
var visionOCRBudget = 15 * time.Second

// refKind distinguishes the two things an @reference can resolve to.
type refKind int

const (
	refResource refKind = iota // an MCP resource: @<server>:<uri>
	refFile                    // a local file or directory: @<path>
	refImage                   // a local image attachment: @.gaea/attachments/<file>
	refSession                 // a cross-session reference: @会话:<关键词>（10.3）
)

// ref is a resolved @reference found in a submitted line.
type ref struct {
	kind   refKind
	server string // refResource
	uri    string // refResource
	path   string // refFile
	raw    string // the original token after '@', for labelling
}

// refTokenRe matches an @reference token: '@' then a run of non-space chars.
var refTokenRe = regexp.MustCompile(`@([^\s]+)`)

// parseRefTokens extracts the deduped, punctuation-trimmed tokens following '@'
// in a line. Pure: classification (server? file?) happens in classifyRef.
func parseRefTokens(line string) []string {
	var toks []string
	seen := map[string]bool{}
	for _, g := range refTokenRe.FindAllStringSubmatch(line, -1) {
		t := strings.TrimRight(g[1], ".,;!?)]}")
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		toks = append(toks, t)
	}
	return toks
}

// classifyRef decides what a token refers to. A "server:uri" token whose server
// is connected is an MCP resource; otherwise a token that names an existing path
// is a file. Anything else (an @mention, an email) is not a reference. exists is
// injected so the rule is testable without touching the filesystem.
func classifyRef(token string, known map[string]bool, exists func(string) bool) (ref, bool) {
	if i := strings.Index(token, ":"); i > 0 && i+1 < len(token) && known[token[:i]] {
		return ref{kind: refResource, server: token[:i], uri: token[i+1:], raw: token}, true
	}
	if strings.HasPrefix(filepath.ToSlash(token), ".gaea/attachments/") && exists(token) {
		return ref{kind: refImage, path: token, raw: token}, true
	}
	if looksLikeSessionRef(token) {
		return ref{kind: refSession, raw: token}, true
	}
	if exists(token) {
		return ref{kind: refFile, path: token, raw: token}, true
	}
	return ref{}, false
}

// detectRefs finds the @references in a line: MCP resources for connected
// servers, and local paths that exist on disk.
func (c *Controller) detectRefs(line string) []ref {
	known := map[string]bool{}
	if c.host != nil {
		for _, n := range c.host.ServerNames() {
			known[n] = true
		}
	}
	exists := func(p string) bool { _, err := os.Stat(p); return err == nil }

	var refs []ref
	for _, tok := range parseRefTokens(line) {
		if r, ok := classifyRef(tok, known, exists); ok {
			refs = append(refs, r)
		}
	}
	return refs
}

// ResolveRefs resolves the @references in a line into a single tagged context
// block (file/dir contents, MCP resource bodies), plus per-reference error
// strings for any that failed. An empty block means no references resolved.
// Safe to call off a frontend's event loop; honours ctx for the resource reads.
func (c *Controller) ResolveRefs(ctx context.Context, line string) (block string, errs []string) {
	var b strings.Builder
	for _, r := range c.detectRefs(line) {
		switch r.kind {
		case refResource:
			text, err := c.host.ReadResource(ctx, r.server, r.uri)
			if err != nil {
				errs = append(errs, "@"+r.raw+" — "+err.Error())
				continue
			}
			appendRefBlock(&b, "resource", `ref="@`+r.raw+`"`, text)
		case refFile:
			text, isDir, err := readFileRef(r.path)
			if err != nil {
				errs = append(errs, "@"+r.raw+" — "+err.Error())
				continue
			}
			if isDir {
				appendRefBlock(&b, "dir", `path="`+r.path+`"`, text)
				continue
			}
			if isImagePath(r.path) {
				c.appendImageBlock(ctx, &b, r)
				continue
			}
			appendRefBlock(&b, "file", `path="`+r.path+`"`, text)
		case refImage:
			c.appendImageBlock(ctx, &b, r)
		case refSession:
			text, errMsg := c.resolveSessionRef(r.raw)
			if errMsg != "" {
				errs = append(errs, "@"+r.raw+" — "+errMsg)
				continue
			}
			appendRefBlock(&b, "referenced-session", `ref="@`+r.raw+`"`, text)
		}
	}
	return b.String(), errs
}

// appendImageBlock 识别图片并把结果注入引用块。识图（看懂图）与全文 OCR
// （提取图中文字，本地 uocr2 链）两腿并联执行互不阻塞；OCR 腿只在预算内
// 等待，失败/超时/空结果静默降级，识图腿行为与历史一致。识图失败时仍回退
// 占位提示，保证 @图片 引用始终能进入上下文。
func (c *Controller) appendImageBlock(ctx context.Context, b *strings.Builder, r ref) {
	ocrCh := make(chan ocrResult, 1)
	go func() {
		text, ocrErr := ocrImageText(r.path)
		ocrCh <- ocrResult{text: text, err: ocrErr}
	}()

	desc, err := visionRecognize(ctx, r.path, "")
	// OCR 腿只等预算时长：引擎冷启动时不拖死本轮消息（放弃的 goroutine 会
	// 继续在后台拉起服务，就绪后下一轮秒回）。
	ocr := <-waitOCR(ocrCh)

	if err != nil || strings.TrimSpace(desc) == "" {
		msg := "[image attachment available at @" + r.path + "; use an image/OCR/vision MCP tool if visual understanding is needed]"
		if err != nil {
			msg = "[图片附件 @" + r.path + " 识图失败：" + err.Error() + "]"
		}
		// 识图腿倒了但 OCR 腿活着时，全文照样值钱——有就带上。
		if ocrBody := ocrBlockText(ocr); ocrBody != "" {
			msg += "\n\n" + ocrBody
		}
		appendRefBlock(b, "image", `path="`+r.path+`"`, msg)
		return
	}
	body := "【画面理解】\n" + truncateVision(desc)
	if ocrBody := ocrBlockText(ocr); ocrBody != "" {
		body += "\n\n" + ocrBody
	}
	appendRefBlock(b, "image", `path="`+r.path+`"`, body)
}

// ocrResult 全文 OCR 腿的产出。
type ocrResult struct {
	text     string
	err      error
	timedOut bool
}

// waitOCR 等待 OCR 腿产出，最多等 visionOCRBudget，超时返回 timedOut 结果。
func waitOCR(ch <-chan ocrResult) <-chan ocrResult {
	out := make(chan ocrResult, 1)
	timer := time.NewTimer(visionOCRBudget)
	defer timer.Stop()
	select {
	case r := <-ch:
		out <- r
	case <-timer.C:
		out <- ocrResult{timedOut: true}
	}
	return out
}

// ocrBlockText 把 OCR 腿产出整理成注入块：空/噪声/失败不注入，超时给提示行，
// 成功给全文（截到 maxVisionOCRBytes）。返回空串表示该腿没有可注入内容。
func ocrBlockText(r ocrResult) string {
	if r.timedOut {
		return "【全文 OCR】未就绪（引擎冷启动中），可调用 ocr 工具提取文字全文"
	}
	text := strings.TrimSpace(r.text)
	if r.err != nil || utf8.RuneCountInString(text) < 12 {
		return ""
	}
	if len(text) > maxVisionOCRBytes {
		text = truncateUTF8(text, maxVisionOCRBytes) + "…[已截断；全文可调用 ocr 工具提取]"
	}
	return "【全文 OCR】\n" + text
}

// isImagePath 按扩展名判断是否为图片文件（refFile 命中图片时也走识图）。
func isImagePath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".tiff", ".tif":
		return true
	}
	return false
}

// truncateVision 截断识图文本到最大长度（按 UTF-8 边界，不劈多字节字符）。
func truncateVision(s string) string {
	if len(s) <= maxVisionTextBytes {
		return s
	}
	return truncateUTF8(s, maxVisionTextBytes) + "…[已截断]"
}

func appendRefBlock(b *strings.Builder, tag, attr, body string) {
	if b.Len() > 0 {
		b.WriteString("\n\n")
	}
	fmt.Fprintf(b, "<%s %s>\n%s\n</%s>", tag, attr, body, tag)
}

// readFileRef reads an @-referenced path for injection. A directory yields a
// recursive listing (walked depth-first so the model sees the full tree); a
// binary file (NUL in the first 8 KiB) is noted rather than dumped; a large file
// is truncated to maxFileRefBytes with a marker. isDir lets the caller pick the
// wrapping tag. Common noise directories (.git, node_modules, .DS_Store) are
// skipped during the walk.
func readFileRef(path string) (content string, isDir bool, err error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", false, err
	}
	if info.IsDir() {
		var b strings.Builder
		err := filepath.WalkDir(path, func(p string, d os.DirEntry, wErr error) error {
			if wErr != nil {
				return wErr
			}
			// Skip the root itself — we only list its children.
			if p == path {
				return nil
			}
			name := d.Name()
			// Skip common noise directories.
			if d.IsDir() {
				switch name {
				case ".git", "node_modules", ".DS_Store", "__pycache__", ".idea", ".vscode":
					return filepath.SkipDir
				}
			}
			// Render the path relative to the referenced directory so the
			// listing is concise and unambiguous. Use forward slashes for
			// cross-platform consistency.
			rel, rErr := filepath.Rel(path, p)
			if rErr != nil {
				rel = p
			}
			rel = strings.ReplaceAll(rel, string(os.PathSeparator), "/")
			if d.IsDir() {
				rel += "/"
			}
			b.WriteString(rel)
			b.WriteByte('\n')
			return nil
		})
		if err != nil {
			return "", true, err
		}
		return b.String(), true, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, maxFileRefBytes+1)
	n, rerr := io.ReadFull(f, buf)
	if rerr != nil && rerr != io.ErrUnexpectedEOF && rerr != io.EOF {
		return "", false, rerr
	}
	data := buf[:n]

	if mime := imageMime(data, path); mime != "" {
		return fmt.Sprintf("[image file %s, mime=%s, %d bytes — image bytes are not inlined. Use an available MCP image/OCR/vision tool with this path when visual understanding is needed.]", path, mime, info.Size()), false, nil
	}
	if isOfficePath(path) {
		// 办公文档：转 Markdown 注入头部（PDF 限前 refOfficeMaxPages 页），
		// 而不是把 zip/PDF 二进制当 "not shown" 丢弃。转换带预算（扫描件
		// OCR 分钟级，超预算放弃本轮、后台继续）。
		md, total, truncated, convErr, timedOut := convertOfficeDocBudgeted(path, refOfficeMaxPages)
		if timedOut {
			return fmt.Sprintf("[office document %s, %d bytes — 转换超预算（扫描件 OCR 仍在后台进行）；可调用 format_convert（支持 pages 分段，扫描件自动 OCR）或 summarize_file 提取]", path, info.Size()), false, nil
		}
		if convErr == nil && strings.TrimSpace(md) != "" {
			return officeRefBlock(md, path, info.Size(), total, truncated), false, nil
		}
		reason := "text extraction failed"
		if convErr != nil {
			reason = convErr.Error() // 真因透传：pdftoppm 缺失/引擎不可用一眼可见
		}
		return fmt.Sprintf("[office document %s, %d bytes — text extraction failed (%s); use format_convert (scanned PDFs auto-OCR, pages param supported) or summarize_file]", path, info.Size(), reason), false, nil
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return fmt.Sprintf("[binary file %s, %d bytes — not shown]", path, info.Size()), false, nil
	}
	if n > maxFileRefBytes {
		return string(data[:maxFileRefBytes]) + fmt.Sprintf("\n…[truncated; file is %d bytes]…\n[文件较大：可调用 summarize_file 获取全文摘要，或用 read_file 按 offset 分页读取]", info.Size()), false, nil
	}
	return string(data), false, nil
}

// convertOfficeDocBudgeted 带预算的办公文档转换（同图片 OCR 腿的预算模式：
// goroutine + timer select，超时放弃的 goroutine 后台继续跑完）。返回末参
// timedOut=true 时前四个值无意义。
func convertOfficeDocBudgeted(path string, maxPages int) (md string, total int, truncated bool, err error, timedOut bool) {
	type convResult struct {
		md        string
		total     int
		truncated bool
		err       error
	}
	ch := make(chan convResult, 1)
	go func() {
		m, t, tr, e := convertOfficeDoc(path, "", maxPages)
		ch <- convResult{m, t, tr, e}
	}()
	timer := time.NewTimer(officeConvBudget)
	defer timer.Stop()
	select {
	case r := <-ch:
		return r.md, r.total, r.truncated, r.err, false
	case <-timer.C:
		return "", 0, false, nil, true
	}
}

// isOfficePath 判断是否为 docmd 可转换的办公文档。
func isOfficePath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".docx", ".doc", ".xls", ".xlsx", ".pdf":
		return true
	}
	return false
}

// officeRefBlock 把办公文档转换后的 Markdown 注入 @ 引用：截到注入上限并带明确
// 标记（总大小/页数/可用的深入读取方式），引导模型用 summarize_file 读全文。
func officeRefBlock(md, path string, size int64, totalPages int, truncated bool) string {
	md = truncateUTF8(md, maxFileRefBytes)
	var b strings.Builder
	b.WriteString(md)
	fmt.Fprintf(&b, "\n\n[office document %s, %d bytes", path, size)
	if totalPages > 0 {
		fmt.Fprintf(&b, ", %d 页", totalPages)
	}
	if truncated {
		fmt.Fprintf(&b, "，已注入前 %d 页", refOfficeMaxPages)
	}
	b.WriteString("；可调用 summarize_file 获取全文摘要，或 format_convert 转换指定页]")
	return b.String()
}

// truncateUTF8 按字节截断但不切断多字节 UTF-8 字符。
func truncateUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	i := maxBytes
	for i > 0 && !utf8.RuneStart(s[i]) {
		i--
	}
	if i == 0 {
		return s[:maxBytes]
	}
	return s[:i]
}

func imageMime(data []byte, path string) string {
	mime := http.DetectContentType(data[:min(len(data), 512)])
	if strings.HasPrefix(mime, "image/") {
		return mime
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".tiff", ".tif":
		return "image/tiff"
	}
	return ""
}
