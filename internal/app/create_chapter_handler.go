package app

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
	"github.com/gaea/gaea/internal/util"
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
		skillMD = skillName // 保存 skill 名称供后续注入
		slog.Info("CreateChapter 将注入 Skill", "name", skillName)
	}

	// 3. 从章节节点树提取前文摘要
	of, err := pm.ReadOutlines()
	if err != nil {
		of = &types.OutlineFile{Nodes: []types.OutlineNode{}}
	}
	var prevParts []string
	limitChapter := chapterNum
	if limitChapter <= 0 {
		limitChapter = len(of.Nodes) + 1
	}
	for _, n := range of.Nodes {
		cn := n.OrderIndex
		if cn > 0 && cn < limitChapter && n.Summary != "" {
			prevParts = append(prevParts, fmt.Sprintf("第%d章：%s", cn, util.Truncate(n.Summary, 100)))
		}
	}
	prevSummary = strings.Join(prevParts, "\n\n")

	// 4. 构建 prompt（通过模板 + Skill 注入）
	tmpl := a.eng.Get("create-chapter")
	if tmpl == nil {
		return nil, fmt.Errorf("缺少 create-chapter 模板文件")
	}

	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"plot_req":     plotReq,
		"setting":      setting,
		"characters":   a.buildCharacterSummary(pm),
		"prev_summary": prevSummary,
	})

	systemPrompt := tmpl.BuildSystemPrompt("")
	if skillMD != "" && a.skillLoader != nil {
		if s := a.skillLoader.Get(skillMD); s != nil {
			systemPrompt = a.skillLoader.InjectSkill(systemPrompt, skillMD)
			slog.Info("CreateChapter Skill 已注入", "name", s.Name, "version", s.Version)
		} else {
			slog.Warn("CreateChapter Skill 未找到", "skill", skillMD)
		}
	}
	const minWords = 5000
	const maxContinues = 20
	// 5. 确定/创建节点（同步，前端立即可用）
	targetNum, nodeID, branch := a.ensureChapterNode(pm, of, chapterNum, branchFromNodeID)
	if nodeID == "" {
		return nil, fmt.Errorf("创建章节节点失败")
	}

	// 启动流式生成 + 字数守卫（续写模式）
	go a.streamCreateChapter(pm, of, setting, prevSummary, plotReq, chapterNum, branchFromNodeID, systemPrompt, userPrompt, minWords, maxContinues, targetNum, nodeID, branch)

	return map[string]interface{}{
		"streaming":   true,
		"chapterNum":  targetNum,
		"nodeId":      nodeID,
		"branch":      branch,
	}, nil
}

