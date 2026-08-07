package app

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gaea/gaea/internal/assistant"
	"github.com/gaea/gaea/internal/characterlib"
	"github.com/gaea/gaea/internal/types"
)

// ── 全局角色库：统一角色资产（小说 × 聊天）────────────────────

// CharacterList 分页查询全局角色库。
func (a *App) CharacterList(query, kind string, chatOnly bool, page, pageSize int) map[string]interface{} {
	empty := map[string]interface{}{"items": []characterlib.Character{}, "total": 0}
	if a.charLib == nil {
		return empty
	}
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}
	items, total, err := a.charLib.List(query, kind, chatOnly, pageSize, (page-1)*pageSize)
	if err != nil {
		empty["error"] = err.Error()
		return empty
	}
	return map[string]interface{}{"items": items, "total": total}
}

// CharacterGet 读取单个角色（含引用它的项目列表）。
func (a *App) CharacterGet(id string) (map[string]interface{}, error) {
	if a.charLib == nil {
		return nil, fmt.Errorf("角色库未初始化")
	}
	c, err := a.charLib.Get(id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("角色不存在")
	}
	projects, err := a.charLib.ProjectIDsForCharacter(id)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"character": c, "projects": projects}, nil
}

// CharacterSave 保存/新建统一角色。可聊天角色自动同步 assistant 记录（微信通道兼容）。
func (a *App) CharacterSave(cJSON string) (characterlib.Character, error) {
	var c characterlib.Character
	if err := json.Unmarshal([]byte(cJSON), &c); err != nil {
		return c, fmt.Errorf("解析角色数据失败: %w", err)
	}
	if a.charLib == nil {
		return c, fmt.Errorf("角色库未初始化")
	}
	if c.ID == "" {
		c.ID = fmt.Sprintf("lib_%d", time.Now().UnixMilli())
	}
	if c.Name == "" {
		return c, fmt.Errorf("角色名称不能为空")
	}
	if c.Kind == "" {
		c.Kind = characterlib.KindCustom
	}

	// 聊天能力 ↔ assistant 记录同步（assistant 是微信通道配置的事实源）
	if a.assistantMgr != nil {
		if c.ChatEnabled {
			if c.AssistantID == "" {
				ast := assistant.Assistant{
					ID:            "ast_" + c.ID,
					Name:          c.Name,
					PersonalityID: c.ID,
					Enabled:       true,
					VoiceGuide:    c.VoiceGuide,
					Gender:        c.Gender,
					Tags:          append([]string(nil), c.Tags...),
					Dims:          c.Dims,
					PortraitURL:   c.PortraitURL,
				}
				if err := a.assistantMgr.Add(ast); err != nil {
					return c, fmt.Errorf("创建聊天通道失败: %w", err)
				}
				c.AssistantID = ast.ID
			} else if ast := a.assistantMgr.Get(c.AssistantID); ast != nil {
				ast.Name = c.Name
				ast.PersonalityID = c.ID
				ast.Enabled = true
				ast.VoiceGuide = c.VoiceGuide
				ast.Gender = c.Gender
				ast.Tags = append([]string(nil), c.Tags...)
				ast.Dims = c.Dims
				ast.PortraitURL = c.PortraitURL
				if err := a.assistantMgr.Update(ast.ID, *ast); err != nil {
					return c, fmt.Errorf("更新聊天通道失败: %w", err)
				}
			}
		} else if c.AssistantID != "" {
			if ast := a.assistantMgr.Get(c.AssistantID); ast != nil {
				ast.Enabled = false
				_ = a.assistantMgr.Update(ast.ID, *ast)
			}
		}
	}

	if err := a.charLib.Upsert(&c); err != nil {
		return c, err
	}
	return c, nil
}

// CharacterDelete 删除角色：内置角色软隐藏，其余硬删（级联清理项目关联与助手通道）。
func (a *App) CharacterDelete(id string) error {
	if a.charLib == nil {
		return fmt.Errorf("角色库未初始化")
	}
	c, err := a.charLib.Get(id)
	if err != nil {
		return err
	}
	if c == nil {
		return nil
	}
	if c.AssistantID != "" && a.assistantMgr != nil {
		_ = a.assistantMgr.Delete(c.AssistantID)
	}
	return a.charLib.Delete(id)
}

// CharacterImportProject 把当前小说项目的 characters.json 导入全局库并建立引用（幂等）。
func (a *App) CharacterImportProject() (int, error) {
	pm := a.getPM()
	if pm == nil {
		return 0, fmt.Errorf("请先打开小说项目")
	}
	if a.charLib == nil {
		return 0, fmt.Errorf("角色库未初始化")
	}
	cf, err := pm.ReadCharacters()
	if err != nil {
		return 0, err
	}
	return a.charLib.ImportProjectCharacters(pm.Dir, cf.Characters)
}

// CharacterListByProject 当前项目已引用的角色。
func (a *App) CharacterListByProject() []characterlib.ProjectCharacter {
	pm := a.getPM()
	if pm == nil || a.charLib == nil {
		return nil
	}
	items, err := a.charLib.ListByProject(pm.Dir)
	if err != nil {
		return nil
	}
	return items
}

// CharacterAssociate 把库内角色加入当前项目（角色本身仍属于全局库）。
func (a *App) CharacterAssociate(charID, role string) error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开小说项目")
	}
	if a.charLib == nil {
		return fmt.Errorf("角色库未初始化")
	}
	c, err := a.charLib.Get(charID)
	if err != nil {
		return err
	}
	if c == nil {
		return fmt.Errorf("角色不存在")
	}
	if role == "" {
		role = c.RoleType
	}
	return a.charLib.Associate(pm.Dir, charID, role, c.Arc, c.Status)
}

// CharacterDissociate 把角色从当前项目移除（角色保留在全局库）。
func (a *App) CharacterDissociate(charID string) error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开小说项目")
	}
	if a.charLib == nil {
		return fmt.Errorf("角色库未初始化")
	}
	return a.charLib.Dissociate(pm.Dir, charID)
}

// CharacterSyncProject 把项目引用的库角色物化回 characters.json（小说 Agent 消费的工作副本）。
func (a *App) CharacterSyncProject() error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开小说项目")
	}
	if a.charLib == nil {
		return fmt.Errorf("角色库未初始化")
	}
	chars, err := a.charLib.ProjectCharactersForNovel(pm.Dir)
	if err != nil {
		return err
	}
	cf, err := pm.ReadCharacters()
	if err != nil {
		return err
	}
	if cf == nil {
		cf = &types.CharacterFile{}
	}
	cf.Characters = chars
	return pm.WriteCharacters(cf)
}

// syncCurrentProjectToLibrary 小说页写角色后回写全局库（幂等，保持单一事实源）。
func (a *App) syncCurrentProjectToLibrary() error {
	pm := a.getPM()
	if pm == nil || a.charLib == nil {
		return nil
	}
	cf, err := pm.ReadCharacters()
	if err != nil {
		return err
	}
	_, err = a.charLib.ImportProjectCharacters(pm.Dir, cf.Characters)
	return err
}
