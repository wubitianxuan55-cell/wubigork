package app

import (
	"encoding/json"
	"fmt"
	"time"

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
		"characters":    cf.Characters,
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
		"characters":    cf.Characters,
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
	return a.characterAgent.SaveCharacter(ch)
}

// DeleteCharacter 删除角色
func (a *writingState) DeleteCharacter(id string) error {
	if a.characterAgent == nil {
		return fmt.Errorf("请先打开项目")
	}
	return a.characterAgent.DeleteCharacter(id)
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
		"characters":    cf.Characters,
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
	cf := a.characterAgent.GetCharacters()
	if cf == nil {
		return nil
	}
	return map[string]interface{}{
		"characters":    cf.Characters,
		"organizations": cf.Organizations,
		"relationships": cf.Relationships,
	}
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
	return a.characterAgent.SetPortrait(charID, imageData)
}

// SaveCharactersBatch 批量创建角色（仅名称，其余字段留空）
// 前端确认新角色后调用。namesJSON: JSON 字符串数组，如 `["林风","苏婉"]`
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
