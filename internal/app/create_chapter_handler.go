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

func (a *App) CreateChapter(setting, prevSummary, plotReq string, chapterNum int, branchFromNodeID string, skillName string) (map[string]interface{}, error) {
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

	// 2. 加载 Skill 写作指导
	skillMD := ""
	if skillName != "" && a.skillLoader != nil {
		if s := a.skillLoader.Get(skillName); s != nil {
			skillMD = s.Body
			slog.Info("CreateChapter Skill 已注入", "name", s.Name, "version", s.Version)
		} else {
			slog.Warn("CreateChapter Skill 未找到", "skill", skillName)
		}
	}

	// 3. 从章节节点树提取前文摘要（每章一个节点摘要）
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

	// 4. 构建 user prompt（通过模板）
	tmpl := a.eng.Get("create-chapter")
	if tmpl == nil {
		return nil, fmt.Errorf("缺少 create-chapter 模板文件")
	}

	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"plot_req":     plotReq,
		"setting":      setting,
		"prev_summary": prevSummary,
	})

	systemPrompt := tmpl.BuildSystemPrompt("")
	if skillMD != "" {
		systemPrompt += "\n\n---\n## 额外写作指导\n" + skillMD
	}
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
	prevFullLen := 0 // 续写前长度，用于去重

	for attempt := 0; attempt <= maxContinues; attempt++ {
		if attempt > 0 {
			// 续写模式：基于已有内容继续
			prevFullLen = len([]rune(fullText))
			bodyLen := prevFullLen
			need := minWords - bodyLen
			currentPrompt = fmt.Sprintf("【续写指令】当前已写%d字，请从断点处直接继续写至少%d字。不要重复已写内容，不要加章节标题或前言，直接接着写正文。\n\n已有内容末尾：\n%s\n\n请继续：",
				bodyLen, need, util.Truncate(fullText, 200))
			slog.Info("章节字数不足，启动续写", "current", bodyLen, "need", need, "attempt", attempt)
		}

		// 向 AI 控制台发送请求日志
		a.emit("xai-output", map[string]interface{}{
			"type":   "request",
			"model":  a.cfg.Model,
			"system": systemPrompt,
			"user":   currentPrompt,
		})

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

				if attempt == 0 && !summaryStarted {
					// 首次生成：流式发送正文（续写时不发，统一在循环后去重发送）
					if idx := strings.Index(fullText, "---CHAPTER_SUMMARY---"); idx >= 0 {
						summaryStarted = true
						bodyLen := idx - (len(fullText) - len(chunk.Content))
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

	// 续写增量：一次性发送本 attempt 产生的新内容（去重）
	if prevFullLen > 0 {
		newRunes := []rune(fullText)[prevFullLen:]
		if len(newRunes) > 0 {
			a.emit("create-chapter-stream", map[string]interface{}{
				"type": "chunk", "content": string(newRunes), "total": len([]rune(fullText)),
			})
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

	// AI 控制台响应日志
	a.emit("xai-output", map[string]interface{}{
		"type":    "response",
		"model":   a.cfg.Model,
		"content": util.Truncate(content, 200),
		"length":  len([]rune(content)),
	})

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
