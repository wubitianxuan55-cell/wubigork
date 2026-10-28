package cad

import (
	"context"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRenderPDFPage 单页渲染（ask 档）：落盘在 cacheDir/ask 前缀下、DPI 真实
// 传播（两档渲染宽度比≈DPI 比——问图纸清晰度档的根基断言）。pdftoppm 缺席
// 环境跳过（与 OpenPDF 系用例同口径）。
func TestRenderPDFPage(t *testing.T) {
	path := buildPDFPages(t, 3) // MediaBox 612×792pt = 8.5×11in
	cache := t.TempDir()

	lo, err := RenderPDFPage(context.Background(), path, cache, 1, 36)
	if err != nil {
		if strings.Contains(err.Error(), "pdftoppm") {
			t.Skipf("pdftoppm unavailable: %v", err)
		}
		t.Fatal(err)
	}
	// 渲完立刻量：两渲同前缀同名输出（单页无零填充），后渲会覆写前渲的文件
	loW := pngWidth(t, lo)
	hi, err := RenderPDFPage(context.Background(), path, cache, 1, 144)
	if err != nil {
		t.Fatal(err)
	}
	hiW := pngWidth(t, hi)
	for _, p := range []string{lo, hi} {
		if filepath.Dir(p) != cache || !strings.Contains(filepath.Base(p), "ask") {
			t.Fatalf("render out = %q, want under cacheDir with ask prefix", p)
		}
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("rendered png missing: %v", err)
		}
	}
	// 8.5in × 36dpi ≈ 306px，×144dpi ≈ 1224px——比值≈DPI 比（容差 1px 抖动）
	if loW < 300 || loW > 312 {
		t.Fatalf("36dpi width = %d, want ≈306", loW)
	}
	if hiW < 1220 || hiW > 1232 {
		t.Fatalf("144dpi width = %d, want ≈1224", hiW)
	}

	// 越界页：如实报错（不静默回别的页）
	if _, err := RenderPDFPage(context.Background(), path, cache, 99, 72); err == nil {
		t.Fatal("page 越界应报错")
	}
}

func pngWidth(t *testing.T, p string) int {
	t.Helper()
	f, err := os.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return img.Bounds().Dx()
}
