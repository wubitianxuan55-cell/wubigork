//go:build !windows

package app

// office_com_probe_other.go — 非 Windows 平台无 COM 自动化对象，整族不注册
//（gaeaExtraTools 条件装配依赖此探测恒 false）。

func officeComProbeSuite() bool { return false }
