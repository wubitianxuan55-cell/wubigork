package ai

// 扩图合成器测试（阶段二刀 C，规格 进度计划/gaea-outpaint-20260923.md）：
// 画布尺寸/锚点/蒙版环、边界（四边 0/钳制/非 data URL）、补正方数学。
import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// solidPNGDataURL 纯色 PNG data URL（测试夹具）。
func solidPNGDataURL(t *testing.T, w, h int, c color.Color) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func decodePNGDataURL(t *testing.T, dataURL string) image.Image {
	t.Helper()
	raw, _, err := decodeDataURLBytes(dataURL)
	if err != nil {
		t.Fatalf("decodeDataURLBytes: %v", err)
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("image.Decode: %v", err)
	}
	return img
}

func TestComposeOutpaint_Shape(t *testing.T) {
	// 2×2 红图 + 左 100 / 右 100 → 6×2 画布；原图锚点 x∈[2,4)；蒙版白=左右各 2、黑=中间 2
	canvasURL, maskURL, err := ComposeOutpaint(solidPNGDataURL(t, 2, 2, color.RGBA{R: 255, A: 255}), OutpaintExpand{Left: 100, Right: 100})
	if err != nil {
		t.Fatalf("ComposeOutpaint: %v", err)
	}
	cv := decodePNGDataURL(t, canvasURL)
	if cv.Bounds().Dx() != 6 || cv.Bounds().Dy() != 2 {
		t.Fatalf("画布应 6x2: %v", cv.Bounds())
	}
	r, g, b, _ := cv.At(2, 0).RGBA()
	if r == 0 || g != 0 || b != 0 {
		t.Fatalf("锚点应为原图红: %d %d %d", r, g, b)
	}
	rg, gg, bg, _ := cv.At(0, 0).RGBA()
	if uint8(rg>>8) != 128 || uint8(gg>>8) != 128 || uint8(bg>>8) != 128 {
		t.Fatalf("扩边应为中性灰 128: %d %d %d", uint8(rg>>8), uint8(gg>>8), uint8(bg>>8))
	}
	m := decodePNGDataURL(t, maskURL)
	if m.Bounds().Dx() != 6 || m.Bounds().Dy() != 2 {
		t.Fatalf("蒙版应与画布同尺寸: %v", m.Bounds())
	}
	for x := 0; x < 6; x++ {
		lum, _, _, _ := m.At(x, 0).RGBA()
		isExpand := x < 2 || x >= 4
		if isExpand && lum == 0 {
			t.Fatalf("扩边区蒙版应白: x=%d", x)
		}
		if !isExpand && lum != 0 {
			t.Fatalf("原图区蒙版应黑: x=%d lum=%v", x, lum)
		}
	}
}

func TestComposeOutpaint_BoundsAndClamp(t *testing.T) {
	src := solidPNGDataURL(t, 2, 1, color.RGBA{R: 255, A: 255})
	// 四边全 0
	if _, _, err := ComposeOutpaint(src, OutpaintExpand{}); err == nil {
		t.Fatal("四边全 0 应报错")
	}
	// 非 data URL
	if _, _, err := ComposeOutpaint("/tmp/a.png", OutpaintExpand{Left: 50}); err == nil {
		t.Fatal("非 data URL 应报错")
	}
	// 钳制：Left 999→200、Right -5→0 → 画布 2+2*200/100=6
	canvasURL, _, err := ComposeOutpaint(src, OutpaintExpand{Left: 999, Right: -5})
	if err != nil {
		t.Fatalf("钳制后应成功: %v", err)
	}
	if cv := decodePNGDataURL(t, canvasURL); cv.Bounds().Dx() != 6 {
		t.Fatalf("Left 钳 200、Right 钳 0 → 画布 6x1: %v", cv.Bounds())
	}
	// 上限边界：2048² 四边 50 → 2048+2048×100/100=4096² 恰等于上限（> 判定放行）；
	// 四边 55 → 4300² 超限应拒。
	sq := solidPNGDataURL(t, 2048, 2048, color.RGBA{R: 255, A: 255})
	if _, _, err := ComposeOutpaint(sq, OutpaintExpand{Left: 50, Right: 50, Top: 50, Bottom: 50}); err != nil {
		t.Fatalf("恰好等于上限应放行: %v", err)
	}
	if _, _, err := ComposeOutpaint(sq, OutpaintExpand{Left: 55, Right: 55, Top: 55, Bottom: 55}); err == nil {
		t.Fatal("超上限应报错")
	}
}

func TestOutpaintToSquare(t *testing.T) {
	// 3×1 → 补 3×3：上下各 +1px=各 100%（合计 200%）
	e := OutpaintToSquare(3, 1)
	if e.Top != 100 || e.Bottom != 100 || e.Left != 0 || e.Right != 0 {
		t.Fatalf("3x1 补方应上下各 100%%: %+v", e)
	}
	// 1×3 → 左右合计 200%
	e = OutpaintToSquare(1, 3)
	if e.Left+e.Right != 200 || e.Top != 0 || e.Bottom != 0 {
		t.Fatalf("1x3 补方应左右合计 200%%: %+v", e)
	}
	// 方图 → 零
	if e := OutpaintToSquare(4, 4); e != (OutpaintExpand{}) {
		t.Fatalf("方图应零扩展: %+v", e)
	}
}
