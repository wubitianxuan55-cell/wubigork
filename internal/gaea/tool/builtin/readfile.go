// Package builtin provides Tianxuan's compile-time built-in tools. Each tool
// self-registers via init(); main blank-imports this package to wire them in.
package builtin

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/transform"

	fileenc "github.com/gaea/gaea/internal/gaea/fileutil/encoding"
	"github.com/gaea/gaea/internal/gaea/proc"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(readFile{}) }

// readFile reads a text file. workDir, when non-empty, is the directory a
// relative path is resolved against (see resolveIn); the zero value registered
// at init resolves against the process working directory.
type readFile struct{ workDir string }

const (
	readFileDefaultLimit    = 2000      // lines returned when limit is unset
	readFileBinaryPeek      = 8 * 1024  // bytes scanned for a NUL to flag binary
	readFileDetectSample    = 256 << 10 // bytes sampled for encoding detection
	readFileSectionMaxBytes = 64 << 20  // section 模式整载上限（防巨文件整读）
)

// reMDHeading Markdown 标题行（ATX 形态 `#..###### 标题文本`）。
var reMDHeading = regexp.MustCompile(`^(#{1,6})[ \t]+(.+?)[ \t]*$`)

// mdSectionSpan 一个节窗口：lines[start:end)（0-based，含命中标题行，end 为
// 下一个同级或更高级标题行或文件尾）。level 为命中标题的 # 数。
type mdSectionSpan struct {
	start int
	end   int
	level int
	title string
}

// findMDSection 按选择子找首个命中标题（对标题文本大小写不敏感子串匹配，
// 支持编号 "3.2" 或文本 "风险评估"），返回其节窗口。
func findMDSection(lines []string, sel string) (mdSectionSpan, bool) {
	low := strings.ToLower(strings.TrimSpace(sel))
	if low == "" {
		return mdSectionSpan{}, false
	}
	for i, ln := range lines {
		m := reMDHeading.FindStringSubmatch(ln)
		if m == nil {
			continue
		}
		if strings.Contains(strings.ToLower(m[2]), low) {
			lvl := len(m[1])
			end := len(lines)
			for j := i + 1; j < len(lines); j++ {
				if m2 := reMDHeading.FindStringSubmatch(lines[j]); m2 != nil && len(m2[1]) <= lvl {
					end = j
					break
				}
			}
			return mdSectionSpan{start: i, end: end, level: lvl, title: m[2]}, true
		}
	}
	return mdSectionSpan{}, false
}

// noSectionMatchError 未命中时的错误：回显全文标题清单（行号+级别+文本，
// 最多 80 行）——一次调用即拿到文档大纲，选节不再盲猜。
func noSectionMatchError(path, sel string, lines []string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "section %q 未命中 %s 的任何标题。标题清单（行号 级别 文本）：", sel, path)
	n := 0
	for i, ln := range lines {
		m := reMDHeading.FindStringSubmatch(ln)
		if m == nil {
			continue
		}
		if n >= 80 {
			fmt.Fprintf(&b, "\n  …（标题超 80 个，余略）")
			break
		}
		fmt.Fprintf(&b, "\n  %6d  %s  %s", i+1, m[1], m[2])
		n++
	}
	if n == 0 {
		b.WriteString("\n  （全文无 Markdown 标题——非结构化文档请用 offset/limit 翻页）")
	}
	return fmt.Errorf("%s", b.String())
}

func (readFile) Name() string { return "read_file" }

func (readFile) Description() string {
	return "Read a text file with optional line offset/limit. By default each line is prefixed with its absolute 1-based line number (e.g. `   42→...`). Use a non-negative offset for forward pagination, or a negative offset to start from the end (`offset=-30` returns lines from 30 lines before EOF); `limit` still caps returned lines. Set line_numbers=false for raw text. For Markdown files, pass `section` (heading number or text, e.g. \"3.2\" or \"风险评估\") to return that whole section structurally — no sampling, exact numbers — instead of paging; offset (non-negative) then pages within the section. When no heading matches, the error lists all headings with line numbers so one call reveals the document outline."
}

