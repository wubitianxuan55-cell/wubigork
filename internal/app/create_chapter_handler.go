package app

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/wubigork/wubigork/internal/ai"
	"github.com/wubigork/wubigork/internal/project"
	"github.com/wubigork/wubigork/internal/types"
	"github.com/wubigork/wubigork/internal/util"
)

func (a *App) CreateChapter(setting, prevSummary, plotReq string, chapterNum int, branchFromNodeID string) (map[string]interface{}, error) {
	if a.client == nil {
		return nil, fmt.Errorf("AI client not ready")
	}

	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("project not open")
	}

	// 1. 小说设定不允许为空
	if setting == "" {
		return nil, fmt.Errorf("小说设定为空，请先在「设定」页面填写世界观")
	}

	// 2. 从章节节点树提取前文摘要（每章一个节点摘要）
	of, err := pm.ReadOutlines()
	if err != nil {
		of = &types.OutlineFile{Nodes: []types.OutlineNode{}}
	}
	var prevParts []string
	limitChapter := chapterNum
	if limitChapter <= 0 {
		limitChapter = len(of.Nodes) + 1 // 追加模式：取所有已有序号节点
	}
	for _, n := range of.Nodes {
		cn := n.OrderIndex
		if cn > 0 && cn < limitChapter && n.Summary != "" {
			prevParts = append(prevParts, fmt.Sprintf("第%d章：%s", cn, util.Truncate(n.Summary, 100)))
		}
	}
	prevSummary = strings.Join(prevParts, "\n\n")

	// 3. 构建 user prompt
	var parts []string
	parts = append(parts, fmt.Sprintf("【本章剧情要求】\n%s", plotReq))
	parts = append(parts, fmt.Sprintf("【小说设定】\n%s", setting))
	if prevSummary != "" {
		parts = append(parts, fmt.Sprintf("【前文章节摘要】\n%s", prevSummary))
	}
	// 注入角色信息
	if chars, err := pm.ReadCharacters(); err == nil && chars != nil && len(chars.Characters) > 0 {
		var charLines []string
		for _, c := range chars.Characters {
			if c.Name == "" {
				continue
			}
			line := fmt.Sprintf("- %s", c.Name)
			if c.RoleType != "" {
				line += fmt.Sprintf("（%s）", c.RoleType)
			}
			if c.Personality != "" {
				line += fmt.Sprintf("：%s", c.Personality)
			}
			charLines = append(charLines, line)
		}
		if len(charLines) > 0 {
			parts = append(parts, fmt.Sprintf("【角色信息】\n%s", strings.Join(charLines, "\n")))
		}
	}

	parts = append(parts, `请直接撰写本章正文（2000-4000字），然后在文末附加一段章节摘要，格式如下：

---CHAPTER_SUMMARY---
关键事件：（一句话概括本章核心情节）
人物变化：（角色关系或状态的变化）
伏笔/悬念：（本章埋下的伏笔或结尾悬念）
（以上摘要总字数不超过100字）`)

	userPrompt := strings.Join(parts, "\n\n")

	systemPrompt := "你是一位专业小说作家。请保持与前文一致的文风、人物性格和世界观设定。直接撰写小说正文，不要任何前言。文末按要求附加章节摘要。文笔流畅自然。"
	const minWords = 5000
	const maxContinues = 2

	// 启动流式生成 + 字数守卫（续写模式）
	go a.streamCreateChapter(pm, of, setting, prevSummary, plotReq, chapterNum, branchFromNodeID, systemPrompt, userPrompt, minWords, maxContinues)

	return map[string]interface{}{"streaming": true}, nil
}

