package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"image/png"

	"github.com/gaea/gaea/internal/gaea/vision"
	"github.com/gaea/gaea/internal/screen"
)

// GaeaCaptureScreen 捕获整个屏幕，返回 PNG data URL（办公板块截图用）。
func (a *App) GaeaCaptureScreen() (string, error) {
	img, err := screen.Capture()
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// GaeaRecognizeImage 用本地视觉模型识别图片，返回文本描述（办公板块识图用）。
func (a *App) GaeaRecognizeImage(imagePath, prompt string) (string, error) {
	return vision.RecognizeImage(context.Background(), imagePath, prompt)
}