// streamCreateChapter 在后台 goroutine 中流式生成章节，字数不足时续写
func (a *App) streamCreateChapter(pm *project.Manager, of *types.OutlineFile, setting, prevSummary, plotReq string, chapterNum int, branchFromNodeID, systemPrompt, userPrompt string, minWords, maxContinues int, targetNum int, nodeID string, branch string) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("CreateChapter stream panic", "panic", r)
			a.emit("create-chapter-stream", map[string]interface{}{"type": "error", "error": fmt.Sprintf("内部错误: %v", r)})
		}
	}()

	var fullText string  // 全部原始内容（含摘要标记，用于提取 summary）
	var bodyText string  // 纯正文（不含 ---CHAPTER_SUMMARY--- 及摘要，用于字数和最终保存）
	currentPrompt := userPrompt

	for attempt := 0; attempt <= maxContinues; attempt++ {
		if attempt > 0 {
			// 续写模式：基于已有内容继续
			bodyLen := len([]rune(bodyText))
			need := minWords - bodyLen
			// 截取已有内容末尾作为续写上下文（避免截开头导致重复）
			allRunes := []rune(bodyText)
			tailRunes := allRunes
			const tailCtx = 500
			if len(allRunes) > tailCtx {
				tailRunes = allRunes[len(allRunes)-tailCtx:]
			}
			tailText := string(tailRunes)
			if len(allRunes) > tailCtx {
				tailText = "…(前文省略)\n" + tailText
			}
			currentPrompt = fmt.Sprintf("【续写指令】当前已写%d字，请从断点处直接继续写至少%d字。不要重复已写内容，不要加章节标题或前言，直接接着写正文。\n\n已有内容末尾：\n%s\n\n请继续：",
				bodyLen, need, tailText)
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
				fullText += chunk.Content

				if !summaryStarted {
					if attempt == 0 {
						// 首次生成：流式发送正文（需处理摘要标记）
						if idx := strings.Index(fullText, "---CHAPTER_SUMMARY---"); idx >= 0 {
							summaryStarted = true
							bodyText = fullText[:idx] // 锁定纯正文（标记之前）
							bodyLen := idx - (len(fullText) - len(chunk.Content))
							if bodyLen > 0 {
								bodyPart := string([]rune(chunk.Content)[:bodyLen])
								a.emit("create-chapter-stream", map[string]interface{}{
									"type": "chunk", "content": bodyPart, "total": len([]rune(bodyText)),
								})
							}
						} else {
							a.emit("create-chapter-stream", map[string]interface{}{
								"type": "chunk", "content": chunk.Content, "total": len([]rune(fullText)),
							})
						}
					} else {
						// 续写：直接流式发送（无摘要标记），同时追加到纯正文
						bodyText += chunk.Content
						a.emit("create-chapter-stream", map[string]interface{}{
							"type": "chunk", "content": chunk.Content, "total": len([]rune(bodyText)),
						})
					}
				}
			}
		}

		// 首次生成无摘要标记时，bodyText = fullText
		if attempt == 0 && !summaryStarted {
			bodyText = fullText
		}

		// 检查字数（用纯正文 bodyText）
		if len([]rune(bodyText)) >= minWords {
			break // 达标
		}
		}

	// 提取正文和摘要
	content := strings.TrimSpace(bodyText) // 纯正文（含续写内容）
	summary := ""
	if idx := strings.Index(fullText, "---CHAPTER_SUMMARY---"); idx >= 0 {
		summary = strings.TrimSpace(fullText[idx+len("---CHAPTER_SUMMARY---"):])
		summary = util.Truncate(summary, 100)
	}

	// 更新节点摘要并保存
	for i := range of.Nodes {
		if of.Nodes[i].ID == nodeID {
			of.Nodes[i].Summary = summary
			of.Nodes[i].Status = "written"
			break
		}
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
		"content": content,
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

	// 异步提取章节角色（不阻塞 done 事件）
	go a.extractCharactersAfterChapter(pm, content, targetNum)
}
// ensureChapterNode 确定章节号并创建/复用节点（同步，在 AI 生成前执行）
func (a *App) ensureChapterNode(pm *project.Manager, of *types.OutlineFile, chapterNum int, branchFromNodeID string) (targetNum int, nodeID string, branch string) {
	if chapterNum > 0 {
		targetNum = chapterNum
		for _, n := range of.Nodes {
			if n.OrderIndex == chapterNum && n.Branch == "" {
				return targetNum, n.ID, ""
			}
		}
		nodeID = fmt.Sprintf("n_%d", time.Now().UnixMilli())
		of.Nodes = append(of.Nodes, types.OutlineNode{
			ID: nodeID, Title: fmt.Sprintf("第%d章", targetNum),
			OrderIndex: targetNum, Status: "generating",
		})
		pm.WriteOutlines(of)
		return
	}
	if branchFromNodeID != "" {
		var parent *types.OutlineNode
		for i := range of.Nodes {
			if of.Nodes[i].ID == branchFromNodeID {
				parent = &of.Nodes[i]
				break
			}
		}
		if parent == nil {
			return 0, "", ""
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
			return 0, "", ""
		}
		nodeID = fmt.Sprintf("n_%d", time.Now().UnixMilli())
		of.Nodes = append(of.Nodes, types.OutlineNode{
			ID: nodeID, ParentID: branchFromNodeID,
			Title: fmt.Sprintf("第%d%s章", targetNum, branch),
			OrderIndex: targetNum, Branch: branch,
			Status: "generating", ChapterFile: fmt.Sprintf("%03d%s.md", targetNum, branch),
		})
		pm.WriteOutlines(of)
		return
	}
	targetNum = len(of.Nodes) + 1
	nodeID = fmt.Sprintf("n_%d", time.Now().UnixMilli())
	of.Nodes = append(of.Nodes, types.OutlineNode{
		ID: nodeID, Title: fmt.Sprintf("第%d章", targetNum),
		OrderIndex: targetNum, Status: "generating",
	})
	pm.WriteOutlines(of)
	return
}


