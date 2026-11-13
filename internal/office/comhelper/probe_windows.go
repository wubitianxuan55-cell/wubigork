//go:build windows

package comhelper

// probe_windows.go — Office 套件可用性探测（下沉自 internal/app，2026-11-13
// 波3.5 preferLive 引导回需要内核侧（boot 提示词段）可达——internal/app 是壳层，
// internal/gaea/boot 不能反向 import；internal/office 是基础库，方向合法）。
// 探测只回答「装没装」：WPS（Kingsoft 注册表键）或微软 Office（App Paths
// 三件）。套件选择交给 helper 的 suite=auto。

import (
	"sync"

	"golang.org/x/sys/windows/registry"
)

var (
	probeOnce sync.Once
	probeOK   bool
)

// ProbeSuiteAvailable 本机是否有可用的 Office 套件（MS Office 或 WPS）。
// 进程内单次探测缓存。
func ProbeSuiteAvailable() bool {
	probeOnce.Do(func() {
		probeOK = probeSuite()
	})
	return probeOK
}

func probeSuite() bool {
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
