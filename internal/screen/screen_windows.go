//go:build windows

// Package screen 提供屏幕捕获能力（供 gaea 截图工具与桌面端绑定共用）。
package screen

import (
	"errors"
	"fmt"
	"image"
	"syscall"
	"unsafe"
)

var (
	modUser32 = syscall.NewLazyDLL("user32.dll")
	modGdi32  = syscall.NewLazyDLL("gdi32.dll")

	procGetSystemMetrics   = modUser32.NewProc("GetSystemMetrics")
	procGetDC              = modUser32.NewProc("GetDC")
	procReleaseDC          = modUser32.NewProc("ReleaseDC")
	procCreateCompatibleDC = modGdi32.NewProc("CreateCompatibleDC")
	procDeleteDC           = modGdi32.NewProc("DeleteDC")
	procCreateDIBSection   = modGdi32.NewProc("CreateDIBSection")
	procSelectObject       = modGdi32.NewProc("SelectObject")
	procDeleteObject       = modGdi32.NewProc("DeleteObject")
	procBitBlt             = modGdi32.NewProc("BitBlt")

	// 多显示器枚举（读屏纵深 v4.8）。
	procEnumDisplayMonitors = modUser32.NewProc("EnumDisplayMonitors")
	procGetMonitorInfoW     = modUser32.NewProc("GetMonitorInfoW")
)

const (
	smXVirtualScreen  = 76
	smYVirtualScreen  = 77
	smCxVirtualScreen = 78
	smCyVirtualScreen = 79
	srcCopy           = 0x00CC0020
	biRGB             = 0
	dibRGBColors      = 0

	monitorInfofPrimary = 0x1 // MONITORINFOF_PRIMARY

	// hgdiError 是 GDI 约定的失败返回值 (HGDIOBJ)-1（SelectObject 常规失败
	// 返回 NULL=0，区域选择失败返回 HGDI_ERROR）。
	hgdiError = ^uintptr(0)
)

// gdiCallError 归一化 GDI 调用失败原因：优先取 syscall.GetLastError()（本
// 线程最近一次失败的错误码），退化用 LazyProc.Call 返回的错误。GDI 成功时
// 错误码常是残留值，故只在 r==0 的失败分支调用。
func gdiCallError(callErr error) error {
	if e := syscall.GetLastError(); e != nil {
		return e
	}
	if callErr != nil {
		return callErr
	}
	return errors.New("未知 GDI 错误")
}

type bitmapInfoHeader struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type bitmapInfo struct {
	BmiHeader bitmapInfoHeader
	BmiColors [1]uint32
}

// Monitor 单块显示器的虚拟桌面矩形（读屏纵深 v4.8）。
type Monitor struct {
	X, Y    int
	W, H    int
	Primary bool
}

// gdiRect 与 Win32 RECT 对齐。
type gdiRect struct {
	Left, Top, Right, Bottom int32
}

// monitorInfo 与 Win32 MONITORINFO 对齐（GetMonitorInfoW 用）。
type monitorInfo struct {
	CbSize    uint32
	RcMonitor gdiRect
	RcWork    gdiRect
	DwFlags   uint32
}

// Monitors 枚举所有显示器（虚拟桌面坐标系）。单屏机器返回 1 条且 Primary=true；
// 失败返回错误（调用方诚实回退整屏捕获）。
func Monitors() ([]Monitor, error) {
	var out []Monitor
	cb := syscall.NewCallback(func(hmon, hdc, clip, lparam uintptr) uintptr {
		var mi monitorInfo
		mi.CbSize = uint32(unsafe.Sizeof(mi))
		r, _, _ := procGetMonitorInfoW.Call(hmon, uintptr(unsafe.Pointer(&mi)))
		if r == 0 {
			return 1 // 单块信息获取失败不中断枚举（宁漏勿误：少一块好过硬失败）
		}
		out = append(out, Monitor{
			X:       int(mi.RcMonitor.Left),
			Y:       int(mi.RcMonitor.Top),
			W:       int(mi.RcMonitor.Right - mi.RcMonitor.Left),
			H:       int(mi.RcMonitor.Bottom - mi.RcMonitor.Top),
			Primary: mi.DwFlags&monitorInfofPrimary != 0,
		})
		return 1 // 继续枚举
	})
	r, _, callErr := procEnumDisplayMonitors.Call(0, 0, cb, 0)
	if r == 0 {
		return nil, fmt.Errorf("枚举显示器失败: %v", callErr)
	}
	if len(out) == 0 {
		return nil, errors.New("未枚举到任何显示器")
	}
	return out, nil
}

// Capture 捕获整个虚拟屏幕（多显示器合并区域），返回 RGBA 图像。
func Capture() (image.Image, error) {
	x := int(sysMetrics(smXVirtualScreen))
	y := int(sysMetrics(smYVirtualScreen))
	w := int(sysMetrics(smCxVirtualScreen))
	h := int(sysMetrics(smCyVirtualScreen))
	return CaptureArea(x, y, w, h)
}

