package ai

// 抠图合成器测试（阶段二刀 E）：白=保留原像素 alpha255、黑=透明、尺寸校验。
import (
	"image/color"
	"testing"
)

func TestComposeCutout(t *testing.T) {
	// 2×1：左红右蓝；蒙版左白右黑 → 左保留右透明
	src := solidPNGDataURL(t, 2, 1, color.RGBA{R: 255, A: 255})
	// 蒙版须左右异色——手写灰度（realGrayPNGDataURL 是半幅白，正好：左白右黑）
	mask := realGrayPNGDataURL(t, 2, 1, true)
	out, err := ComposeCutout(src, mask)
	if err != nil {
		t.Fatalf("ComposeCutout: %v", err)
	}
	m := decodePNGDataURL(t, out)
	if m.Bounds().Dx() != 2 || m.Bounds().Dy() != 1 {
		t.Fatalf("尺寸应不变: %v", m.Bounds())
	}
	r, g, b, a := m.At(0, 0).RGBA()
	if r == 0 || g != 0 || a != 0xffff {
		t.Fatalf("白区应保留原红像素不透明: %d %d %d %d", r, g, b, a)
	}
	_, _, _, a1 := m.At(1, 0).RGBA()
	if a1 != 0 {
		t.Fatalf("黑区应透明: %d", a1)
	}
}

func TestComposeCutout_Bounds(t *testing.T) {
	src := solidPNGDataURL(t, 2, 1, color.RGBA{R: 255, A: 255})
	// 尺寸不一致
	if _, err := ComposeCutout(src, realGrayPNGDataURL(t, 4, 1, true)); err == nil {
		t.Fatal("尺寸不一致应报错")
	}
	// 空 mask / 非 data URL
	if _, err := ComposeCutout(src, ""); err == nil {
		t.Fatal("空蒙版应报错")
	}
	if _, err := ComposeCutout("/tmp/a.png", realGrayPNGDataURL(t, 2, 1, true)); err == nil {
		t.Fatal("非 data URL 原图应报错")
	}
}
