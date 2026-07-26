package app

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/wubigork/wubigork/internal/types"
)

// ── 章节创作 ─────────────────────────────────────────────────

// GenerateChapter 流式生成章节
func (a *App) GenerateChapter(outlineNodeID string, skillName string, targetWords int) (map[string]interface{}, error) {
	if a.chapterAgent == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	// 解析 Skill 名称 → 注入的写作指导 markdown
	skillMD := ""
	if skillName != "" && a.skillLoader != nil {
		if s := a.skillLoader.Get(skillName); s != nil {
			skillMD = s.Body
			slog.Info("Skill 已注入", "name", s.Name, "version", s.Version)
		} else {
			slog.Warn("Skill 未找到，将以原始名称传入", "skill", skillName)
			skillMD = skillName // 回退：可能是前端传来的自定义指导文本
		}
	}

	chunks, chapterNum, err := a.chapterAgent.Generate(a.ctx, outlineNodeID, skillMD, true, targetWords)
	if err != nil {
		return nil, err
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("章节流式 goroutine panic", "panic", r)
				a.emit("chapter-stream", map[string]interface{}{
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
				a.emit("chapter-stream", map[string]interface{}{
					"type": "error", "error": chunk.Error,
				})
				return
			}
			if chunk.Done {
				break
			}
			fullText += chunk.Content
			a.emit("chapter-stream", map[string]interface{}{
				"type": "chunk", "content": chunk.Content, "total": len([]rune(fullText)),
			})
		}
		pm := a.getPM()
		if pm == nil {
			return
		}
		summary, err := pm.ReadChapterSummary(chapterNum)
		if err != nil {
			slog.Warn("流式生成后读取摘要失败", "chapter", chapterNum, "error", err)
		}

		// 自我演化：分析新章节 → 建议词条/更新世界观/记录伏笔
		doneData := map[string]interface{}{
			"type":    "done",
			"content": fullText,
			"total":   len([]rune(fullText)),
			"summary": summary,
		}
		if a.analysisAgent != nil {
			evolution, evErr := a.analysisAgent.EvolveAfterChapter(chapterNum, fullText, summary)
			if evErr == nil && evolution != nil {
				doneData["evolution"] = evolution
			}
		}
		a.emit("chapter-stream", doneData)
	}()

	return map[string]interface{}{"status": "started"}, nil
}

// GetChapter 读取已生成章节
func (a *App) GetChapter(num int) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	content, err := pm.ReadChapter(num)
	if err != nil {
		return nil, err
	}
	summary, err := pm.ReadChapterSummary(num)
	if err != nil {
		slog.Warn("读取章节摘要失败", "chapter", num, "error", err)
	}
	return map[string]interface{}{
		"content": content,
		"summary": summary,
	}, nil
}

// SaveChapterContent 手动保存章节内容
func (a *App) SaveChapterContent(num int, content string) error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}
	return pm.WriteChapter(num, content)
}

// ChatChapter 与写作 Agent 讨论特定章节
func (a *App) ChatChapter(chapterNum int, userMsg string) (map[string]interface{}, error) {
	if a.chapterAgent == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	content, err := pm.ReadChapter(chapterNum)
	if err != nil {
		slog.Warn("读取章节内容失败", "chapter", chapterNum, "error", err)
	}
	summary, err := pm.ReadChapterSummary(chapterNum)
	if err != nil {
		slog.Warn("读取章节摘要失败", "chapter", chapterNum, "error", err)
	}

	// 构建上下文
	ctx := a.ctx
	model := a.cfg.Model
	client := a.client

	sumText := ""
	if summary != nil {
		sumText = summary.Summary
	}

	lorebookCtx := a.buildLorebookContext(userMsg)
	systemPrompt := fmt.Sprintf(`你是专业的小说写作顾问。用户正在讨论第 %d 章。
章节摘要: %s
章节内容: %s
%s
请帮助作者改进这一章：给出具体的修改建议、指出节奏问题、建议更好的表达方式。`, chapterNum, sumText, content, lorebookCtx)

	reply, err := client.ChatSimpleStream(ctx, model, systemPrompt, userMsg)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"reply": reply}, nil
}

// ReviewChapter 审查章节质量（Reviser-Reviewer 模式）
func (a *App) ReviewChapter(chapterNum int) (map[string]interface{}, error) {
	if a.chapterAgent == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	content, err := pm.ReadChapter(chapterNum)
	if err != nil {
		return nil, fmt.Errorf("读取章节失败: %w", err)
	}

	summary, err := pm.ReadChapterSummary(chapterNum)
	outlineTitle := ""
	if err == nil && summary != nil {
		outlineTitle = summary.Title
	}

	// 获取上一章结尾作为上下文
	prevHint := ""
	if chapterNum > 1 {
		prevContent, err := pm.ReadChapter(chapterNum - 1)
		if err == nil {
			trimmed := strings.TrimRight(prevContent, "\n ")
			if len([]rune(trimmed)) > 150 {
				trimmed = string([]rune(trimmed)[len([]rune(trimmed))-150:])
			}
			prevHint = trimmed
		}
	}

	result, err := a.chapterAgent.ReviewChapter(a.ctx, content, outlineTitle, prevHint)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"score":      result.Score,
		"strengths":  result.Strengths,
		"weaknesses": result.Weaknesses,
		"revisePlan": result.RevisePlan,
	}, nil
}

