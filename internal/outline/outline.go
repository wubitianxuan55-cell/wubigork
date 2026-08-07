package outline

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/prompt"
	"github.com/gaea/gaea/internal/types"
	"github.com/gaea/gaea/internal/util"
)

// Agent 大纲子代理 — 卷-章分层规划、节点对话、续写展开
type Agent struct {
	client ai.LLMClient
	pm     *project.Manager
	cfg    *config.Config
	eng    *prompt.Engine
}

// New 创建大纲 Agent
func New(client ai.LLMClient, pm *project.Manager, cfg *config.Config, eng *prompt.Engine) *Agent {
	return &Agent{client: client, pm: pm, cfg: cfg, eng: eng}
}

// featureModel 小说功能级模型（持久化绑定 func_novel，运行中切换即时生效；空=全局）
func (a *Agent) featureModel() (engine, model string) {
	return a.cfg.GetFeatureModel("novel")
}

// novelModelName 小说功能绑定模型名（空 = 让客户端按引擎解析默认/全局模型）
func (a *Agent) novelModelName() string {
	_, m := a.featureModel()
	return m
}

// novelEngineName 小说功能绑定引擎名（空 = 全局激活引擎）
func (a *Agent) novelEngineName() string {
	e, _ := a.featureModel()
	return e
}

// chat 功能级对话：带 novel 引擎覆盖
func (a *Agent) chat(ctx context.Context, system, user string) (string, error) {
	eng, model := a.featureModel()
	// 未绑定（model 为空）时留空，由客户端按活跃引擎解析默认模型（等价 routeModel 全局路径），
	// 避免把全局 cfg.Model 发给非 xAI 引擎导致 404（E03）。
	return a.client.ChatSimpleStreamWithOptions(ctx, model, system, user, ai.ChatSimpleOptions{EngineID: eng})
}

// ── 对话 ────────────────────────────────────────────────────

// Chat 全局大纲对话
func (a *Agent) Chat(ctx context.Context, userMsg string) (string, error) {
	of, err := a.pm.ReadOutlines()
	if err != nil {
		slog.Warn("大纲: 读取失败", "error", err)
	}
	outlineJSON := string(util.MustMarshal(of))

	tmpl := a.eng.Get("outline-chat")
	if tmpl == nil {
		return "", fmt.Errorf("缺少 outline-chat 模板文件")
	}

	systemPrompt := tmpl.BuildSystemPrompt("")
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"current_outline": string(outlineJSON),
		"user_request":    userMsg,
	})

	return a.chat(ctx, systemPrompt, userPrompt)
}

// ChatNode 针对特定大纲节点对话
func (a *Agent) ChatNode(ctx context.Context, nodeID, userMsg string) (string, error) {
	of, err := a.pm.ReadOutlines()
	if err != nil {
		slog.Warn("大纲: 读取失败", "error", err)
	}
	if of == nil {
		return "", fmt.Errorf("无大纲数据")
	}

	var target *types.OutlineNode
	for i := range of.Nodes {
		if n := FindNodeByID(&of.Nodes[i], nodeID); n != nil {
			target = n
			break
		}
	}
	if target == nil {
		return "", fmt.Errorf("未找到大纲节点: %s", nodeID)
	}

	wvCtx := a.loadWorldviewContext()
	charsCtx := a.loadCharsContext()

	tmpl := a.eng.Get("outline-chat-node")
	if tmpl == nil {
		return "", fmt.Errorf("缺少 outline-chat-node 模板文件")
	}

	systemPrompt := tmpl.BuildSystemPrompt("")
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"target_node":  fmt.Sprintf("节点「%s」\n摘要: %s\n要点: %s", target.Title, target.Summary, strings.Join(target.KeyPoints, "、")),
		"worldview":    wvCtx,
		"characters":   charsCtx,
		"user_request": userMsg,
	})

	return a.chat(ctx, systemPrompt, userPrompt)
}

// ── 续写与展开 ──────────────────────────────────────────────

