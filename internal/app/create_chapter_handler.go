package app

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/characterlib"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
	"github.com/gaea/gaea/internal/util"
)

// CreateChapter 生成章节。minWords 为单章目标字数（<=0 使用默认 5000）；
// temperature 为生成温度（<=0 使用模型服务端默认值）。
func (a *writingState) CreateChapter(setting, prevSummary, plotReq string, chapterNum int, branchFromNodeID string, skillName string, minWords int, temperature float64) (map[string]interface{}, error) {
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
			// 每章摘要截断 200 rune（原 100，前文上下文加厚）
			prevParts = append(prevParts, fmt.Sprintf("第%d章：%s", cn, util.Truncate(n.Summary, 200)))
		}
	}
	prevSummary = strings.Join(prevParts, "\n\n")

	if minWords <= 0 {
		minWords = 5000
	}

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
	// 上下文增强：追加未回收伏笔 + 世界观要点区段。全部容错注入——读取失败或
	// 无数据时静默跳过（不追加空区段），绝不因增强失败中断章节生成主链路；
	// 新增区段合计受 ctxBudgetTotal 预算约束，超出逐段截断。
	if extra := buildChapterContextSections(pm); extra != "" {
		userPrompt += extra + "\n"
	}

	systemPrompt := tmpl.BuildSystemPrompt("")
	// create-chapter 模板以 {word_count} 占位符声明目标字数（prompts/create-chapter.json），
	// 用户可在创作页调整目标字数；用实际 minWords 精确替换占位符，避免模型仍按
	// 固定字数生成，同时杜绝旧 ReplaceAll("5000") 误伤模板中其他 "5000" 字样。
	systemPrompt = substituteWordCount(systemPrompt, minWords)
	if skillMD != "" && a.skillLoader != nil {
		if s := a.skillLoader.Get(skillMD); s != nil {
			systemPrompt = a.skillLoader.InjectSkill(systemPrompt, skillMD)
			slog.Info("CreateChapter Skill 已注入", "name", s.Name, "version", s.Version)
		} else {
			slog.Warn("CreateChapter Skill 未找到", "skill", skillMD)
		}
	}
	const maxContinues = 20
	// 5. 确定/创建节点（同步，前端立即可用）
	targetNum, nodeID, branch := a.ensureChapterNode(pm, of, chapterNum, branchFromNodeID)
	if nodeID == "" {
		return nil, fmt.Errorf("创建章节节点失败")
	}

	// 6. 按章节互斥（T6-7.2）：同一章节（同一 NNN.md 目标文件）并发生成直接拒绝，
	//    不同章节可并行。登记时创建请求级 context，供前端 CancelCreateChapter 取消，
	//    取消经该 context 传播到 ChatStream 与流读取循环。
	genKey := chapterGenKey(targetNum, branch)
	genCtx, genCancel, err := a.registerChapterGen(genKey, targetNum, branch)
	if err != nil {
		return nil, err
	}

	// 启动流式生成 + 字数守卫（续写模式）
	go func() {
		defer a.unregisterChapterGen(genKey, genCancel)
		a.streamCreateChapter(genCtx, pm, of, setting, prevSummary, plotReq, chapterNum, branchFromNodeID, systemPrompt, userPrompt, minWords, maxContinues, temperature, targetNum, nodeID, branch)
	}()

	return map[string]interface{}{
		"streaming":  true,
		"chapterNum": targetNum,
		"nodeId":     nodeID,
		"branch":     branch,
	}, nil
}

// chapterGenKey 章节生成任务标识：目标章节文件（NNN.md / NNN{branch}.md）。
func chapterGenKey(chapterNum int, branch string) string {
	return fmt.Sprintf("%d|%s", chapterNum, branch)
}

