//go:build windows

package screen

import (
	"bytes"
	"image/png"
	"testing"
)

// TestCapture 验证 GDI 截图能产出可解码的非空图像（真实屏幕）。
func TestCapture(t *testing.T) {
	img, err := Capture()
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		t.Fatalf("截图尺寸异常: %dx%d", b.Dx(), b.Dy())
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("PNG 编码失败: %v", err)
	}
	if len(buf.Bytes()) < 8 || !bytes.Equal(buf.Bytes()[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		t.Fatalf("输出不是 PNG（%d 字节）", len(buf.Bytes()))
	}
	t.Logf("截图尺寸: %dx%d, %d 字节", b.Dx(), b.Dy(), len(buf.Bytes()))
}