func (readFile) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"File path"},
  "offset":{"type":"integer","description":"0-based forward line offset (default 0). Negative values start that many lines before EOF, e.g. -30 reads the final 30-line window."},
  "limit":{"type":"integer","description":"Maximum lines to return (default 2000)","minimum":1},
  "line_numbers":{"type":"boolean","description":"Prefix each line with its 1-based line number (default true). Set false for raw text."},
  "section":{"type":"string","description":"Markdown heading selector: substring (case-insensitive) matched against heading text, e.g. \"3.2\" or \"风险评估\". Returns the whole section (heading line through just before the next same-or-higher level heading) with exact line numbers — for precise number-checking reads of long documents. Non-negative offset pages within the section. Non-Markdown or oversized files fall back with a clear error."}
},
"required":["path"]
}`)
}

func (readFile) ReadOnly() bool { return true }

func (r readFile) ReadTarget(args json.RawMessage) (string, bool) {
	var p struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &p); err != nil || strings.TrimSpace(p.Path) == "" {
		return "", false
	}
	return resolveIn(r.workDir, p.Path), true
}

func (readFile) CompactDescription() string     { return compactDesc["read_file"] }
func (readFile) CompactSchema() json.RawMessage { return compactSchema["read_file"] }

func (r readFile) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Path        string `json:"path"`
		Offset      int    `json:"offset,omitempty"`
		Limit       int    `json:"limit,omitempty"`
		LineNumbers *bool  `json:"line_numbers,omitempty"`
		Section     string `json:"section,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("read_file 参数解析失败: %w", err)
	}
	if p.Path == "" {
		return "", fmt.Errorf("path is required")
	}
	p.Path = resolveIn(r.workDir, p.Path)
	if p.Limit <= 0 {
		p.Limit = readFileDefaultLimit
	}
	const readFileMaxLimit = 10000
	if p.Limit > readFileMaxLimit {
		p.Limit = readFileMaxLimit
	}
	// V10.5: line_numbers defaults to true (backward-compatible)
	showLineNumbers := true
	if p.LineNumbers != nil {
		showLineNumbers = *p.LineNumbers
	}

	// A directory can be os.Open'd but not read as text — catch it up front with
	// an actionable message (and avoid the doubled "read X: read X:" the scanner's
	// error would otherwise produce) so the model switches to the ls tool.
	if info, err := os.Stat(p.Path); err == nil && info.IsDir() {
		return "", fmt.Errorf("%s is a directory, not a file — use the ls tool to list it, or read a specific file inside it", p.Path)
	}

	// B1 凭据围栏（dsh-ops 蒸馏）：凭据形态路径结构化拒绝（错误即值+自愈指引）。
	if rule, hit := credentialFenceRule(p.Path); hit {
		return "", credentialDenied("read_file", p.Path, rule)
	}

	// 会话件围栏（2026-11-16 实弹）：.gaea/sessions/ 子树结构化拒绝——
	// 历史会话走 @会话/session_search，生啃原始 jsonl 必撞 D1 输出预算。
	if SessionDirFenceRule(p.Path) {
		return "", SessionDirDenied("read_file", p.Path)
	}

	f, err := os.Open(p.Path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", p.Path, err)
	}
	defer func() { _ = f.Close() }()

	peek := make([]byte, readFileBinaryPeek)
	pn, perr := io.ReadFull(f, peek)
	peek = peek[:pn]
	peekEOF := perr != nil // whole file fit in the peek (EOF / ErrUnexpectedEOF)

	var src io.Reader

	// BOM check first: UTF-16 files contain 0x00 for every ASCII character, so a
	// naive NUL check would misidentify them as binary.
	if k := fileenc.DetectQuick(peek); k != fileenc.UTF8 {
		// UTF-16 is not self-synchronising — buffer it fully.
		rest, rerr := io.ReadAll(f)
		if rerr != nil {
			return "", fmt.Errorf("read %s: %w", p.Path, rerr)
		}
		src = bytes.NewReader(fileenc.Decode(append(peek, rest...), fileenc.DetectQuick(append(peek, rest...))))
	} else if k, ok := fileenc.DetectUTF16NoBOM(peek); ok {
		// BOM-less UTF-16 (Windows source files) — recognise by NUL pattern.
		rest, rerr := io.ReadAll(f)
		if rerr != nil {
			return "", fmt.Errorf("read %s: %w", p.Path, rerr)
		}
		src = bytes.NewReader(fileenc.Decode(append(peek, rest...), k))
	} else {
		// V5.9: 二进制文件 → 尝试用 markitdown 转为 Markdown
		if bytes.IndexByte(peek, 0) >= 0 {
			// 图片不在 markitdown 白名单里，直接给分工引导：
			// 提文字用 ocr（本地 uocr2 链），看懂图用 vision。
			if isImageExt(p.Path) {
				return "", fmt.Errorf(
					"image file %s; call the \"ocr\" tool to extract its text (tables, receipts, scanned pages), or the \"vision\" tool to understand the picture",
					p.Path,
				)
			}
			if markdown, ok := tryMarkItDownFn(ctx, p.Path); ok {
				return markdown, nil
			}
			// 扫描件 PDF（无文本层）：markitdown 转不出文本。不在此真跑 OCR
			// （分钟级与快速读工具形态不符）——指给 format_convert（内置扫描件
			// OCR 兜底，可 pages 分段）或 summarize_file；未装 markitdown 的
			// 文本 PDF 也顺带被引到正解入口。
			if strings.EqualFold(filepath.Ext(p.Path), ".pdf") {
				return "", fmt.Errorf(
					"binary file %s; text extraction failed — likely a scanned PDF (no text layer). Use the format_convert tool (auto-OCR for scanned PDFs, optional pages param) or summarize_file",
					p.Path,
				)
			}
			return "", fmt.Errorf(
				"binary file %s (NUL byte detected); install markitdown (pip install markitdown) to auto-convert PDF/Word/Excel/PPT to readable Markdown, or use `bash hexdump` for raw bytes",
				p.Path,
			)
		}

		// Read up to a bounded sample for encoding detection (GB18030/GBK).
		head := peek
		if !peekEOF {
			more := make([]byte, readFileDetectSample-len(peek))
			mn, _ := io.ReadFull(f, more)
			head = append(peek, more[:mn]...)
		}
		// 截断采样可能切在多字节 UTF-8 序列中间：utf8.Valid 对残尾判非，
		// Detect 级联随之误落 GB18030（宽松接受几乎任意字节）→ 整档按 GBK
		// 流式解码成系统性乱码（2026-11-20 实测：315KB 严格 UTF-8 中文文件
		// 256KB 边界切中汉字 → 「对象」变「瀵硅薄」，解码还「成功」——静默
		// 乱码比报错危险）。把采样尾回退到 rune 边界再判（最多 3 字节，只为
		// 判型，检测后的数据流始终从完整 head 起读，不受裁剪影响）。
		sample := head
		for i := 1; i <= 3 && i <= len(sample); i++ {
			if utf8.RuneStart(sample[len(sample)-i]) {
				if i > 1 {
					sample = sample[:len(sample)-i]
				}
				break
			}
		}
		enc, _ := fileenc.Detect(sample)
		switch enc {
		case fileenc.UTF8, fileenc.LossyUTF8:
			// Plain UTF-8 — stream directly.
			src = io.MultiReader(bytes.NewReader(head), f)
		case fileenc.GB18030:
			// GB18030/GBK — decode on the fly via streaming decoder.
			src = transform.NewReader(io.MultiReader(bytes.NewReader(head), f),
				fileenc.Decoder(enc))
		default:
			// Other encodings — full decode via Decoder.
			src = io.MultiReader(bytes.NewReader(head), f)
			if dec := fileenc.Decoder(enc); dec != nil {
				src = transform.NewReader(src, dec)
			}
		}
	}

	// 1 MB per-line cap matches what other scanners in this package allow —
	// well above any reasonable source line.
	scanner := bufio.NewScanner(src)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// Check for cancellation before potentially long scan
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	var collected []string
	startLine := p.Offset + 1
	hasMore := false
	sectionMode := false
	if p.Section != "" {
		// section 模式（实测反馈③）：长文档混合采样对「精确核对数字」的任务
		// 危险（采样片段会被误读为连续原文），此前用户被迫 awk/sed 分段精读
		// 十余次。按标题结构整节返回，行号精确，一次到位。
		if p.Offset < 0 {
			return "", fmt.Errorf("section 不能与负 offset 组合（负 offset 是文件尾部窗口语义）；section 定位节首后用非负 offset 在节内翻页")
		}
		if info, err := os.Stat(p.Path); err == nil && info.Size() > readFileSectionMaxBytes {
			return "", fmt.Errorf("section 只支持 ≤%d MB 的文件（%s 为 %d MB），请改用 offset/limit 翻页", readFileSectionMaxBytes>>20, p.Path, info.Size()>>20)
		}
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("read %s: %w", p.Path, err)
		}
		span, ok := findMDSection(lines, p.Section)
		if !ok {
			return "", noSectionMatchError(p.Path, p.Section, lines)
		}
		sec := lines[span.start:span.end] // 含命中标题行
		if p.Offset >= len(sec) {
			return fmt.Sprintf("(section %q has %d lines; offset %d is past its end)", p.Section, len(sec), p.Offset), nil
		}
		win := sec[p.Offset:]
		if len(win) > p.Limit {
			win = win[:p.Limit]
			hasMore = true
		}
		collected = win
		startLine = span.start + p.Offset + 1
		sectionMode = true
	} else if p.Offset < 0 {
		tailLines := -p.Offset
		if tailLines > readFileMaxLimit {
			return "", fmt.Errorf("negative offset magnitude %d exceeds the supported %d-line tail window; use a smaller negative offset or page forward with a non-negative offset", tailLines, readFileMaxLimit)
		}
		ring := make([]string, tailLines)
		total := 0
		for scanner.Scan() {
			ring[total%tailLines] = scanner.Text()
			total++
		}
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("read %s: %w", p.Path, err)
		}
		if total == 0 {
			return "(empty file)", nil
		}
		start := total - tailLines
		if start < 0 {
			start = 0
		}
		n := total - start
		if n > p.Limit {
			n = p.Limit
		}
		collected = make([]string, 0, n)
		for i := 0; i < n; i++ {
			collected = append(collected, ring[(start+i)%tailLines])
		}
		startLine = start + 1
		hasMore = start+n < total
	} else {
		// Scan up to offset+limit+1 lines (the extra tells us whether another
		// line remains without draining large files).
		upTo := p.Offset + p.Limit + 1
		lineNo := 0
		for scanner.Scan() {
			lineNo++
			if lineNo > p.Offset && len(collected) < p.Limit {
				collected = append(collected, scanner.Text())
			}
			if lineNo >= upTo {
				break
			}
		}
		hasMore = scanner.Scan()
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("read %s: %w", p.Path, err)
		}
		if lineNo == 0 && !hasMore {
			return "(empty file)", nil
		}
		if len(collected) == 0 {
			return fmt.Sprintf("(offset %d is past EOF — file has %d lines)", p.Offset, lineNo), nil
		}
	}

	var b strings.Builder

	if showLineNumbers {
		// Right-align line numbers to the largest one we'll print, so the arrow
		// "→" column lines up.
		maxShown := startLine + len(collected) - 1
		w := len(fmt.Sprint(maxShown))
		for i, line := range collected {
			fmt.Fprintf(&b, "%*d→%s\n", w, startLine+i, line)
		}
	} else {
		// V10.5: raw text mode — no line numbers, useful for copying to edit_file
		for _, line := range collected {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}

	if hasMore {
		if sectionMode {
			// section 模式的 offset 是节内相对量，与普通模式的绝对行号提示
			// 语义不同，分开说以免误导翻页。
			fmt.Fprintf(&b, "\n[more lines in section %q; pass offset=%d (relative to section start) to continue]\n",
				p.Section, p.Offset+len(collected))
		} else {
			fmt.Fprintf(&b, "\n[more lines available; pass offset=%d to continue]\n",
				startLine-1+len(collected))
		}
	}
	// A3 revision 尾标（dsh-ops 蒸馏）：模型经此取 expected_revision 做 CAS 写。
	// marker 恒最后追加（hasMore 提示行之前的内容在两种截断形态下都可存活）；
	// 取不到哈希（竞态删除）静默省略——模型不传 expected_revision 即旧行为。
	if rev, rerr := RevisionSHA256(p.Path); rerr == nil {
		fmt.Fprintf(&b, "\n[revision sha256:%s]\n", rev)
	}
	return b.String(), nil
}

