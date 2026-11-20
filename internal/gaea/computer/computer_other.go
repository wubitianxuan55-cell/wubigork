//go:build !windows

package computer

// 非 Windows 平台：全部能力返回 ErrUnsupported（对齐 internal/screen/screen_other.go
// 范式）。gaea 运行面只在 Windows，本文件保障 linux 交叉编译门禁与平台语义完整。
// W2+ 每落一个平台函数，这里同步补对侧桩。

func init() { platformObserve = observeUnsupported }

// observeUnsupported 观察平台桩。
func observeUnsupported(windowID int, opt ObserveOpt) (*Snapshot, error) {
	return nil, ErrUnsupported
}
