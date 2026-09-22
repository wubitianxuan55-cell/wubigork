package ai

// 抠图合成器（阶段二刀 E，规格 进度计划/gaea-cutout-20260923.md）：涂选保留
// 主体 → 蒙版即 alpha 通道，透明底 PNG 导出（T3「对齐 NovelAI V5 原生透明」）。
// 零新模型零引擎调用——纯图像合成。**抠图蒙版语义与编辑相反：白=保留主体、
// 黑=背景（转透明）**——两个原语（edit 的白=重绘 / cutout 的白=保留）各自契约
// 文档化，蒙版产物都由前端同一涂选组件产出，语义由消费方定义。

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
)

// colorTransparent 全透明像素（抠图背景区）。
func colorTransparent() color.RGBA { return color.RGBA{R: 0, G: 0, B: 0, A: 0} }

// ComposeCutout 抠图合成：原图 + 灰度蒙版（白=保留主体、黑=背景）→ RGBA PNG
// data URL（保留区原像素 alpha=255、背景区 alpha=0）。蒙版与原图尺寸必须一致
// （前端同一画布坐标构造性成立）；边缘硬切（羽化半径挂观察池）。
// 空 mask / 非 data URL / 解码失败 / 尺寸不一致 → 诚实报错。
func ComposeCutout(initImage, maskDataURL string) (string, error) {
	srcRaw, _, err := decodeDataURLBytes(initImage)
	if err != nil {
		return "", fmt.Errorf("抠图需要原图（data URL）: %w", err)
	}
	maskRaw, _, err := decodeDataURLBytes(maskDataURL)
	if err != nil {
		return "", fmt.Errorf("抠图需要蒙版（data URL）: %w", err)
	}
	src, _, err := image.Decode(bytes.NewReader(srcRaw))
	if err != nil {
		return "", fmt.Errorf("抠图原图解析失败: %w", err)
	}
	m, _, err := image.Decode(bytes.NewReader(maskRaw))
	if err != nil {
		return "", fmt.Errorf("抠图蒙版解析失败: %w", err)
	}
	b := src.Bounds()
	mb := m.Bounds()
	if b.Dx() != mb.Dx() || b.Dy() != mb.Dy() {
		return "", fmt.Errorf("蒙版与原图尺寸不一致（%dx%d vs %dx%d）", mb.Dx(), mb.Dy(), b.Dx(), b.Dy())
	}

	// 逐像素合成：蒙版亮度 >127 判保留（白）——红通道即灰度
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(out, out.Bounds(), src, b.Min, draw.Src)
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			r, _, _, _ := m.At(mb.Min.X+x, mb.Min.Y+y).RGBA()
			if uint8(r>>8) <= 127 {
				out.SetRGBA(x, y, colorTransparent())
			}
		}
	}
	return encodePNGDataURL(out)
}