// buildCharacterSummary 构建角色摘要字符串（用于注入章节生成 prompt）
// 格式：每角色一行「姓名 · 身份 · 性格关键词 · 状态」
func (a *App) buildCharacterSummary(pm *project.Manager) string {
	cf, err := pm.ReadCharacters()
	if err != nil || cf == nil || len(cf.Characters) == 0 {
		return "（暂无角色设定）"
	}
	var lines []string
	for _, ch := range cf.Characters {
		roleLabel := map[string]string{
			"protagonist": "主角", "antagonist": "反派",
			"supporting": "配角", "minor": "次要",
		}[ch.RoleType]
		if roleLabel == "" {
			roleLabel = ch.RoleType
		}
		personality := ch.Personality
		if len([]rune(personality)) > 20 {
			personality = string([]rune(personality)[:20]) + "…"
		}
		line := fmt.Sprintf("- %s：%s·%s·%s", ch.Name, roleLabel, personality, ch.Status)
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// extractNewCharacters 从章节摘要中提取新角色，与已有角色对照去重
// 返回新角色名列表，如果全部已知则返回空
func (a *App) extractNewCharacters(pm *project.Manager, summary *types.ChapterSummary) []string {
	if summary == nil || len(summary.CharactersAppeared) == 0 {
		return nil
	}
	cf, err := pm.ReadCharacters()
	if err != nil || cf == nil {
		return summary.CharactersAppeared // 无已有角色，全部算新
	}
	existingNames := make(map[string]bool)
	for _, ch := range cf.Characters {
		existingNames[ch.Name] = true
	}
	var newChars []string
	for _, name := range summary.CharactersAppeared {
		if !existingNames[name] && name != "" {
			newChars = append(newChars, name)
		}
	}
	return newChars
}

// extractCharactersAfterChapter 章节生成后异步提取角色
// 调用 chapter-summary AI → 提取 characters_appeared → 对照去重 → 通知前端
func (a *App) extractCharactersAfterChapter(pm *project.Manager, content string, chapterNum int) {
	if a.chapterAgent == nil {
		slog.Warn("extractCharactersAfterChapter: chapterAgent 未初始化")
		return
	}

	// 1. 调用 AI 提取结构化摘要（含 characters_appeared）
	slog.Info("开始提取章节角色", "chapter", chapterNum)
	summary, err := a.chapterAgent.GenerateSummary(a.ctx, content)
	if err != nil {
		slog.Warn("提取章节角色失败", "chapter", chapterNum, "error", err)
		return
	}

	// 2. 对照已有角色去重
	newChars := a.extractNewCharacters(pm, summary)
	if len(newChars) == 0 {
		slog.Info("章节角色提取完成，无新角色", "chapter", chapterNum, "appeared", len(summary.CharactersAppeared))
		return
	}

	// 3. 通知前端发现新角色
	slog.Info("发现新角色", "chapter", chapterNum, "new", newChars)
	a.emit("new-characters-discovered", map[string]interface{}{
		"chapterNum": chapterNum,
		"characters": newChars,
	})
}
