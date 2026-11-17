// app_shutdown.go 是优雅关闭与关停步骤工具（codex 蒸馏 B1，第四批 P1）。
//
// Shutdown 的两条纪律：
//   - 每个管理外部进程/循环的步骤带超时——挂起时记日志并继续整体退出
//     （用户点了关闭不该被单个步骤卡死）；数据库关闭类必须完成的步骤
//     不走本包装（A4：DB 步骤除外）；
//   - 步骤错误文本截尾 4KiB——外部进程的 stderr 可能极长，关停日志只留
//     尾部（错误与结论通常在尾）。
package app

import (
	"log/slog"
	"time"

	"github.com/gaea/gaea/internal/gaea/control"
)

// stderrTailLimit 是关停日志里错误文本的保留上限（尾部 4KiB，同上游
// POST_EXIT stderr 收尾口径）。
const stderrTailLimit = 4 << 10

// shutdownStep 执行一个带超时的关停步骤：完成（含失败）即返回；超时记
// warn 后继续——挂起步骤的 goroutine 随进程退出回收，不阻塞整体。
func shutdownStep(name string, timeout time.Duration, fn func() error) {
	done := make(chan error, 1)
	go func() { done <- fn() }()
	select {
	case err := <-done:
		if err != nil {
			slog.Error("shutdown step failed", "step", name, "error", tailStderr(err.Error()))
		}
	case <-time.After(timeout):
		slog.Warn("shutdown step timed out, continuing", "step", name, "timeout", timeout.String())
	}
}

// tailStderr 保留错误文本尾部（头部通常是重复的环境回显）。
func tailStderr(s string) string {
	if len(s) <= stderrTailLimit {
		return s
	}
	return "…" + s[len(s)-stderrTailLimit:]
}

// drainControllers 是 B1 的两板块控制器收口：关闸拒收新消息 → 等在飞回合
// 收敛（默认 3s，超时强退并落 TurnDone{Err}）→ Close（SessionEnd 钩子与
// jobs 收尾）。仅 [agent] drain_enabled=true 时由 Shutdown 调用；关闭时
// Shutdown 完全不碰控制器（现状行为）。
func (a *App) drainControllers() {
	ga.mu.Lock()
	office := ga.ctrl
	ga.mu.Unlock()
	wx.mu.Lock()
	lit := wx.ctrl
	wx.mu.Unlock()
	boards := []struct {
		name string
		c    *control.Controller
	}{{"office", office}, {"wenxue", lit}}
	for _, b := range boards {
		if b.c == nil {
			continue
		}
		if err := b.c.BeginDrain(control.DrainTimeoutDefault); err != nil {
			slog.Warn("优雅关闭超时，在飞回合已强制取消", "board", b.name, "error", err)
		}
		b.c.Close()
	}
}
