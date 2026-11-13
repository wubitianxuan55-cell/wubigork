//go:build windows

package app

// office_com_probe_windows.go — office 实时协同档可用性探测（Windows 实现）：
// WPS（Kingsoft 注册表键，本机 2026-11-13 实证存在）或微软 Office
//（App Paths 三件）。探测只回答「装没装」，套件选择交给 helper 的
// suite=auto（微软优先，缺了退 WPS）。

import (
	"golang.org/x/sys/windows/registry"
)

func officeComProbeSuite() bool {
	for _, root := range []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER} {
		k, err := registry.OpenKey(root, `SOFTWARE\Kingsoft\Office\6.0\Common`, registry.QUERY_VALUE)
		if err == nil {
			_ = k.Close()
			return true
		}
	}
	for _, exe := range []string{"Winword.exe", "Excel.exe", "Powerpnt.exe"} {
		k, err := registry.OpenKey(registry.LOCAL_MACHINE,
			`SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\`+exe, registry.QUERY_VALUE)
		if err == nil {
			_ = k.Close()
			return true
		}
	}
	return false
}