// V5.9: markitdown 集成 —— 将二进制文档自动转为 Markdown
// 3.0 Step 3d #7：markitdown CLI→python 两级回退收敛为 MarkdownConverter
// 注册表 + config 选择（默认 kind=cli，行为不变）；切换文档转换后端只改配置。

// MarkdownConverterKindCLI markitdown CLI 后端（markitdown 可执行文件 →
// `python -m markitdown` 两级回退，与旧实现一致）。
const MarkdownConverterKindCLI = "cli"

// MarkdownConverter 文档→Markdown 转换能力接口。
type MarkdownConverter interface {
	// Convert 把 path 指向的二进制文档转为 Markdown；失败返回错误。
	Convert(ctx context.Context, path string) (string, error)
}

// MarkdownConverterConfig 是转换后端实例配置（注册表 New 入参）。
// 当前 kind 无参数；预留结构以便未来后端（如库内转换器）带参注册。
type MarkdownConverterConfig struct{}

// MarkdownConverterFactory 按实例配置构建转换后端（kind → 实例）。
type MarkdownConverterFactory func(cfg MarkdownConverterConfig) (MarkdownConverter, error)

// markdownConverterRegistry kind → 工厂注册表。各实现 init() 自注册；互斥注册，
// 重复即 panic（编译期接线错误）。
var markdownConverterRegistry = map[string]MarkdownConverterFactory{}

