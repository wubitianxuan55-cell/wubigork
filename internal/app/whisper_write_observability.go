package app

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ── T6-5.3 异步写可观测：记忆写入/持久化错误回传 ──────────────────────
//
// whisper_handler 的记忆写入/持久化走 fire-and-forget 协程，正常错误路径
// （LLM 失败/落库失败/JSON 解析失败）原本不可观测。这里在 whisper 状态侧
// 维护 WriteErrors 计数器（原子计数 + 最近错误摘要），handler 以内部方法
// 暴露读取（不新增 Wails 绑定）。

// whisperWriteErrors 异步记忆写入/持久化错误统计（T6-5.3）。
type whisperWriteErrors struct {
	mu    sync.Mutex
	count int64
	last  string
	at    time.Time
}

// record 记录一次错误，返回摘要（协程并发安全）。
func (w *whisperWriteErrors) record(phase string, err error) string {
	summary := fmt.Sprintf("[%s] %v", phase, err)
	w.mu.Lock()
	w.count++
	w.last = summary
	w.at = time.Now()
	w.mu.Unlock()
	return summary
}

// stats 返回（错误计数, 最近错误摘要, 最近错误时间）。
func (w *whisperWriteErrors) stats() (int64, string, time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.count, w.last, w.at
}

// recordMemoryWriteError 异步记忆写入错误回传（whisper.MemoryWriteErrorSink）：
// 所有错误路径（LLM 失败/JSON 解析失败/落库失败/panic）统一计入 WriteErrors
// 计数并 slog.Error。日志先于计数落盘，保证测试轮询到计数时日志已可断言。
func (a *whisperState) recordMemoryWriteError(sessionID, phase string, err error) {
	slog.Error("[whisper] 异步记忆写入失败", "sessionID", sessionID, "phase", phase, "error", err)
	a.writeErrors.record(phase, err)
}

// whisperWriteErrorStats 暴露异步写错误统计（内部读取：诊断/自检用，不新增 Wails 绑定）。
// 返回（错误计数, 最近错误摘要, 最近错误时间）。
func (a *whisperState) whisperWriteErrorStats() (int64, string, time.Time) {
	return a.writeErrors.stats()
}