// registerChapterGen 登记进行中的章节生成并创建请求级 context。
// 同一章节已在生成时返回明确错误（拒绝并发写同一 NNN.md）。
func (a *writingState) registerChapterGen(key string, chapterNum int, branch string) (context.Context, context.CancelFunc, error) {
	a.chapterGenMu.Lock()
	defer a.chapterGenMu.Unlock()
	if a.chapterGenCancels == nil {
		a.chapterGenCancels = make(map[string]context.CancelFunc)
	}
	if _, running := a.chapterGenCancels[key]; running {
		label := fmt.Sprintf("第%d章", chapterNum)
		if branch != "" {
			label = fmt.Sprintf("第%d%s章", chapterNum, branch)
		}
		return nil, nil, fmt.Errorf("%s 正在生成中，请等待完成或先取消", label)
	}
	parent := a.ctx
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	a.chapterGenCancels[key] = cancel
	return ctx, cancel, nil
}

// unregisterChapterGen 生成结束（成功/失败/取消）后移除登记并释放取消函数。
// cancel 幂等：生成已结束，仅释放 context 资源。
func (a *writingState) unregisterChapterGen(key string, cancel context.CancelFunc) {
	a.chapterGenMu.Lock()
	delete(a.chapterGenCancels, key)
	a.chapterGenMu.Unlock()
	cancel()
}

// CancelCreateChapter 取消指定章节的进行中生成（T6-7.2）。
// 取消后 streamCreateChapter 会把已生成部分落盘并向前端发 cancelled 事件。
// 幂等：目标章节没有进行中生成时返回 false。
func (a *writingState) CancelCreateChapter(chapterNum int, branch string) bool {
	key := chapterGenKey(chapterNum, branch)
	a.chapterGenMu.Lock()
	cancel, running := a.chapterGenCancels[key]
	if running {
		delete(a.chapterGenCancels, key)
	}
	a.chapterGenMu.Unlock()
	if !running {
		return false
	}
	cancel()
	return true
}

// wordCountPlaceholder create-chapter 模板中目标字数的占位符
// （prompts/create-chapter.json 的 task 与 output 两处字数声明均使用它）。
const wordCountPlaceholder = "{word_count}"

// substituteWordCount 将模板中的 {word_count} 占位符精确替换为实际目标字数。
// 旧实现 strings.ReplaceAll("5000") 会误伤模板中其他 "5000" 字样；占位符替换
// 只命中字数声明位，其余数字字样原样保留。
func substituteWordCount(prompt string, minWords int) string {
	return strings.ReplaceAll(prompt, wordCountPlaceholder, strconv.Itoa(minWords))
}

// chapterCurrentBody 计算当前已生成的纯正文（不含 ---CHAPTER_SUMMARY--- 摘要）。
// 首次生成且未见摘要标记时正文在 fullText 中；出现标记或进入续写后正文在 bodyText 中。
func chapterCurrentBody(fullText, bodyText string, attempt int, summaryStarted bool) string {
	if attempt > 0 || summaryStarted {
		return bodyText
	}
	if idx := strings.Index(fullText, "---CHAPTER_SUMMARY---"); idx >= 0 {
		return fullText[:idx]
	}
	return fullText
}

// saveCancelledPartial 取消生成时把已生成部分原子落盘并通知前端（T6-7.2）。
// 保持正常完成的落盘路径不变，仅补充取消场景的部分写入；未生成任何内容
// （partial 为空）时只发 cancelled 事件、不写空文件，避免覆盖既有章节。
func (a *writingState) saveCancelledPartial(pm *project.Manager, fullText, bodyText string, attempt int, summaryStarted bool, targetNum int, nodeID, branch string) {
	partial := strings.TrimSpace(chapterCurrentBody(fullText, bodyText, attempt, summaryStarted))
	payload := map[string]interface{}{
		"type":       "cancelled",
		"chapterNum": targetNum,
		"branch":     branch,
		"nodeId":     nodeID,
		"total":      len([]rune(partial)),
	}
	if partial != "" {
		var err error
		if branch != "" {
			err = pm.WriteChapterBranch(targetNum, branch, partial)
		} else {
			err = pm.WriteChapter(targetNum, partial)
		}
		if err != nil {
			slog.Warn("取消生成：已生成部分落盘失败", "chapter", targetNum, "branch", branch, "error", err)
		} else {
			payload["content"] = partial
		}
	}
	a.emit("create-chapter-stream", payload)
}

