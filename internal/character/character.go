package character

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/wubigork/wubigork/internal/ai"
	"github.com/wubigork/wubigork/internal/config"
	"github.com/wubigork/wubigork/internal/project"
	"github.com/wubigork/wubigork/internal/prompt"
	"github.com/wubigork/wubigork/internal/types"
	"github.com/wubigork/wubigork/internal/util"
)

// Agent 角色子代理 — 对话式创建、修改角色及关系网
type Agent struct {
	client ai.LLMClient
	pm     *project.Manager
	cfg    *config.Config
	eng    *prompt.Engine
}

// New 创建角色 Agent
func New(client ai.LLMClient, pm *project.Manager, cfg *config.Config, eng *prompt.Engine) *Agent {
	return &Agent{client: client, pm: pm, cfg: cfg, eng: eng}
}

// ── 对话 ────────────────────────────────────────────────────

// Chat 对话式编辑角色（注入世界观+角色上下文）
func (a *Agent) Chat(ctx context.Context, userMsg string) (string, error) {
	cf, err := a.pm.ReadCharacters()
	if err != nil {
		slog.Warn("角色 Chat: 读取角色失败", "error", err)
	}
	charsJSON := string(util.MustMarshal(cf))

	tmpl := a.eng.Get("character-agent")
	if tmpl == nil {
		return "", fmt.Errorf("缺少 character-agent 模板文件")
	}

	systemPrompt := tmpl.BuildSystemPrompt("")
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"user_request":        userMsg,
		"existing_characters": string(charsJSON),
	})

	return a.client.ChatSimpleStream(ctx, a.cfg.Model, systemPrompt, userPrompt)
}

// ChatWithAutoSave 对话 + 自动解析并保存角色
func (a *Agent) ChatWithAutoSave(ctx context.Context, userMsg string) (string, error) {
	reply, err := a.Chat(ctx, userMsg)
	if err != nil {
		return "", fmt.Errorf("AI 调用失败: %w", err)
	}
	a.applyUpdates(reply)
	return reply, nil
}

// ChatCharacterDetail 针对特定角色对话（上下文聚焦该角色）
func (a *Agent) ChatCharacterDetail(ctx context.Context, charID, userMsg string) (string, error) {
	cf, err := a.pm.ReadCharacters()
	if err != nil {
		slog.Warn("角色 Chat: 读取角色失败", "error", err)
	}
	if cf == nil {
		return "", fmt.Errorf("无角色数据")
	}

	// 找到目标角色
	var target *types.Character
	for i := range cf.Characters {
		if cf.Characters[i].ID == charID {
			target = &cf.Characters[i]
			break
		}
	}
	if target == nil {
		return "", fmt.Errorf("未找到角色: %s", charID)
	}

	targetJSON := string(util.MustMarshal(target))
	wvCtx := a.loadWorldviewContext()

	tmpl := a.eng.Get("character-detail")
	if tmpl == nil {
		return "", fmt.Errorf("缺少 character-detail 模板文件")
	}

	systemPrompt := tmpl.BuildSystemPrompt("")
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"target_character": fmt.Sprintf("角色「%s」\n%s", target.Name, string(targetJSON)),
		"worldview":        wvCtx,
		"user_request":     userMsg,
	})

	reply, err := a.client.ChatSimpleStream(ctx, a.cfg.Model, systemPrompt, userPrompt)
	if err != nil {
		return "", err
	}
	a.applyUpdates(reply)
	return reply, nil
}

func (a *Agent) applyUpdates(reply string) {
	updates := extractCharacterUpdates(reply)
	if len(updates) > 0 {
		cf, err := a.pm.ReadCharacters()
		if err != nil {
			slog.Warn("character: 读取角色失败", "error", err)
		}
		if cf == nil {
			cf = &types.CharacterFile{}
		}
		a.mergeCharacters(cf, updates)
		a.pm.WriteCharacters(cf)
	}
}

// ── 角色剧照 ────────────────────────────────────────────────

var roleTypeCN = map[string]string{
	"protagonist": "主角", "antagonist": "反派",
	"supporting": "配角", "minor": "次要角色",
}

