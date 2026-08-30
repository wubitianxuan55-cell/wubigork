package app

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// writePNG 生成一张 WxH 的测试图（fill = 全图填充色，改色区域 rect）。
func writePNG(t *testing.T, path string, w, h int, fill color.RGBA, rect image.Rectangle, rectColor color.RGBA) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, fill)
		}
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.Set(x, y, rectColor)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
}

// v4.6 Verifier 通道 B 真视觉 diff：像素差异率纯函数。
func TestPixelDiffRatio(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.png")
	b := filepath.Join(dir, "b.png")
	c := filepath.Join(dir, "c.png")

	// 相同图 → 0
	writePNG(t, a, 200, 120, color.RGBA{255, 255, 255, 255}, image.Rect(0, 0, 0, 0), color.RGBA{})
	writePNG(t, b, 200, 120, color.RGBA{255, 255, 255, 255}, image.Rect(0, 0, 0, 0), color.RGBA{})
	r, err := pixelDiffRatio(a, b)
	if err != nil {
		t.Fatalf("pixelDiffRatio: %v", err)
	}
	if r != 0 {
		t.Fatalf("相同图差异率 = %f, want 0", r)
	}

	// 右半图涂黑 → 约 0.5（采样网格 100x60 = 6000 点，100% 采样比）
	writePNG(t, c, 200, 120, color.RGBA{255, 255, 255, 255}, image.Rect(100, 0, 200, 120), color.RGBA{0, 0, 0, 255})
	r, err = pixelDiffRatio(a, c)
	if err != nil {
		t.Fatalf("pixelDiffRatio: %v", err)
	}
	if r < 0.45 || r > 0.55 {
		t.Fatalf("半图差异率 = %f, want ≈0.5", r)
	}

	// 尺寸不同（整页缩放模拟）也能比对且不崩溃
	d := filepath.Join(dir, "d.png")
	writePNG(t, d, 100, 60, color.RGBA{255, 255, 255, 255}, image.Rect(0, 0, 0, 0), color.RGBA{})
	r, err = pixelDiffRatio(a, d)
	if err != nil {
		t.Fatalf("尺寸不同: %v", err)
	}
	if r != 0 {
		t.Fatalf("等比缩放差异率 = %f, want 0", r)
	}

	// 不存在文件 → 错误
	if _, err := pixelDiffRatio(a, filepath.Join(dir, "missing.png")); err == nil {
		t.Fatal("缺失文件应报错")
	}
}

// v4.6 通道 B 汇总：渲染/比对 seam 注入后，真视觉 diff 产出 pass/warn/fail。
func TestRunVisualDiffVerdicts(t *testing.T) {
	dir := t.TempDir()
	verifyDir := filepath.Join(dir, "verify")
	if err := os.MkdirAll(verifyDir, 0o755); err != nil {
		t.Fatalf("mkdir verify: %v", err)
	}
	base := filepath.Join(dir, "base.xlsx")
	target := filepath.Join(dir, "target.xlsx")
	writePNG(t, base, 4, 4, color.RGBA{255, 255, 255, 255}, image.Rect(0, 0, 0, 0), color.RGBA{})
	writePNG(t, target, 4, 4, color.RGBA{255, 255, 255, 255}, image.Rect(0, 0, 0, 0), color.RGBA{})

	oldConv, oldRender, oldDiff := verifyConvertToPdf, verifyRenderPages, verifyPixelDiff
	t.Cleanup(func() {
		verifyConvertToPdf, verifyRenderPages, verifyPixelDiff = oldConv, oldRender, oldDiff
	})

	// 转换与渲染 stub：直接把预置 PNG 当 PDF 处理（通道 B 只看渲染产物）
	verifyConvertToPdf = func(src, out string) error {
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		return os.WriteFile(out, data, 0o644)
	}
	// 渲染 stub：按 PDF 文件名配置页数（默认 1；before/after 可不同以模拟
	// 版式页数变化）
	pageCounts := map[string]int{}
	verifyRenderPages = func(pdf, prefix string, dpi int) ([]string, error) {
		n := pageCounts[filepath.Base(pdf)]
		if n <= 0 {
			n = 1
		}
		out := make([]string, 0, n)
		for i := 1; i <= n; i++ {
			p := filepath.Join(filepath.Dir(prefix), filepath.Base(prefix)+"-"+string(rune('0'+i))+".png")
			if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
				return nil, err
			}
			out = append(out, p)
		}
		return out, nil
	}
	// 相同内容 → pass（像素差异 0）
	verifyPixelDiff = func(a, b string) (float64, error) { return 0, nil }
	if msg, st := runVisualDiff(base, target, verifyDir); st != "pass" {
		t.Fatalf("相同内容 = %q/%q, want pass", st, msg)
	}

	// 视觉变化（页数相同）→ warn
	verifyPixelDiff = func(a, b string) (float64, error) { return 0.05, nil }
	if msg, st := runVisualDiff(base, target, verifyDir); st != "warn" {
		t.Fatalf("中改 = %q/%q, want warn", st, msg)
	}

	// 大改 + 页数变化 → fail
	pageCounts["before.pdf"] = 2
	pageCounts["after.pdf"] = 3
	verifyPixelDiff = func(a, b string) (float64, error) { return 0.5, nil }
	msg, st := runVisualDiff(base, target, verifyDir)
	if st != "fail" {
		t.Fatalf("大改+页数变化 = %q/%q, want fail", st, msg)
	}
	if !filepath.IsAbs(verifyDir) || verifyDir == "" {
		t.Fatalf("verifyDir 异常: %q", verifyDir)
	}

	// 渲染不可用 → 降级 warn（不误判 fail）
	pageCounts = map[string]int{}
	verifyRenderPages = func(pdf, prefix string, dpi int) ([]string, error) {
		return nil, os.ErrNotExist
	}
	if msg, st := runVisualDiff(base, target, verifyDir); st != "warn" {
		t.Fatalf("渲染降级 = %q/%q, want warn", st, msg)
	}
}