// Continue 续写大纲节点
func (a *Agent) Continue(ctx context.Context, count int) (*types.OutlineFile, error) {
	of, err := a.pm.ReadOutlines()
	if err != nil {
		slog.Warn("大纲: 读取失败，以空大纲继续", "error", err)
		of = &types.OutlineFile{Nodes: []types.OutlineNode{}}
	}
	if of == nil {
		of = &types.OutlineFile{Nodes: []types.OutlineNode{}}
	}
	existingJSON := string(util.MustMarshal(of))

	allSummaries, err := a.pm.ReadAllChapterSummaries()
	if err != nil {
		slog.Warn("outline: 读取章节摘要失败", "error", err)
	}
	var summaries []string
	for _, s := range allSummaries {
		summaries = append(summaries, s.Summary)
	}

	tmpl := a.eng.Get("outline-continue")
	if tmpl == nil {
		return nil, fmt.Errorf("缺少 outline-continue 模板文件")
	}

	systemPrompt := tmpl.BuildSystemPrompt("")
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"existing_outline":  string(existingJSON),
		"chapter_summaries": strings.Join(summaries, "\n"),
		"continue_count":    fmt.Sprintf("%d", count),
		"worldview":         a.loadWorldviewContext(),
		"characters":        a.loadCharsContext(),
		"story_thread":      of.StoryThread,
	})

	reply, err := a.chat(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	var newOF types.OutlineFile
	if err := json.Unmarshal([]byte(util.ExtractJSON(reply)), &newOF); err != nil {
		return nil, fmt.Errorf("解析 AI 生成的大纲 JSON 失败: %w", err)
	}

	if of != nil {
		// 恰好 5 卷 → 全量替换（五阶段固定卷模式）
		if count == 5 {
			// 收集已有节点 ID，防御 AI 在"全量替换"模式下仍拷贝旧数据
			oldIDs := make(map[string]bool)
			var collectOldIDs func(nodes []types.OutlineNode)
			collectOldIDs = func(nodes []types.OutlineNode) {
				for _, n := range nodes {
					oldIDs[n.ID] = true
					if len(n.Children) > 0 {
						collectOldIDs(n.Children)
					}
				}
			}
			collectOldIDs(of.Nodes)

			// 显式清空旧节点
			of.Nodes = nil

			// 只保留 AI 真正新生成的节点（不是旧 ID 的拷贝）
			for i := range newOF.Nodes {
				if oldIDs[newOF.Nodes[i].ID] {
					slog.Warn("大纲: AI 在替换模式下返回了旧卷 ID，已过滤", "id", newOF.Nodes[i].ID, "title", newOF.Nodes[i].Title)
					continue
				}
				newOF.Nodes[i].Status = types.OutlinePlanned
				// 也过滤子节点中的旧 ID
				filtered := make([]types.OutlineNode, 0, len(newOF.Nodes[i].Children))
				for _, ch := range newOF.Nodes[i].Children {
					if oldIDs[ch.ID] {
						slog.Warn("大纲: AI 在替换模式下返回了旧章 ID，已过滤", "id", ch.ID, "title", ch.Title)
						continue
					}
					ch.Status = types.OutlinePlanned
					filtered = append(filtered, ch)
				}
				newOF.Nodes[i].Children = filtered
				of.Nodes = append(of.Nodes, newOF.Nodes[i])
			}

			// 确保恰好 count 卷（截断或报错）
			if len(of.Nodes) != count {
				slog.Warn("大纲: AI 返回卷数与预期不符，已截断", "expected", count, "got", len(of.Nodes))
				if len(of.Nodes) > count {
					of.Nodes = of.Nodes[:count]
				}
			}
		} else {
			// 追加模式：收集已有 ID 避免重复
			existingIDs := make(map[string]bool)
			collectIDs := func(nodes []types.OutlineNode) {}
			collectIDs = func(nodes []types.OutlineNode) {
				for _, n := range nodes {
					existingIDs[n.ID] = true
					if len(n.Children) > 0 {
						collectIDs(n.Children)
					}
				}
			}
			collectIDs(of.Nodes)

			for _, n := range newOF.Nodes {
				if !existingIDs[n.ID] {
					n.Status = types.OutlinePlanned
					if len(n.Children) > 0 {
						filtered := make([]types.OutlineNode, 0, len(n.Children))
						for _, ch := range n.Children {
							if !existingIDs[ch.ID] {
								ch.Status = types.OutlinePlanned
								filtered = append(filtered, ch)
								existingIDs[ch.ID] = true
							}
						}
						n.Children = filtered
					}
					of.Nodes = append(of.Nodes, n)
					existingIDs[n.ID] = true
				}
			}
		}
		reindexOutlineNodes(of.Nodes, "")
		return of, nil
	}
	return &newOF, nil
}

