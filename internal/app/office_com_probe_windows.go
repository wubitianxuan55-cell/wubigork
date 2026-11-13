//go:build windows

package app

// office_com_probe_windows.go — office 实时协同档可用性探测（app 层薄转调：
// 探测实现 2026-11-13 波3.5 下沉到 internal/office/comhelper——boot 提示词段
// 引导回需要内核侧可达；本文件保 officeComAvailable 接口兼容）。

import comhelper "github.com/gaea/gaea/internal/office/comhelper"

func officeComProbeSuite() bool {
	return comhelper.ProbeSuiteAvailable()
}
