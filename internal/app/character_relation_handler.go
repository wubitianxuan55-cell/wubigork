package app

// character_relation_handler.go — v4.454 小说项目角色「与主角的关系」AI 随机生成。
//
// 口径（用户拍板）：通用角色库不固化关系——项目级字段只落本书 characters.json
// （types.Character.ProtagonistRelation），不写全局角色库。范围三态：
//
//	all     全部：本书除主角本人外的全部角色，覆盖重写
//	missing 剩余全部：只生成该字段为空的角色
//	one     个人：name 指定的单个角色（覆盖重写）
//
// 锚定本书 RoleType=protagonist 的角色；无主角卡诚实报错指路。生成结果经
// characterSummaryLine 注入章节生成的角色名册（·与主角：X）。

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/types"
	"github.com/gaea/gaea/internal/util"
)

// relationPhraseMaxRunes 关系短语长度上限（rune）——模型超长时确定性截断兜底，
// 不让句子灌进角色档案。
const relationPhraseMaxRunes = 16

// GenerateProjectProtagonistRelations 批量 AI 随机生成本书角色「与主角的关系」。
func (a *writingState) GenerateProjectProtagonistRelations(mode, name string) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	if a.clientRef() == nil {
		return nil, fmt.Errorf("AI 客户端未初始化")
	}
	if a.eng == nil {
		return nil, fmt.Errorf("prompt 引擎未初始化")
	}
	tmpl := a.eng.Get("character-relation")
	if tmpl == nil {
		return nil, fmt.Errorf("缺少 character-relation 模板文件（prompts/character-relation.json）")
	}
	mode = strings.TrimSpace(mode)
	switch mode {
	case "all", "missing", "one":
	default:
		return nil, fmt.Errorf("未知范围（%s）：可选 all（全部）/ missing（剩余全部）/ one（个人）", mode)
	}

	cf, err := pm.ReadCharacters()
	if err != nil {
		return nil, fmt.Errorf("读取角色失败: %w", err)
	}
	if cf == nil || len(cf.Characters) == 0 {
		return nil, fmt.Errorf("本书还没有角色：先在角色面板添加或抽卡")
	}
	var proto *types.Character
	for i := range cf.Characters {
		if cf.Characters[i].RoleType == "protagonist" {
			proto = &cf.Characters[i]
			break
		}
	}
	if proto == nil {
		return nil, fmt.Errorf("未找到主角：先在本书角色里把某个角色的「定位」设为「主角」，再生成与主角的关系")
	}

	var targets []*types.Character
	for i := range cf.Characters {
		c := &cf.Characters[i]
		if mode == "one" {
			if c.Name == strings.TrimSpace(name) {
				if c.Name == proto.Name {
					return nil, fmt.Errorf("「%s」本人即主角：无需生成与主角的关系", c.Name)
				}
				targets = []*types.Character{c}
				break
			}
			continue
		}
		if c.Name == proto.Name {
			continue // 主角本人不参与
		}
		if mode == "missing" && strings.TrimSpace(c.ProtagonistRelation) != "" {
			continue
		}
		targets = append(targets, c)
	}
	if mode == "one" && len(targets) == 0 {
		return nil, fmt.Errorf("未找到角色「%s」", strings.TrimSpace(name))
	}

	// 已用关系避重清单（全部他人，生成过程中不追加——批量内部重复由避重段
	// 尽量约束，逐角色重算成本高于收益）
	used := make([]string, 0, 8)
	seen := make(map[string]bool, 8)
	for i := range cf.Characters {
		r := strings.TrimSpace(cf.Characters[i].ProtagonistRelation)
		if r != "" && !seen[r] {
			used = append(used, r)
			seen[r] = true
		}
	}

	worldview := "（暂无世界观）"
	if wf, err := pm.ReadWorldviewFile(); err == nil && wf != nil {
		if md := wf.ToMarkdown(); strings.TrimSpace(md) != "" {
			worldview = md
		}
	}

	eng, model, _ := a.routeModel("novel")
	// 关系短语是极短 JSON 输出；0.85 随机度与角色生成同基线，护栏只降不升。
	g := playGuardrails()
	systemPrompt := tmpl.BuildSystemPrompt("")
	updated, failed := 0, 0
	var failNames []string
	for i, c := range targets {
		a.emit("protagonist-relation-progress", map[string]interface{}{
			"current": i + 1,
			"total":   len(targets),
			"name":    c.Name,
		})
		userPrompt := tmpl.BuildUserPrompt(map[string]string{
			"character":      projectCharDigest(*c),
			"protagonist":    projectCharDigest(*proto),
			"used_relations": strings.Join(used, "、"),
			"worldview":      util.Truncate(worldview, 1200),
		})
		reply, err := a.clientRef().ChatSimpleStreamWithOptions(a.ctx, model, systemPrompt, userPrompt, ai.ChatSimpleOptions{
			EngineID:    eng,
			Feature:     "novel",
			Temperature: clampPlayTemperature(0.85, g.TemperatureMax),
			MaxTokens:   clampPlayMaxTokens(256, g.MaxOutputTokens),
		})
		if err != nil {
			failed++
			failNames = append(failNames, c.Name)
			continue
		}
		relation, err := parseRelationReply(reply)
		if err != nil {
			failed++
			failNames = append(failNames, c.Name)
			continue
		}
		c.ProtagonistRelation = relation
		if err := pm.WriteCharacters(cf); err != nil {
			failed++
			failNames = append(failNames, c.Name)
			continue
		}
		updated++
	}
	return map[string]interface{}{
		"mode":      mode,
		"total":     len(targets),
		"updated":   updated,
		"failed":    failed,
		"failNames": failNames,
	}, nil
}

// parseRelationReply 解析关系短语回复：{"relation":"…"}；空短语如实报错。
func parseRelationReply(reply string) (string, error) {
	raw := strings.TrimSpace(reply)
	if raw == "" {
		return "", fmt.Errorf("模型返回为空")
	}
	var out struct {
		Relation string `json:"relation"`
	}
	if err := json.Unmarshal([]byte(util.ExtractJSON(raw)), &out); err != nil {
		return "", fmt.Errorf("解析关系短语失败: %w", err)
	}
	relation := strings.TrimSpace(out.Relation)
	if relation == "" {
		return "", fmt.Errorf("模型未给出关系短语")
	}
	if r := []rune(relation); len(r) > relationPhraseMaxRunes {
		relation = string(r[:relationPhraseMaxRunes])
	}
	return relation, nil
}

// projectCharDigest 角色设定摘要（关系生成的锚点输入：姓名/定位/性格/背景/动机）。
func projectCharDigest(c types.Character) string {
	var b strings.Builder
	b.WriteString("「" + c.Name + "」·" + characterRoleLabel(c.RoleType))
	if p := strings.TrimSpace(c.Personality); p != "" {
		b.WriteString("·性格：" + util.Truncate(p, 60))
	}
	if bg := strings.TrimSpace(c.Background); bg != "" {
		b.WriteString("·身份：" + util.Truncate(bg, 60))
	}
	if m := strings.TrimSpace(c.Motivation); m != "" {
		b.WriteString("·目标：" + util.Truncate(m, 40))
	}
	return b.String()
}
