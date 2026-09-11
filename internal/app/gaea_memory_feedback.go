package app

import (
	"fmt"
	"time"

	"github.com/gaea/gaea/internal/gaea/memory"
)

// GaeaMemoryFeedback 助手回答反馈（点赞/点踩）能力层：反馈作为记忆事件
// （op=feedback）追加进 memory_events 日志——「用户对哪条回答满意与否」是
// 个人记忆的一部分，事件节点随 v4.210 图谱投影自然可查（投影对 feedback
// 不建实体，仅事件节点，与 cite/pin 同边界）。
//
// 参数逐项显式（Wails 要求逐参传递，v4.237 定论）：messageID=助手消息 id
// （会话日志内稳定）；rating=up/down；excerpt=回答正文摘要（后端按
// eventExcerptLimit 截断，供事件列表/图节点 hover 展示）；space=当前空间
// （work/play 事件行标记）。经 hubOfficeStore 路由（尊重测试 override 缝）；
// 文件后端诚实报错（Pin 先例）。
func (a *App) GaeaMemoryFeedback(messageID string, rating string, excerpt string, space string) error {
	if rating != "up" && rating != "down" {
		return fmt.Errorf("rating 须为 up 或 down")
	}
	e := memory.Event{
		At:            time.Now().UnixMilli(),
		Op:            memory.OpFeedback,
		Name:          messageID,
		Title:         "点赞回答",
		Desc:          excerpt,
		Space:         space,
		SourceSession: resolveGaeaSessionPath(nil),
		SourceMessage: messageID,
		Actor:         "panel",
	}
	if rating == "down" {
		e.Title = "点踩回答"
	}
	if err := a.hubOfficeStore().AppendFeedbackEvent(e); err != nil {
		return fmt.Errorf("反馈落库: %w", err)
	}
	return nil
}
