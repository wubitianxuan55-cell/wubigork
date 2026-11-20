package provider

// W0-6 编码件测试：缩边几何、JPEG 重编码、data URL 形态、token 估算。

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestPNG(t *testing.T, dir, name string, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), A: 255})
		}
	}
	p := filepath.Join(dir, name)
	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return p
}

func TestEncodeImagePart_NoUpscale(t *testing.T) {
	dir := t.TempDir()
	p := writeTestPNG(t, dir, "small.png", 640, 400)
	u, err := EncodeImagePart(p, DefaultImageMaxEdge)
	if err != nil {
		t.Fatalf("EncodeImagePart: %v", err)
	}
	if !strings.HasPrefix(u, "data:image/jpeg;base64,") {
		t.Fatalf("应为 jpeg data URL: %s…", u[:40])
	}
}

func TestEncodeImagePart_ShrinksLongEdge(t *testing.T) {
	dir := t.TempDir()
	p := writeTestPNG(t, dir, "big.png", 2000, 1000)
	u, err := EncodeImagePart(p, DefaultImageMaxEdge)
	if err != nil {
		t.Fatalf("EncodeImagePart: %v", err)
	}
	img := decodeDataURL(t, u)
	b := img.Bounds()
	if b.Dx() != 1280 || b.Dy() != 640 {
		t.Fatalf("长边缩放几何错: got %dx%d want 1280x640", b.Dx(), b.Dy())
	}
}

func decodeDataURL(t *testing.T, u string) image.Image {
	t.Helper()
	const prefix = "data:image/jpeg;base64,"
	if !strings.HasPrefix(u, prefix) {
		t.Fatalf("非 jpeg data URL: %.40s…", u)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(u, prefix))
	if err != nil {
		t.Fatalf("base64: %v", err)
	}
	img, err := jpeg.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("jpeg decode: %v", err)
	}
	return img
}

func TestEncodeImagePart_MissingFile(t *testing.T) {
	if _, err := EncodeImagePart(filepath.Join(t.TempDir(), "nope.png"), DefaultImageMaxEdge); err == nil {
		t.Fatal("缺失文件必须报错（调用方决定跳过）")
	}
}

func TestImageTokenEstimate(t *testing.T) {
	cases := []struct{ w, h, want int }{
		{0, 0, 0},
		{512, 512, 255},    // 1×1=1 tile
		{1024, 512, 425},   // 2×1=2 tiles
		{1280, 640, 1105},  // ceil(1280/512)=3 × ceil(640/512)=2 = 6 tiles
		{2000, 1000, 1445}, // 4×2=8 tiles
	}
	for _, c := range cases {
		if got := ImageTokenEstimate(c.w, c.h); got != c.want {
			t.Fatalf("ImageTokenEstimate(%d,%d)=%d want %d", c.w, c.h, got, c.want)
		}
	}
}
