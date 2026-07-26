package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/wubigork/wubigork/internal/types"
)

// chapterNum: 0=追加末尾, >0=覆盖该章
// branchFromNodeID: 非空=作为该节点的子分支追加
func (a *App) CreateChapter(setting, prevSummary, plotReq string, chapterNum int, branchFromNodeID string) (map[string]interface{}, error) {
	if a.client == nil {
		return nil, fmt.Errorf("AI client not ready")
	}

	userPrompt := fmt.Sprintf(`【本章剧情要求】
%s

【小说设定】
%s

【前文章节摘要】
%s

请直接撰写本章正文（2000-4000字），然后在文末附加一段章节摘要，格式如下：

---CHAPTER_SUMMARY---
关键事件：（一句话概括本章核心情节）
人物变化：（角色关系或状态的变化）
伏笔/悬念：（本章埋下的伏笔或结尾悬念）`, plotReq, setting, prevSummary)

	raw, err := a.client.ChatSimpleStream(a.ctx, a.cfg.Model,
		"你是一位专业小说作家。直接撰写小说正文，不要任何前言。文末按要求附加章节摘要。文笔流畅自然。", userPrompt)
	if err != nil {
		return nil, fmt.Errorf("generate failed: %w", err)
	}

	content := raw
	summary := ""
	if idx := strings.Index(raw, "---CHAPTER_SUMMARY---"); idx >= 0 {
		content = strings.TrimSpace(raw[:idx])
		summary = strings.TrimSpace(raw[idx+len("---CHAPTER_SUMMARY---"):])
	}

	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("project not open")
	}

	of, err := pm.ReadOutlines()
	if err != nil {
		of = &types.OutlineFile{Nodes: []types.OutlineNode{}}
	}

	var targetNum int
	var nodeID string
	var title string

	if chapterNum > 0 && chapterNum <= len(of.Nodes) {
		// 覆盖已有章节
		targetNum = chapterNum
		for i, n := range of.Nodes {
			if n.OrderIndex == chapterNum {
				nodeID = n.ID
				of.Nodes[i].Summary = summary
				break
			}
		}
	} else {
		// 追加新节点
		targetNum = len(of.Nodes) + 1
		nodeID = fmt.Sprintf("n_%d", time.Now().UnixMilli())
		title = fmt.Sprintf("第%d章", targetNum)
		parentID := branchFromNodeID
		if parentID == "" {
			parentID = "" // 根节点
		}
		of.Nodes = append(of.Nodes, types.OutlineNode{
			ID: nodeID, ParentID: parentID,
			Title: title, OrderIndex: targetNum,
			Status: "written", Summary: summary,
		})
	}

	if err := pm.WriteOutlines(of); err != nil {
		return nil, fmt.Errorf("save outline failed: %w", err)
	}
	if err := pm.WriteChapter(targetNum, content); err != nil {
		return nil, fmt.Errorf("save chapter failed: %w", err)
	}

	return map[string]interface{}{
		"content":    content,
		"nodeId":     nodeID,
		"chapterNum": targetNum,
		"summary":    summary,
	}, nil
}
