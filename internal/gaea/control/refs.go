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
	"unicode/utf8"

	"github.com/gaea/gaea/internal/gaea/vision"
	"github.com/gaea/gaea/internal/office/docmd"
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

// maxVisionTextBytes 限制注入的识图文本长度，避免撑爆上下文。
const maxVisionTextBytes = 1200

// refKind distinguishes the two things an @reference can resolve to.
type refKind int

const (
	refResource refKind = iota // an MCP resource: @<server>:<uri>
	refFile                    // a local file or directory: @<path>
	refImage                   // a local image attachment: @.gaea/attachments/<file>
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

// HasRefs reports whether a line contains any resolvable @references, so a
// frontend can decide to resolve off its event loop only when needed.
func (c *Controller) HasRefs(line string) bool {
	return len(c.detectRefs(line)) > 0
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
		}
	}
	return b.String(), errs
}

// appendImageBlock 识别图片并把结果注入引用块；识别失败时回退占位提示。
func (c *Controller) appendImageBlock(ctx context.Context, b *strings.Builder, r ref) {
	desc, err := visionRecognize(ctx, r.path, "")
	if err != nil || strings.TrimSpace(desc) == "" {
		msg := "[image attachment available at @" + r.path + "; use an image/OCR/vision MCP tool if visual understanding is needed]"
		if err != nil {
			msg = "[图片附件 @" + r.path + " 识图失败：" + err.Error() + "]"
		}
		appendRefBlock(b, "image", `path="`+r.path+`"`, msg)
		return
	}
	appendRefBlock(b, "image", `path="`+r.path+`"`, "【图片识别】\n"+truncateVision(desc))
}

// isImagePath 按扩展名判断是否为图片文件（refFile 命中图片时也走识图）。
func isImagePath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".tiff", ".tif":
		return true
	}
	return false
}

// truncateVision 截断识图文本到最大长度。
func truncateVision(s string) string {
	if len(s) <= maxVisionTextBytes {
		return s
	}
	return s[:maxVisionTextBytes] + "…[已截断]"
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
	defer f.Close()

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
		// 而不是把 zip/PDF 二进制当 "not shown" 丢弃。
		if md, total, truncated, convErr := docmd.ConvertLimit(path, "", refOfficeMaxPages); convErr == nil && strings.TrimSpace(md) != "" {
			return officeRefBlock(md, path, info.Size(), total, truncated), false, nil
		}
		// 转换失败（缺依赖等）时退回通用二进制提示，保证引用不失效
	}
	if bytes.IndexByte(data[:min(n, 8192)], 0) >= 0 {
		return fmt.Sprintf("[binary file %s, %d bytes — not shown]", path, info.Size()), false, nil
	}
	if n > maxFileRefBytes {
		return string(data[:maxFileRefBytes]) + fmt.Sprintf("\n…[truncated; file is %d bytes]…\n[文件较大：可调用 summarize_file 获取全文摘要，或用 read_file 按 offset 分页读取]", info.Size()), false, nil
	}
	return string(data), false, nil
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
