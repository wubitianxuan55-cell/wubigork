// Package ooxml — docx/pptx 共用的 OOXML zip 容器与纯文本原语（AP3-02+IN3-06）。
//
// docxedit 与 pptxedit 的 zip 读写（条目序保留 + 临时文件原子替换）与段落
// 定位纯函数（locateSpan/extractAttrs 等）历史上逐行同构各存一份；本包收拢
// 唯一实现。约束：字节级行为冻结——zip 条目顺序、压缩方法（Deflate）、错误
// 信息前缀、转义字节（docx=WML 口径，pptx=DML 口径）均与原两包逐字节一致；
// 各 part 名与 XML 语义（WML 修订制 / DrawingML 直改）仍留 docxedit/pptxedit。
//
// xlsx 不接入：xlsxedit 全程经 excelize 读写，无自有 zip 层与可共享纯函数。
package ooxml

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/gaea/gaea/internal/gaea/fileutil"
)

// ── zip 容器读写（同原 docxedit/pptxedit：条目序保留 + 原子替换） ──

// ReadZip 打开磁盘上的 OOXML 包，读出全部条目：files 按条目名取字节，
// order 保留 zip 原始条目序（写回时逐字节保序）。label 用于错误信息前缀
// （"docx"/"pptx"）；必需 part（word/document.xml、ppt/presentation.xml）
// 的存在性校验由调用方按各自 part 名做。
func ReadZip(path, label string) (files map[string][]byte, order []string, err error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, nil, fmt.Errorf("打开 %s 失败: %w", label, err)
	}
	defer func() { _ = r.Close() }()
	files = map[string][]byte{}
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return nil, nil, fmt.Errorf("读取 %s 失败: %w", f.Name, err)
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("读取 %s 失败: %w", f.Name, err)
		}
		files[f.Name] = b
		order = append(order, f.Name)
	}
	return files, order, nil
}

// WriteAtomic 原子写回：条目按 order 逐个 Deflate 写入同目录临时文件
// （tmpPattern 定临时名，如 ".gaea-docxedit-*.docx"），再 RenameWithRetry
// 原子替换。条目顺序、压缩方法与错误信息（"打包 <label> 失败"）与原
// docxedit/pptxedit 实现逐字节一致。
func WriteAtomic(path string, files map[string][]byte, order []string, tmpPattern, label string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), tmpPattern)
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	zw := zip.NewWriter(tmp)
	for _, name := range order {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate})
		if err != nil {
			_ = zw.Close()
			_ = tmp.Close()
			return fmt.Errorf("写回 %s 失败: %w", name, err)
		}
		if _, err := w.Write(files[name]); err != nil {
			_ = zw.Close()
			_ = tmp.Close()
			return fmt.Errorf("写回 %s 失败: %w", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("打包 %s 失败: %w", label, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("关闭临时文件失败: %w", err)
	}
	if err := fileutil.RenameWithRetry(tmpName, path); err != nil {
		return fmt.Errorf("替换原文件失败: %w", err)
	}
	return nil
}

// ── 定位纯函数（两包逐行同构的原实现收拢） ─────────────────────────

// ExtractAttrs 从 t 起始标签原始字节提取属性子串（不含标签名与 >）。
func ExtractAttrs(raw []byte) string {
	s := string(raw)
	// 找到元素名结束位置（空格或 >）
	idx := strings.IndexAny(s, " \t\n>")
	if idx < 0 {
		return ""
	}
	if s[idx] == '>' {
		return ""
	}
	rest := s[idx:]
	end := strings.Index(rest, ">")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(rest[:end])
}

// LocateSpan 返回 target 在段落拼接文本 text 中的 rune 区间 [s,e)；精确优先，
// 失败折叠空白兜底（还原到原始区间）。段落模型本身（WML/DrawingML 语义）
// 留在各编辑包，这里只消费拼接文本。
func LocateSpan(text, target string) (int, int, bool) {
	if idx := strings.Index(text, target); idx >= 0 {
		// 统一使用 rune 偏移（重建阶段按 rune 切分）
		s := len([]rune(text[:idx]))
		return s, s + len([]rune(target)), true
	}
	// 折叠空白兜底：需要 rune 级别映射
	runes := []rune(text)
	norm := make([]int, 0, len(runes))
	for i, r := range runes {
		if unicode.IsSpace(r) {
			if i > 0 && unicode.IsSpace(runes[i-1]) {
				continue
			}
		}
		norm = append(norm, i)
	}
	normText := make([]rune, 0, len(norm))
	for _, i := range norm {
		normText = append(normText, runes[i])
	}
	// 同样折叠 target
	var tNorm []rune
	prevSpace := false
	for _, r := range target {
		if unicode.IsSpace(r) {
			if prevSpace {
				continue
			}
			prevSpace = true
			tNorm = append(tNorm, ' ')
		} else {
			prevSpace = false
			tNorm = append(tNorm, r)
		}
	}
	if len(tNorm) == 0 {
		return 0, 0, false
	}
	// normText 中的空白已被折叠为单个空格
	ni := IndexRunes(normText, tNorm)
	if ni < 0 {
		return 0, 0, false
	}
	s := norm[ni]
	e := norm[ni+len(tNorm)-1] + 1
	// 去掉首尾纯空白，避免吃掉相邻空白
	for s < e && unicode.IsSpace(runes[s]) {
		s++
	}
	for e > s && unicode.IsSpace(runes[e-1]) {
		e--
	}
	if s >= e {
		return 0, 0, false
	}
	return s, e, true
}

// IndexRunes 返回 needle 在 hay 中的首个下标；空 needle 恒 0，未命中 -1。
func IndexRunes(hay, needle []rune) int {
	if len(needle) == 0 {
		return 0
	}
	for i := 0; i+len(needle) <= len(hay); i++ {
		ok := true
		for j := range needle {
			if hay[i+j] != needle[j] {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}

// MaxInt 返回较大者。
func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// MinInt 返回较小者。
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ── 文本元素生成（WML/DML 两套历史字节口径，方言参数钉死） ─────────

// TextElement 生成 <elem attrs>text</elem>：文本 XML 转义、首尾空白补
// xml:space="preserve"。wmlDialect 钉死两套历史字节口径，逐字节对齐原
// docxedit/pptxedit 各自实现：
//   - WML（docx）：tab 也算首尾空白；`"`→&quot;；`'` 不转义；
//   - DML（pptx）：仅空格算首尾空白；`"`→&#34;；`'`→&#39;。
func TextElement(elem, text, attrs string, wmlDialect bool) string {
	esc := EscapeXML(text, wmlDialect)
	needsPreserve := strings.HasPrefix(text, " ") || strings.HasSuffix(text, " ")
	if wmlDialect {
		needsPreserve = needsPreserve || strings.HasPrefix(text, "\t") || strings.HasSuffix(text, "\t")
	}
	if needsPreserve && !strings.Contains(attrs, "xml:space") {
		attrs = strings.TrimSpace(attrs)
		if attrs != "" {
			attrs += " "
		}
		attrs += `xml:space="preserve"`
	}
	if attrs != "" {
		attrs = " " + attrs
	}
	return "<" + elem + attrs + ">" + esc + "</" + elem + ">"
}

// EscapeXML 按方言转义（见 TextElement：两口径仅 `"`/`'` 的字节不同）。
func EscapeXML(s string, wmlDialect bool) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			if wmlDialect {
				b.WriteString("&quot;")
			} else {
				b.WriteString("&#34;")
			}
		case '\'':
			if !wmlDialect {
				b.WriteString("&#39;")
			} else {
				b.WriteRune(r)
			}
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
