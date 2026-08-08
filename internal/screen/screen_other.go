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