// GenerateSceneIllustration 为指定章节生成场景插图（Aurora）
func (a *App) GenerateSceneIllustration(chapterNum int) (map[string]interface{}, error) {
	if a.chapterAgent == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	content, err := pm.ReadChapter(chapterNum)
	if err != nil {
		return nil, fmt.Errorf("读取章节失败: %w", err)
	}
	summary, err := pm.ReadChapterSummary(chapterNum)
	if err != nil {
		slog.Warn("读取章节摘要失败", "chapter", chapterNum, "error", err)
	}

	// 获取角色和世界观
	chars, err := pm.ReadCharacters()
	if err != nil {
		slog.Warn("读取角色失败", "error", err)
	}
	var characterList []types.Character
	if chars != nil {
		characterList = chars.Characters
	}

	wv, err := pm.ReadWorldview()
	if err != nil {
		slog.Warn("读取世界观失败", "error", err)
	}

	resp, err := a.chapterAgent.GenerateSceneIllustration(a.ctx, content, summary, characterList, wv)
	if err != nil {
		return nil, fmt.Errorf("生成插图失败: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("未生成图片")
	}

	return map[string]interface{}{
		"url":            resp.Data[0].URL,
		"revised_prompt": resp.Data[0].RevisedPrompt,
	}, nil
}

// ── v4 场景 API ──────────────────────────────────────────────

// GetChapterScenes 获取章节的场景列表
func (a *App) GetChapterScenes(chapterNum int) ([]map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	if !pm.IsV4() {
		// v3 项目：返回单场景视图
		content, err := pm.ReadChapter(chapterNum)
		if err != nil {
			return nil, err
		}
		return []map[string]interface{}{{
			"id":      fmt.Sprintf("%03d-chapter", chapterNum),
			"title":   fmt.Sprintf("第%d章", chapterNum),
			"content": content,
			"status":  "done",
			"order":   1,
		}}, nil
	}

	sm := pm.SceneManager(chapterNum)
	scenes, err := sm.List()
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, meta := range scenes {
		scene, err := sm.Read(meta.ID)
		if err != nil {
			continue
		}
		result = append(result, map[string]interface{}{
			"id":        scene.Meta.ID,
			"slug":      scene.Meta.Slug,
			"title":     scene.Meta.Title,
			"summary":   scene.Meta.Summary,
			"povCharId": scene.Meta.POVCharID,
			"location":  scene.Meta.Location,
			"timeOfDay": scene.Meta.TimeOfDay,
			"emotion":   scene.Meta.Emotion,
			"tags":      scene.Meta.Tags,
			"status":    string(scene.Meta.Status),
			"wordCount": scene.Meta.WordCount,
			"order":     scene.Meta.Order,
			"content":   scene.Content,
		})
	}
	return result, nil
}

// SaveScene 保存单个场景
func (a *App) SaveScene(chapterNum int, sceneID string, content string) error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}

	if !pm.IsV4() {
		return pm.WriteChapter(chapterNum, content)
	}

	sm := pm.SceneManager(chapterNum)
	scene, err := sm.Read(sceneID)
	if err != nil {
		return err
	}
	scene.Content = content
	return sm.Write(scene)
}

// ReorderScenes 重排场景顺序
func (a *App) ReorderScenes(chapterNum int, sceneIDs []string) error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}
	if !pm.IsV4() {
		return nil // v3 项目无场景可排
	}
	return pm.SceneManager(chapterNum).Reorder(sceneIDs)
}

// CreateSnapshot 手动创建场景快照
func (a *App) CreateSnapshot(sceneID string, chapterNum int, label string) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	var content string
	if pm.IsV4() {
		scene, err := pm.SceneManager(chapterNum).Read(sceneID)
		if err != nil {
			return nil, err
		}
		content = scene.Content
	} else {
		var err error
		content, err = pm.ReadChapter(chapterNum)
		if err != nil {
			return nil, err
		}
	}

	store := pm.SnapshotStore(chapterNum)
	snap, err := store.Capture(sceneID, content, label, "manual")
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":        snap.ID,
		"timestamp": snap.Timestamp,
		"label":     snap.Label,
		"wordCount": snap.WordCount,
	}, nil
}

// ListSnapshots 列出场景的所有快照
func (a *App) ListSnapshots(sceneID string, chapterNum int) ([]map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	store := pm.SnapshotStore(chapterNum)
	snaps, err := store.List(sceneID)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, snap := range snaps {
		result = append(result, map[string]interface{}{
			"id":        snap.ID,
			"timestamp": snap.Timestamp,
			"label":     snap.Label,
			"trigger":   snap.Trigger,
			"wordCount": snap.WordCount,
		})
	}
	return result, nil
}

// RestoreSnapshot 恢复到指定快照
func (a *App) RestoreSnapshot(snapshotID string, sceneID string, chapterNum int) error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}

	store := pm.SnapshotStore(chapterNum)
	content, err := store.Restore(snapshotID, sceneID)
	if err != nil {
		return err
	}

	if pm.IsV4() {
		scene, err := pm.SceneManager(chapterNum).Read(sceneID)
		if err != nil {
			return err
		}
		scene.Content = content
		return pm.SceneManager(chapterNum).Write(scene)
	}

	return pm.WriteChapter(chapterNum, content)
}

// MigrateProjectToV4 手动触发项目迁移到 v4
func (a *App) MigrateProjectToV4() error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}
	return pm.MigrateV3ToV4()
}

// IsProjectV4 检查当前项目是否为 v4 结构
func (a *App) IsProjectV4() bool {
	pm := a.getPM()
	if pm == nil {
		return false
	}
	return pm.IsV4()
}
