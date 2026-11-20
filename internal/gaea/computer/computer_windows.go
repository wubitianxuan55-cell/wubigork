//go:build windows

package computer

// Windows 平台观察入口。W1 骨架期：UIA/WGC 实装在 W2（window_windows.go /
// tree_windows.go）落地，本文件届时收编真实现；骨架期返回明确的未实现错误，
// 不假装成功。
func init() { platformObserve = observeWindowsStub }

func observeWindowsStub(windowID int, opt ObserveOpt) (*Snapshot, error) {
	return nil, ErrUnsupported
}