// ExpandNode 展开节点为子章节
func (a *Agent) ExpandNode(ctx context.Context, nodeID string, subCount int) (*types.OutlineFile, error) {
	of, err := a.pm.ReadOutlines()
	if err != nil {
		slog.Warn("大纲: 读取失败", "error", err)
	}
	if of == nil {
		return nil, fmt.Errorf("无大纲数据，请先创建大纲")
	}
	if len(of.Nodes) == 0 {
		return nil, fmt.Errorf("大纲为空，请先创建卷结构")
	}

	var target *types.OutlineNode
	for i := range of.Nodes {
		if n := FindNodeByID(&of.Nodes[i], nodeID); n != nil {
			target = n
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("未找到大纲节点 %s", nodeID)
	}

	tmpl := a.eng.Get("outline-expand")
	if tmpl == nil {
		return nil, fmt.Errorf("缺少 outline-expand 模板文件")
	}

	systemPrompt := tmpl.BuildSystemPrompt("")

	// 构建已有章节信息，让 AI 知道从哪里续写
	var existingChapters string
	if len(target.Children) > 0 {
		var chInfos []string
		for _, ch := range target.Children {
			chInfos = append(chInfos, fmt.Sprintf("第%d章「%s」— %s", ch.OrderIndex, ch.Title, ch.Summary))
		}
		existingChapters = fmt.Sprintf("卷内已有 %d 章：\n%s\n\n请从第 %d 章之后续写，不要重复已有章节。",
			len(target.Children), strings.Join(chInfos, "\n"), len(target.Children))
	} else {
		existingChapters = "该卷暂无章节，请从头规划章节。"
	}

	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"parent_node":       fmt.Sprintf("卷「%s」— %s", target.Title, target.Summary),
		"existing_chapters": existingChapters,
		"expand_count":      fmt.Sprintf("%d", subCount),
		"worldview":         a.loadWorldviewContext(),
		"story_thread":      of.StoryThread,
	})

	reply, err := a.chat(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	var expandResult struct {
		Children []types.OutlineNode `json:"children"`
	}
	if err := json.Unmarshal([]byte(util.ExtractJSON(reply)), &expandResult); err != nil {
		return nil, fmt.Errorf("解析展开结果 JSON 失败: %w", err)
	}

	target.Children = append(target.Children, expandResult.Children...)
	reindexOutlineNodes(of.Nodes, "")
	return of, nil
}

// ── CRUD ────────────────────────────────────────────────────
// ── CRUD ────────────────────────────────────────────────────

// Save 保存完整大纲文件
func (a *Agent) Save(of *types.OutlineFile) error {
	return a.pm.WriteOutlines(of)
}

// SaveNode 保存/更新单个大纲节点（顶级或嵌套在 Children 中）
func (a *Agent) SaveNode(node types.OutlineNode) error {
	if node.ID == "" {
		return fmt.Errorf("SaveNode: 节点 ID 为空")
	}
	if node.OrderIndex < 1 {
		node.OrderIndex = 1
	}
	of, err := a.pm.ReadOutlines()
	if err != nil {
		slog.Warn("大纲: 读取失败，新建空大纲", "error", err)
		of = &types.OutlineFile{Nodes: []types.OutlineNode{}}
	}
	if of == nil {
		of = &types.OutlineFile{Nodes: []types.OutlineNode{}}
	}
	// 递归查找并更新
	if updateNodeInTree(of.Nodes, node) {
		slog.Debug("SaveNode: 已更新节点", "id", node.ID, "title", node.Title)
		return a.pm.WriteOutlines(of)
	}
	// 未找到则追加到根节点
	slog.Debug("SaveNode: 未找到节点，追加到根", "id", node.ID)
	of.Nodes = append(of.Nodes, node)
	return a.pm.WriteOutlines(of)
}

// AddNode 添加一个子节点到指定父节点下
func (a *Agent) AddNode(node types.OutlineNode) error {
	if node.ID == "" {
		return fmt.Errorf("AddNode: 节点 ID 为空")
	}
	if node.OrderIndex < 1 {
		node.OrderIndex = 1
	}
	of, err := a.pm.ReadOutlines()
	if err != nil {
		slog.Warn("大纲: 读取失败，新建空大纲", "error", err)
		of = &types.OutlineFile{Nodes: []types.OutlineNode{node}}
		return a.pm.WriteOutlines(of)
	}
	if of == nil {
		of = &types.OutlineFile{Nodes: []types.OutlineNode{node}}
		return a.pm.WriteOutlines(of)
	}

	if node.ParentID == "" {
		of.Nodes = append(of.Nodes, node)
	} else {
		// 找到父节点并追加
		found := false
		for i := range of.Nodes {
			if addChildToNode(&of.Nodes[i], node.ParentID, node) {
				found = true
				break
			}
		}
		if !found {
			slog.Warn("AddNode: 未找到父节点，追加到根", "parentID", node.ParentID)
			of.Nodes = append(of.Nodes, node)
		}
	}
	return a.pm.WriteOutlines(of)
}

