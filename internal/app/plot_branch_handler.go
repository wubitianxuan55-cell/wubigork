package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/gaea/fileutil"
	"github.com/gaea/gaea/internal/outline"
	"github.com/gaea/gaea/internal/project"
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

	branches, err := a.brainstormBranchesForNode(pm, target)
	if err != nil {
		return nil, err
	}

	// 持久化本次 AI 生成的分支（branches.json sidecar）：ApplyBranch 读存储结果
	// 按索引应用，不再二次重调 AI——AI 非确定性，重生成会导致「用户预览的第 N 条」
	// 与「实际应用的第 N 条」不一致。落盘失败只记日志，不阻断构思结果返回。
	persistPlotBranchStore(pm, nodeID, nodeID, branches)

	return map[string]interface{}{
		"branches": branches,
	}, nil
}

// brainstormBranchesForNode 针对指定大纲节点调用 AI 构思剧情分支
// （BrainstormBranches 与 ApplyBranch 的旧数据回退路径共用）。
func (a *writingState) brainstormBranchesForNode(pm *project.Manager, target *types.OutlineNode) ([]PlotBranch, error) {
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

	eng, model, _ := a.routeModel("novel")
	// S1.5-B play 内容护栏：max_output_tokens 钳制（temperature_max 不适用
	// ——该点未显式传温度，钳制不注入新参数）。未配置 = 零值 = 现状逐字节。
	opts := ai.ChatSimpleOptions{EngineID: eng}
	applyChatSimpleMaxTokens(&opts, playGuardrails().MaxOutputTokens)
	reply, err := a.client.ChatSimpleStreamWithOptions(a.ctx, model, systemPrompt, userPrompt, opts)
	if err != nil {
		return nil, fmt.Errorf("剧情分支推理失败: %w", err)
	}

	return parsePlotBranchReply(reply)
}

// parsePlotBranchReply 从 AI 回复中提取并校验分支 JSON。
func parsePlotBranchReply(reply string) ([]PlotBranch, error) {
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

	return result.Branches, nil
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

	res := map[string]interface{}{}
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
		// 应用 brainstorm 时持久化的分支（branches.json sidecar）；
		// 仅旧数据无持久化时回退现场重生成（返回中注明 applied_from/branch_note）。
		branch, appliedFrom, note, err := a.storedOrRegeneratedBranch(pm, target, branchIndex)
		if err != nil {
			return nil, err
		}

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

		res["branch"] = branch
		res["applied_from"] = appliedFrom
		if note != "" {
			res["branch_note"] = note
		}
	}

	// 保存大纲
	if err := pm.WriteOutlines(of); err != nil {
		return nil, fmt.Errorf("保存大纲失败: %w", err)
	}

	// 同步角色卡：把节点引用的新角色幂等物化进 characters.json
	// （失败只记日志，不阻断分支应用）
	a.syncCharactersFromOutline(target)

	res["node"] = target
	res["outlines"] = of.Nodes
	return res, nil
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

	eng, model, _ := a.routeModel("novel")
	// S1.5-B play 内容护栏：同 BrainstormBranches（max_output_tokens 钳制；
	// 未配置 = 零值 = 现状逐字节）。
	opts := ai.ChatSimpleOptions{EngineID: eng}
	applyChatSimpleMaxTokens(&opts, playGuardrails().MaxOutputTokens)
	reply, err := a.client.ChatSimpleStreamWithOptions(a.ctx, model, systemPrompt, userPrompt, opts)
	if err != nil {
		return nil, fmt.Errorf("剧情分支推理失败: %w", err)
	}

	branches, err := parsePlotBranchReply(reply)
	if err != nil {
		return nil, err
	}

	// 轻量构思无节点上下文，落固定键 quick（最新覆盖），供 ApplyBranch 兜底读取
	persistPlotBranchStore(pm, plotBranchQuickKey, "", branches)

	return map[string]interface{}{
		"branches": branches,
	}, nil
}
// ── 分支结果持久化（branches.json sidecar）───────────────────
//
// BrainstormBranches / QuickBrainstormBranches 成功后把本次 AI 生成的分支原子
// 落盘到项目根 branches.json；ApplyBranch 按 nodeID（节点无存储时兜底读轻量
// 构思 quick 会话）读持久化结果应用，完全去掉二次 AI 重调。旧项目无该文件 =
// 无持久化 → ApplyBranch 回退现场重生成（原行为），并在返回的 branch_note 里
// 注明。写盘走 fileutil.AtomicWrite（临时文件 + rename 原子替换，与 project
// 包 writeFileAtomic 同一约定，崩溃/并发写不会写坏用户数据）。

const (
	// plotBranchStoreFile 分支结果 sidecar 文件名（项目根目录）
	plotBranchStoreFile = "branches.json"
	// plotBranchQuickKey 轻量构思（无节点上下文）的固定存储键，最新覆盖
	plotBranchQuickKey = "quick"
)

// plotBranchStore branches.json 完整结构：按 nodeID（或 quick）存最近一次构思
type plotBranchStore struct {
	Version  int                          `json:"version"`
	Sessions map[string]plotBranchSession `json:"sessions"`
}