func init() {
	RegisterMarkdownConverter(MarkdownConverterKindCLI, func(cfg MarkdownConverterConfig) (MarkdownConverter, error) {
		return &markitdownCLIConverter{}, nil
	})
}

// RegisterMarkdownConverter 注册文档转换后端 kind（如 "cli"）。供各实现 init()
// 自注册；kind 为空或重复注册直接 panic。
func RegisterMarkdownConverter(kind string, factory MarkdownConverterFactory) {
	if kind == "" {
		panic("builtin: markdown converter kind must not be empty")
	}
	if _, dup := markdownConverterRegistry[kind]; dup {
		panic("builtin: duplicate markdown converter kind " + kind)
	}
	markdownConverterRegistry[kind] = factory
}

// NewMarkdownConverter 按 kind 经注册表构建转换后端；未知 kind 返回错误
// （fail-closed，附已注册 kind 列表）。
func NewMarkdownConverter(kind string, cfg MarkdownConverterConfig) (MarkdownConverter, error) {
	factory, ok := markdownConverterRegistry[kind]
	if !ok {
		return nil, fmt.Errorf("builtin: unknown markdown converter kind %q (registered: %v)", kind, MarkdownConverterKinds())
	}
	c, err := factory(cfg)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("builtin: markdown converter factory %q returned nil", kind)
	}
	return c, nil
}

