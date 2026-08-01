package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/gaea/gaea/internal/types"
)

// ── 大纲 Agent ──────────────────────────────────────────────

// GetOutlines 获取大纲树（已排序，安全返回）
func (a *writingState) GetOutlines() map[string]interface{} {
	if a.outlineAgent == nil {
		slog.Warn("GetOutlines: outlineAgent 未初始化")
		return map[string]interface{}{"nodes": []interface{}{}, "story_thread": ""}
	}
	of := a.outlineAgent.GetOutlines()
	if of == nil {
		slog.Warn("GetOutlines: outlineAgent 返回 nil")
		return map[string]interface{}{"nodes": []interface{}{}, "story_thread": ""}
	}
	return map[string]interface{}{"nodes": of.Nodes, "story_thread": of.StoryThread}
}

// ChatOutline 与大纲 Agent 对话
func (a *writingState) ChatOutline(userMsg string) (map[string]interface{}, error) {
	if a.outlineAgent == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	reply, err := a.outlineAgent.Chat(a.ctx, userMsg)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"reply":    reply,
		"outlines": a.outlineAgent.GetOutlines(),
	}, nil
}

// ChatOutlineNode 针对特定大纲节点对话
func (a *writingState) ChatOutlineNode(nodeID, userMsg string) (map[string]interface{}, error) {
	if a.outlineAgent == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	reply, err := a.outlineAgent.ChatNode(a.ctx, nodeID, userMsg)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"reply":    reply,
		"outlines": a.outlineAgent.GetOutlines(),
	}, nil
}

// SaveOutlineNode 保存/更新单个大纲节点
func (a *writingState) SaveOutlineNode(nodeJSON string) error {
	if a.outlineAgent == nil {
		return fmt.Errorf("请先打开项目")
	}
	var node types.OutlineNode
	if err := json.Unmarshal([]byte(nodeJSON), &node); err != nil {
		return fmt.Errorf("解析节点数据失败: %w", err)
	}
	return a.outlineAgent.SaveNode(node)
}

// AddOutlineNode 添加大纲节点
func (a *writingState) AddOutlineNode(nodeJSON string) error {
	if a.outlineAgent == nil {
		return fmt.Errorf("请先打开项目")
	}
	var node types.OutlineNode
	if err := json.Unmarshal([]byte(nodeJSON), &node); err != nil {
		return fmt.Errorf("解析节点数据失败: %w", err)
	}
	return a.outlineAgent.AddNode(node)
}

// DeleteOutlineNode 删除大纲节点
func (a *writingState) DeleteOutlineNode(nodeID string) error {
	if a.outlineAgent == nil {
		return fmt.Errorf("请先打开项目")
	}
	return a.outlineAgent.DeleteNode(nodeID)
}

// ContinueOutline AI 续写大纲
func (a *writingState) ContinueOutline(count int) (map[string]interface{}, error) {
	if a.outlineAgent == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	of, err := a.outlineAgent.Continue(a.ctx, count)
	if err != nil {
		return nil, err
	}
	if err := a.outlineAgent.Save(of); err != nil {
		slog.Error("ContinueOutline: 保存大纲失败", "error", err)
		return nil, fmt.Errorf("保存大纲失败: %w", err)
	}
	return map[string]interface{}{
		"nodes":   of.Nodes,
		"total":   len(of.Nodes),
		"added":   count,
		"message": fmt.Sprintf("共 %d 卷", len(of.Nodes)),
	}, nil
}

// ExpandOutlineNode 展开大纲节点为子章节
func (a *writingState) ExpandOutlineNode(nodeID string, subCount int) (map[string]interface{}, error) {
	if a.outlineAgent == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	of, err := a.outlineAgent.ExpandNode(a.ctx, nodeID, subCount)
	if err != nil {
		return nil, err
	}
	if err := a.outlineAgent.Save(of); err != nil {
		slog.Error("ExpandOutlineNode: 保存大纲失败", "error", err)
		return nil, fmt.Errorf("保存大纲失败: %w", err)
	}
	return map[string]interface{}{"nodes": of.Nodes}, nil
}

// GenerateOutlineWithDialogue 使用学生-专家 LLM 对话生成大纲
func (a *writingState) GenerateOutlineWithDialogue(storyPrompt string, numChapters int, maxTurns int) (map[string]interface{}, error) {
	if a.outlineAgent == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	result, err := a.outlineAgent.GenerateOutlineWithDialogue(a.ctx, storyPrompt, numChapters, maxTurns)
	if err != nil {
		return nil, err
	}

	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	of, err := pm.ReadOutlines()
	if err != nil {
		slog.Warn("读取大纲失败，创建新大纲", "error", err)
		of = &types.OutlineFile{Nodes: []types.OutlineNode{}}
	}

	var savedNodes []types.OutlineNode
	for i, ch := range result.Chapters {
		if i >= numChapters {
			break
		}
		node := types.OutlineNode{
			ID:         fmt.Sprintf("dialogue_%d_%d", time.Now().UnixMilli(), i),
			ParentID:   "",
			Title:      ch.Title,
			Summary:    ch.Summary,
			OrderIndex: i + 1,
			Status:     "planned",
			Characters: []string{},
			KeyPoints:  []string{},
			Children:   []types.OutlineNode{},
		}
		savedNodes = append(savedNodes, node)
	}
	of.Nodes = append(of.Nodes, savedNodes...)
	if err := pm.WriteOutlines(of); err != nil {
		return nil, fmt.Errorf("保存大纲失败: %w", err)
	}

	return map[string]interface{}{
		"storyTitle": result.StoryTitle,
		"nodes":      savedNodes,
		"dialogue":   result.Dialogue,
		"chapters":   result.Chapters,
	}, nil
}
