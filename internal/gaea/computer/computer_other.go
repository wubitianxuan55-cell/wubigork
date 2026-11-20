//go:build !windows

package computer

import "unsafe"

// 非 Windows 平台：全部能力返回 ErrUnsupported（对齐 internal/screen/screen_other.go
// 范式）。gaea 运行面只在 Windows，本文件保障 linux 交叉编译门禁与平台语义完整。
// W2+ 每落一个平台函数，这里同步补对侧桩。

func init() { platformObserve = observeUnsupported }

// observeUnsupported 观察平台桩。
func observeUnsupported(windowID int, opt ObserveOpt) (*Snapshot, error) {
	return nil, ErrUnsupported
}

// ListWindows 平台桩。
func ListWindows() ([]Window, error) {
	return nil, ErrUnsupported
}

// LaunchApp 平台桩。
func LaunchApp(app string) (Window, error) {
	return Window{}, ErrUnsupported
}

// 输入原语平台桩（W5；门禁在 Session 方法层，非 Windows 走不到这里）。
func ClickAtScreen(button string, count int, sx, sy int) error {
	return ErrUnsupported
}

var clickElement = func(elem unsafe.Pointer) error {
	return ErrUnsupported
}

func TypeTextRaw(text string) error {
	return ErrUnsupported
}

func PressKeyChord(chord string) error {
	return ErrUnsupported
}

func ScrollAtScreen(sx, sy int, dx, dy int) error {
	return ErrUnsupported
}

func FocusWindow(windowID int) {}

func WindowRelativeToScreen(win Window, x, y int) (int, int) {
	return win.X + x, win.Y + y
}
