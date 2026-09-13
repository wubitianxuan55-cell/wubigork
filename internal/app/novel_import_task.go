package app

// 长书后台任务态（v4.291，拆书线欠账）：AI 反推大纲是 1-3 分钟的长操作，
// 同步绑定页面一关结果就丢。本文件把反推接进 tasks 任务队列——任务中心可见、
// 页面关闭不丢、轮询取结果。同步绑定 NovelOutlineReconstruct 保留（既有调用
// 与测试零变化），任务化是新入口。

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/tasks"
)

// NovelOutlineReconstructTaskState 反推任务状态（轮询返回）。
type NovelOutlineReconstructTaskState struct {
	TaskID  string                      `json:"taskId"`
	Status  string                      `json:"status"`
	Message string                      `json:"message,omitempty"`
	Error   string                      `json:"error,omitempty"`
	Preview *OutlineReconstructPreview  `json:"preview,omitempty"` // succeeded 时携带
}

// outlineReconstructTaskHandler 反推任务 handler：跑与同步绑定同一套
// outlineReconstructCore，预览 JSON 存任务 Result（不透明载荷，前端按 kind 解析）。
func (a *App) outlineReconstructTaskHandler(ctx context.Context, tk *tasks.Task, p *tasks.Progress) error {
	p.Report(5, "正在读取章节并立项反推")
	ctx, cancel := context.WithTimeout(ctx, reconstructTimeout)
	defer cancel()
	preview, err := a.outlineReconstructCore(ctx)
	if err != nil {
		return err
	}
	p.Report(90, "反推完成，写入预览")
	raw, err := json.Marshal(preview)
	if err != nil {
		return fmt.Errorf("预览序列化失败: %w", err)
	}
	p.Result(string(raw))
	return nil
}

// NovelOutlineReconstructStart 提交反推后台任务（同空间去重：已有在跑/排队的
// 反推任务时显式报错，不重复入队）。taskMgr 挂在 App 级，故接收者为 *App。
func (a *App) NovelOutlineReconstructStart() (NovelOutlineReconstructTaskState, error) {
	if a.getPM() == nil {
		return NovelOutlineReconstructTaskState{}, fmt.Errorf("请先打开项目")
	}
	m := a.taskMgr()
	if m == nil || !m.Available() {
		return NovelOutlineReconstructTaskState{}, fmt.Errorf("任务队列不可用，请直接使用同步反推")
	}
	tk, err := m.SubmitSpaceSession(tasks.KindOutlineReconstruct, "AI 反推大纲", map[string]any{}, spaces.SpacePlay, "")
	if err != nil {
		return NovelOutlineReconstructTaskState{}, err
	}
	return NovelOutlineReconstructTaskState{TaskID: tk.ID, Status: string(tk.Status)}, nil
}

// NovelOutlineReconstructTaskGet 查询最近一次反推任务的状态；succeeded 时携带预览。
func (a *App) NovelOutlineReconstructTaskGet() (NovelOutlineReconstructTaskState, error) {
	m := a.taskMgr()
	if m == nil || !m.Available() {
		return NovelOutlineReconstructTaskState{}, fmt.Errorf("任务队列不可用")
	}
	list, err := m.List(50)
	if err != nil {
		return NovelOutlineReconstructTaskState{}, err
	}
	for _, tk := range list {
		if tk.Kind != string(tasks.KindOutlineReconstruct) {
			continue
		}
		state := NovelOutlineReconstructTaskState{TaskID: tk.ID, Status: string(tk.Status), Message: tk.Message, Error: tk.Error}
		if tk.Status == string(tasks.StatusSucceeded) && tk.Result != "" {
			var preview OutlineReconstructPreview
			if err := json.Unmarshal([]byte(tk.Result), &preview); err != nil {
				return NovelOutlineReconstructTaskState{}, fmt.Errorf("任务结果解析失败: %w", err)
			}
			state.Preview = &preview
		}
		return state, nil
	}
	return NovelOutlineReconstructTaskState{}, fmt.Errorf("尚无反推任务")
}
