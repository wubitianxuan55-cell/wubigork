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

// TestCaptureAreaOffVirtualScreenErrors 审计 IN3-15：完全落在虚拟桌面外的
// 源矩形会被 GDI 裁剪成全黑位图、BitBlt 返回非 0，只看返回码查不出「成功但
// 空白」。修复前这里返回 err=nil 的全黑图（探针实测非黑像素 0/256）。
func TestCaptureAreaOffVirtualScreenErrors(t *testing.T) {
	if _, err := CaptureArea(1<<20, 1<<20, 16, 16); err == nil {
		t.Error("区域与虚拟桌面无交集时必须返回错误，实际 err=nil（全黑图被当成成功）")
	}
	if _, err := CaptureArea(-(1 << 20), -(1 << 20), 16, 16); err == nil {
		t.Error("负向桌面外区域必须返回错误，实际 err=nil")
	}
}

// TestCaptureAreaPartialOverlapAllowed 对照：与桌面只是部分相交（一半在外）
// 的区域仍照旧捕获——越界保护只拒绝「零交集」，不缩小既有可行路径。
func TestCaptureAreaPartialOverlapAllowed(t *testing.T) {
	vx := int(sysMetrics(smXVirtualScreen))
	vy := int(sysMetrics(smYVirtualScreen))
	img, err := CaptureArea(vx-8, vy-8, 16, 16)
	if err != nil {
		t.Fatalf("部分相交区域应可捕获: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 16 || b.Dy() != 16 {
		t.Errorf("捕获尺寸 = %dx%d, want 16x16", b.Dx(), b.Dy())
	}
}

// TestGdiFailureYieldsError 钉死 CaptureArea 里 r==0 分支的形状：无法靠
// CaptureArea 的参数稳定注入 GDI 失败（源矩形越界实测 BitBlt 返回非 0 的
// 全黑帧），故直接对空 DC 调用同两个 GDI 函数——r==0 时 gdiCallError 必须
// 给出非 nil 错误，而不是把 DIB 内存当有效像素返回（审计 IN3-15）。
func TestGdiFailureYieldsError(t *testing.T) {
	r, _, selErr := procSelectObject.Call(0, 0)
	if r != 0 {
		t.Fatalf("空 DC 上 SelectObject 应失败，实际 r=%d", r)
	}
	if gdiCallError(selErr) == nil {
		t.Error("SelectObject 失败必须产出非 nil 错误")
	}
	br, _, bltErr := procBitBlt.Call(0, 0, 0, 1, 1, 0, 0, 0, srcCopy)
	if br != 0 {
		t.Fatalf("空 DC 上 BitBlt 应失败，实际 r=%d", br)
	}
	if gdiCallError(bltErr) == nil {
		t.Error("BitBlt 失败必须产出非 nil 错误")
	}
}