// DeleteNode 删除节点及其子节点
func (a *Agent) DeleteNode(nodeID string) error {
	if nodeID == "" {
		return fmt.Errorf("DeleteNode: 节点 ID 为空")
	}
	of, err := a.pm.ReadOutlines()
	if err != nil {
		slog.Warn("大纲: 读取失败", "error", err)
	}
	if of == nil {
		return nil // 无可删除
	}
	of.Nodes = removeNodeFromList(of.Nodes, nodeID)
	return a.pm.WriteOutlines(of)
}

// GetOutlines 获取当前大纲（已排序，永不返回 nil）
func (a *Agent) GetOutlines() *types.OutlineFile {
	of, err := a.pm.ReadOutlines()
	if err != nil {
		slog.Warn("大纲: 读取失败，返回空大纲", "error", err)
		return &types.OutlineFile{Nodes: []types.OutlineNode{}}
	}
	if of == nil {
		return &types.OutlineFile{Nodes: []types.OutlineNode{}}
	}
	sortOutlineNodes(of.Nodes)
	return of
}

// ── 辅助 ────────────────────────────────────────────────────

// sortOutlineNodes 递归按 order_index 排序，并对子节点排序
func sortOutlineNodes(nodes []types.OutlineNode) {
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].OrderIndex < nodes[j].OrderIndex })
	for i := range nodes {
		if len(nodes[i].Children) > 0 {
			sortOutlineNodes(nodes[i].Children)
		}
	}
}

// reindexOutlineNodes 递归重新分配 order_index（基于数组位置）
func reindexOutlineNodes(nodes []types.OutlineNode, parentID string) {
	for i := range nodes {
		nodes[i].OrderIndex = i + 1
		if parentID != "" {
			nodes[i].ParentID = parentID
		}
		if len(nodes[i].Children) > 0 {
			reindexOutlineNodes(nodes[i].Children, nodes[i].ID)
		}
	}
}

func (a *Agent) loadWorldviewContext() string {
	wf, err := a.pm.ReadWorldviewFile()
	if err != nil {
		slog.Warn("大纲: 读取世界观失败", "error", err)
	}
	if wf == nil {
		return "（暂无）"
	}
	return wf.ToMarkdown()
}

func (a *Agent) loadCharsContext() string {
	cf, err := a.pm.ReadCharacters()
	if err != nil {
		slog.Warn("大纲: 读取角色失败", "error", err)
	}
	if cf == nil || len(cf.Characters) == 0 {
		return "（暂无角色）"
	}
	b := util.MustMarshalCompact(cf.PromptView())
	return string(b)
}

func updateNodeInTree(nodes []types.OutlineNode, target types.OutlineNode) bool {
	for i := range nodes {
		if nodes[i].ID == target.ID {
			// 保留 children
			target.Children = nodes[i].Children
			nodes[i] = target
			return true
		}
		if updateNodeInTree(nodes[i].Children, target) {
			return true
		}
	}
	return false
}

func addChildToNode(node *types.OutlineNode, parentID string, child types.OutlineNode) bool {
	if node.ID == parentID {
		node.Children = append(node.Children, child)
		return true
	}
	for i := range node.Children {
		if addChildToNode(&node.Children[i], parentID, child) {
			return true
		}
	}
	return false
}

func removeNodeFromList(nodes []types.OutlineNode, targetID string) []types.OutlineNode {
	var result []types.OutlineNode
	for _, n := range nodes {
		if n.ID == targetID {
			continue
		}
		n.Children = removeNodeFromList(n.Children, targetID)
		result = append(result, n)
	}
	return result
}

func FindNodeByID(node *types.OutlineNode, targetID string) *types.OutlineNode {
	if node.ID == targetID {
		return node
	}
	for i := range node.Children {
		if found := FindNodeByID(&node.Children[i], targetID); found != nil {
			return found
		}
	}
	return nil
}

// ── QA Dialogue 大纲生成（蒸馏自 MM-StoryAgent 的学生-专家对话模式） ──

// OutlineDialogueResult 对话式大纲生成结果
type OutlineDialogueResult struct {
	StoryTitle string `json:"story_title"`
	Chapters   []struct {
		Title   string `json:"chapter_title"`
		Summary string `json:"chapter_summary"`
	} `json:"story_outline"`
	Dialogue []string `json:"dialogue"` // 对话历史（调试用）
}

