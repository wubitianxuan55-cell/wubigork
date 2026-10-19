package modelengine

// stats_recent.go — 近期调用环形缓冲（「运行监测」板块的最近请求表面板）：
// statsRecorder.record 在聚合计账之外保留最近 recentCallsCap 条单次调用
// 快照（旧→新），进程内存态、不落盘——监测语义关心「现在」，持久账目仍由
// 聚合计账（models/trends + 落盘）承担。云引擎调用同样入环（最近请求表
// 覆盖 gaea 全部引擎流量，不只是本地引擎）。

// recentCallsCap 环形容量：前端表格一屏 12~15 行 + 翻看余量；60 条足够且
// 内存占用可忽略（每条 ~200B）。
const recentCallsCap = 60

// appendRecentLocked 追加一条近期调用（调用方须持 r.mu；超容量滑出最旧）。
func (r *statsRecorder) appendRecentLocked(u ModelCallUsage) {
	if len(r.recent) >= recentCallsCap {
		copy(r.recent, r.recent[1:])
		r.recent[len(r.recent)-1] = u
		return
	}
	r.recent = append(r.recent, u)
}

// recentSnapshot 返回最近 n 条调用快照的拷贝（新→旧；n<=0 或超量 = 全部留存）。
func (r *statsRecorder) recentSnapshot(n int) []ModelCallUsage {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n <= 0 || n > len(r.recent) {
		n = len(r.recent)
	}
	out := make([]ModelCallUsage, n)
	for i := 0; i < n; i++ {
		out[i] = r.recent[len(r.recent)-1-i] // 尾部最新，倒序输出
	}
	return out
}

// RecentCalls 返回最近 n 条模型调用快照（新→旧；ai.Client 每次请求结束
// 经 RecordCall 上报，环在 record 内追加）。运行监测页数据源。
func (m *Manager) RecentCalls(n int) []ModelCallUsage {
	return m.stats().recentSnapshot(n)
}
