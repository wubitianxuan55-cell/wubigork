package config

import "sync"

// modelMu 保护 Config.Model 的运行期内存读写。
//
// 背景（2026-09-19 后端审计）：Model 是「UI 写路径 × 请求热路径」交汇的唯一
// 裸字段——模型中心切换引擎/设默认模型/SaveConfig（绑定 goroutine 写）∥ 每条
// 请求 resolveModelName 等读。string 是 ptr+len 双字，并发读写未同步是形式
// data race。funcMu 只覆盖功能级绑定字段，Model 不在其内，故单独立锁。
//
// 纪律：启动期 config.Load 单线程装配不受此限；运行期对全局 Config 实例的
// Model 读写一律走 SetModelMem / GetModelMem（持久化由调用方另行 Save）。
var modelMu sync.RWMutex

// SetModelMem 写内存态全局回退模型（并发安全；不落盘）。
func SetModelMem(cfg *Config, model string) {
	modelMu.Lock()
	cfg.Model = model
	modelMu.Unlock()
}

// GetModelMem 读内存态全局回退模型（并发安全）。
func GetModelMem(cfg *Config) string {
	modelMu.RLock()
	model := cfg.Model
	modelMu.RUnlock()
	return model
}
