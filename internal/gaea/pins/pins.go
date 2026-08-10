// Package pins 管理工作区的「常用资料」固定清单（对标 Claude Projects 知识库 /
// aily 关联知识的本地版）：用户把常用文件钉住后，新会话启动时自动把
// 文件清单（小文本文件含正文）装进系统提示词，供模型直接参考。
//
// 清单落在 <cwd>/.gaea/pinned.json（每工作区一份），正文提取遵循
// 「装配而非灌输」：文本文件带内容，办公文档只列名，需要时由模型
// read_file / format_convert 按需读取。
package pins

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	maxPins       = 20   // 清单上限
	maxBlockPins  = 10   // 注入上下文的上限
	maxFileRunes  = 1600 // 单文件正文注入上限（字符）
	maxTotalRunes = 8000 // 整块注入上限
)

// Path 返回当前工作区的固定清单文件路径。
func Path(cwd string) string {
	return filepath.Join(cwd, ".gaea", "pinned.json")
}

// Load 读取固定清单（相对路径、去重、保序）；文件不存在返回空列表。
func Load(cwd string) ([]string, error) {
	b, err := os.ReadFile(Path(cwd))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var raw []string
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	return normalize(raw), nil
}

// Save 写入固定清单（自动去重、限长）。
func Save(cwd string, paths []string) error {
	dir := filepath.Join(cwd, ".gaea")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(normalize(paths), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(cwd), b, 0o644)
}

// Add 固定一个文件并持久化，返回更新后的清单。
func Add(cwd, rel string) ([]string, error) {
	cur, err := Load(cwd)
	if err != nil {
		return nil, err
	}
	rel = cleanRel(rel)
	if rel == "" {
		return cur, nil
	}
	next := append([]string{}, cur...)
	found := false
	for _, p := range next {
		if p == rel {
			found = true
			break
		}
	}
	if !found {
		next = append(next, rel)
	}
	if len(next) > maxPins {
		next = next[:maxPins]
	}
	if err := Save(cwd, next); err != nil {
		return cur, err
	}
	return next, nil
}

// Remove 取消固定并持久化，返回更新后的清单。
func Remove(cwd, rel string) ([]string, error) {
	cur, err := Load(cwd)
	if err != nil {
		return nil, err
	}
	rel = cleanRel(rel)
	next := cur[:0]
	for _, p := range cur {
		if p != rel {
			next = append(next, p)
		}
	}
	if err := Save(cwd, next); err != nil {
		return cur, err
	}
	return next, nil
}

// cleanRel 规范化相对路径：正斜杠、去首尾 / 与 ./，拒绝绝对路径和 .. 逃逸。
func cleanRel(rel string) string {
	rel = strings.TrimSpace(rel)
	rel = filepath.ToSlash(rel)
	if rel == "" || strings.HasPrefix(rel, "/") || filepath.IsAbs(rel) {
		return ""
	}
	rel = strings.TrimPrefix(rel, "./")
	rel = strings.Trim(rel, "/")
	if rel == "" || strings.HasPrefix(rel, "../") || strings.Contains(rel, "/../") {
		return ""
	}
	return rel
}

func normalize(paths []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, p := range paths {
		p = cleanRel(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
		if len(out) >= maxPins {
			break
		}
	}
	return out
}

// 文本类扩展名：正文可直接注入。
var textExts = map[string]bool{
	".md": true, ".markdown": true, ".txt": true, ".csv": true,
}

// Block 构建注入系统提示词的「常用资料」块；无固定或全部失败返回 ""。
// 最佳努力：单个文件失败只跳过，绝不阻塞会话启动。
func Block(cwd string) string {
	paths, err := Load(cwd)
	if err != nil || len(paths) == 0 {
		return ""
	}
	if len(paths) > maxBlockPins {
		paths = paths[:maxBlockPins]
	}

	var b strings.Builder
	b.WriteString("## 常用资料（已固定，自动带入上下文）\n")
	b.WriteString("以下是你固定为常用资料的文件：文本类已附正文摘要，办公文档请按需用 read_file 或 format_convert 读取完整内容。\n")
	total := 0
	wroteAny := false
	for _, rel := range paths {
		abs := filepath.Join(cwd, filepath.FromSlash(rel))
		name := filepath.Base(abs)
		ext := strings.ToLower(filepath.Ext(name))
		head := "- " + name + "（" + rel + "）"
		if !textExts[ext] {
			head += "：办公文档"
			b.WriteString(head + "\n")
			wroteAny = true
			continue
		}
		body, ok := readText(abs)
		if !ok {
			continue
		}
		wroteAny = true
		body = truncateRunes(body, maxFileRunes)
		if total+utf8.RuneCountInString(body) > maxTotalRunes {
			body = truncateRunes(body, maxTotalRunes-total)
		}
		b.WriteString("\n### " + name + "（" + rel + "）\n")
		b.WriteString(body + "\n")
		total += utf8.RuneCountInString(body)
		if total >= maxTotalRunes {
			break
		}
	}
	if !wroteAny {
		return ""
	}
	return strings.TrimSpace(b.String())
}

// readText 读取文本文件正文（限 5MB，UTF-8 修复）。
func readText(abs string) (string, bool) {
	info, err := os.Stat(abs)
	if err != nil || info.IsDir() || info.Size() > 5<<20 {
		return "", false
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", false
	}
	s := strings.TrimPrefix(string(b), "\ufeff")
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "\ufffd")
	}
	return s, true
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 0 {
		return ""
	}
	return string(r[:n])
}
