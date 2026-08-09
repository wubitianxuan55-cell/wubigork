package app

import (
	"fmt"
	"os"

	"github.com/gaea/gaea/internal/office/docmd"
)

// GaeaOCRText 用本地 OvisOCR2 常驻服务提取图片中的文字（办公板块「提取文字」用）。
func (a *App) GaeaOCRText(imagePath string) (string, error) {
	if imagePath == "" {
		return "", fmt.Errorf("缺少图片路径")
	}
	if _, err := os.Stat(imagePath); err != nil {
		return "", fmt.Errorf("图片不存在：%s", imagePath)
	}
	return docmd.OCRImageText(imagePath)
}
