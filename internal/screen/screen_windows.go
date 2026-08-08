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
)

const (
	smXVirtualScreen  = 76
	smYVirtualScreen  = 77
	smCxVirtualScreen = 78
	smCyVirtualScreen = 79
	srcCopy           = 0x00CC0020
	biRGB             = 0
	dibRGBColors      = 0
)

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

// Capture 捕获整个虚拟屏幕（多显示器合并区域），返回 RGBA 图像。
func Capture() (image.Image, error) {
	x := int(sysMetrics(smXVirtualScreen))
	y := int(sysMetrics(smYVirtualScreen))
	w := int(sysMetrics(smCxVirtualScreen))
	h := int(sysMetrics(smCyVirtualScreen))
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("截图：无效屏幕尺寸 %dx%d", w, h)
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

	procSelectObject.Call(hdcMem, hbm)
	procBitBlt.Call(hdcMem, 0, 0, uintptr(w), uintptr(h), hdcScreen, uintptr(x), uintptr(y), srcCopy)

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
