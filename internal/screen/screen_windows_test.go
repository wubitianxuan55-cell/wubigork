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

// TestMonitors 显示器枚举（读屏纵深 v4.8）：至少 1 块、矩形非空、恰好一块主屏。
func TestMonitors(t *testing.T) {
	mons, err := Monitors()
	if err != nil {
		t.Fatalf("Monitors: %v", err)
	}
	if len(mons) == 0 {
		t.Fatal("至少应枚举到 1 块显示器")
	}
	primary := 0
	for i, m := range mons {
		if m.W <= 0 || m.H <= 0 {
			t.Errorf("显示器 %d 尺寸异常: %dx%d", i, m.W, m.H)
		}
		if m.Primary {
			primary++
		}
	}
	if primary != 1 {
		t.Errorf("主屏数量 = %d, want 1", primary)
	}
	t.Logf("显示器: %+v", mons)
}

// TestCaptureArea 区域捕获：主屏矩形应产出与矩形同尺寸的可解码图像。
func TestCaptureArea(t *testing.T) {
	mons, err := Monitors()
	if err != nil || len(mons) == 0 {
		t.Skipf("显示器枚举不可用（err=%v）", err)
	}
	m := mons[0]
	img, err := CaptureArea(m.X, m.Y, m.W, m.H)
	if err != nil {
		t.Fatalf("CaptureArea: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != m.W || b.Dy() != m.H {
		t.Errorf("捕获尺寸 = %dx%d, want %dx%d", b.Dx(), b.Dy(), m.W, m.H)
	}
	// 无效尺寸应被拒绝（CaptureArea 参数校验）
	if _, err := CaptureArea(0, 0, 0, 0); err == nil {
		t.Error("CaptureArea(0,0,0,0) 应返回错误")
	}
}