// GeneratePortrait 生成角色剧照（通过 xAI Image API）
func (a *Agent) GeneratePortrait(ctx context.Context, charID string) (string, error) {
	cf, err := a.pm.ReadCharacters()
	if err != nil {
		return "", fmt.Errorf("读取角色失败: %w", err)
	}
	var target *types.Character
	for i := range cf.Characters {
		if cf.Characters[i].ID == charID {
			target = &cf.Characters[i]
			break
		}
	}
	if target == nil {
		return "", fmt.Errorf("未找到角色: %s", charID)
	}

	roleCN := roleTypeCN[target.RoleType]
	if roleCN == "" {
		roleCN = target.RoleType
	}

	// 组装中文 prompt — 适配本地 Z-Image-Turbo / Flux
	prompt := fmt.Sprintf(
		"角色概念艺术肖像，%s，%s。"+
			"外貌: %s。身材: %s。性格: %s。背景: %s。"+
			"电影级光影，8K超高清，半身肖像，深色氛围背景。",
		target.Name, roleCN, target.Appearance, target.Figure, target.Personality, target.Background,
	)

	req := &ai.ImageGenerationRequest{
		Model:  a.cfg.ImageModel, // 使用配置的模型 (flux / z-image-turbo)
		Prompt: prompt,
		N:      1,
		Size:   "1024x1024",
	}

	resp, err := a.client.GenerateImage(ctx, req)
	if err != nil {
		return "", fmt.Errorf("图片生成失败: %w", err)
	}
	if len(resp.Data) == 0 {
		return "", fmt.Errorf("图片生成返回为空")
	}

	portraitURL := resp.Data[0].URL
	if portraitURL == "" {
		portraitURL = resp.Data[0].B64JSON
	}

	// 保存剧照到小说项目文件夹 portraits/ 子目录（文件备份）
	a.savePortraitToProject(portraitURL, charID)
	// 注意：不覆盖 portraitURL 为本地路径！
	// Wails WebView 无法通过 file:// 访问本地文件，前端需要 data URL 或远程 URL 才能显示 <img>

	// 保存到角色
	for i := range cf.Characters {
		if cf.Characters[i].ID == charID {
			cf.Characters[i].PortraitURL = portraitURL
			break
		}
	}
	if err := a.pm.WriteCharacters(cf); err != nil {
		return "", fmt.Errorf("保存角色剧照失败: %w", err)
	}

	return portraitURL, nil
}

// savePortraitToProject 将剧照 base64 数据保存到项目 portraits/ 子目录
func (a *Agent) savePortraitToProject(imageData string, charID string) string {
	if !strings.HasPrefix(imageData, "data:") {
		return "" // 远程 URL 不处理，保留原值
	}
	commaIdx := strings.Index(imageData, ",")
	if commaIdx < 0 {
		return ""
	}
	data, err := base64.StdEncoding.DecodeString(imageData[commaIdx+1:])
	if err != nil {
		return ""
	}

	portraitDir := filepath.Join(a.pm.Dir, "portraits")
	if err := os.MkdirAll(portraitDir, 0755); err != nil {
		return ""
	}

	filename := charID + ".png"
	fullPath := filepath.Join(portraitDir, filename)
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return ""
	}
	return fullPath
}

// SetPortrait 将外部图片数据设为角色剧照（来自 AI 绘梦）
func (a *Agent) SetPortrait(charID string, imageData string) error {
	cf, err := a.pm.ReadCharacters()
	if err != nil {
		return fmt.Errorf("读取角色失败: %w", err)
	}

	found := false
	for i := range cf.Characters {
		if cf.Characters[i].ID == charID {
			a.savePortraitToProject(imageData, charID)
			// 保持原始 URL（data URL 或远程 URL），前端 WebView 才能显示
			cf.Characters[i].PortraitURL = imageData
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("未找到角色: %s", charID)
	}

	return a.pm.WriteCharacters(cf)
}

// ── 批量生成 ───────────────────────────────────────────────

// GenerateSingleCharacter 随机生成单个角色的全部字段
func (a *Agent) GenerateSingleCharacter(ctx context.Context, ch types.Character, genre string) (*types.Character, error) {
	wvCtx := a.loadWorldviewContext()

	tmpl := a.eng.Get("character-generate-single")
	if tmpl == nil {
		return nil, fmt.Errorf("缺少 character-generate-single 模板文件")
	}

	systemPrompt := tmpl.BuildSystemPrompt("")
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"genre_and_character": fmt.Sprintf("题材: %s\n当前角色: %s (ID: %s, 类型: %s)",
			genre, ch.Name, ch.ID, ch.RoleType),
		"worldview": wvCtx,
	})

	reply, err := a.client.ChatSimpleStream(ctx, a.cfg.Model, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	var result types.Character
	if err := json.Unmarshal([]byte(util.ExtractJSON(reply)), &result); err != nil {
		return nil, fmt.Errorf("解析角色 JSON 失败: %w", err)
	}
	if result.Name == "" {
		result.Name = ch.Name
	}
	if result.ID == "" {
		result.ID = ch.ID
	}
	return &result, nil
}

