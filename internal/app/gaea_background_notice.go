package app

// 后台任务可见 notice 的共用收口（自 gaea_dream.go 抽出保留——2026-11-20
// 用户拍板删除 auto-dream 做梦机制，但本收口被轻语主动关心/子代理追问等
// 后台 goroutine 共用，不随文件删除）。

import (
	"fmt"
	"log/slog"

	"github.com/gaea/gaea/internal/gaea/event"
)

// gaeaNoticeSink 是后台任务可见 notice 的发射缝（测试注入；nil = 走内核 emit →
// gaea-event 通道，与其他 gaea 事件同路，**不新造通道**）。
var gaeaNoticeSink func(eventName string, data map[string]interface{})

// gaeaBackgroundPanicNotice 是「后台任务 panic」的统一收口（审计 P2 AP5-16，
// 多处后台 goroutine 共用）：
//
//	① slog.Error 留痕（保留原有行为）；
//	② 复位各自的单飞/运行中状态位——由调用方的 defer 负责（本函数不碰状态，
//	   避免跨文件争用同一把锁时产生复位顺序歧义）；
//	③ 经既有 gaea-event 通道发一条 Notice（LevelWarn），前端/面板可见——此前
//	   recover 只写日志就吞掉，用户看到的是「功能开着但再也不工作」，只能重启
//	   进程（轻语 ticker 的原实现更是命中一次即永久退出循环）。
//
// task 为中文任务名（直接进用户可见文案）；fields 只进日志，便于事后定位。
func (c *core) gaeaBackgroundPanicNotice(task string, r any, fields map[string]interface{}) {
	attrs := []any{"task", task, "panic", r}
	for k, v := range fields {
		attrs = append(attrs, k, v)
	}
	slog.Error("后台任务 panic（已复位状态并通知前端）", attrs...)
	payload := gaeaEventMap(event.Event{
		Kind:  event.Notice,
		Level: event.LevelWarn,
		Text:  fmt.Sprintf("后台任务「%s」发生异常已中断（%v），状态已复位，稍后可重试", task, r),
	})
	if gaeaNoticeSink != nil {
		gaeaNoticeSink("gaea-event", payload)
		return
	}
	c.emit("gaea-event", payload)
}

// gaeaSessionSpace 当前办公会话的写入侧空间（自 gaea_dream.go 抽出保留——
// auto-dream 已删但子代理追问/DAG 等仍消费）：
//   - 引擎已初始化 → ctrl.SessionSpace()（"work"/"play"；space.mode=off 为 ""）；
//   - 引擎未初始化 → 回退配置生效空间 gaeaEffectiveSpace()。
func gaeaSessionSpace() string {
	ga.mu.Lock()
	ctrl := ga.ctrl
	ga.mu.Unlock()
	if ctrl != nil {
		return ctrl.SessionSpace()
	}
	return gaeaEffectiveSpace()
}
