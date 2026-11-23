//go:build !windows

package app

// wenxue_privacy_other.go — 图文私密保护目录属性面（非 Windows 桩）。
// 桌面端只在 Windows 运行；此文件仅为 linux CI 编译面成立（属性面 no-op）。

import "errors"

const fileAttributeHidden = uint32(0)

func windowsGetFileAttributes(path string) (uint32, error) {
	return 0, errors.New("目录隐藏属性仅 Windows 支持")
}

func windowsSetFileAttributes(path string, attrs uint32) error {
	return errors.New("目录隐藏属性仅 Windows 支持")
}
