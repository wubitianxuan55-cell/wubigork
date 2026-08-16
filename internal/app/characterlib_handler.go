package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/assistant"
	"github.com/gaea/gaea/internal/characterlib"
	"github.com/gaea/gaea/internal/project"
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
				if err := a.assistantMgr.Update(ast.ID, *ast); err != nil {
					slog.Error("角色停用聊天通道更新失败", "assistantID", ast.ID, "error", err)
				}
			}
		}
	}

	if err := a.charLib.Upsert(&c); err != nil {
		return c, err
	}
	// 剧照 URL 可能在 Upsert 内被本地化（远程 xAI 临时图 → 本地路径），
	// 回写助手记录保持一致，避免聊天人格头像仍引用过期链接。
	if a.assistantMgr != nil && c.ChatEnabled && c.AssistantID != "" {
		if ast := a.assistantMgr.Get(c.AssistantID); ast != nil && ast.PortraitURL != c.PortraitURL {
			ast.PortraitURL = c.PortraitURL
			if err := a.assistantMgr.Update(ast.ID, *ast); err != nil {
				slog.Warn("保存角色后同步助手剧照失败", "assistantID", ast.ID, "error", err)
			}
		}
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
		if err := a.assistantMgr.Delete(c.AssistantID); err != nil {
			slog.Error("角色删除时清理聊天通道失败", "assistantID", c.AssistantID, "error", err)
		}
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

// CharacterAssociateTo 把库内角色加入指定小说项目（不改变当前打开的项目）。
// projectDir 可以是书架下子目录名（如 "星落之城"）或完整路径；必须位于书架目录内
// 且是有效项目（存在 project.json）。
func (a *App) CharacterAssociateTo(projectDir, charID, role string) error {
	if a.charLib == nil {
		return fmt.Errorf("角色库未初始化")
	}
	if strings.TrimSpace(projectDir) == "" {
		return fmt.Errorf("请选择要加入的小说项目")
	}
	abs := projectDir
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(a.cfg.NovelsDir, abs)
	}
	abs, err := filepath.Abs(abs)
	if err != nil {
		return fmt.Errorf("路径解析失败: %w", err)
	}
	novelsDir, err := filepath.Abs(a.cfg.NovelsDir)
	if err != nil {
		return fmt.Errorf("书架目录解析失败: %w", err)
	}
	rel, err := filepath.Rel(novelsDir, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("项目必须在书架目录内")
	}
	if _, err := os.Stat(filepath.Join(abs, "project.json")); err != nil {
		return fmt.Errorf("小说项目不存在或已损坏：%s", abs)
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
	if err := a.charLib.Associate(abs, charID, role, c.Arc, c.Status); err != nil {
		return err
	}
	// 物化到目标小说 characters.json：角色库关联表与小说面板读的
	// 工作副本是两个数据面，只有写回 characters.json 面板才可见。
	if _, err := a.mergeLibraryRefsIntoProject(abs); err != nil {
		return fmt.Errorf("角色已加入「%s」，但写入小说角色文件失败：%w", abs, err)
	}
	return nil
}

// mergeLibraryRefsIntoProject 把项目已关联的库角色按 ID 幂等合入 characters.json：
// 缺失则追加、已存在则跳过（保留项目内既有角色/组织/关系与本地编辑）。
// 返回是否发生了写入。
func (a *App) mergeLibraryRefsIntoProject(dir string) (bool, error) {
	if a.charLib == nil {
		return false, fmt.Errorf("角色库未初始化")
	}
	chars, err := a.charLib.ProjectCharactersForNovel(dir)
	if err != nil {
		return false, err
	}
	if len(chars) == 0 {
		return false, nil
	}
	meta, err := loadProjectMeta(filepath.Join(dir, "project.json"))
	if err != nil {
		return false, fmt.Errorf("小说项目不存在或已损坏：%s", dir)
	}
	pm := &project.Manager{Dir: dir, Meta: meta}
	cf, err := pm.ReadCharacters()
	if err != nil {
		return false, err
	}
	if cf == nil {
		cf = &types.CharacterFile{}
	}
	byID := make(map[string]int, len(cf.Characters))
	for i := range cf.Characters {
		byID[cf.Characters[i].ID] = i
	}
	changed := false
	for _, ch := range chars {
		if _, ok := byID[ch.ID]; ok {
			continue
		}
		cf.Characters = append(cf.Characters, ch)
		byID[ch.ID] = len(cf.Characters) - 1
		changed = true
	}
	if !changed {
		return false, nil
	}
	return true, pm.WriteCharacters(cf)
}

// CharacterDissociate 把角色从当前项目移除（角色保留在全局库）。
// 同时清理项目 characters.json 中的物化副本及相关组织成员/关系引用，
// 避免小说角色面板残留已移出的角色。
func (a *App) CharacterDissociate(charID string) error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开小说项目")
	}
	if a.charLib == nil {
		return fmt.Errorf("角色库未初始化")
	}
	// 1. 移除项目关联
	if err := a.charLib.Dissociate(pm.Dir, charID); err != nil {
		return err
	}
	// 2. 同步从 characters.json 移除物化副本（角色本体保留在全局库）
	cf, err := pm.ReadCharacters()
	if err != nil {
		return err
	}
	if cf == nil {
		return nil
	}
	filtered := cf.Characters[:0]
	for _, ch := range cf.Characters {
		if ch.ID != charID {
			filtered = append(filtered, ch)
		}
	}
	cf.Characters = filtered
	// 清理组织成员引用
	for i := range cf.Organizations {
		members := cf.Organizations[i].Members[:0]
		for _, m := range cf.Organizations[i].Members {
			if m != charID {
				members = append(members, m)
			}
		}
		cf.Organizations[i].Members = members
	}
	// 清理关系引用
	rels := cf.Relationships[:0]
	for _, r := range cf.Relationships {
		if r.FromID != charID && r.ToID != charID {
			rels = append(rels, r)
		}
	}
	cf.Relationships = rels
	return pm.WriteCharacters(cf)
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
	// 防误清保护：项目里还有未入库角色时拒绝覆盖，先走一次性导入
	refs, err := a.charLib.ListByProject(pm.Dir)
	if err != nil {
		return err
	}
	refSet := make(map[string]bool, len(refs))
	for _, r := range refs {
		refSet[r.CharacterID] = true
	}
	var legacy []string
	for _, ch := range cf.Characters {
		if !refSet[ch.ID] {
			legacy = append(legacy, ch.Name)
		}
	}
	if len(legacy) > 0 {
		return fmt.Errorf("项目还有 %d 个角色未入库（%s），请先在角色库「导入项目」完成一次性迁移", len(legacy), strings.Join(legacy[:min(len(legacy), 3)], "、"))
	}
	cf.Characters = chars
	return pm.WriteCharacters(cf)
}

// CharacterSetProjectState 更新角色在当前项目的状态（项目内覆盖，不影响全局角色）。
// 这是小说面板唯一允许的角色写入：只写关联表的 role/arc_state/status。
func (a *App) CharacterSetProjectState(charID, role, arcState, status string) error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开小说项目")
	}
	if a.charLib == nil {
		return fmt.Errorf("角色库未初始化")
	}
	return a.charLib.Associate(pm.Dir, charID, role, arcState, status)
}

// CharacterDrawRandom 从角色库随机抽卡（小说角色面板不再自行生成角色）。
func (a *App) CharacterDrawRandom(count int, gender, tags string, chatOnly bool) []characterlib.Character {
	if a.charLib == nil {
		return nil
	}
	items, err := a.charLib.DrawRandom(count, gender, tags, chatOnly)
	if err != nil {
		return nil
	}
	return items
}
