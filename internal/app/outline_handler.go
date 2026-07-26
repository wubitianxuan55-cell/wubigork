package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/wubigork/wubigork/internal/types"
)

// ── 大纲 Agent ──────────────────────────────────────────────

// GetOutlines 获取大纲树（已排序，安全返回）
func (a *App) GetOutlines() map[string]interface{} {
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
func (a *App) ChatOutline(userMsg string) (map[string]interface{}, error) {
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
func (a *App) ChatOutlineNode(nodeID, userMsg string) (map[string]interface{}, error) {
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
func (a *App) SaveOutlineNode(nodeJSON string) error {
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
func (a *App) AddOutlineNode(nodeJSON string) error {
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
func (a *App) DeleteOutlineNode(nodeID string) error {
	if a.outlineAgent == nil {
		return fmt.Errorf("请先打开项目")
	}
	return a.outlineAgent.DeleteNode(nodeID)
}

// GenerateOutlineNodeDetail 一键生成大纲节点详情
func (a *App) GenerateOutlineNodeDetail(nodeID string) (map[string]interface{}, error) {
	if a.outlineAgent == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	node, err := a.outlineAgent.GenerateNodeDetail(a.ctx, nodeID)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"node": node,
	}, nil
}

// ContinueOutline AI 续写大纲
func (a *App) ContinueOutline(count int) (map[string]interface{}, error) {
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
		"added":   count, // 请求数量（实际新增由前端对比计算）
		"message": fmt.Sprintf("共 %d 卷", len(of.Nodes)),
	}, nil
}

// ExpandOutlineNode 展开大纲节点为子章节
func (a *App) ExpandOutlineNode(nodeID string, subCount int) (map[string]interface{}, error) {
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

// SaveStoryThread 保存故事主线
func (a *App) SaveStoryThread(storyThread string) error {
	if a.outlineAgent == nil {
		return fmt.Errorf("请先打开项目")
	}
	return a.outlineAgent.SaveStoryThread(storyThread)
}

// GetStoryThread 读取故事主线
func (a *App) GetStoryThread() string {
	if a.outlineAgent == nil {
		return ""
	}
	return a.outlineAgent.GetStoryThread()
}

// GenerateStoryThread AI 一键生成故事主线
func (a *App) GenerateStoryThread(userHint string) (map[string]interface{}, error) {
	if a.outlineAgent == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	storyThread, err := a.outlineAgent.GenerateStoryThread(a.ctx, userHint)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"story_thread": storyThread}, nil
}

// ChatStoryThread 与 AI 讨论故事主线
func (a *App) ChatStoryThread(userMsg string) (map[string]interface{}, error) {
	if a.outlineAgent == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	reply, err := a.outlineAgent.ChatStoryThread(a.ctx, userMsg)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"reply":        reply,
		"story_thread": a.outlineAgent.GetStoryThread(),
	}, nil
}

// ImportStoryThreadFile 导入文本文件到故事主线编辑区
func (a *App) ImportStoryThreadFile() (map[string]interface{}, error) {
	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "导入故事主线文件",
		Filters: []runtime.FileFilter{
			{DisplayName: "文本文件 (*.txt, *.md)", Pattern: "*.txt;*.md"},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("打开文件对话框失败: %w", err)
	}
	if filePath == "" {
		return nil, nil // 用户取消
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	content := string(data)
	if len(content) > 50000 {
		content = content[:50000] + "\n\n…（文件过大，已截断至前 50000 字符）"
	}

	return map[string]interface{}{
		"content":  content,
		"filename": filepath.Base(filePath),
		"size":     len(data),
	}, nil
}

// GenerateOutlineWithDialogue 使用学生-专家 LLM 对话生成大纲（蒸馏自 MM-StoryAgent）
func (a *App) GenerateOutlineWithDialogue(storyPrompt string, numChapters int, maxTurns int) (map[string]interface{}, error) {
	if a.outlineAgent == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	result, err := a.outlineAgent.GenerateOutlineWithDialogue(a.ctx, storyPrompt, numChapters, maxTurns)
	if err != nil {
		return nil, err
	}

	// 将生成的章节转为 OutlineNode 并保存
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
		if i >= numChapters { break }
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
		"nodes":       savedNodes,
		"dialogue":    result.Dialogue,
		"chapters":    result.Chapters,
	}, nil
}
