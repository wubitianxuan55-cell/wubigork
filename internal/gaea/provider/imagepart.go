package provider

// W0 多模态旁路的编码件（电脑操控施工单 2026-11-19 W0-6）。
// 会话里消息只存本地路径；本文件负责在请求序列化边界把路径转成
// OpenAI 兼容 image_url 所需的 data URL，并给出保守的 token 估算。

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/gif" // 注册解码器
	_ "image/png" // 注册解码器
	"os"

	"golang.org/x/image/draw"
)

const (
	// DefaultImageMaxEdge 进请求的图片长边上限（施工单 W0：长边 ≤1280）。
	DefaultImageMaxEdge = 1280
	// imagePartJPEGQuality 统一 JPEG 重编码质量（对齐 codex 截图编码口径）。
	imagePartJPEGQuality = 70
)

// EncodeImagePart 把本地图片编码为 data URL。长边超过 maxEdge 时等比缩到
// maxEdge（CatmullRom）；无论原图格式统一 JPEG 重编码（q=70，截图体积可控）。
// 文件缺失/非图片返回错误，由调用方决定跳过（不拖垮整个请求）。
func EncodeImagePart(path string, maxEdge int) (string, error) {
	if maxEdge <= 0 {
		maxEdge = DefaultImageMaxEdge
	}
	f, err := os.Open(path) //nolint:gosec // 路径来自工具结果落盘，非用户输入的任意读
	if err != nil {
		return "", fmt.Errorf("image part: %w", err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return "", fmt.Errorf("image part: decode %s: %w", path, err)
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return "", fmt.Errorf("image part: empty image %s", path)
	}
	if w > maxEdge || h > maxEdge {
		scale := float64(maxEdge) / float64(max(w, h))
		nw := max(1, int(float64(w)*scale))
		nh := max(1, int(float64(h)*scale))
		dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
		img = dst
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: imagePartJPEGQuality}); err != nil {
		return "", fmt.Errorf("image part: encode %s: %w", path, err)
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// ImageTokenEstimate 按 OpenAI 高细节 tile 公式保守估算图片 token
// （85 基础 + 每 512×512 tile 170）。仅供预算/审计参考，不参与请求体。
func ImageTokenEstimate(w, h int) int {
	if w <= 0 || h <= 0 {
		return 0
	}
	tiles := ((w + 511) / 512) * ((h + 511) / 512)
	return 85 + tiles*170
}