// GenerateCharacters 根据世界观一键批量生成角色
// extraContext 可选：传入世界观摘要 + 参考素材等额外上下文
func (a *Agent) GenerateCharacters(ctx context.Context, count int, genre string, extraContext ...string) (*types.CharacterFile, error) {
	wvCtx := a.loadWorldviewContext()
	currentCF, err := a.pm.ReadCharacters()
	if err != nil {
		slog.Warn("character: 读取角色文件失败", "error", err)
	}

	tmpl := a.eng.Get("character-generate-batch")
	if tmpl == nil {
		return nil, fmt.Errorf("缺少 character-generate-batch 模板文件")
	}

	systemPrompt := tmpl.BuildSystemPrompt("")
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"genre_and_count": fmt.Sprintf("题材: %s\n生成 %d 个角色", genre, count),
		"worldview":       wvCtx,
	})
	extraUser := "请生成角色。"
	if currentCF != nil && len(currentCF.Characters) > 0 {
		b := util.MustMarshalCompact(currentCF)
		extraUser = fmt.Sprintf("现有角色:\n%s\n\n请补充生成 %d 个新角色（不重复）。", string(b), count)
	}
	if len(extraContext) > 0 && extraContext[0] != "" {
		extraUser = extraUser + "\n\n【参考上下文】\n" + extraContext[0]
	}
	userPrompt = userPrompt + "\n" + extraUser

	// ── 调用 LLM + JSON 解析重试（蒸馏自 MM-StoryAgent）──
	caller := func(ctx context.Context, sys, usr string) (string, error) {
		return a.client.ChatSimpleStreamWithOptions(ctx, a.cfg.Model, sys, usr, ai.ChatSimpleOptions{
			Temperature: 0.7,
			MaxTokens:   4096,
		})
	}
	jsonStr, err := util.RetryJSON(ctx, caller, systemPrompt, userPrompt, 2)
	if err != nil {
		return nil, err
	}

	var generated types.CharacterFile
	if err := json.Unmarshal([]byte(jsonStr), &generated); err != nil {
		return nil, fmt.Errorf("解析角色 JSON 失败: %w\nJSON: %s", err, util.Truncate(jsonStr, 300))
	}

	// 合并到现有角色
	if currentCF != nil {
		existingIDs := make(map[string]bool)
		for _, c := range currentCF.Characters {
			existingIDs[c.ID] = true
		}
		for _, c := range generated.Characters {
			if !existingIDs[c.ID] {
				currentCF.Characters = append(currentCF.Characters, c)
			}
		}
		currentCF.Organizations = append(currentCF.Organizations, generated.Organizations...)
		currentCF.Relationships = append(currentCF.Relationships, generated.Relationships...)
		a.pm.WriteCharacters(currentCF)
		return currentCF, nil
	}

	a.pm.WriteCharacters(&generated)
	return &generated, nil
}

// ── 保存 ────────────────────────────────────────────────────

// Save 保存角色到文件
func (a *Agent) Save(cf *types.CharacterFile) error {
	return a.pm.WriteCharacters(cf)
}

// SaveCharacter 保存/更新单个角色
func (a *Agent) SaveCharacter(ch types.Character) error {
	cf, err := a.pm.ReadCharacters()
	if err != nil {
		slog.Warn("角色 Chat: 读取角色失败", "error", err)
	}
	if cf == nil {
		cf = &types.CharacterFile{}
	}
	found := false
	for i := range cf.Characters {
		if cf.Characters[i].ID == ch.ID {
			cf.Characters[i] = ch
			found = true
			break
		}
	}
	if !found {
		cf.Characters = append(cf.Characters, ch)
	}
	return a.pm.WriteCharacters(cf)
}

// DeleteCharacter 删除角色
func (a *Agent) DeleteCharacter(id string) error {
	cf, err := a.pm.ReadCharacters()
	if err != nil {
		slog.Warn("角色 Chat: 读取角色失败", "error", err)
	}
	if cf == nil {
		return nil
	}
	var filtered []types.Character
	for _, c := range cf.Characters {
		if c.ID != id {
			filtered = append(filtered, c)
		}
	}
	cf.Characters = filtered
	return a.pm.WriteCharacters(cf)
}

// BatchGenerate 批量生成角色（Bootstrap 兼容旧接口）
// worldview 参数传入额外上下文（世界观摘要 + 参考素材），传递给 GenerateCharacters
func (a *Agent) BatchGenerate(ctx context.Context, count int, genre string, worldview string) (*types.CharacterFile, error) {
	return a.GenerateCharacters(ctx, count, genre, worldview)
}

// ── 组织与关系管理 ──────────────────────────────────────────

// SaveOrganization 保存/更新组织
func (a *Agent) SaveOrganization(org types.Organization) error {
	cf, err := a.pm.ReadCharacters()
	if err != nil {
		slog.Warn("角色 Chat: 读取角色失败", "error", err)
	}
	if cf == nil {
		cf = &types.CharacterFile{}
	}
	for i := range cf.Organizations {
		if cf.Organizations[i].ID == org.ID {
			cf.Organizations[i] = org
			return a.pm.WriteCharacters(cf)
		}
	}
	cf.Organizations = append(cf.Organizations, org)
	return a.pm.WriteCharacters(cf)
}

