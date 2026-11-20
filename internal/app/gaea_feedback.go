package app

// AI 反馈库绑定（2026-11-20 立项：ai_feedback 工具收集模型侧反馈 →
// <userDir>/ai-feedback.jsonl；记忆面板「AI 反馈」区经此两绑定查看/删除，
// 导出 md 由前端组文本走 saveFileAs（不新增后端导出面））。

import (
	"fmt"

	"github.com/gaea/gaea/internal/gaea/feedback"
)

// GaeaAIFeedbackList 返回全部 AI 反馈（新→旧）。库是全局的（跨板块收集），
// 不做空间过滤——工具入参 board 已记录板块，面板按需筛选。
func (a *App) GaeaAIFeedbackList() []feedback.Item {
	c := gaeaCtrl()
	if c == nil {
		return []feedback.Item{}
	}
	set := c.Memory()
	if set == nil || set.UserDir == "" {
		return []feedback.Item{}
	}
	return feedback.List(set.UserDir)
}

// GaeaAIFeedbackDelete 按 ID 删除一条反馈（面板逐条清理）。
func (a *App) GaeaAIFeedbackDelete(id string) error {
	c := gaeaCtrl()
	if c == nil {
		return fmt.Errorf("记忆系统未就绪")
	}
	set := c.Memory()
	if set == nil || set.UserDir == "" {
		return fmt.Errorf("记忆系统未就绪")
	}
	ok, err := feedback.Delete(set.UserDir, id)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("反馈条目 %s 不存在（可能已被删除）", id)
	}
	return nil
}
