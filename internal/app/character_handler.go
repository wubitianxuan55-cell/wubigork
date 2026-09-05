package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/characterlib"
	"github.com/gaea/gaea/internal/types"
	"github.com/gaea/gaea/internal/util"
)

// ── 角色 ──────────────────────────────────────────────────────

// ChatCharacter 与角色 Agent 对话（自动解析并保存角色）
func (a *writingState) ChatCharacter(userMsg string) (map[string]interface{}, error) {
	if a.characterAgent == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	reply, err := a.characterAgent.ChatWithAutoSave(a.ctx, userMsg)
	if err != nil {
		return nil, err
	}
	cf := a.characterAgent.GetCharacters()
	if cf == nil {
		return map[string]interface{}{"reply": reply}, nil
	}
	return map[string]interface{}{
		"reply":         reply,
		"characters":    capCharacters(cf.Characters),
		"organizations": cf.Organizations,
		"relationships": cf.Relationships,
	}, nil
}

// GenerateCharacters AI 一键批量生成角色
func (a *writingState) GenerateCharacters(count int) (map[string]interface{}, error) {
	if a.characterAgent == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	cf, err := a.characterAgent.GenerateCharacters(a.ctx, count, pm.Meta.Genre)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"characters":    capCharacters(cf.Characters),
		"organizations": cf.Organizations,
		"relationships": cf.Relationships,
	}, nil
}

// SaveCharacter 保存/更新单个角色
func (a *writingState) SaveCharacter(chJSON string) error {
	if a.characterAgent == nil {
		return fmt.Errorf("请先打开项目")
	}
	var ch types.Character
	if err := json.Unmarshal([]byte(chJSON), &ch); err != nil {
		return fmt.Errorf("解析角色数据失败: %w", err)
	}
	if err := a.characterAgent.SaveCharacter(ch); err != nil {
		return err
	}
	// 单向约束：小说只写自己的 characters.json，绝不回写全局角色库
	return nil
}

// DeleteCharacter 删除角色
func (a *writingState) DeleteCharacter(id string) error {
	if a.characterAgent == nil {
		return fmt.Errorf("请先打开项目")
	}
	if err := a.characterAgent.DeleteCharacter(id); err != nil {
		return err
	}
	// 小说页删除 = 从项目移除引用；角色本身保留在全局库
	if err := a.app.CharacterDissociate(id); err != nil {
		slog.Warn("删除角色后同步角色库失败（角色保留在库中）", "error", err)
	}
	return nil
}

// GenerateSingleCharacter 随机生成单个角色详情
func (a *writingState) GenerateSingleCharacter(chJSON string) (map[string]interface{}, error) {
	if a.characterAgent == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	var ch types.Character
	if err := json.Unmarshal([]byte(chJSON), &ch); err != nil {
		return nil, fmt.Errorf("解析角色数据失败: %w", err)
	}
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	result, err := a.characterAgent.GenerateSingleCharacter(a.ctx, ch, pm.Meta.Genre)
	if err != nil {
		return nil, err
	}
	b := util.MustMarshalCompact(result)
	return map[string]interface{}{
		"character": string(b),
	}, nil
}

// ChatCharacterDetail 针对特定角色对话
func (a *writingState) ChatCharacterDetail(charID, userMsg string) (map[string]interface{}, error) {
	if a.characterAgent == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	reply, err := a.characterAgent.ChatCharacterDetail(a.ctx, charID, userMsg)
	if err != nil {
		return nil, err
	}
	cf := a.characterAgent.GetCharacters()
	if cf == nil {
		return map[string]interface{}{"reply": reply}, nil
	}
	return map[string]interface{}{
		"reply":         reply,
		"characters":    capCharacters(cf.Characters),
		"organizations": cf.Organizations,
		"relationships": cf.Relationships,
	}, nil
}

// SaveOrganization 保存/更新组织
func (a *writingState) SaveOrganization(orgJSON string) error {
	if a.characterAgent == nil {
		return fmt.Errorf("请先打开项目")
	}
	var org types.Organization
	if err := json.Unmarshal([]byte(orgJSON), &org); err != nil {
		return fmt.Errorf("解析组织数据失败: %w", err)
	}
	return a.characterAgent.SaveOrganization(org)
}

// DeleteOrganization 删除组织
func (a *writingState) DeleteOrganization(id string) error {
	if a.characterAgent == nil {
		return fmt.Errorf("请先打开项目")
	}
	return a.characterAgent.DeleteOrganization(id)
}

// ToggleOrgMember 切换角色-组织成员关系
func (a *writingState) ToggleOrgMember(charID, orgID string) error {
	if a.characterAgent == nil {
		return fmt.Errorf("请先打开项目")
	}
	return a.characterAgent.ToggleOrgMember(charID, orgID)
}

