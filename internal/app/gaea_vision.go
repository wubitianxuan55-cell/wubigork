package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"image/png"
	"strings"

	"github.com/gaea/gaea/internal/gaea/vision"
	"github.com/gaea/gaea/internal/screen"
)

// visionRecognize 可注入 seam（测试替换；生产恒为 vision.RecognizeImage）。
var visionRecognize = vision.RecognizeImage

// visionOCRTextPrompt 微信识图提示词：只提取文字，不描述画面（识别结果直接
// 进入对话管线，描述性输出会污染用户消息语义）。
const visionOCRTextPrompt = "请提取这张图片中的所有文字，按原有版式逐行输出文字内容；不要描述画面、不要添加任何评论。图中没有文字时，用一句话概括画面内容。"

// visionOCRText 微信识图（v4.8.3）：优先多模态主模型（vision 运行时，默认
// 本机 Qwen3.6-35B——真机探针实测视觉链路完好），手写体/低质量图显著强于
// PaddleOCR 专线；主模型失败（服务忙/冷启动超时等）回退 GaeaOCRText
// （PaddleOCR → MinerU → OvisOCR2 三级既有链）。
func (a *App) visionOCRText(imagePath string) (string, error) {
	text, err := visionRecognize(context.Background(), imagePath, visionOCRTextPrompt)
	if err == nil && strings.TrimSpace(text) != "" {
		return strings.TrimSpace(text), nil
	}
	return a.GaeaOCRText(imagePath)
}

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
