package app

// v4.6 Verifier 通道 B 真视觉 diff（审计补课：docs/audit-2026-08-30 §C ①——
// 「通道 B 是页数对比非视觉 diff」）。渲染侧用 docmd.RenderPDFPages（poppler
// pdftoppm），比对侧用纯 Go 像素差异率，不引入新依赖。

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
)

// diffSampleGridMin 是最小采样网格边长：低于它不再缩小（防过小图片退化）。
const diffSampleGridMin = 16

// diffLuminanceTolerance 是像素差异容差（0-255 亮度差）：低于它视为同色，
// 吸收抗锯齿/亚像素平移造成的微小色差，避免静态页误报。
const diffLuminanceTolerance = 24

// pixelDiffRatio 计算两张图片的归一化像素差异率（0..1）。
//
// 尺寸不一致时统一缩放到「较小尺寸的一半」采样网格（最低 16px），再做
// 最近邻采样比对——整页缩放/换行导致的尺寸变化不会被简单丢弃，而是以
// 内容网格形式参与比对；逐像素按亮度差 > 容差计入差异像素。
func pixelDiffRatio(aPath, bPath string) (float64, error) {
	a, err := loadImageFile(aPath)
	if err != nil {
		return 0, fmt.Errorf("before 渲染图解析失败: %w", err)
	}
	b, err := loadImageFile(bPath)
	if err != nil {
		return 0, fmt.Errorf("after 渲染图解析失败: %w", err)
	}
	gridW := max(diffSampleGridMin, min(a.Bounds().Dx(), b.Bounds().Dx())/2)
	gridH := max(diffSampleGridMin, min(a.Bounds().Dy(), b.Bounds().Dy())/2)

	changed := 0
	total := gridW * gridH
	for y := 0; y < gridH; y++ {
		for x := 0; x < gridW; x++ {
			// 网格坐标 → 原图像素坐标（最近邻采样；网格端点映射到图内）
			ax := a.Bounds().Min.X + (x * (a.Bounds().Dx() - 1) / max(gridW-1, 1))
			ay := a.Bounds().Min.Y + (y * (a.Bounds().Dy() - 1) / max(gridH-1, 1))
			bx := b.Bounds().Min.X + (x * (b.Bounds().Dx() - 1) / max(gridW-1, 1))
			by := b.Bounds().Min.Y + (y * (b.Bounds().Dy() - 1) / max(gridH-1, 1))
			ar, ag, ab, _ := a.At(ax, ay).RGBA()
			br, bg, bb, _ := b.At(bx, by).RGBA()
			if lumDiff(ar, ag, ab, br, bg, bb) > diffLuminanceTolerance {
				changed++
			}
		}
	}
	if total == 0 {
		return 0, fmt.Errorf("采样网格为空")
	}
	return float64(changed) / float64(total), nil
}

func loadImageFile(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

// lumDiff 计算两像素的感知亮度差（RGBA 16bit → 8bit 亮度，0-255）。
func lumDiff(ar, ag, ab, br, bg, bb uint32) float64 {
	la := 0.299*float64(ar>>8) + 0.587*float64(ag>>8) + 0.114*float64(ab>>8)
	lb := 0.299*float64(br>>8) + 0.587*float64(bg>>8) + 0.114*float64(bb>>8)
	return math.Abs(la - lb)
}
