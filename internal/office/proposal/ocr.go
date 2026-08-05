// Package proposal — 扫描件 OCR 管线（可插拔，默认 Python fitz + rapidocr）
package proposal

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// OCRProvider 扫描件/图片 OCR 提供者
type OCRProvider interface {
	OCR(ctx context.Context, filePath string) (string, error)
}

// SetOCRProviderForTest 覆盖 OCR 提供者（测试隔离）
func (s *Service) SetOCRProviderForTest(p OCRProvider) { s.ocr = p }

// ensureOCR 懒加载默认 OCR（Python fitz + rapidocr_onnxruntime）
func (s *Service) ensureOCR() {
	if s.ocr != nil {
		return
	}
	if p, ok := DetectPythonOCR(); ok {
		s.ocr = p
	}
}

// pythonOCR 基于 Python 的 OCR：PyMuPDF 渲染 PDF 页面 → rapidocr 识别
type pythonOCR struct {
	pythonBin string
}

const pythonOCRScript = `
import sys, tempfile, os
import fitz
from rapidocr_onnxruntime import RapidOCR

engine = RapidOCR()
path = sys.argv[1]
pages = []

def ocr_image(img_path):
    result, _ = engine(img_path)
    if result:
        return "\n".join(line[1] for line in result)
    return ""

if path.lower().endswith(".pdf"):
    doc = fitz.open(path)
    for i in range(len(doc)):
        pix = doc[i].get_pixmap(dpi=300)
        fd, tmp = tempfile.mkstemp(suffix=".png")
        os.close(fd)
        pix.save(tmp)
        try:
            text = ocr_image(tmp)
            if text:
                pages.append("【第 %d 页】\n%s" % (i + 1, text))
        finally:
            os.unlink(tmp)
else:
    text = ocr_image(path)
    if text:
        pages.append(text)

print("\n\n".join(pages))
`

// DetectPythonOCR 检测 python + fitz + rapidocr 是否可用
func DetectPythonOCR() (OCRProvider, bool) {
	for _, bin := range []string{"python", "python3"} {
		cmd := exec.Command(bin, "-c", "import fitz, rapidocr_onnxruntime")
		if cmd.Run() == nil {
			return &pythonOCR{pythonBin: bin}, true
		}
	}
	return nil, false
}

func (p *pythonOCR) OCR(ctx context.Context, filePath string) (string, error) {
	if !isOCRFile(filePath) {
		return "", fmt.Errorf("OCR 仅支持 PDF/图片: %s", filePath)
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, p.pythonBin, "-c", pythonOCRScript, filePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("OCR 识别失败: %w\n输出: %s", err, truncate(string(out), 500))
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		return "", fmt.Errorf("OCR 未识别到文字（请检查扫描清晰度或安装中文语言包）")
	}
	return text, nil
}

func isOCRFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".pdf", ".png", ".jpg", ".jpeg", ".bmp", ".tif", ".tiff":
		return true
	}
	return false
}
