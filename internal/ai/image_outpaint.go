package ai

// 扩图合成器（阶段二刀 C，规格 进度计划/gaea-outpaint-20260923.md）：
// 扩图 ≡ 合成大画布（原图锚定 + 四边扩展填中性灰）+ 灰度蒙版（扩边环白=重绘），
// 即 v4.393 已打通的 edit+mask 请求形态——两个后端（OpenAI 兼容 /images/edits 的
// mask 字段 + ComfyUI SetLatentNoiseMask 链）零引擎改动即可消费。本文件只做
// 纯函数合成，供 app 层在 mode=outpaint 时转换为 edit 请求下发。

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
)

// OutpaintExpand 四边扩展百分比（相对原图对应边长，0–200）。
type OutpaintExpand struct {
	Left   int `json:"left"`
	Top    int `json:"top"`
	Right  int `json:"right"`
	Bottom int `json:"bottom"`
}

// outpaintMaxPixels 合成画布像素上限（4096²≈16.7MP）：防 base64 传输爆炸；
// ComfyUI 分支会经 FluxKontextImageScale 重标到 1MP 档，上限只约束合成输入。
const outpaintMaxPixels = 4096 * 4096

// ComposeOutpaint 扩图合成：返回画布与蒙版（均为 PNG data URL，gaea 统一
// 蒙版契约白=重绘区）。画布新区域填中性灰 128（对噪声初始化最中性）；
// 锚点由四边扩展量自然决定（Left 在左留白、Top 在上留白）。
// 四边全 0 / 超上限 / 非 data URL / 解码失败 → 诚实报错。
func ComposeOutpaint(initImage string, e OutpaintExpand) (canvas, mask string, err error) {
	raw, _, err := decodeDataURLBytes(initImage)
	if err != nil {
		return "", "", fmt.Errorf("扩图需要原图（data URL）: %w", err)
	}
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return "", "", fmt.Errorf("扩图原图解析失败: %w", err)
	}
	clamp := func(v int) int {
		if v < 0 {
			return 0
		}
		if v > 200 {
			return 200
		}
		return v
	}
	l, t, r, b := clamp(e.Left), clamp(e.Top), clamp(e.Right), clamp(e.Bottom)
	if l+t+r+b == 0 {
		return "", "", fmt.Errorf("扩图需要至少一边扩展量大于 0")
	}
	ow, oh := src.Bounds().Dx(), src.Bounds().Dy()
	offX := ow * l / 100
	offY := oh * t / 100
	cw := ow + ow*(l+r)/100
	ch := oh + oh*(t+b)/100
	if cw <= 0 || ch <= 0 || cw > outpaintMaxPixels || ch > outpaintMaxPixels || cw*ch > outpaintMaxPixels {
		return "", "", fmt.Errorf("扩图后画布过大（%dx%d，上限 %d 像素）", cw, ch, outpaintMaxPixels)
	}

	// 画布：整幅中性灰 128 → 原图贴锚点
	out := image.NewRGBA(image.Rect(0, 0, cw, ch))
	draw.Draw(out, out.Bounds(), &image.Uniform{C: color.Gray{Y: 128}}, image.Point{}, draw.Src)
	draw.Draw(out, image.Rect(offX, offY, offX+ow, offY+oh), src, src.Bounds().Min, draw.Src)

	// 蒙版：黑底（保留）→ 扩边环白（重绘）——四条矩形与原图区互补
	m := image.NewGray(image.Rect(0, 0, cw, ch))
	white := &image.Uniform{C: color.Gray{Y: 255}}
	draw.Draw(m, image.Rect(0, 0, cw, offY), white, image.Point{}, draw.Src)             // 上
	draw.Draw(m, image.Rect(0, offY+oh, cw, ch), white, image.Point{}, draw.Src)         // 下
	draw.Draw(m, image.Rect(0, offY, offX, offY+oh), white, image.Point{}, draw.Src)     // 左
	draw.Draw(m, image.Rect(offX+ow, offY, cw, offY+oh), white, image.Point{}, draw.Src) // 右

	canvas, err = encodePNGDataURL(out)
	if err != nil {
		return "", "", fmt.Errorf("扩图画布编码失败: %w", err)
	}
	mask, err = encodePNGDataURL(m)
	if err != nil {
		return "", "", fmt.Errorf("扩图蒙版编码失败: %w", err)
	}
	return canvas, mask, nil
}

// OutpaintToSquare 计算把 w×h 补成正方形（1:1）的四边扩展量：长边不动、
// 短边方向均分差额（结果百分比可能非整数向下取整，容差 1px 由上游消化）。
func OutpaintToSquare(w, h int) OutpaintExpand {
	if w <= 0 || h <= 0 {
		return OutpaintExpand{}
	}
	if w == h {
		return OutpaintExpand{}
	}
	if w > h {
		pct := int(math.Round(float64(w-h) / float64(h) * 100))
		return OutpaintExpand{Top: pct / 2, Bottom: pct - pct/2}
	}
	pct := int(math.Round(float64(h-w) / float64(w) * 100))
	return OutpaintExpand{Left: pct / 2, Right: pct - pct/2}
}

// encodePNGDataURL 便捷编码：image → PNG → data URL。
func encodePNGDataURL(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
