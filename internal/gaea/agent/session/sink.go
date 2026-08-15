package session

// 3.0 Step 1: 持久化事件 sink。挂在 boot 的 Sync 包装之后（sink 链
// Sync→logSink→前端），事件在转发前端之前先落盘事件日志。
// 「模型可见必入日志」：事件先写日志、再转发前端，日志写入失败只降级
// 记录（不阻断实时 UI），日志本身是恢复/派生的真相源。

import (
	"log/slog"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/event"
)

// EventLogSink 把 event.Event 流追加写入当前会话的事件日志。
// 会话路径由 pathSrc 在 Emit 时解析（boot 在控制构建完成后注入），
// 为空（尚未建立会话）时只转发不落盘。
type EventLogSink struct {
	mu      sync.Mutex
	inner   event.Sink
	pathSrc func() string
	writer  *LogWriter
	logPath string
	openErr error // 打开失败时记一次，避免每事件重试刷屏
}

// NewEventLogSink 构造事件日志 sink。dir 是会话目录；inner 是下一环 sink。
func NewEventLogSink(dir string, inner event.Sink) *EventLogSink {
	return &EventLogSink{inner: inner}
}

// SetPathSource 注入会话路径解析器（boot 在 control.New 之后调用）。
func (s *EventLogSink) SetPathSource(fn func() string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pathSrc = fn
}

// Emit 先写日志再转发前端。并发由调用方（event.Sync 包装）串行保证，
// 本实现自身也持锁以防御直接并发调用。
func (s *EventLogSink) Emit(e event.Event) {
	s.mu.Lock()
	path := ""
	if s.pathSrc != nil {
		path = s.pathSrc()
	}
	if path != "" {
		s.logTo(path, e)
	}
	s.mu.Unlock()
	if s.inner != nil {
		s.inner.Emit(e)
	}
}

// logTo 把事件追加到 path 对应的事件日志（懒打开 + 旧格式自动迁移）。
func (s *EventLogSink) logTo(path string, e event.Event) {
	lp := LogPathFor(path)
	if lp == "" {
		return
	}
	if s.writer == nil || s.logPath != lp {
		if s.writer != nil {
			_ = s.writer.Close()
			s.writer = nil
		}
		s.logPath = lp
		s.openErr = nil
		w, err := OpenLog(lp, path)
		if err != nil {
			s.openErr = err
			slog.Warn("session event log: open failed", "log", lp, "error", err)
			return
		}
		s.writer = w
	}
	entry, err := EntryFromEvent(e, time.Now().Unix())
	if err != nil {
		slog.Warn("session event log: entry build failed", "error", err)
		return
	}
	if _, err := s.writer.AppendRaw(entry.Kind, entry.Payload); err != nil {
		slog.Warn("session event log: append failed", "log", lp, "error", err)
		return
	}
	// 回合边界（turn_done）落盘后关闭写入器：回合间日志处于持久、可被外部
	// 工具删除/迁移的状态；下一事件再懒打开（OpenLog 修复 torn-tail 并续 seq）。
	if entry.Kind == "turn_done" {
		_ = s.writer.Close()
		s.writer = nil
	}
}

// Close 关闭当前日志写入器。
func (s *EventLogSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.writer != nil {
		err := s.writer.Close()
		s.writer = nil
		return err
	}
	return nil
}

// Flush 是 fail-closed 检查点落盘挂钩：确保日志已追加（close 刷盘）并返回
// 当前已写 seq。模型调用前调用方（后续接线）用它 flush 检查点。
func (s *EventLogSink) Flush() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.writer != nil {
		return s.writer.Seq()
	}
	return 0
}