// streamCreateChapter 在后台 goroutine 中流式生成章节，字数不足时续写
func (a *App) streamCreateChapter(pm *project.Manager, of *types.OutlineFile, setting, prevSummary, plotReq string, chapterNum int, branchFromNodeID, systemPrompt, userPrompt string, minWords, maxContinues int) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("CreateChapter stream panic", "panic", r)
			a.emit("create-chapter-stream", map[string]interface{}{"type": "error", "error": fmt.Sprintf("内部错误: %v", r)})
		}
	}()

	var fullText string
	currentPrompt := userPrompt

	for attempt := 0; attempt <= maxContinues; attempt++ {
		if attempt > 0 {
			// 续写模式：基于已有内容继续
			bodyLen := len([]rune(fullText))
			need := minWords - bodyLen
			currentPrompt = fmt.Sprintf("【续写指令】当前已写%d字，请从断点处直接继续写至少%d字。不要重复已写内容，不要加章节标题或前言，直接接着写正文。\n\n已有内容末尾：\n%s\n\n请继续：",
				bodyLen, need, util.Truncate(fullText, 200))
			slog.Info("章节字数不足，启动续写", "current", bodyLen, "need", need, "attempt", attempt)
		}

		chunks, err := a.client.ChatStream(a.ctx, &ai.ChatRequest{
			Model:    a.cfg.Model,
			Messages: []ai.ChatMessage{{Role: "system", Content: systemPrompt}, {Role: "user", Content: currentPrompt}},
		})
		if err != nil {
			a.emit("create-chapter-stream", map[string]interface{}{"type": "error", "error": err.Error()})
			return
		}

		var segmentText string
		var summaryStarted bool
	loop:
		for {
			select {
			case <-a.ctx.Done():
				return
			case chunk, ok := <-chunks:
				if !ok {
					break loop
				}
				if chunk.Error != "" {
					a.emit("create-chapter-stream", map[string]interface{}{"type": "error", "error": chunk.Error})
					return
				}
				if chunk.Done {
					break loop
				}
				segmentText += chunk.Content
				fullText += chunk.Content

				if !summaryStarted {
					// 检测摘要分隔符，只输出正文部分
					if idx := strings.Index(fullText, "---CHAPTER_SUMMARY---"); idx >= 0 {
						summaryStarted = true
						// 计算本次 chunk 中正文部分的长度
						bodyLen := idx - (len([]rune(fullText)) - len([]rune(chunk.Content)))
						if bodyLen > 0 {
							bodyPart := string([]rune(chunk.Content)[:bodyLen])
							a.emit("create-chapter-stream", map[string]interface{}{
								"type": "chunk", "content": bodyPart, "total": len([]rune(fullText)),
							})
						}
					} else {
						a.emit("create-chapter-stream", map[string]interface{}{
							"type": "chunk", "content": chunk.Content, "total": len([]rune(fullText)),
						})
					}
				}
			}
		}

		// 检查字数（仅计正文，去掉摘要）
		bodyContent := fullText
		if idx := strings.Index(fullText, "---CHAPTER_SUMMARY---"); idx >= 0 {
			bodyContent = strings.TrimSpace(fullText[:idx])
		}
		if len([]rune(bodyContent)) >= minWords {
			break // 达标
		}
	}

	// 提取正文和摘要
	content := fullText
	summary := ""
	if idx := strings.Index(fullText, "---CHAPTER_SUMMARY---"); idx >= 0 {
		content = strings.TrimSpace(fullText[:idx])
		summary = strings.TrimSpace(fullText[idx+len("---CHAPTER_SUMMARY---"):])
		summary = util.Truncate(summary, 100)
	}

	// 保存节点和章节文件
	var targetNum int
	var nodeID string
	var title string
	var branch string

	if chapterNum > 0 {
		targetNum = chapterNum
		for i, n := range of.Nodes {
			if n.OrderIndex == chapterNum && n.Branch == "" {
				nodeID = n.ID
				of.Nodes[i].Summary = summary
				break
			}
		}
	} else if branchFromNodeID != "" {
		var parent *types.OutlineNode
		for i := range of.Nodes {
			if of.Nodes[i].ID == branchFromNodeID {
				parent = &of.Nodes[i]
				break
			}
		}
		if parent == nil {
			a.emit("create-chapter-stream", map[string]interface{}{"type": "error", "error": fmt.Sprintf("父节点不存在: %s", branchFromNodeID)})
			return
		}
		targetNum = parent.OrderIndex
		used := map[string]bool{}
		for _, n := range of.Nodes {
			if n.OrderIndex == targetNum && n.Branch != "" {
				used[n.Branch] = true
			}
		}
		for _, l := range []string{"a", "b", "c"} {
			if !used[l] { branch = l; break }
		}
		if branch == "" {
			a.emit("create-chapter-stream", map[string]interface{}{"type": "error", "error": fmt.Sprintf("第%d章已达到最大分支数（3个）", targetNum)})
			return
		}
		nodeID = fmt.Sprintf("n_%d", time.Now().UnixMilli())
		title = fmt.Sprintf("第%d%s章", targetNum, branch)
		chapterFile := fmt.Sprintf("%03d%s.md", targetNum, branch)
		of.Nodes = append(of.Nodes, types.OutlineNode{
			ID: nodeID, ParentID: branchFromNodeID,
			Title: title, OrderIndex: targetNum, Branch: branch,
			Status: "written", Summary: summary,
			ChapterFile: chapterFile,
		})
	} else {
		targetNum = len(of.Nodes) + 1
		nodeID = fmt.Sprintf("n_%d", time.Now().UnixMilli())
		title = fmt.Sprintf("第%d章", targetNum)
		of.Nodes = append(of.Nodes, types.OutlineNode{
			ID: nodeID, ParentID: "",
			Title: title, OrderIndex: targetNum,
			Status: "written", Summary: summary,
		})
	}

	if err := pm.WriteOutlines(of); err != nil {
		a.emit("create-chapter-stream", map[string]interface{}{"type": "error", "error": fmt.Sprintf("save outline: %v", err)})
		return
	}
	if branch != "" {
		if err := pm.WriteChapterBranch(targetNum, branch, content); err != nil {
			a.emit("create-chapter-stream", map[string]interface{}{"type": "error", "error": fmt.Sprintf("save chapter: %v", err)})
			return
		}
	} else {
		if err := pm.WriteChapter(targetNum, content); err != nil {
			a.emit("create-chapter-stream", map[string]interface{}{"type": "error", "error": fmt.Sprintf("save chapter: %v", err)})
			return
		}
	}

	a.emit("create-chapter-stream", map[string]interface{}{
		"type":       "done",
		"content":    content,
		"chapterNum": targetNum,
		"branch":     branch,
		"summary":    summary,
		"nodeId":     nodeID,
		"total":      len([]rune(content)),
	})
}