// GenerateOutlineWithDialogue 使用学生-专家 LLM 对话模式生成大纲。
// storyPrompt: 故事设定描述（题材、主角、世界观提示等）
// numChapters: 期望的大纲章节数
// maxTurns: 最多对话轮次（默认 3）
func (a *Agent) GenerateOutlineWithDialogue(ctx context.Context, storyPrompt string, numChapters int, maxTurns int) (*OutlineDialogueResult, error) {
	if maxTurns <= 0 {
		maxTurns = 3
	}
	if numChapters <= 0 {
		numChapters = 4
	}

	// ── Phase 1: 学生提问 → 专家回答（多轮对话）──
	var dialogue []string

	studentSystem := "你是一位小说作者，正在构思一个故事。请向资深编辑提问，每次只问一个问题，帮助自己理清故事方向。问题应围绕角色动机、情节冲突、世界观细节、主题深度等。如果你觉得已经获得足够信息，说「谢谢，我已经有足够想法了。」"
	expertSystem := "你是资深小说编辑，擅长指导作者构思故事。请根据作者的故事设定和问题，给出具体、有建设性的回答。每次回答不超过 200 字，聚焦于激发作者的创作灵感。"

	for turn := 0; turn < maxTurns; turn++ {
		dialogueHistory := strings.Join(dialogue, "\n")

		// 学生提问
		studentPrompt := fmt.Sprintf("故事设定：%s\n\n对话历史：\n%s\n\n请提出你的下一个问题：", storyPrompt, dialogueHistory)
		question, err := a.client.ChatSimpleStreamWithOptions(ctx, a.novelModelName(), studentSystem, studentPrompt, ai.ChatSimpleOptions{EngineID: a.novelEngineName(),
			Temperature: 0.8,
			MaxTokens:   256,
		})
		if err != nil {
			return nil, fmt.Errorf("学生提问失败 (turn %d): %w", turn, err)
		}
		question = strings.TrimSpace(question)
		if strings.Contains(question, "谢谢") || strings.Contains(question, "足够想法") {
			break
		}
		dialogue = append(dialogue, fmt.Sprintf("作者: %s", question))

		// 专家回答
		expertPrompt := fmt.Sprintf("故事设定：%s\n\n作者提问：%s\n\n请回答：", storyPrompt, question)
		answer, err := a.client.ChatSimpleStreamWithOptions(ctx, a.novelModelName(), expertSystem, expertPrompt, ai.ChatSimpleOptions{EngineID: a.novelEngineName(),
			Temperature: 0.7,
			MaxTokens:   512,
		})
		if err != nil {
			return nil, fmt.Errorf("专家回答失败 (turn %d): %w", turn, err)
		}
		answer = strings.TrimSpace(answer)
		dialogue = append(dialogue, fmt.Sprintf("编辑: %s", answer))
	}

	// ── Phase 2: 基于对话生成大纲 ──
	slog.Info("对话完成，生成大纲", "turns", len(dialogue)/2)

	writerSystem := `你是一位专业的小说大纲规划师。基于作者与编辑的对话内容，生成一个结构清晰的故事大纲。

## 输出格式
输出 JSON：
{
  "story_title": "故事标题",
  "story_outline": [
    {"chapter_title": "第1章标题", "chapter_summary": "本章摘要"},
    ...
  ]
}

## 要求
1. 参考对话中的核心创意和编辑建议
2. 章节之间要有逻辑递进
3. 每章摘要简洁（50-100字）
4. 首章要有吸引人的开场\n5. 严格按照要求的章节数量生成，绝不多生成`

	writerPrompt := fmt.Sprintf("故事设定：%s\n\n作者与编辑的对话：\n%s\n\n请**严格只生成 %d 章**的大纲，不要多也不要少。", storyPrompt, strings.Join(dialogue, "\n"), numChapters)

	reply, err := a.client.ChatSimpleStreamWithOptions(ctx, a.novelModelName(), writerSystem, writerPrompt, ai.ChatSimpleOptions{EngineID: a.novelEngineName(),
		Temperature: 0.3,
		MaxTokens:   4096,
	})
	if err != nil {
		return nil, fmt.Errorf("大纲生成失败: %w", err)
	}

	var result OutlineDialogueResult
	if err := json.Unmarshal([]byte(util.ExtractJSON(reply)), &result); err != nil {
		return nil, fmt.Errorf("解析大纲 JSON 失败: %w\n原始回复: %s", err, util.Truncate(reply, 500))
	}
	result.Dialogue = dialogue

	return &result, nil
}
