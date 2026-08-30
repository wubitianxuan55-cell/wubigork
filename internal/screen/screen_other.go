//go:build !windows

package screen

import (
	"errors"
	"image"
)

// Capture 非 Windows 平台不支持屏幕捕获。
func Capture() (image.Image, error) {
	return nil, errors.New("截图仅支持 Windows")
}

// CaptureArea 非 Windows 平台不支持屏幕捕获。
func CaptureArea(x, y, w, h int) (image.Image, error) {
	return nil, errors.New("截图仅支持 Windows")
}

// Monitor 单块显示器矩形（非 Windows 平台不填充）。
type Monitor struct {
	X, Y    int
	W, H    int
	Primary bool
}

// Monitors 非 Windows 平台不支持显示器枚举。
func Monitors() ([]Monitor, error) {
	return nil, errors.New("显示器枚举仅支持 Windows")
}