// SaveRelationship 保存/更新关系
func (a *writingState) SaveRelationship(relJSON string) error {
	if a.characterAgent == nil {
		return fmt.Errorf("请先打开项目")
	}
	var rel types.Relationship
	if err := json.Unmarshal([]byte(relJSON), &rel); err != nil {
		return fmt.Errorf("解析关系数据失败: %w", err)
	}
	return a.characterAgent.SaveRelationship(rel)
}

// DeleteRelationship 删除关系
func (a *writingState) DeleteRelationship(fromID, toID string) error {
	if a.characterAgent == nil {
		return fmt.Errorf("请先打开项目")
	}
	return a.characterAgent.DeleteRelationship(fromID, toID)
}

// SaveCharacters 保存角色文件
func (a *writingState) SaveCharacters(cfJSON string) error {
	if a.characterAgent == nil {
		return fmt.Errorf("请先打开项目")
	}
	var cf types.CharacterFile
	if err := json.Unmarshal([]byte(cfJSON), &cf); err != nil {
		return fmt.Errorf("解析角色数据失败: %w", err)
	}
	return a.characterAgent.Save(&cf)
}

// GetCharacters 获取角色文件
func (a *writingState) GetCharacters() map[string]interface{} {
	if a.characterAgent == nil {
		return nil
	}
	// 自愈：角色库已关联但尚未物化进 characters.json 的角色（旧版本从角色库
	// 「加入项目」只写关联表），读取时按 ID 幂等合入，保证小说角色面板立即可见。
	if a.app != nil {
		if pm := a.getPM(); pm != nil {
			if _, err := a.app.mergeLibraryRefsIntoProject(pm.Dir); err != nil {
				slog.Warn("GetCharacters: 合并角色库引用失败", "error", err)
			}
		}
	}
	cf := a.characterAgent.GetCharacters()
	if cf == nil {
		return nil
	}
	// 项目 characters.json 里可能存着 1MB+ 的 base64 剧照（抽卡/入库时带入）。
	// 全量返回会撑爆 Wails IPC 导致界面卡死：超大内联头像不随列表响应返回。
	return map[string]interface{}{
		"characters":    capCharacters(cf.Characters),
		"organizations": cf.Organizations,
		"relationships": cf.Relationships,
	}
}

// capCharacters 批量截断超大内联头像（复制一份，避免改动 agent 缓存）。
func capCharacters(chars []types.Character) []types.Character {
	out := make([]types.Character, len(chars))
	for i, c := range chars {
		c.PortraitURL = capPortraitURL(c.PortraitURL)
		out[i] = c
	}
	return out
}

// capPortraitURL 超大内联头像置空（远程 URL 不受影响），防止巨型 base64
// 载荷进入 Wails IPC / WebView2 渲染导致界面卡死。
func capPortraitURL(s string) string {
	if strings.HasPrefix(s, "data:") && len(s) > 300*1024 {
		return ""
	}
	return s
}

// GenerateCharacterPortrait 生成角色剧照
func (a *writingState) GenerateCharacterPortrait(charID string, model string) (string, error) {
	if a.characterAgent == nil {
		return "", fmt.Errorf("请先打开项目")
	}
	return a.characterAgent.GeneratePortrait(a.ctx, charID, model)
}

// SetCharacterPortrait 将外部图片数据设为角色剧照（来自 AI 绘梦）
func (a *writingState) SetCharacterPortrait(charID string, imageData string) error {
	if a.characterAgent == nil {
		return fmt.Errorf("请先打开项目")
	}
	if err := a.characterAgent.SetPortrait(charID, imageData); err != nil {
		return err
	}
	// T1：剧照保存成功后登记进图像域 ledger（失败只 warn，不影响主流程）。
	if cf := a.characterAgent.GetCharacters(); cf != nil {
		registerCharacterPortraitAsset(gaeaCwd(), cf.Characters, charID)
	}
	return nil
}

