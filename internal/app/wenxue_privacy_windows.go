//go:build windows

package app

// wenxue_privacy_windows.go — 图文私密保护的目录属性面（Windows 实现）。
// x/sys/windows 直调（先例 herdsman_disk.go）；非 Windows 平台见
// wenxue_privacy_other.go 同名桩。

import (
	"golang.org/x/sys/windows"
)

// fileAttributeHidden 目录系统隐藏属性位。
const fileAttributeHidden = uint32(windows.FILE_ATTRIBUTE_HIDDEN)

// windowsGetFileAttributes 读目录属性字（供加/摘单属性位）。
func windowsGetFileAttributes(path string) (uint32, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	return windows.GetFileAttributes(p)
}

// windowsSetFileAttributes 写回目录属性字。
func windowsSetFileAttributes(path string, attrs uint32) error {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	return windows.SetFileAttributes(p, attrs)
}
