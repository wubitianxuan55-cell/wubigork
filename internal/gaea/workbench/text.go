package workbench

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/gaea/fileutil"
	fileenc "github.com/gaea/gaea/internal/gaea/fileutil/encoding"
	"github.com/gaea/gaea/internal/gaea/wspath"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

const (
	// EditableBytes is the upper bound of a document that can be loaded into
	// the browser editor. Larger text opens in a paged read-only view.
	EditableBytes = 5 << 20
	// PageBytes is one paged-read response.
	PageBytes = 256 << 10
	// sampleBytes is enough to detect encoding and EOL without reading the file.
	sampleBytes = 64 << 10
)

// TextFile is a fully decoded editable document.
type TextFile struct {
	Path      string `json:"path"`
	Text      string `json:"text"`
	Size      int64  `json:"size"`
	MTimeMs   int64  `json:"mtimeMs"`
	Encoding  string `json:"encoding"`
	EOL       string `json:"eol"`
	TooLarge  bool   `json:"tooLarge,omitempty"`
	Binary    bool   `json:"binary,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
}

// TextPage is one byte-window read converted to UTF-8 and aligned to whole
// lines whenever the file has a line break inside the requested window.
type TextPage struct {
	Path       string `json:"path"`
	Text       string `json:"text"`
	Offset     int64  `json:"offset"`
	NextOffset int64  `json:"nextOffset"`
	Size       int64  `json:"size"`
	Encoding   string `json:"encoding"`
	EOL        string `json:"eol"`
	Binary     bool   `json:"binary,omitempty"`
	EOF        bool   `json:"eof"`
}

// WriteResult is returned after an optimistic-concurrency write.
type WriteResult struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	MTimeMs int64  `json:"mtimeMs"`
}

// ReadText loads a full editable document. expectedMtimeMs is not involved in
// reads; it exists on the write path where stale drafts are rejected.
func ReadText(root, rel string) (TextFile, error) {
	abs, cleanRel, err := resolveReadable(root, rel)
	if err != nil {
		return TextFile{}, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return TextFile{}, fmt.Errorf("读取文件失败: %w", err)
	}
	out := TextFile{Path: slashRel(cleanRel), Size: info.Size(), MTimeMs: info.ModTime().UnixMilli()}
	if info.Size() > EditableBytes {
		out.TooLarge = true
		return out, nil
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		return TextFile{}, fmt.Errorf("读取文件失败: %w", err)
	}
	kind := detectEncodingSample(raw)
	out.Encoding = encodingName(kind)
	out.EOL = detectEOL(raw, kind)
	if likelyBinary(raw, kind) {
		out.Binary = true
		return out, nil
	}
	out.Text = string(fileenc.Decode(raw, kind))
	return out, nil
}

// ReadPage returns one read-only page from a text file of any size.
func ReadPage(root, rel string, offset int64) (TextPage, error) {
	abs, cleanRel, err := resolveReadable(root, rel)
	if err != nil {
		return TextPage{}, err
	}
	f, err := os.Open(abs)
	if err != nil {
		return TextPage{}, fmt.Errorf("打开文件失败: %w", err)
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return TextPage{}, fmt.Errorf("读取文件信息失败: %w", err)
	}
	if offset < 0 {
		offset = 0
	}
	if offset > info.Size() {
		offset = info.Size()
	}
	sample := make([]byte, minInt64(int64(sampleBytes), info.Size()))
	if _, err := f.ReadAt(sample, 0); err != nil && !errors.Is(err, io.EOF) {
		return TextPage{}, fmt.Errorf("读取编码样本失败: %w", err)
	}
	kind := detectEncodingSample(sample)
	out := TextPage{
		Path:     slashRel(cleanRel),
		Offset:   offset,
		Size:     info.Size(),
		Encoding: encodingName(kind),
		EOL:      detectEOL(sample, kind),
	}
	if likelyBinary(sample, kind) {
		out.Binary = true
		out.NextOffset = info.Size()
		out.EOF = true
		return out, nil
	}
	unit := encodingUnit(kind)
	if offset > 0 {
		offset -= offset % int64(unit)
		out.Offset = offset
	}
	readN := minInt64(int64(PageBytes), info.Size()-offset)
	raw := make([]byte, readN)
	if _, err := f.ReadAt(raw, offset); err != nil && !errors.Is(err, io.EOF) {
		return TextPage{}, fmt.Errorf("读取分页失败: %w", err)
	}
	chunk := raw
	// Find the last complete line break inside the page; retain at most 64 KiB
	// after it so a pathological single line still makes forward progress.
	if end := lastLineBreakEnd(raw, kind); end > 0 && end < len(raw) {
		chunk = raw[:end]
	}
	if len(chunk) == 0 && offset < info.Size() {
		chunk = raw
	}
	if offset == 0 {
		out.Text = string(fileenc.Decode(chunk, kind))
	} else {
		out.Text = string(fileenc.Decode(chunk, fileenc.Kind(bomlessKind(kind))))
	}
	out.NextOffset = offset + int64(len(chunk))
	out.EOF = out.NextOffset >= info.Size()
	return out, nil
}

// WriteText writes a UTF-8 editor document back in the file's original
// encoding and line-ending convention. A non-zero expectedMtimeMs turns a
// changed or vanished target into a conflict rather than an overwrite.
func WriteText(root, rel, content string, expectedMtimeMs int64) (WriteResult, error) {
	abs, cleanRel, err := resolveWritable(root, rel)
	if err != nil {
		return WriteResult{}, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		if expectedMtimeMs != 0 && errors.Is(err, fs.ErrNotExist) {
			return WriteResult{}, fmt.Errorf("文件已在磁盘上被删除或改名，请重新打开")
		}
		return WriteResult{}, fmt.Errorf("目标不是已存在的文本文件: %s", slashRel(cleanRel))
	}
	if !info.Mode().IsRegular() {
		return WriteResult{}, fmt.Errorf("目标不是普通文件: %s", slashRel(cleanRel))
	}
	if expectedMtimeMs != 0 && info.ModTime().UnixMilli() != expectedMtimeMs {
		return WriteResult{}, fmt.Errorf(
			"文件已在磁盘上改变（打开时 %d，当前 %d），请重新载入后再保存",
			expectedMtimeMs, info.ModTime().UnixMilli(),
		)
	}
	if len(content) > EditableBytes {
		return WriteResult{}, fmt.Errorf("内容过大（上限 5MB，当前 %d 字节）", len(content))
	}
	f, err := os.Open(abs)
	if err != nil {
		return WriteResult{}, fmt.Errorf("读取文件编码失败: %w", err)
	}
	sample := make([]byte, minInt64(int64(sampleBytes), info.Size()))
	if _, err := f.ReadAt(sample, 0); err != nil && !errors.Is(err, io.EOF) {
		_ = f.Close()
		return WriteResult{}, fmt.Errorf("读取编码样本失败: %w", err)
	}
	_ = f.Close()
	kind := detectEncodingSample(sample)
	eol := detectEOL(sample, kind)
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	if eol == "crlf" {
		normalized = strings.ReplaceAll(normalized, "\n", "\r\n")
	} else if eol == "cr" {
		normalized = strings.ReplaceAll(normalized, "\n", "\r")
	}
	encoded := fileenc.Encode(normalized, kind)
	if err := fileutil.AtomicWriteSync(abs, encoded, info.Mode().Perm()); err != nil {
		return WriteResult{}, fmt.Errorf("写入失败: %w", err)
	}
	written, err := os.Stat(abs)
	if err != nil {
		return WriteResult{}, nil
	}
	// Windows ms 粒度 CAS 失明防线（2026-11-13 实锤，先例=config 全零事件
	// 「同纳秒互覆备份须 CAS 递增时间戳」）：原子写（temp+rename）落在打开
	// 基线的同一毫秒内时，下一次保存的 stale 判定（UnixMilli 相等）会放过
	// 外部改动——把落盘 mtime 顶到基线之后，保基线链严格单调，编辑器拿到的
	// 新基线即递增后的值。
	if mod := written.ModTime(); expectedMtimeMs != 0 && mod.UnixMilli() <= expectedMtimeMs {
		next := time.UnixMilli(expectedMtimeMs + 1)
		_ = os.Chtimes(abs, next, next)
		if bumped, serr := os.Stat(abs); serr == nil {
			written = bumped
		}
	}
	return WriteResult{
		Path:    slashRel(cleanRel),
		Size:    written.Size(),
		MTimeMs: written.ModTime().UnixMilli(),
	}, nil
}

// DecodePreview decodes up to limit bytes for the existing preview surface and
// reports the original byte count plus encoding metadata.
func DecodePreview(raw []byte, limit int64) (body string, truncated bool, encoding, eol string) {
	kind := detectEncodingSample(raw)
	encoding = encodingName(kind)
	eol = detectEOL(raw, kind)
	cut := raw
	if int64(len(cut)) > limit {
		cut = cut[:limit]
		truncated = true
		if i := bytes.LastIndexByte(cut, '\n'); i >= 0 {
			cut = cut[:i+1]
		}
	}
	// Avoid cutting in the middle of a UTF-8 rune. Wide encodings are bounded
	// by their fixed unit below.
	if kind == fileenc.UTF8 || kind == fileenc.UTF8BOM || kind == fileenc.LossyUTF8 {
		for len(cut) > 0 && !utf8.Valid(cut) {
			cut = cut[:len(cut)-1]
		}
	}
	if kind == fileenc.GB18030 {
		for len(cut) > 0 {
			if _, _, err := transform.Bytes(simplifiedchinese.GB18030.NewDecoder(), cut); err == nil {
				break
			}
			cut = cut[:len(cut)-1]
		}
	}
	return string(fileenc.Decode(cut, kind)), truncated, encoding, eol
}

func resolveReadable(root, rel string) (string, string, error) {
	abs, cleanRel, err := resolveWithin(root, rel)
	if err != nil {
		return "", "", err
	}
	rootAbs, _ := filepath.Abs(root)
	realRoot, err := wspath.RealPath(rootAbs)
	if err != nil {
		return "", "", err
	}
	realPath, err := wspath.RealPath(abs)
	if err != nil {
		return "", "", fmt.Errorf("读取文件失败: %w", err)
	}
	if !wspath.Within(realRoot, realPath) {
		return "", "", fmt.Errorf("文件通过符号链接越出工作区")
	}
	return abs, cleanRel, nil
}

func resolveWritable(root, rel string) (string, string, error) {
	abs, cleanRel, err := resolveWithin(root, rel)
	if err != nil {
		return "", "", err
	}
	if cleanRel == "" {
		return "", "", fmt.Errorf("不能写入工作区根目录")
	}
	if err := ensureExistingDirWithin(root, filepath.Dir(abs)); err != nil {
		return "", "", err
	}
	return abs, cleanRel, nil
}

func likelyBinary(sample []byte, kind fileenc.Kind) bool {
	switch kind {
	case fileenc.UTF16LE, fileenc.UTF16BE, fileenc.UTF16LENoBOM, fileenc.UTF16BENoBOM:
		return false
	}
	if len(sample) == 0 {
		return false
	}
	var controls, nuls int
	for _, b := range sample {
		if b == 0 {
			nuls++
		}
		if b < 0x09 || (b > 0x0D && b < 0x20) {
			controls++
		}
	}
	return nuls > 0 || controls*100 > len(sample)*5
}

func encodingName(kind fileenc.Kind) string {
	switch kind {
	case fileenc.UTF8BOM:
		return "utf-8-bom"
	case fileenc.UTF16LE:
		return "utf-16le"
	case fileenc.UTF16BE:
		return "utf-16be"
	case fileenc.GB18030:
		return "gb18030"
	case fileenc.LossyUTF8:
		return "utf-8-lossy"
	case fileenc.UTF16LENoBOM:
		return "utf-16le"
	case fileenc.UTF16BENoBOM:
		return "utf-16be"
	default:
		return "utf-8"
	}
}

func encodingUnit(kind fileenc.Kind) int {
	switch kind {
	case fileenc.UTF16LE, fileenc.UTF16BE, fileenc.UTF16LENoBOM, fileenc.UTF16BENoBOM:
		return 2
	default:
		return 1
	}
}

func detectEOL(raw []byte, kind fileenc.Kind) string {
	if encodingUnit(kind) == 2 {
		var crlf, lf, cr int
		for i := 0; i+3 < len(raw); i += 2 {
			a0, a1 := raw[i], raw[i+1]
			b0, b1 := raw[i+2], raw[i+3]
			le := (kind == fileenc.UTF16LE || kind == fileenc.UTF16LENoBOM)
			if le {
				if a0 == '\r' && a1 == 0 && b0 == '\n' && b1 == 0 {
					crlf++
					i += 2
				} else if a0 == '\n' && a1 == 0 {
					lf++
				} else if a0 == '\r' && a1 == 0 {
					cr++
				}
			} else {
				if a0 == 0 && a1 == '\r' && b0 == 0 && b1 == '\n' {
					crlf++
					i += 2
				} else if a0 == 0 && a1 == '\n' {
					lf++
				} else if a0 == 0 && a1 == '\r' {
					cr++
				}
			}
		}
		return preferredEOL(crlf, lf, cr)
	}
	crlf := bytes.Count(raw, []byte("\r\n"))
	lf := bytes.Count(raw, []byte("\n")) - crlf
	cr := bytes.Count(raw, []byte("\r")) - crlf
	return preferredEOL(crlf, lf, cr)
}

func preferredEOL(crlf, lf, cr int) string {
	if crlf >= lf && crlf >= cr && crlf > 0 {
		return "crlf"
	}
	if cr > lf && cr > 0 {
		return "cr"
	}
	return "lf"
}

func lastLineBreakEnd(raw []byte, kind fileenc.Kind) int {
	if encodingUnit(kind) == 2 {
		last := 0
		le := kind == fileenc.UTF16LE || kind == fileenc.UTF16LENoBOM
		for i := 0; i+1 < len(raw); i += 2 {
			isLF := (le && raw[i] == '\n' && raw[i+1] == 0) || (!le && raw[i] == 0 && raw[i+1] == '\n')
			if isLF {
				last = i + 2
			}
		}
		return last
	}
	if i := bytes.LastIndexByte(raw, '\n'); i >= 0 {
		return i + 1
	}
	return 0
}

func bomlessKind(kind fileenc.Kind) fileenc.Kind {
	switch kind {
	case fileenc.UTF8BOM:
		return fileenc.UTF8
	case fileenc.UTF16LE:
		return fileenc.UTF16LENoBOM
	case fileenc.UTF16BE:
		return fileenc.UTF16BENoBOM
	default:
		return kind
	}
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func detectEncodingSample(raw []byte) fileenc.Kind {
	kind, _ := fileenc.Detect(raw)
	if kind != fileenc.LossyUTF8 {
		return kind
	}
	// A bounded sample can end in the middle of a multi-byte CJK sequence.
	// Retry without up to three trailing bytes before declaring the sample
	// lossy; the full-file read/write path remains authoritative.
	for cut := 1; cut <= 3 && len(raw)-cut >= 16; cut++ {
		if candidate, _ := fileenc.Detect(raw[:len(raw)-cut]); candidate != fileenc.LossyUTF8 {
			return candidate
		}
	}
	return kind
}