// SaveCharactersBatch 批量创建章节新发现角色。
// 先让 AI 生成完整档案（性格/背景/外貌等），再写入项目 characters.json；
// AI 生成失败时降级为仅名称（配角·存活），不阻塞添加。
// 只写项目文件：角色入库（全局角色库）由用户在「角色」面板手动「一次性迁移」，
// 捕获角色不自动进入全局库。
func (a *writingState) SaveCharactersBatch(namesJSON string) (map[string]interface{}, error) {
	if a.characterAgent == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	var names []string
	if err := json.Unmarshal([]byte(namesJSON), &names); err != nil {
		return nil, fmt.Errorf("解析角色名列表失败: %w", err)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("角色名列表为空")
	}

	// 读取已有角色做去重（按 Name）
	cf := a.characterAgent.GetCharacters()
	if cf == nil {
		cf = &types.CharacterFile{}
	}
	existingNames := make(map[string]bool)
	for _, ch := range cf.Characters {
		existingNames[ch.Name] = true
	}

	// 项目题材（用于角色档案生成，保持与世界观一致）
	genre := ""
	if a.pm != nil && a.pm.Meta != nil {
		genre = a.pm.Meta.Genre
	}

	var created []types.Character
	for _, name := range names {
		if name == "" || existingNames[name] {
			continue
		}
		ch := types.Character{
			ID:       fmt.Sprintf("ch_%d", time.Now().UnixNano()+int64(len(created))),
			Name:     name,
			RoleType: "supporting",
			Status:   "Alive",
		}
		// AI 生成完整档案；失败保留裸名字
		if a.ctx != nil {
			gen, err := a.characterAgent.GenerateSingleCharacter(a.ctx, ch, genre)
			if err == nil && gen != nil {
				mergeGeneratedProfile(&ch, gen)
			} else {
				slog.Warn("角色档案生成失败，保存基础信息", "name", name, "error", err)
			}
		}
		if err := a.characterAgent.SaveCharacter(ch); err != nil {
			return nil, fmt.Errorf("保存角色 %s 失败: %w", name, err)
		}
		created = append(created, ch)
		existingNames[name] = true
	}

	return map[string]interface{}{
		"created": created,
		"total":   len(cf.Characters) + len(created),
	}, nil
}

// projectCharToLib 项目角色 → 角色库字段（补全复用角色库 AI 逻辑）。
func projectCharToLib(c types.Character) characterlib.Character {
	return characterlib.Character{
		ID:          c.ID,
		Name:        c.Name,
		Gender:      c.Gender,
		Age:         c.Age,
		PortraitURL: c.PortraitURL,
		RoleType:    c.RoleType,
		Personality: c.Personality,
		Background:  c.Background,
		Appearance:  c.Appearance,
		Figure:      c.Figure,
		Motivation:  c.Motivation,
		Arc:         c.Arc,
		Status:      c.Status,
		Notes:       c.Notes,
	}
}

// libCharToProject 角色库字段 → 项目角色。
func libCharToProject(c characterlib.Character) types.Character {
	return types.Character{
		ID:          c.ID,
		Name:        c.Name,
		Gender:      c.Gender,
		Age:         c.Age,
		PortraitURL: c.PortraitURL,
		RoleType:    c.RoleType,
		Personality: c.Personality,
		Background:  c.Background,
		Appearance:  c.Appearance,
		Figure:      c.Figure,
		Motivation:  c.Motivation,
		Arc:         c.Arc,
		Status:      c.Status,
		Notes:       c.Notes,
	}
}

// GenerateProjectCharacterFill 为小说项目角色 AI 补齐空缺字段（只填空缺，保留已有）。
// 复用角色库的补全逻辑，但只写项目 characters.json，绝不回写全局角色库。
func (a *writingState) GenerateProjectCharacterFill(chJSON string) (string, error) {
	if a.characterAgent == nil {
		return "", fmt.Errorf("请先打开项目")
	}
	var ch types.Character
	if err := json.Unmarshal([]byte(chJSON), &ch); err != nil {
		return "", fmt.Errorf("解析角色数据失败: %w", err)
	}
	if strings.TrimSpace(ch.Name) == "" {
		return "", fmt.Errorf("角色名称不能为空")
	}
	if a.app == nil || a.app.client == nil || a.app.eng == nil {
		return "", fmt.Errorf("AI 客户端未初始化")
	}
	merged, err := a.app.characterGenerate(string(util.MustMarshal(projectCharToLib(ch))), "fill", nil)
	if err != nil {
		return "", err
	}
	var next characterlib.Character
	if err := json.Unmarshal([]byte(merged), &next); err != nil {
		return "", fmt.Errorf("解析补全结果失败: %w", err)
	}
	updated := libCharToProject(next)
	updated.ID = ch.ID // 保护身份，AI 不得改名/换 ID
	if err := a.characterAgent.SaveCharacter(updated); err != nil {
		return "", fmt.Errorf("保存角色失败: %w", err)
	}
	return string(util.MustMarshal(updated)), nil
}

