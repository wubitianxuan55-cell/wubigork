// engine_activity.go 引擎活跃度记账：外部引擎（天璇/Strata 等 gaea 代启进程）
// 的「最近一次真实使用」时间戳，供 app 层空闲自动退出机制判定（model_idle_exit_minutes）。
// 记账点=真实使用：BuildChatURL（每次 LLM 请求的端点解析）+ 引擎启动；探活
// （health_probe/fetchModels/引擎状态点）刻意不记账——探活不能把空闲模型续命。
package modelengine

import (
	"sync"
	"time"
)

// engineActivityMgr 活跃记账（Manager.activity 组合字段；独立小锁——记账在
// 每请求热路径上，避免与引擎表 m.mu 竞争；last 只被本文件方法读写）。
type engineActivityMgr struct {
	mu   sync.Mutex
	last map[string]time.Time
}

// TouchEngine 记一次引擎真实使用（now）。
func (m *Manager) TouchEngine(engineID string) {
	m.TouchEngineAt(engineID, time.Now())
}

// TouchEngineAt 记一次引擎真实使用（显式时间；测试回拨用）。
func (m *Manager) TouchEngineAt(engineID string, t time.Time) {
	if engineID == "" {
		return
	}
	m.activity.mu.Lock()
	defer m.activity.mu.Unlock()
	if m.activity.last == nil {
		m.activity.last = make(map[string]time.Time)
	}
	m.activity.last[engineID] = t
}

// EngineLastActivity 返回引擎最近一次真实使用时间；从未使用返回零值。
func (m *Manager) EngineLastActivity(engineID string) time.Time {
	m.activity.mu.Lock()
	defer m.activity.mu.Unlock()
	return m.activity.last[engineID]
}
