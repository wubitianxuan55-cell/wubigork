package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gaea/gaea/internal/outline"
	"github.com/gaea/gaea/internal/types"
	"github.com/gaea/gaea/internal/util"
)

// ── 剧情分支 ─────────────────────────────────────────────────

// PlotBranch 一个剧情分支方向
type PlotBranch struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Summary            string   `json:"summary"`
	CharactersInvolved []string `json:"characters_involved"`
	CoreConflict       string   `json:"core_conflict"`
	ForeshadowImpact   string   `json:"foreshadow_impact"`
	Tone               string   `json:"tone"`
}

// BrainstormBranches 为指定大纲节点生成 3-5 个剧情分支
func (a *writingState) BrainstormBranches(nodeID string) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	of, err := pm.ReadOutlines()
	if err != nil {
		return nil, fmt.Errorf("读取大纲失败: %w", err)
	}

	// 找到目标节点
	var target *types.OutlineNode
	for i := range of.Nodes {
		if n := outline.FindNodeByID(&of.Nodes[i], nodeID); n != nil {
			target = n
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("未找到大纲节点 %s", nodeID)
	}

	// 收集上下文
	// 1. 章节摘要
	summaries, err := pm.ReadAllChapterSummaries()
	if err != nil {
		slog.Warn("BrainstormBranches: 读取章节摘要失败", "error", err)
	}
	var recentSummaries []string
	for i := len(summaries) - 1; i >= 0 && len(recentSummaries) < 3; i-- {
		recentSummaries = append(recentSummaries, summaries[i].Summary)
	}
	// reverse
	for i, j := 0, len(recentSummaries)-1; i < j; i, j = i+1, j-1 {
		recentSummaries[i], recentSummaries[j] = recentSummaries[j], recentSummaries[i]
	}
	previousCtx := strings.Join(recentSummaries, "\n")
	if previousCtx == "" {
		previousCtx = "（暂无前章摘要）"
	}

	// 2. 世界观
	wv, err := pm.ReadWorldview()
	if err != nil {
		slog.Warn("BrainstormBranches: 读取世界观失败", "error", err)
	}
	if wv == "" {
		wv = "（暂无世界观）"
	}
	// 当前节点信息

	// 当前节点信息
	nodeJSON, err := json.Marshal(target)
	if err != nil {
		slog.Warn("BrainstormBranches: 序列化大纲节点失败", "error", err)
	}

	tmpl := a.eng.Get("plot-branch-browser")
	if tmpl == nil {
		return nil, fmt.Errorf("缺少 plot-branch-browser 模板文件")
	}

	systemPrompt := tmpl.BuildSystemPrompt("")
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"current_outline_node":      string(nodeJSON),
		"previous_chapters_summary": previousCtx,
		"worldview":                 wv,
		"characters":                a.buildCharacterSummary(pm),
	})

	reply, err := a.client.ChatSimpleStream(a.ctx, a.cfg.Model, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("剧情分支推理失败: %w", err)
	}

	jsonStr := util.ExtractJSON(reply)
	var result struct {
		Branches []PlotBranch `json:"branches"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("解析分支 JSON 失败: %w", err)
	}

	if len(result.Branches) == 0 {
		return nil, fmt.Errorf("AI 未生成任何分支")
	}

	return map[string]interface{}{
		"branches": result.Branches,
	}, nil
}

// ApplyBranch 将选中分支写入大纲节点 + 同步角色和世界观
// userInput 可选：用户在分支基础上的手工补充
func (a *writingState) ApplyBranch(nodeID string, branchIndex int, userInput string) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	of, err := pm.ReadOutlines()
	if err != nil {
		return nil, fmt.Errorf("读取大纲失败: %w", err)
	}

	var target *types.OutlineNode
	for i := range of.Nodes {
		if n := outline.FindNodeByID(&of.Nodes[i], nodeID); n != nil {
			target = n
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("未找到大纲节点 %s", nodeID)
	}

	// 再次生成分支以获取完整信息（简化版直接重新生成）
	if branchIndex < 0 {
		// 手工录入模式：userInput 直接作为 summary + key_points
		lines := strings.Split(strings.TrimSpace(userInput), "\n")
		if len(lines) > 0 {
			target.Summary = lines[0]
		}
		if len(lines) > 1 {
			target.KeyPoints = lines[1:]
		}
	} else {
		// 从 AI 获取第 branchIndex 个分支
		// 简化：重新调 BrainstormBranches 取结果
		branchesRes, err := a.BrainstormBranches(nodeID)
		if err != nil {
			return nil, fmt.Errorf("获取分支失败: %w", err)
		}
		branches, ok := branchesRes["branches"].([]PlotBranch)
		if !ok || branchIndex >= len(branches) {
			return nil, fmt.Errorf("分支索引 %d 超出范围", branchIndex)
		}
		branch := branches[branchIndex]

		// 写入节点
		target.Summary = branch.Summary
		target.KeyPoints = []string{branch.CoreConflict, branch.ForeshadowImpact}
		target.Characters = branch.CharactersInvolved
		if branch.Tone != "" {
			target.Emotion = branch.Tone
		}

		// 如果用户有额外输入，追加到 summary
		if userInput != "" {
			target.Summary = target.Summary + "\n（用户补充：" + userInput + "）"
		}
	}

	// 保存大纲
	if err := pm.WriteOutlines(of); err != nil {
		return nil, fmt.Errorf("保存大纲失败: %w", err)
	}

	// 更新角色卡：确保所有 characters_involved 在角色列表中
	if a.characterAgent != nil {
		syncCharactersFromOutline(target)
	}

	return map[string]interface{}{
		"node":     target,
		"outlines": of.Nodes,
	}, nil
}

// QuickBrainstormBranches 轻量分支构思——直接接收小说设定和前文摘要，不需要大纲节点
func (a *writingState) QuickBrainstormBranches(setting, prevSummary string) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	tmpl := a.eng.Get("plot-branch-browser")
	if tmpl == nil {
		return nil, fmt.Errorf("缺少 plot-branch-browser 模板文件")
	}

	systemPrompt := tmpl.BuildSystemPrompt("")
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"current_outline_node":      "（请根据小说设定和前文摘要构思下一章方向）",
		"previous_chapters_summary": prevSummary,
		"worldview":                 setting,
		"characters":                a.buildCharacterSummary(pm),
	})

	reply, err := a.client.ChatSimpleStream(a.ctx, a.cfg.Model, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("剧情分支推理失败: %w", err)
	}

	jsonStr := util.ExtractJSON(reply)
	var result struct {
		Branches []PlotBranch `json:"branches"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("解析分支 JSON 失败: %w", err)
	}

	if len(result.Branches) == 0 {
		return nil, fmt.Errorf("AI 未生成任何分支")
	}

	return map[string]interface{}{
		"branches": result.Branches,
	}, nil
}
func syncCharactersFromOutline(node *types.OutlineNode) {
	if node == nil || len(node.Characters) == 0 {
		return
	}
	slog.Info("大纲节点引用角色", "node", node.Title, "chars", strings.Join(node.Characters, ","))
}
