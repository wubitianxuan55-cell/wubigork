package app

// H0-1 Herdsman 环境探测：启动时对本机 herdsman 服务依赖的多个非公开契约
// （config.yaml、herdsman.exe CLI、HTTP /v1/models、launch_records/ 等数据文件）
// 做一次性探测，供前端「启动诊断」展示。herdsman 升级可能静默改变这些契约，
// 探测结果以结构化 JSON 返回，错误均放入结构内，方法本身不返回 error。

import (
	"github.com/gaea/gaea/internal/herdsman"
)

// HerdsmanProbe 返回本机 Herdsman 环境与数据契约探测结果。
//
// 使用默认根目录（%USERPROFILE%\.herdsman）与默认 baseURL
// （http://localhost:8080/v1）；默认参数下不发真实 HTTP 请求
// （APIReachable=false、APIError="skipped"），设置环境变量
// HERDSMAN_PROBE_LIVE=1 时才真实探测 API 可达性。
func (a *App) HerdsmanProbe() herdsman.Probe {
	return herdsman.NewProbe("", "").Run()
}