// streamCreateChapter 在后台 goroutine 中流式生成章节，字数不足时续写。
// ctx 为请求级 context（T6-7.2）：由 CreateChapter 绑定入口创建，前端调用
// CancelCreateChapter 时取消；取消传播到 ChatStream 与流读取循环。
func (a *writingState) streamCreateChapter(ctx context.Context, pm *project.Manager, of *types.OutlineFile, setting, prevSummary, plotReq string, chapterNum int, branchFromNodeID, systemPrompt, userPrompt string, minWords, maxContinues int, temperature float64, targetNum int, nodeID string, branch string) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("CreateChapter stream panic", "panic", r)
			a.emit("create-chapter-stream", map[string]interface{}{"type": "error", "error": fmt.Sprintf("内部错误: %v", r)})
		}
	}()

	var fullText string // 全部原始内容（含摘要标记，用于提取 summary）
	var bodyText string // 纯正文（不含 ---CHAPTER_SUMMARY--- 及摘要，用于字数和最终保存）
	currentPrompt := userPrompt

	// S1.5-B play 内容护栏：temperature_max/max_output_tokens 钳制（未配置
	// = 零值 = 请求与现状逐字节一致），每次续写尝试均按同一钳制值下发。
	g := playGuardrails()

	for attempt := 0; attempt <= maxContinues; attempt++ {
		if attempt > 0 {
			a.emit("create-chapter-stream", map[string]interface{}{
				"type":    "phase",
				"phase":   "continuing",
				"attempt": attempt,
				"current": len([]rune(bodyText)),
				"target":  minWords,
			})
			// 续写模式：基于已有内容继续
			bodyLen := len([]rune(bodyText))
			need := minWords - bodyLen
			// 截取已有内容末尾作为续写上下文（避免截开头导致重复）
			// 续写尾部 1500 rune（原 500）；正文不足 1500 时 tailRunes 保持全量（带全章）。
			allRunes := []rune(bodyText)
			tailRunes := allRunes
			const tailCtx = 1500
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
		} else {
			a.emit("create-chapter-stream", map[string]interface{}{
				"type":   "phase",
				"phase":  "writing",
				"target": minWords,
			})
		}

		// 向 AI 控制台发送请求日志
		featEng, featModel, _ := a.routeModel("novel")
		a.emit("xai-output", map[string]interface{}{
			"type":   "request",
			"model":  featModel,
			"system": systemPrompt,
			"user":   currentPrompt,
		})

		req := &ai.ChatRequest{
			Model:    featModel,
			EngineID: featEng,
			Messages: []ai.ChatMessage{{Role: "system", Content: systemPrompt}, {Role: "user", Content: currentPrompt}},
		}
		applyChapterGuardrails(req, temperature, g)
		chunks, err := a.client.ChatStream(ctx, req)
		if err != nil {
			a.emit("create-chapter-stream", map[string]interface{}{"type": "error", "error": err.Error()})
			return
		}

		var summaryStarted bool
	loop:
		for {
			select {
			case <-ctx.Done():
				// T6-7.2 用户取消：把已生成部分落盘后退出（不再续写）
				a.saveCancelledPartial(pm, fullText, bodyText, attempt, summaryStarted, targetNum, nodeID, branch)
				return
			case chunk, ok := <-chunks:
				if !ok {
					break loop
				}
				if chunk.Error != "" {
					if ctx.Err() != nil {
						// 取消导致的流中断（parseStreamEvents 在 ctx 取消时发 error 帧）：
						// 与上方 ctx.Done 分支等价，同样落盘已生成部分。
						a.saveCancelledPartial(pm, fullText, bodyText, attempt, summaryStarted, targetNum, nodeID, branch)
						return
					}
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
							// 计算当前 chunk 中位于标记之前的字节数，用字节切片；
							// 原实现按 rune 下标切片，遇到中文等多字节字符会越界。
							prefixLen := idx - (len(fullText) - len(chunk.Content))
							if prefixLen > 0 && prefixLen <= len(chunk.Content) {
								bodyPart := chunk.Content[:prefixLen]
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

	// 将摘要落盘，分支章节使用独立摘要文件，避免分支复用主线摘要
	if summary != "" {
		label := fmt.Sprintf("第%d章", targetNum)
		if branch != "" {
			label = fmt.Sprintf("第%d%s章", targetNum, branch)
		}
		chapterSummary := &types.ChapterSummary{Title: label, Summary: summary}
		var summaryErr error
		if branch != "" {
			summaryErr = pm.WriteChapterBranchSummary(targetNum, branch, chapterSummary)
		} else {
			summaryErr = pm.WriteChapterSummary(targetNum, chapterSummary)
		}
		if summaryErr != nil {
			slog.Warn("章节摘要落盘失败", "chapter", targetNum, "branch", branch, "error", summaryErr)
		}
	}

	// 更新节点摘要并保存
	for i := range of.Nodes {
		if of.Nodes[i].ID == nodeID {
			of.Nodes[i].Summary = summary
			of.Nodes[i].Status = types.OutlineDone
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
	_, respModel, _ := a.routeModel("novel")
	a.emit("xai-output", map[string]interface{}{
		"type":    "response",
		"model":   respModel,
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
func (a *writingState) ensureChapterNode(pm *project.Manager, of *types.OutlineFile, chapterNum int, branchFromNodeID string) (targetNum int, nodeID string, branch string) {
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
			OrderIndex: targetNum, Status: types.OutlineWriting,
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
			if !used[l] {
				branch = l
				break
			}
		}
		if branch == "" {
			return 0, "", ""
		}
		nodeID = fmt.Sprintf("n_%d", time.Now().UnixMilli())
		of.Nodes = append(of.Nodes, types.OutlineNode{
			ID: nodeID, ParentID: branchFromNodeID,
			Title:      fmt.Sprintf("第%d%s章", targetNum, branch),
			OrderIndex: targetNum, Branch: branch,
			Status: types.OutlineWriting, ChapterFile: fmt.Sprintf("%03d%s.md", targetNum, branch),
		})
		pm.WriteOutlines(of)
		return
	}
	targetNum = len(of.Nodes) + 1
	nodeID = fmt.Sprintf("n_%d", time.Now().UnixMilli())
	of.Nodes = append(of.Nodes, types.OutlineNode{
		ID: nodeID, Title: fmt.Sprintf("第%d章", targetNum),
		OrderIndex: targetNum, Status: types.OutlineWriting,
	})
	pm.WriteOutlines(of)
	return
}

// ── 章节生成上下文增强（伏笔 / 世界观 / 角色卡）──────────────────────
// 以下注入全部容错：文件缺失、JSON 损坏一律静默跳过对应区段，
// 绝不让上下文增强失败导致章节生成失败。

// 上下文注入预算（均为 rune 数）。各区段先按条目/维度截断，合计超过
// ctxBudgetTotal 时再逐段对半压缩，保证 prompt 不爆炸。
const (
	ctxBudgetTotal        = 4000 // 新增区段（伏笔+世界观）合计预算
	ctxForeshadowBudget   = 1600 // 伏笔区正文上限
	ctxWorldviewBudget    = 1600 // 世界观区正文上限
	ctxForeshadowLineLen  = 100  // 单条伏笔描述截断
	ctxForeshadowMaxItems = 15   // 最多注入的伏笔条数
	ctxWorldviewDimLen    = 150  // 世界观单维度截断
)

// 角色摘要字段预算。性格截断由原 20 rune 放宽到 60，另补身份/目标/关系要点。
const (
	charPersonalityLen  = 60   // 性格截断上限（原 20）
	charFieldLen        = 60   // 身份背景/目标等字段截断
	charRelationDescLen = 30   // 单条关系描述截断
	charMaxRelations    = 3    // 每角色最多注入关系数
	charSummaryBudget   = 2400 // 角色摘要整体预算（rune）
)

// runeLen 字符串的 rune 长度
func runeLen(s string) int {
	return len([]rune(s))
}

// truncateBudget 将 s 截断到不超过 budget 个 rune（省略号计入预算，
// util.Truncate 的 "..." 后缀占 3 rune）。
func truncateBudget(s string, budget int) string {
	if budget < 3 {
		budget = 3
	}
	if runeLen(s) <= budget {
		return s
	}
	return util.Truncate(s, budget-3)
}

// buildChapterContextSections 组装章节生成 prompt 的增强上下文区段
// （未回收伏笔 + 世界观要点）。无数据或读取失败时返回 ""，调用方不追加
// 任何内容（prompt 中不出现空区段）。
func buildChapterContextSections(pm *project.Manager) string {
	sections := make([]string, 0, 2)
	if body := buildForeshadowSection(pm); body != "" {
		sections = append(sections, "## 未回收伏笔（创作约束）\n"+
			"以下伏笔已埋设尚未回收，写作时不得与之矛盾；可自然推进，不要强行提前揭穿：\n"+body)
	}
	if body := buildWorldviewSection(pm); body != "" {
		sections = append(sections, "## 世界观要点\n"+body)
	}
	return joinWithBudget(ctxBudgetTotal, sections...)
}

// joinWithBudget 用空行拼接区段；合计超过 budget（rune）时对每段截断至
// budget/段数（扣除拼接符开销），保证新增上下文不会撑爆生成 prompt。
func joinWithBudget(budget int, sections ...string) string {
	if len(sections) == 0 {
		return ""
	}
	joined := strings.Join(sections, "\n\n")
	if runeLen(joined) > budget {
		joiner := 2 * (len(sections) - 1) // "\n\n" 拼接开销
		per := (budget - joiner) / len(sections)
		for i := range sections {
			sections[i] = truncateBudget(sections[i], per)
		}
		joined = strings.Join(sections, "\n\n")
	}
	return joined
}

// buildForeshadowSection 读 foreshadows.json，把未回收（planted/hinted，
// 即「未回收/进行中」）的伏笔整理为创作约束区正文：每条一行、描述截断、
// 最多 ctxForeshadowMaxItems 条、整体不超过 ctxForeshadowBudget。
// 读失败或没有未回收伏笔时返回 ""。
func buildForeshadowSection(pm *project.Manager) string {
	if pm == nil {
		return ""
	}
	ff, err := pm.ReadForeshadows()
	if err != nil || ff == nil || len(ff.Items) == 0 {
		return ""
	}
	statusLabel := map[types.ForeshadowStatus]string{
		types.ForeshadowPlanted: "已埋设",
		types.ForeshadowHinted:  "已暗示·进行中",
	}
	var lines []string
	for _, f := range ff.Items {
		if f.Status == types.ForeshadowRevealed {
			continue // 已回收的伏笔不再注入
		}
		label := statusLabel[f.Status]
		if label == "" {
			label = string(f.Status)
		}
		longTag := ""
		if f.IsLongTerm {
			longTag = "（长线）"
		}
		desc := util.Truncate(strings.TrimSpace(f.Description), ctxForeshadowLineLen)
		lines = append(lines, fmt.Sprintf("- [%s] %s%s（状态：%s）", f.Category, desc, longTag, label))
		if len(lines) >= ctxForeshadowMaxItems {
			break
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return truncateBudget(strings.Join(lines, "\n"), ctxForeshadowBudget)
}

// buildWorldviewSection 读世界观并拆分为「维度要点」正文，每维度截断
// ctxWorldviewDimLen、整体不超过 ctxWorldviewBudget。复用 pm.ReadWorldview
// （worldview.json 优先，旧 worldview.md 兜底）；旧 md 无 "## " 维度标题时
// 整块压缩为单条要点。读失败或全空时返回 ""。
func buildWorldviewSection(pm *project.Manager) string {
	if pm == nil {
		return ""
	}
	md, err := pm.ReadWorldview()
	if err != nil || strings.TrimSpace(md) == "" {
		return ""
	}
	var lines []string
	curTitle := ""
	var cur []string
	flush := func() {
		content := strings.TrimSpace(strings.Join(cur, " "))
		if curTitle != "" && content != "" {
			lines = append(lines, fmt.Sprintf("- %s：%s", curTitle, util.Truncate(content, ctxWorldviewDimLen)))
		}
		cur = cur[:0]
	}
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(line, "## ") {
			flush()
			curTitle = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			continue
		}
		if curTitle != "" {
			cur = append(cur, line)
		}
	}
	flush()
	if len(lines) == 0 {
		// 旧 md 没有任何 "## " 维度标题：整块作为单条要点压缩注入
		return "- 世界观：" + util.Truncate(strings.TrimSpace(md), ctxWorldviewDimLen*4)
	}
	return truncateBudget(strings.Join(lines, "\n"), ctxWorldviewBudget)
}

// buildCharacterSummary 构建角色摘要字符串（用于注入章节生成 prompt）
// 格式：每角色一行「姓名：身份·性格·身份背景·目标·关系要点·状态」，
// 各字段按 rune 截断（性格上限 charPersonalityLen），整体不超过
// charSummaryBudget；无背景/目标/关系的角色保持原有精简形态。
func (a *writingState) buildCharacterSummary(pm *project.Manager) string {
	cf, err := pm.ReadCharacters()
	if err != nil || cf == nil || len(cf.Characters) == 0 {
		return "（暂无角色设定）"
	}
	relDigest := buildRelationDigest(cf)
	var lines []string
	for _, ch := range cf.Characters {
		roleLabel := map[string]string{
			"protagonist": "主角", "antagonist": "反派",
			"supporting": "配角", "minor": "次要",
		}[ch.RoleType]
		if roleLabel == "" {
			roleLabel = ch.RoleType
		}
		personality := util.Truncate(strings.TrimSpace(ch.Personality), charPersonalityLen)
		line := fmt.Sprintf("- %s：%s·%s", ch.Name, roleLabel, personality)
		if bg := strings.TrimSpace(ch.Background); bg != "" {
			line += "·身份：" + util.Truncate(bg, charFieldLen)
		}
		if mo := strings.TrimSpace(ch.Motivation); mo != "" {
			line += "·目标：" + util.Truncate(mo, charFieldLen)
		}
		if rel := relDigest[ch.ID]; rel != "" {
			line += "·关系：" + rel
		}
		line += "·" + ch.Status
		lines = append(lines, line)
	}
	return truncateBudget(strings.Join(lines, "\n"), charSummaryBudget)
}

// buildRelationDigest 把 characters.json 的 relationships 折叠为
// 「角色ID → 关系要点」，只保留双方都能对照到项目角色的关系；
// 每角色最多 charMaxRelations 条。无关系数据时返回空 map。
func buildRelationDigest(cf *types.CharacterFile) map[string]string {
	if cf == nil || len(cf.Relationships) == 0 {
		return nil
	}
	nameByID := make(map[string]string, len(cf.Characters))
	for _, ch := range cf.Characters {
		nameByID[ch.ID] = ch.Name
	}
	relationLabel := map[string]string{
		"friend": "好友", "enemy": "敌对", "family": "亲人", "mentor": "师徒",
		"rival": "宿敌", "lover": "恋人", "member": "隶属", "leader": "统领",
	}
	rels := make(map[string][]string)
	for _, r := range cf.Relationships {
		fromName, toName := nameByID[r.FromID], nameByID[r.ToID]
		if fromName == "" || toName == "" {
			continue
		}
		label := relationLabel[r.RelationType]
		if label == "" {
			label = r.RelationType
		}
		item := fmt.Sprintf("与%s为%s", toName, label)
		if d := strings.TrimSpace(r.Description); d != "" {
			item += "（" + util.Truncate(d, charRelationDescLen) + "）"
		}
		rels[r.FromID] = append(rels[r.FromID], item)
	}
	digest := make(map[string]string, len(rels))
	for id, items := range rels {
		if len(items) > charMaxRelations {
			items = items[:charMaxRelations]
		}
		digest[id] = strings.Join(items, "、")
	}
	return digest
}

// extractNewCharacters 从章节摘要中提取新角色，与项目角色 + 全局角色库对照去重。
// 返回两部分：
//
//	newChars       项目与角色库都没有的全新角色名
//	libraryMatches 角色库中已有同名角色（前端提示直接关联，不新建）
func (a *writingState) extractNewCharacters(pm *project.Manager, summary *types.ChapterSummary) (newChars []string, libraryMatches []characterlib.Character) {
	if summary == nil || len(summary.CharactersAppeared) == 0 {
		return nil, nil
	}

	// 1. 对照项目 characters.json 去重
	existingNames := make(map[string]bool)
	if cf, err := pm.ReadCharacters(); err == nil && cf != nil {
		for _, ch := range cf.Characters {
			existingNames[ch.Name] = true
		}
	}
	var unknown []string
	for _, name := range summary.CharactersAppeared {
		if name == "" || existingNames[name] {
			continue
		}
		unknown = append(unknown, name)
	}
	if len(unknown) == 0 {
		return nil, nil
	}

	// 2. 对照全局角色库去重：库内已有同名 → 走关联而非新建
	if a.app != nil && a.app.charLib != nil {
		var remain []string
		for _, name := range unknown {
			c, err := a.app.charLib.FindByName(name)
			if err != nil || c == nil {
				remain = append(remain, name)
				continue
			}
			libraryMatches = append(libraryMatches, *c)
		}
		unknown = remain
	}
	return unknown, libraryMatches
}

// extractCharactersAfterChapter 章节生成后异步提取角色
// 调用 chapter-summary AI → 提取 characters_appeared → 对照去重 → 通知前端
func (a *writingState) extractCharactersAfterChapter(pm *project.Manager, content string, chapterNum int) {
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

	// 2. 对照项目 + 角色库去重
	newChars, libMatches := a.extractNewCharacters(pm, summary)
	if len(newChars) == 0 && len(libMatches) == 0 {
		slog.Info("章节角色提取完成，无新角色", "chapter", chapterNum, "appeared", len(summary.CharactersAppeared))
		return
	}

	// 3. 通知前端发现新角色（全新名字 + 库内已有同名角色）
	payload := map[string]interface{}{
		"chapterNum": chapterNum,
		"characters": newChars,
	}
	if len(libMatches) > 0 {
		matches := make([]map[string]interface{}, 0, len(libMatches))
		for _, c := range libMatches {
			matches = append(matches, map[string]interface{}{
				"id":          c.ID,
				"name":        c.Name,
				"roleType":    c.RoleType,
				"portraitUrl": c.PortraitURL,
			})
		}
		payload["libraryMatches"] = matches
	}
	slog.Info("发现新角色", "chapter", chapterNum, "new", newChars, "libraryMatches", len(libMatches))
	a.emit("new-characters-discovered", payload)
}
