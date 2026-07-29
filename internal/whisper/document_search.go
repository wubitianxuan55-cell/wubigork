// Package whisper — document_search.go
// 100% 对齐 ackem desktop-agent/investigation/documentSearch.ts
// 文档搜索：按扩展名 + 关键词递归搜索文件

package whisper

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ─── 扩展名映射 ────────────────────────────────────────────────

// extensionMap 中文关键词 → 扩展名映射
var extensionMap = map[string][]string{
	"pdf":     {".pdf"},
	"word":    {".doc", ".docx"},
	"excel":   {".xls", ".xlsx", ".csv"},
	"ppt":     {".ppt", ".pptx"},
	"图片":     {".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp"},
	"照片":     {".jpg", ".jpeg", ".png", ".heic"},
	"视频":     {".mp4", ".avi", ".mkv", ".mov", ".wmv"},
	"音频":     {".mp3", ".wav", ".flac", ".aac", ".ogg"},
	"文本":     {".txt", ".md", ".log"},
	"代码":     {".go", ".py", ".js", ".ts", ".java", ".cpp", ".c", ".rs", ".html", ".css", ".json", ".yaml"},
	"压缩":     {".zip", ".rar", ".7z", ".tar", ".gz"},
	"笔记":     {".md", ".txt", ".rtf"},
	"表格":     {".xls", ".xlsx", ".csv"},
	"演示":     {".ppt", ".pptx"},
}

// ─── 类型定义 ──────────────────────────────────────────────────

// FileFinding 文件发现
type FileFinding struct {
	DisplayName string `json:"displayName"`
	Path        string `json:"path"`
	Source      string `json:"source"`
	SizeBytes   int64  `json:"sizeBytes,omitempty"`
	Ext         string `json:"ext"`
}

// ─── 查询解析 ──────────────────────────────────────────────────

// ParseExtensionsFromQuery 从用户查询中提取扩展名列表
// 100% 对齐 ackem documentSearch.ts parseExtensionsFromQuery
func ParseExtensionsFromQuery(query string) []string {
	lower := strings.ToLower(query)

	// 中文关键词映射
	for keyword, exts := range extensionMap {
		if strings.Contains(lower, keyword) {
			return exts
		}
	}

	// 直接扩展名匹配（如 ".pdf", ".docx"）
	var exts []string
	words := strings.Fields(lower)
	for _, w := range words {
		if strings.HasPrefix(w, ".") && len(w) >= 2 {
			exts = append(exts, w)
		}
	}
	if len(exts) > 0 {
		return exts
	}

	// 无扩展名关键词 → 返回文档类默认
	return []string{".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".txt", ".md"}
}

// ─── 文件搜索 ──────────────────────────────────────────────────

const (
	maxSearchDepth = 5
	maxSearchResults = 200
)

// skipDirs 跳过的目录名
var skipDirs = map[string]bool{
	"node_modules": true,
	".git":         true,
	"__pycache__":  true,
	"vendor":       true,
	"target":       true,
	"build":        true,
	"dist":         true,
	".idea":        true,
	".vscode":      true,
}

// SearchFilesByExtensions 按扩展名递归搜索文件
// 100% 对齐 ackem documentSearch.ts searchFilesByExtensions
func SearchFilesByExtensions(root string, extensions []string, source string) []FileFinding {
	if root == "" {
		return nil
	}

	extSet := make(map[string]bool)
	for _, e := range extensions {
		extSet[strings.ToLower(e)] = true
	}

	var findings []FileFinding
	searchRecursive(root, extSet, source, 0, &findings)
	return findings
}

// searchRecursive 递归搜索
func searchRecursive(dir string, exts map[string]bool, source string, depth int, findings *[]FileFinding) {
	if depth > maxSearchDepth || len(*findings) >= maxSearchResults {
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		name := entry.Name()

		// 跳过隐藏文件/目录
		if strings.HasPrefix(name, ".") {
			continue
		}

		fullPath := filepath.Join(dir, name)

		if entry.IsDir() {
			if skipDirs[name] {
				continue
			}
			searchRecursive(fullPath, exts, source, depth+1, findings)
			continue
		}

		// 检查扩展名
		ext := strings.ToLower(filepath.Ext(name))
		if !exts[ext] {
			continue
		}

		info, err := entry.Info()
		size := int64(0)
		if err == nil {
			size = info.Size()
		}

		*findings = append(*findings, FileFinding{
			DisplayName: name,
			Path:        fullPath,
			Source:      source,
			SizeBytes:   size,
			Ext:         ext,
		})

		if len(*findings) >= maxSearchResults {
			return
		}
	}
}

// ─── 合并文件发现 ──────────────────────────────────────────────

// MergeFileFindings 按路径去重 + 按名称排序
// 100% 对齐 ackem documentSearch.ts mergeFileFindings
func MergeFileFindings(files []FileFinding) []FileFinding {
	seen := make(map[string]bool)
	var merged []FileFinding
	for _, f := range files {
		key := strings.ToLower(f.Path)
		if seen[key] {
			continue
		}
		seen[key] = true
		merged = append(merged, f)
	}

	// 按显示名排序
	sort.Slice(merged, func(i, j int) bool {
		return strings.ToLower(merged[i].DisplayName) < strings.ToLower(merged[j].DisplayName)
	})

	return merged
}

// ─── 搜索入口 ──────────────────────────────────────────────────

// DefaultSearchRoots 默认搜索根目录
var DefaultSearchRoots = []struct {
	Path   string
	Source string
}{
	{os.Getenv("USERPROFILE") + "\\Desktop", "desktop"},
	{os.Getenv("USERPROFILE") + "\\Documents", "documents"},
	{os.Getenv("USERPROFILE") + "\\Downloads", "downloads"},
}

// SearchDocuments 在默认位置搜索文档
func SearchDocuments(query string) []FileFinding {
	extensions := ParseExtensionsFromQuery(query)

	var allFiles []FileFinding
	for _, root := range DefaultSearchRoots {
		if _, err := os.Stat(root.Path); err == nil {
			files := SearchFilesByExtensions(root.Path, extensions, root.Source)
			allFiles = append(allFiles, files...)
		}
	}

	return MergeFileFindings(allFiles)
}

// SearchDocumentsInDir 在指定目录搜索文档
func SearchDocumentsInDir(dir, query string) []FileFinding {
	extensions := ParseExtensionsFromQuery(query)
	files := SearchFilesByExtensions(dir, extensions, "custom")
	return MergeFileFindings(files)
}
