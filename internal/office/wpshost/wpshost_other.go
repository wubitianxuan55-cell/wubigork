//go:build !windows

// Package wpshost 非 Windows 桩：App 绑定方法在 linux（CI race）可编译，
// 调用一律返回不可用错误。真实实现见 wpshost_windows.go。
package wpshost

// Window 是一个 WPS 相关顶层窗口的描述（桩：恒空）。
type Window struct {
	HWND     uint64 `json:"hwnd"`
	Proc     string `json:"proc"`
	Class    string `json:"class"`
	Title    string `json:"title"`
	Visible  bool   `json:"visible"`
	IsDocWin bool   `json:"is_doc_win"`
}

// List 恒空（非 Windows 无 WPS）。
func List() []Window { return []Window{} }

// Dock 恒不可用。
func Dock(hwnd uint64, x, y, w, h int, borderless bool) error {
	return errUnavailable
}

// Restore 恒不可用。
func Restore(hwnd uint64) error { return errUnavailable }

// RestoreAll 恒 0。
func RestoreAll() int { return 0 }

var errUnavailable = errNotSupported{}

type errNotSupported struct{}

func (errNotSupported) Error() string { return "wpshost 仅支持 Windows" }
