//go:build windows

package computer

// 截图采集 P0 兜底（施工单 W2/W3）：复用 internal/screen 的 GDI BitBlt
// 全屏抓图（window 抓屏/遮挡可用是 P2 的 WGC）。落盘 JPEG q70，
// Shot.Path 供给人看（W6 起 .gaea/uploads）与 bridge 编码。

import (
	"fmt"
	"image/jpeg"
	"os"
	"path/filepath"
	"time"

	"github.com/gaea/gaea/internal/screen"
)

// captureFullScreen 全屏截图 → Shot（Width/Height=快照像素；P0 全屏单区域，
// 窗口相对坐标换算在输入注入层处理）。
func captureFullScreen() (*Shot, error) {
	img, err := screen.Capture()
	if err != nil {
		return nil, fmt.Errorf("computer: capture: %w", err)
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("computer: capture: empty frame")
	}
	path := filepath.Join(os.TempDir(), fmt.Sprintf("computer-%d.jpg", time.Now().UnixNano()))
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("computer: capture save: %w", err)
	}
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 70}); err != nil {
		f.Close()
		os.Remove(path)
		return nil, fmt.Errorf("computer: capture encode: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return nil, fmt.Errorf("computer: capture save: %w", err)
	}
	return &Shot{
		ID:           newUUID() + "-0",
		Path:         path,
		Width:        w,
		Height:       h,
		NativeWidth:  w,
		NativeHeight: h,
	}, nil
}
