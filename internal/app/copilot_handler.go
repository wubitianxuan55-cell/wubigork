package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/wubigork/wubigork/internal/ai"
	"github.com/wubigork/wubigork/internal/types"
)

// ── Ghost Text 内联补全 ─────────────────────────────────────

// GhostComplete 触发流式 Ghost Text 补全
// 结果通过 "ghost-stream" 事件推送给前端
func (a *App) GhostComplete(currentText string, styleProfile string) (map[string]interface{}, error) {
	if a.client == nil {
		return nil, fmt.Errorf("AI 客户端未初始化")
	}

	chunks, err := a.client.GhostComplete(a.ctx, a.cfg.Model, currentText, styleProfile)
	if err != nil {
		return nil, err
	}

	// 创建可取消的子 context，用于 CancelGhost
	ghostCtx, cancel := context.WithCancel(a.ctx)
	a.mu.Lock()
	if a.ghostCancel != nil {
		a.ghostCancel() // 取消上一次未完成的
	}
	a.ghostCancel = cancel
	a.mu.Unlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Ghost 补全 goroutine panic", "panic", r)
				a.emit("ghost-stream", map[string]interface{}{
					"type": "error", "error": fmt.Sprintf("内部错误: %v", r),
				})
			}
		}()

		var fullText string
		for chunk := range chunks {
			select {
			case <-ghostCtx.Done():
				a.emit("ghost-stream", map[string]interface{}{
					"type": "cancelled",
				})
				return
			default:
			}

			if chunk.Error != "" {
				a.emit("ghost-stream", map[string]interface{}{
					"type": "error", "error": chunk.Error,
				})
				return
			}
			if chunk.Done {
				a.emit("ghost-stream", map[string]interface{}{
					"type": "done", "content": fullText,
				})
				return
			}
			fullText += chunk.Content
			a.emit("ghost-stream", map[string]interface{}{
				"type": "chunk", "content": chunk.Content,
			})
		}
	}()

	return map[string]interface{}{"status": "started"}, nil
}

// CancelGhost 取消正在进行的 Ghost 补全
func (a *App) CancelGhost() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.ghostCancel != nil {
		a.ghostCancel()
		a.ghostCancel = nil
		slog.Info("Ghost 补全已取消")
	}
}

// ── Cmd+K 命令编辑 ──────────────────────────────────────────

// CmdKEdit 根据自然语言指令编辑选中文本
func (a *App) CmdKEdit(selectedText string, instruction string, styleProfile string) (map[string]interface{}, error) {
	if a.client == nil {
		return nil, fmt.Errorf("AI 客户端未初始化")
	}

	edited, err := a.client.CmdKEdit(a.ctx, a.cfg.Model, selectedText, instruction, styleProfile)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"edited": edited,
	}, nil
}

// ── Beat-to-Prose ────────────────────────────────────────────

// GenerateBeats 为指定大纲节点生成叙事节拍
func (a *App) GenerateBeats(outlineNodeID string) ([]map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	// 加载大纲信息
	outlineSummary := ""
	projCtx, err := pm.LoadContext(outlineNodeID)
	if err == nil && projCtx.CurrentOutline != nil {
		outlineSummary = fmt.Sprintf("%s: %s", projCtx.CurrentOutline.Title, projCtx.CurrentOutline.Summary)
	}

	// 获取上一章摘要
	prevSummary := ""
	if projCtx != nil && projCtx.PrevSummary != nil {
		prevSummary = projCtx.PrevSummary.Summary
	}

	beats, err := a.client.GenerateBeats(a.ctx, a.cfg.Model, outlineSummary, prevSummary)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, b := range beats {
		result = append(result, map[string]interface{}{
			"id":          b.ID,
			"description": b.Description,
			"order":       b.Order,
		})
	}
	return result, nil
}

// GenerateProseFromBeat 从单个节拍流式生成 Prose
// 结果通过 "beat-prose-stream" 事件推送给前端
func (a *App) GenerateProseFromBeat(beatID string, allBeatJSON string, chapterNum int) (map[string]interface{}, error) {
	if a.client == nil {
		return nil, fmt.Errorf("AI 客户端未初始化")
	}
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	// 解析节拍列表
	var allBeats []ai.Beat
	var targetBeat ai.Beat
	found := false

	// 输入格式: [{"id":"beat-1","description":"...","order":1}, ...]
	var rawBeats []struct {
		ID          string `json:"id"`
		Description string `json:"description"`
		Order       int    `json:"order"`
	}
	if err := json.Unmarshal([]byte(allBeatJSON), &rawBeats); err == nil {
		for _, rb := range rawBeats {
			b := ai.Beat{ID: rb.ID, Description: rb.Description, Order: rb.Order}
			allBeats = append(allBeats, b)
			if b.ID == beatID {
				targetBeat = b
				found = true
			}
		}
	}
	if !found {
		return nil, fmt.Errorf("节拍 %s 未找到", beatID)
	}

	// 构建上下文
	contextMap := map[string]string{}

	// 角色信息
	chars, err := pm.ReadCharacters()
	if err == nil && chars != nil {
		var charDescs []string
		for _, ch := range chars.Characters {
			charDescs = append(charDescs, fmt.Sprintf("%s[%s]: %s", ch.Name, ch.RoleType, ch.Personality))
		}
		contextMap["角色"] = strings.Join(charDescs, "; ")
	}

	// 世界观
	wv, err := pm.ReadWorldview()
	if err == nil && wv != "" {
		contextMap["世界观"] = string([]rune(wv)[:min(200, len([]rune(wv)))])
	}

	chunks, err := a.client.GenerateProseFromBeat(a.ctx, a.cfg.Model, targetBeat, allBeats, contextMap)
	if err != nil {
		return nil, err
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Beat Prose goroutine panic", "panic", r)
				a.emit("beat-prose-stream", map[string]interface{}{
					"type": "error", "error": fmt.Sprintf("内部错误: %v", r),
				})
			}
		}()

		var fullText string
		for chunk := range chunks {
			select {
			case <-a.ctx.Done():
				return
			default:
			}

			if chunk.Error != "" {
				a.emit("beat-prose-stream", map[string]interface{}{
					"type": "error", "error": chunk.Error,
				})
				return
			}
			if chunk.Done {
				// 保存生成的 prose
				if pm.IsV4() && fullText != "" {
					sm := pm.SceneManager(chapterNum)
					// 以 beat ID 作为场景 slug 保存
					sceneObj, createErr := sm.Create(beatID, targetBeat.Description)
					if createErr == nil {
						sceneObj.Content = fullText
						sceneObj.Meta.Status = types.SceneDraft
						sm.Write(sceneObj)
					}
				}

				a.emit("beat-prose-stream", map[string]interface{}{
					"type":    "done",
					"content": fullText,
					"beatID":  beatID,
				})
				return
			}
			fullText += chunk.Content
			a.emit("beat-prose-stream", map[string]interface{}{
				"type": "chunk", "content": chunk.Content, "beatID": beatID,
			})
		}
	}()

	return map[string]interface{}{"status": "started"}, nil
}