// MarkdownConverterKinds 返回已注册转换后端 kind 列表（排序，供诊断/校验）。
func MarkdownConverterKinds() []string {
	out := make([]string, 0, len(markdownConverterRegistry))
	for k := range markdownConverterRegistry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// tryMarkItDownFn tryMarkItDown 的 seam（包级变量以便测试堵住 markitdown 路径：
// 扫描件 PDF 指引分支的机器无关覆盖——真 markitdown 装没装因机而异）。
var tryMarkItDownFn = tryMarkItDown

// ── 运行时配置注入 ─────────────────────────────────────────────

// MarkdownConverterRuntime 是文档转换后端的运行时配置，由 boot 从 gaea.toml
// 注入。Kind 为空 = 关闭转换（二进制文件走原有"提示安装 markitdown"错误路径）。
type MarkdownConverterRuntime struct {
	Kind string // 注册表 kind（默认 MarkdownConverterKindCLI）
}

// markdownConverterRuntime 保存 SetMarkdownConverterRuntime 注入的配置。
var markdownConverterRuntime MarkdownConverterRuntime

// SetMarkdownConverterRuntime 注入文档转换后端配置（boot 装配调用）。
// 切换转换后端只改配置（kind），消费方（read_file）代码零改动。
func SetMarkdownConverterRuntime(cfg MarkdownConverterRuntime) {
	markdownConverterRuntime = cfg
}

// markdownConverterKind 返回生效的转换后端 kind（空配置回落默认 cli）。
func markdownConverterKind() string {
	if markdownConverterRuntime.Kind != "" {
		return markdownConverterRuntime.Kind
	}
	return MarkdownConverterKindCLI
}

// tryMarkItDown 尝试用配置的文档转换后端把二进制文件转为 Markdown。
// 返回 (markdown, true) 表示成功，(_, false) 表示不可用或转换失败
// （调用方沿用旧错误提示路径）。
func tryMarkItDown(ctx context.Context, path string) (string, bool) {
	conv, err := NewMarkdownConverter(markdownConverterKind(), MarkdownConverterConfig{})
	if err != nil {
		return "", false
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	result, err := conv.Convert(ctx, path)
	if err != nil {
		return "", false
	}
	result = strings.TrimSpace(result)
	if result == "" {
		return "", false
	}
	return result, true
}

// markitdownCLIConverter 支持的文档扩展名列表。
//
// effect 声明（2026-11-08 T0 effect 审计，docs/gaea-t0-effect-audit-2026-11-08.md）：
// 对 pdf/docx 等二进制文档起 markitdown 子进程做「转 markdown 文本」的只读
// 转换。动作=子进程+stdout 捕获；位置=目标文件路径经 argv 直传（不经 shell，
// 无注入面）；无写面、无网络。子进程只读输入文件——与 ReadOnly() 一致，
// 机械 sweep 的「起子进程」标记经此声明豁免。
type markitdownCLIConverter struct{}

var markitdownExtensions = map[string]bool{
	".pdf": true, ".docx": true, ".xlsx": true, ".xls": true,
	".pptx": true, ".epub": true, ".html": true, ".htm": true,
	".csv": true, ".ipynb": true,
}

// isImageExt 按扩展名判断是否为图片文件（read_file 对图片不走 markitdown，
// 直接给 ocr/vision 分工引导）。
func isImageExt(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".tiff", ".tif":
		return true
	}
	return false
}

// Convert 优先用 markitdown CLI，未安装则回退到 `python -m markitdown`。
func (c *markitdownCLIConverter) Convert(ctx context.Context, path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if !markitdownExtensions[ext] {
		return "", fmt.Errorf("unsupported extension %q", ext)
	}
	var out []byte
	var runErr error
	if p, err := exec.LookPath("markitdown"); err == nil {
		cmd := exec.CommandContext(ctx, p, path)
		cmd.Env = proc.CommandEnv()
		hideBashWindow(cmd)
		out, runErr = cmd.Output()
	} else if p, err := exec.LookPath("python3"); err == nil {
		cmd := exec.CommandContext(ctx, p, "-m", "markitdown", path)
		cmd.Env = proc.CommandEnv()
		hideBashWindow(cmd)
		out, runErr = cmd.Output()
	} else if p, err := exec.LookPath("python"); err == nil {
		cmd := exec.CommandContext(ctx, p, "-m", "markitdown", path)
		cmd.Env = proc.CommandEnv()
		hideBashWindow(cmd)
		out, runErr = cmd.Output()
	} else {
		return "", fmt.Errorf("markitdown 未安装（pip install markitdown）")
	}
	if runErr != nil {
		return "", runErr
	}
	return string(out), nil
}