// CaptureArea 捕获虚拟桌面坐标系中的指定矩形（读屏纵深 v4.8：单显示器
// 选择的底层能力；Capture() 是它对整个虚拟屏的薄封装，行为不变）。
func CaptureArea(x, y, w, h int) (image.Image, error) {
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("截图：无效屏幕尺寸 %dx%d", w, h)
	}
	// 源矩形必须与虚拟桌面相交。GDI 对被完全裁剪到桌面外的源矩形会「成功」
	// 返回一张全黑位图（BitBlt 返回非 0），只看返回码查不出这种「成功但
	// 空白」（审计 IN3-15 点名的坐标越界一例，实测非黑像素恒为 0）。
	vx := int(sysMetrics(smXVirtualScreen))
	vy := int(sysMetrics(smYVirtualScreen))
	vw := int(sysMetrics(smCxVirtualScreen))
	vh := int(sysMetrics(smCyVirtualScreen))
	if vw > 0 && vh > 0 && (x+w <= vx || x >= vx+vw || y+h <= vy || y >= vy+vh) {
		return nil, fmt.Errorf("截图：区域 (%d,%d %dx%d) 与虚拟桌面 (%d,%d %dx%d) 无交集",
			x, y, w, h, vx, vy, vw, vh)
	}

	hdcScreen := getDC(0)
	if hdcScreen == 0 {
		return nil, errors.New("截图：获取屏幕 DC 失败")
	}
	defer releaseDC(0, hdcScreen)

	hdcMem := createCompatibleDC(hdcScreen)
	if hdcMem == 0 {
		return nil, errors.New("截图：创建内存 DC 失败")
	}
	defer deleteDC(hdcMem)

	// 负高度 = 自上而下 DIB，行序与屏幕一致。
	bmi := bitmapInfo{BmiHeader: bitmapInfoHeader{
		BiSize:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		BiWidth:       int32(w),
		BiHeight:      -int32(h),
		BiPlanes:      1,
		BiBitCount:    32,
		BiCompression: biRGB,
	}}
	var bits unsafe.Pointer
	hbm, _, _ := procCreateDIBSection.Call(hdcScreen, uintptr(unsafe.Pointer(&bmi)), dibRGBColors, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if hbm == 0 {
		return nil, errors.New("截图：创建 DIB 失败")
	}
	defer deleteObject(hbm)

	// SelectObject/BitBlt 的返回码必须检查：失败时下方把 DIB 内存当有效像素
	// 读出来会得到未初始化/全黑图像，却按成功返回（审计 IN3-15）。资源释放
	// 仍由上方 defer 负责（deleteObject/deleteDC/releaseDC）。
	hOld, _, selErr := procSelectObject.Call(hdcMem, hbm)
	if hOld == 0 || hOld == hgdiError {
		return nil, fmt.Errorf("截图：选择 DIB 到内存 DC 失败: %v", gdiCallError(selErr))
	}
	r, _, bltErr := procBitBlt.Call(hdcMem, 0, 0, uintptr(w), uintptr(h), hdcScreen, uintptr(x), uintptr(y), srcCopy)
	if r == 0 {
		return nil, fmt.Errorf("截图：BitBlt 拷贝屏幕区域 (%d,%d %dx%d) 失败: %v",
			x, y, w, h, gdiCallError(bltErr))
	}

	stride := w * 4
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	src := unsafe.Slice((*byte)(bits), stride*h)
	for row := 0; row < h; row++ {
		dst := img.Pix[row*img.Stride : row*img.Stride+stride]
		srcRow := src[row*stride : (row+1)*stride]
		for i := 0; i < stride; i += 4 {
			// GDI 不写 alpha，强制不透明；BGRA → RGBA。
			dst[i+0] = srcRow[i+2]
			dst[i+1] = srcRow[i+1]
			dst[i+2] = srcRow[i+0]
			dst[i+3] = 255
		}
	}
	return img, nil
}

func sysMetrics(index int) int {
	r, _, _ := procGetSystemMetrics.Call(uintptr(index))
	return int(r)
}

func getDC(hwnd int) uintptr {
	r, _, _ := procGetDC.Call(uintptr(hwnd))
	return r
}

func releaseDC(hwnd int, hdc uintptr) {
	_, _, _ = procReleaseDC.Call(uintptr(hwnd), hdc)
}

func createCompatibleDC(hdc uintptr) uintptr {
	r, _, _ := procCreateCompatibleDC.Call(hdc)
	return r
}

func deleteDC(hdc uintptr) {
	_, _, _ = procDeleteDC.Call(hdc)
}

func deleteObject(h uintptr) {
	_, _, _ = procDeleteObject.Call(h)
}
