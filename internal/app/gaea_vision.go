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

// visionOCRTextPrompt 微信识图提示词：完整解读——文字按版式逐行提取优先
// （保持原文），画面主体/对象/布局/图表要点随后；两者都要（用户发图后常
// 用下一条文本追问，注入描述必须足以支撑追问，纯文字提取会丢视觉语境）。
const visionOCRTextPrompt = "请完整解读这张图片：先按原版式逐行提取所有可见文字（保持原文，不翻译不改写）；再简述画面主体、对象、布局与图表要点（如有）。"

// visionOCRText 微信识图（v4.8.3）：优先多模态主模型（vision 运行时，默认
// 本机 Qwen3.6-35B——真机探针实测视觉链路完好），手写体/低质量图显著强于
// PaddleOCR 专线；主模型失败（服务忙/冷启动超时等）回退 GaeaOCRText
// （PaddleOCR → MinerU → OvisOCR2 三级既有链）。
// 注意：本函数是「看懂图」不是纯 OCR——文字提取+画面理解都要；纯文字提取
// 场景（办公提取文字工具）继续走 GaeaOCRText，两者分岗。
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
	// T0 图像域试点：识图-懂 先经域能力注册表校验（可用性恒定，行为不变）。
	if _, err := imageDomainEntry(CapabilityVisionUnderstand); err != nil {
		return "", err
	}
	return vision.RecognizeImage(context.Background(), imagePath, prompt)
}