// MergeCharacters 合并两个实为同一人的项目角色：mergeID 并入 keepID。
// 保留角色的空缺字段用被合并角色填充；关系与组织成员引用全部重定向；
// 被合并角色从 characters.json 移除。只改项目文件，不动全局角色库。
func (a *writingState) MergeCharacters(keepID, mergeID string) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	if keepID == "" || mergeID == "" {
		return nil, fmt.Errorf("角色 ID 不能为空")
	}
	if keepID == mergeID {
		return nil, fmt.Errorf("不能合并到自身")
	}
	cf, err := pm.ReadCharacters()
	if err != nil {
		return nil, fmt.Errorf("读取角色失败: %w", err)
	}
	if cf == nil {
		cf = &types.CharacterFile{}
	}
	var keep, merge *types.Character
	mergeIdx := -1
	for i := range cf.Characters {
		switch cf.Characters[i].ID {
		case keepID:
			keep = &cf.Characters[i]
		case mergeID:
			merge, mergeIdx = &cf.Characters[i], i
		}
	}
	if keep == nil {
		return nil, fmt.Errorf("保留角色不存在: %s", keepID)
	}
	if merge == nil {
		return nil, fmt.Errorf("被合并角色不存在: %s", mergeID)
	}

	// 1. 用被合并角色填充保留角色的空缺字段
	fillCharacterGaps(keep, merge)

	// 2. 关系重定向 + 去重 + 去自环
	rels := cf.Relationships[:0]
	seenRel := make(map[string]bool)
	for _, r := range cf.Relationships {
		if r.FromID == mergeID {
			r.FromID = keepID
		}
		if r.ToID == mergeID {
			r.ToID = keepID
		}
		if r.FromID == r.ToID {
			continue // 自身关系无意义
		}
		key := r.FromID + "|" + r.ToID + "|" + r.RelationType
		if seenRel[key] {
			continue
		}
		seenRel[key] = true
		rels = append(rels, r)
	}
	cf.Relationships = rels

	// 3. 组织成员重定向 + 去重
	for i := range cf.Organizations {
		var members []string
		seen := make(map[string]bool)
		for _, m := range cf.Organizations[i].Members {
			if m == mergeID {
				m = keepID
			}
			if seen[m] {
				continue
			}
			seen[m] = true
			members = append(members, m)
		}
		cf.Organizations[i].Members = members
	}

	// 4. 移除被合并角色
	cf.Characters = append(cf.Characters[:mergeIdx], cf.Characters[mergeIdx+1:]...)
	if err := pm.WriteCharacters(cf); err != nil {
		return nil, fmt.Errorf("保存角色失败: %w", err)
	}
	return map[string]interface{}{
		"characters":    capCharacters(cf.Characters),
		"organizations": cf.Organizations,
		"relationships": cf.Relationships,
	}, nil
}

// fillCharacterGaps 用 src 的非空字段填充 dst 的空字段。
func fillCharacterGaps(dst, src *types.Character) {
	if dst == nil || src == nil {
		return
	}
	if dst.Name == "" {
		dst.Name = src.Name
	}
	if dst.RoleType == "" {
		dst.RoleType = src.RoleType
	}
	if dst.Gender == "" {
		dst.Gender = src.Gender
	}
	if dst.Age == "" {
		dst.Age = src.Age
	}
	if dst.Personality == "" {
		dst.Personality = src.Personality
	}
	if dst.Background == "" {
		dst.Background = src.Background
	}
	if dst.Appearance == "" {
		dst.Appearance = src.Appearance
	}
	if dst.Figure == "" {
		dst.Figure = src.Figure
	}
	if dst.Motivation == "" {
		dst.Motivation = src.Motivation
	}
	if dst.Arc == "" {
		dst.Arc = src.Arc
	}
	if dst.Status == "" {
		dst.Status = src.Status
	}
	if dst.Notes == "" {
		dst.Notes = src.Notes
	}
	if dst.PortraitURL == "" {
		dst.PortraitURL = src.PortraitURL
	}
}

// mergeGeneratedProfile 把 AI 生成的档案字段合并进基础角色，
// 保留「配角·存活」默认值，只填充内容字段。
func mergeGeneratedProfile(dst, src *types.Character) {
	if dst == nil || src == nil {
		return
	}
	if src.Gender != "" {
		dst.Gender = src.Gender
	}
	if src.Age != "" {
		dst.Age = src.Age
	}
	if src.Personality != "" {
		dst.Personality = src.Personality
	}
	if src.Background != "" {
		dst.Background = src.Background
	}
	if src.Appearance != "" {
		dst.Appearance = src.Appearance
	}
	if src.Figure != "" {
		dst.Figure = src.Figure
	}
	if src.Motivation != "" {
		dst.Motivation = src.Motivation
	}
	if src.Arc != "" {
		dst.Arc = src.Arc
	}
}
