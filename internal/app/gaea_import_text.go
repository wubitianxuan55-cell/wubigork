package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/docmd"
	"github.com/gaea/gaea/internal/office/cjk"
)

// extractImportTextFile 提取导入文件的文本：md/txt/csv 直读；docx/xlsx/pdf 等经
// docmd 转 Markdown（限 50 页）。progress 在 PDF 逐页 OCR 时回调 (done, total)，
// 不需要进度传 nil；md/txt/csv 不回调。
//
// 原属知识库导入包（knowledgeimport.ExtractTextProgress），2026-11-23 知识库
// 退役后迁到此处，供造价导入等既有消费方使用；提取口径逐字不变。
func extractImportTextFile(path string, progress func(done, total int)) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".markdown", ".txt", ".text":
		b, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(b), nil
	case ".docx", ".xlsx", ".xlsm", ".pdf", ".et", ".ods":
		md, _, _, err := docmd.ConvertLimitProgress(path, "", 50, progress)
		if err != nil {
			return "", err
		}
		// CJK 部首归一（ZCode 蒸馏刀）：Chrome/Skia 打印的 PDF ToUnicode 会把
		// 汉字写成部首兼容字符（⽂/⻩），画面对但检索/引用全错——提取出口
		// 统一归一。
		return cjk.NormalizeCJKCompat(md), nil
	case ".csv", ".tsv":
		b, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(b), nil
	default:
		return "", fmt.Errorf("暂不支持 %s 格式导入", ext)
	}
}