// DeleteOrganization 删除组织
func (a *Agent) DeleteOrganization(id string) error {
	cf, err := a.pm.ReadCharacters()
	if err != nil {
		slog.Warn("角色 Chat: 读取角色失败", "error", err)
	}
	if cf == nil {
		return nil
	}
	var filtered []types.Organization
	for _, o := range cf.Organizations {
		if o.ID != id {
			filtered = append(filtered, o)
		}
	}
	cf.Organizations = filtered
	return a.pm.WriteCharacters(cf)
}

// SaveRelationship 保存/更新关系
func (a *Agent) SaveRelationship(rel types.Relationship) error {
	cf, err := a.pm.ReadCharacters()
	if err != nil {
		slog.Warn("角色 Chat: 读取角色失败", "error", err)
	}
	if cf == nil {
		cf = &types.CharacterFile{}
	}
	for i := range cf.Relationships {
		if cf.Relationships[i].FromID == rel.FromID && cf.Relationships[i].ToID == rel.ToID {
			cf.Relationships[i] = rel
			return a.pm.WriteCharacters(cf)
		}
	}
	cf.Relationships = append(cf.Relationships, rel)
	return a.pm.WriteCharacters(cf)
}

// DeleteRelationship 删除关系
func (a *Agent) DeleteRelationship(fromID, toID string) error {
	cf, err := a.pm.ReadCharacters()
	if err != nil {
		slog.Warn("角色 Chat: 读取角色失败", "error", err)
	}
	if cf == nil {
		return nil
	}
	var filtered []types.Relationship
	for _, r := range cf.Relationships {
		if r.FromID != fromID || r.ToID != toID {
			filtered = append(filtered, r)
		}
	}
	cf.Relationships = filtered
	return a.pm.WriteCharacters(cf)
}

// ToggleOrgMember 切换角色在组织中的成员关系
func (a *Agent) ToggleOrgMember(charID, orgID string) error {
	cf, err := a.pm.ReadCharacters()
	if err != nil {
		slog.Warn("角色 Chat: 读取角色失败", "error", err)
	}
	if cf == nil {
		return nil
	}
	for i := range cf.Organizations {
		if cf.Organizations[i].ID == orgID {
			members := cf.Organizations[i].Members
			found := false
			var newMembers []string
			for _, m := range members {
				if m == charID {
					found = true
				} else {
					newMembers = append(newMembers, m)
				}
			}
			if found {
				cf.Organizations[i].Members = newMembers
			} else {
				cf.Organizations[i].Members = append(members, charID)
			}
			return a.pm.WriteCharacters(cf)
		}
	}
	return nil
}

// ── 读取 ────────────────────────────────────────────────────

// GetCharacters 获取当前角色
func (a *Agent) GetCharacters() *types.CharacterFile {
	cf, err := a.pm.ReadCharacters()
	if err != nil {
		slog.Warn("GetCharacters: 读取角色失败", "error", err)
	}
	return cf
}

// ── 内部辅助 ─────────────────────────────────────────────────

func (a *Agent) loadWorldviewContext() string {
	wf, err := a.pm.ReadWorldviewFile()
	if err != nil {
		slog.Warn("loadWorldviewContext: 读取失败", "error", err)
	}
	if wf == nil {
		return "（暂无世界观）"
	}
	return wf.ToMarkdown()
}

// extractCharacterUpdates 从 AI 回复中提取角色 JSON
func extractCharacterUpdates(reply string) []types.Character {
	marker := "---CHARACTER_UPDATE---"
	endMarker := "---END_UPDATE---"

	var chars []types.Character
	for {
		start := strings.Index(reply, marker)
		if start == -1 {
			break
		}
		end := strings.Index(reply[start:], endMarker)
		if end == -1 {
			break
		}
		jsonStr := reply[start+len(marker) : start+end]
		reply = reply[start+end+len(endMarker):]

		var update struct {
			Characters []types.Character `json:"characters"`
		}
		if err := json.Unmarshal([]byte(strings.TrimSpace(jsonStr)), &update); err == nil {
			chars = append(chars, update.Characters...)
		}
	}
	return chars
}

// mergeCharacters 合并角色更新（同 ID 覆盖，新 ID 追加）
func (a *Agent) mergeCharacters(cf *types.CharacterFile, updates []types.Character) {
	idIndex := make(map[string]int)
	for i, c := range cf.Characters {
		idIndex[c.ID] = i
	}
	for _, u := range updates {
		if idx, ok := idIndex[u.ID]; ok {
			cf.Characters[idx] = u
		} else {
			cf.Characters = append(cf.Characters, u)
			idIndex[u.ID] = len(cf.Characters) - 1
		}
	}
}
