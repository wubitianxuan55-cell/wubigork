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
	"sort"
	"strings"
	"time"

	"golang.org/x/text/transform"

	fileenc "github.com/gaea/gaea/internal/gaea/fileutil/encoding"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(readFile{}) }

// readFile reads a text file. workDir, when non-empty, is the directory a
// relative path is resolved against (see resolveIn); the zero value registered
// at init resolves against the process working directory.
type readFile struct{ workDir string }

const (
	readFileDefaultLimit = 2000      // lines returned when limit is unset
	readFileBinaryPeek   = 8 * 1024  // bytes scanned for a NUL to flag binary
	readFileDetectSample = 256 << 10 // bytes sampled for encoding detection
)

func (readFile) Name() string { return "read_file" }

func (readFile) Description() string {
	return "Read a text file with optional line offset/limit. By default each line is prefixed with its 1-based number (e.g. `   42→...`). Set line_numbers=false to get raw text — useful when copying content for edit_file. Use `offset` and `limit` to page through large files."
}

func (readFile) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"File path"},
  "offset":{"type":"integer","description":"0-based line offset to start reading from (default 0)","minimum":0},
  "limit":{"type":"integer","description":"Maximum lines to return (default 2000)","minimum":1},
  "line_numbers":{"type":"boolean","description":"Prefix each line with its 1-based line number (default true). Set false for raw text."}
},
"required":["path"]
}`)
}

func (readFile) ReadOnly() bool { return true }

func (readFile) CompactDescription() string     { return compactDesc["read_file"] }
func (readFile) CompactSchema() json.RawMessage { return compactSchema["read_file"] }

func (r readFile) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Path        string `json:"path"`
		Offset      int    `json:"offset,omitempty"`
		Limit       int    `json:"limit,omitempty"`
		LineNumbers *bool  `json:"line_numbers,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if p.Path == "" {
		return "", fmt.Errorf("path is required")
	}
	p.Path = resolveIn(r.workDir, p.Path)
	if p.Offset < 0 {
		p.Offset = 0
	}
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

	f, err := os.Open(p.Path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", p.Path, err)
	}
	defer f.Close()

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
			if markdown, ok := tryMarkItDown(p.Path); ok {
				return markdown, nil
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
		sample := head
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

	// Scan up to offset+limit+1 lines (the extra is just to know whether
	// trimming a trailer is warranted). 1 MB per-line cap matches what other
	// scanners in this package allow — well above any reasonable source line.
	scanner := bufio.NewScanner(src)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	upTo := p.Offset + p.Limit + 1

	// Check for cancellation before potentially long scan
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	var collected []string
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		if lineNo > p.Offset && len(collected) < p.Limit {
			collected = append(collected, scanner.Text())
		}
		if lineNo >= upTo {
			// Keep counting to know how many more lines remain.
			break
		}
	}
	// Quick check for remaining lines without draining the entire file.
	// (Avoids O(n) scan over huge files just to compute "more lines" count.)
	hasMore := scanner.Scan()
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read %s: %w", p.Path, err)
	}

	if lineNo == 0 && !hasMore {
		return "(empty file)", nil
	}
	if len(collected) == 0 {
		return fmt.Sprintf("(offset %d is past EOF — file has %d lines)", p.Offset, lineNo), nil
	}

	var b strings.Builder

	if showLineNumbers {
		// Right-align line numbers to the largest one we'll print, so the arrow
		// "→" column lines up. Add 1 for the 1-based display.
		maxShown := p.Offset + len(collected)
		w := len(fmt.Sprint(maxShown))
		for i, line := range collected {
			fmt.Fprintf(&b, "%*d→%s\n", w, p.Offset+i+1, line)
		}
	} else {
		// V10.5: raw text mode — no line numbers, useful for copying to edit_file
		for _, line := range collected {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}

	if hasMore {
		if showLineNumbers {
			fmt.Fprintf(&b, "\n[more lines available; pass offset=%d to continue]\n",
				p.Offset+len(collected))
		} else {
			fmt.Fprintf(&b, "\n[more lines available; pass offset=%d to continue]\n",
				p.Offset+len(collected))
		}
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
func tryMarkItDown(path string) (string, bool) {
	conv, err := NewMarkdownConverter(markdownConverterKind(), MarkdownConverterConfig{})
	if err != nil {
		return "", false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
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
type markitdownCLIConverter struct{}

var markitdownExtensions = map[string]bool{
	".pdf": true, ".docx": true, ".xlsx": true, ".xls": true,
	".pptx": true, ".epub": true, ".html": true, ".htm": true,
	".csv": true, ".ipynb": true,
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
		hideBashWindow(cmd)
		out, runErr = cmd.Output()
	} else if p, err := exec.LookPath("python3"); err == nil {
		cmd := exec.CommandContext(ctx, p, "-m", "markitdown", path)
		hideBashWindow(cmd)
		out, runErr = cmd.Output()
	} else if p, err := exec.LookPath("python"); err == nil {
		cmd := exec.CommandContext(ctx, p, "-m", "markitdown", path)
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
