//go:build !windows

package comhelper

// probe_other.go — 非 Windows 平台无 COM 自动化对象，探测恒 false
//（引导段与工具族条件装配共同依赖此语义）。

func ProbeSuiteAvailable() bool { return false }