// plotBranchSession 一次构思产生的分支集合
type plotBranchSession struct {
	NodeID    string       `json:"node_id,omitempty"` // quick 会话为空
	CreatedAt time.Time    `json:"created_at"`
	Branches  []PlotBranch `json:"branches"`
}

// loadPlotBranchStore 读分支存储；文件缺失（旧项目）或损坏时返回空存储，不报错。
func loadPlotBranchStore(pm *project.Manager) *plotBranchStore {
	store := &plotBranchStore{Version: 1, Sessions: map[string]plotBranchSession{}}
	data, err := os.ReadFile(filepath.Join(pm.Dir, plotBranchStoreFile))
	if err != nil {
		return store
	}
	if err := json.Unmarshal(data, store); err != nil {
		slog.Warn("读取剧情分支存储失败，按空存储处理", "error", err)
		store.Sessions = map[string]plotBranchSession{}
	}
	if store.Sessions == nil {
		store.Sessions = map[string]plotBranchSession{}
	}
	return store
}

// persistPlotBranchStore 覆盖写入指定会话的分支结果（原子写；失败只记日志：
// 构思已成功，落盘失败不应吞掉本次结果）。
func persistPlotBranchStore(pm *project.Manager, key, nodeID string, branches []PlotBranch) {
	store := loadPlotBranchStore(pm)
	store.Sessions[key] = plotBranchSession{
		NodeID:    nodeID,
		CreatedAt: time.Now(),
		Branches:  branches,
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		slog.Warn("持久化剧情分支失败: 序列化失败", "error", err)
		return
	}
	if err := fileutil.AtomicWrite(filepath.Join(pm.Dir, plotBranchStoreFile), data, 0o644); err != nil {
		slog.Warn("持久化剧情分支失败", "error", err)
	}
}

// storedOrRegeneratedBranch 取第 branchIndex 个分支：优先读 brainstorm 持久化
// 结果（nodeID 精确命中 → quick 会话兜底）；均未命中（旧数据）时回退为现场
// 重新生成，from="regenerated" 且 note 非空，由调用方在返回里注明。
func (a *writingState) storedOrRegeneratedBranch(pm *project.Manager, target *types.OutlineNode, branchIndex int) (PlotBranch, string, string, error) {
	store := loadPlotBranchStore(pm)
	for _, key := range []string{target.ID, plotBranchQuickKey} {
		sess, ok := store.Sessions[key]
		if !ok {
			continue
		}
		if branchIndex >= len(sess.Branches) {
			return PlotBranch{}, "", "", fmt.Errorf("分支索引 %d 超出范围（存储共 %d 条）", branchIndex, len(sess.Branches))
		}
		from := "stored"
		if key == plotBranchQuickKey {
			from = "quick"
		}
		return sess.Branches[branchIndex], from, "", nil
	}
	// 旧数据无持久化：回退现场重生成（原行为）
	branches, err := a.brainstormBranchesForNode(pm, target)
	if err != nil {
		return PlotBranch{}, "", "", fmt.Errorf("获取分支失败: %w", err)
	}
	if branchIndex >= len(branches) {
		return PlotBranch{}, "", "", fmt.Errorf("分支索引 %d 超出范围", branchIndex)
	}
	return branches[branchIndex], "regenerated",
		"未找到本次构思持久化的分支结果（旧数据），已现场重新生成，实际应用内容可能与之前预览不一致", nil
}

// syncCharactersFromOutline 把大纲节点引用的角色幂等物化进 characters.json：
// 节点条目按角色 ID 或名字命中已有角色则跳过（不覆盖既有字段），全新名字则
// 创建最小角色卡；既有角色/组织/关系字段原样保留。失败只记日志，不阻断分支应用。
func (a *writingState) syncCharactersFromOutline(node *types.OutlineNode) {
	pm := a.getPM()
	if pm == nil || node == nil || len(node.Characters) == 0 {
		return
	}
	cf, err := pm.ReadCharacters()
	if err != nil {
		slog.Warn("syncCharactersFromOutline: 读取角色文件失败", "error", err)
		return
	}
	if cf == nil {
		cf = &types.CharacterFile{}
	}
	byID := make(map[string]bool, len(cf.Characters))
	byName := make(map[string]bool, len(cf.Characters))
	for _, ch := range cf.Characters {
		byID[ch.ID] = true
		byName[ch.Name] = true
	}
	added := 0
	for _, raw := range node.Characters {
		name := strings.TrimSpace(raw)
		if name == "" || byID[name] || byName[name] {
			continue
		}
		// ID 规则与 character_handler.go 一致（ch_<unixnano>）
		ch := types.Character{
			ID:       fmt.Sprintf("ch_%d", time.Now().UnixNano()+int64(added)),
			Name:     name,
			RoleType: "supporting",
			Status:   "Alive",
		}
		cf.Characters = append(cf.Characters, ch)
		byID[ch.ID] = true
		byName[name] = true
		added++
	}
	if added == 0 {
		return
	}
	if err := pm.WriteCharacters(cf); err != nil {
		slog.Warn("syncCharactersFromOutline: 写入角色文件失败", "error", err)
		return
	}
	slog.Info("大纲节点角色已同步到 characters.json", "node", node.Title, "added", added)
}
