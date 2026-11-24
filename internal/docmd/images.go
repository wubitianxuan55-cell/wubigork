package docmd

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const maxConvertedImageBytes = 20 << 20

// ConvertPreservingImages converts DOCX/PPTX while retaining embedded image
// data URIs for a follow-up ExternalizeDataImages call. Other formats use the
// normal text converter and report zero images.
func ConvertPreservingImages(path, pages string, maxPages int) (md string, total int, truncated bool, images int, err error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".docx", ".doc":
		md, err = convertViaMarkItDownKeepImages(path)
		if err == nil {
			return md, 0, false, countImageDataURIs(md), nil
		}
		if hasOfficeMedia(path) {
			return "", 0, false, 0, fmt.Errorf("原文档含图片，但 markitdown 不可用，无法无损保留：%w", err)
		}
		md, err = docxToMarkdown(path)
		return md, 0, false, 0, err
	case ".pptx", ".ppt":
		md, err = convertViaMarkItDownKeepImages(path)
		if err != nil {
			return "", 0, false, 0, fmt.Errorf("pptx 保留图片需要 markitdown: %w", err)
		}
		return md, 0, false, countImageDataURIs(md), nil
	default:
		md, total, truncated, err = ConvertLimit(path, pages, maxPages)
		return md, total, truncated, 0, err
	}
}

// ExternalizeDataImages writes embedded image data URIs under assetDir and
// rewrites them to paths relative to markdownDir.
func ExternalizeDataImages(markdown, markdownDir, assetDir string) (string, int, error) {
	refs := findImageTargets(markdown)
	if len(refs) == 0 {
		return markdown, 0, nil
	}
	hasData := false
	for _, ref := range refs {
		if strings.HasPrefix(strings.ToLower(ref.target), "data:image/") {
			hasData = true
			break
		}
	}
	if !hasData {
		return markdown, 0, nil
	}
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		return "", 0, fmt.Errorf("创建图片 sidecar 目录: %w", err)
	}
	out := markdown
	written := make(map[string]string)
	count := 0
	for i := len(refs) - 1; i >= 0; i-- {
		ref := refs[i]
		if !strings.HasPrefix(strings.ToLower(ref.target), "data:image/") {
			continue
		}
		raw, err := decodeBase64DataURI(ref.target)
		if err != nil {
			return "", 0, err
		}
		mime := detectedConvertedImageMime(raw)
		if mime == "" {
			return "", 0, fmt.Errorf("不支持的文档图片格式（仅 PNG/JPEG/GIF/WebP）")
		}
		sum := sha256.Sum256(raw)
		hash := hex.EncodeToString(sum[:])
		rel := written[hash]
		if rel == "" {
			abs := filepath.Join(assetDir, hash+convertedImageExt(mime))
			if err := writeBytesAtomic(abs, raw); err != nil {
				return "", 0, fmt.Errorf("写入文档图片: %w", err)
			}
			rel, err = filepath.Rel(markdownDir, abs)
			if err != nil {
				return "", 0, fmt.Errorf("计算图片相对路径: %w", err)
			}
			rel = filepath.ToSlash(rel)
			written[hash] = rel
			count++
		}
		out = out[:ref.targetStart] + rel + out[ref.targetEnd:]
	}
	return out, count, nil
}

type imageTarget struct {
	targetStart int
	targetEnd   int
	target      string
}

func findImageTargets(markdown string) []imageTarget {
	var out []imageTarget
	for offset := 0; offset < len(markdown); {
		start := strings.Index(markdown[offset:], "![")
		if start < 0 {
			break
		}
		start += offset
		altEnd := strings.Index(markdown[start+2:], "](")
		if altEnd < 0 {
			break
		}
		altEnd += start + 2
		targetStart := altEnd + 2
		for targetStart < len(markdown) && (markdown[targetStart] == ' ' || markdown[targetStart] == '\t') {
			targetStart++
		}
		if targetStart >= len(markdown) {
			break
		}
		var (
			target       string
			replaceStart int
			targetEnd    int
			closeParen   int
		)
		if markdown[targetStart] == '<' {
			end := strings.IndexByte(markdown[targetStart+1:], '>')
			if end < 0 {
				break
			}
			end += targetStart + 1
			target = markdown[targetStart+1 : end]
			closeRel := strings.IndexByte(markdown[end+1:], ')')
			if closeRel < 0 {
				break
			}
			closeParen = end + 1 + closeRel
			replaceStart = targetStart + 1
			targetEnd = end
		} else {
			closeRel := strings.IndexByte(markdown[targetStart:], ')')
			if closeRel < 0 {
				break
			}
			closeParen = targetStart + closeRel
			target = markdown[targetStart:closeParen]
			if i := strings.IndexAny(target, " \t\r\n"); i >= 0 {
				target = target[:i]
			}
			replaceStart = targetStart
			targetEnd = targetStart + len(target)
		}
		out = append(out, imageTarget{targetStart: replaceStart, targetEnd: targetEnd, target: target})
		offset = closeParen + 1
	}
	return out
}

func countImageDataURIs(markdown string) int {
	n := 0
	for _, ref := range findImageTargets(markdown) {
		if strings.HasPrefix(strings.ToLower(ref.target), "data:image/") {
			n++
		}
	}
	return n
}

func decodeBase64DataURI(value string) ([]byte, error) {
	comma := strings.IndexByte(value, ',')
	if comma < 0 || !strings.Contains(strings.ToLower(value[:comma]), ";base64") {
		return nil, fmt.Errorf("文档图片 data URI 无效")
	}
	encoded := value[comma+1:]
	encoded = strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, encoded)
	if int64(base64.StdEncoding.DecodedLen(len(encoded))) > maxConvertedImageBytes {
		return nil, fmt.Errorf("文档图片超过 20 MB")
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("文档图片 base64 解码失败")
	}
	if len(raw) == 0 || len(raw) > maxConvertedImageBytes {
		return nil, fmt.Errorf("文档图片大小异常")
	}
	return raw, nil
}

func detectedConvertedImageMime(raw []byte) string {
	head := raw
	if len(head) > 512 {
		head = head[:512]
	}
	mime := strings.ToLower(strings.Split(http.DetectContentType(head), ";")[0])
	if convertedImageExt(mime) == "" {
		return ""
	}
	return mime
}

func convertedImageExt(mime string) string {
	switch mime {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ""
	}
}

func writeBytesAtomic(path string, raw []byte) error {
	if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() && info.Size() == int64(len(raw)) {
		existing, readErr := os.ReadFile(path)
		if readErr == nil {
			want := sha256.Sum256(raw)
			got := sha256.Sum256(existing)
			if want == got {
				return nil
			}
		}
		_ = os.Remove(path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".docmd-image-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(name)
		}
	}()
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		if _, statErr := os.Stat(path); statErr == nil {
			_ = os.Remove(name)
			ok = true
			return nil
		}
		return err
	}
	ok = true
	return nil
}

func hasOfficeMedia(path string) bool {
	r, err := zip.OpenReader(path)
	if err != nil {
		return false
	}
	defer func() { _ = r.Close() }()
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "word/media/") || strings.HasPrefix(f.Name, "ppt/media/") {
			return true
		}
	}
	return false
}
