package app

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/memory"
)

// ── 记忆建议（方法论自动候选 P1-⑥）──────────────────────────────
// 记忆面板「建议」标签页的真实后端：从已沉淀的 procedural 记忆（规则/方法论）
// 聚类出「多次出现同一主题词」的候选，供用户一键沉淀为可复用技能
// （对标千问办公组织级 Skill 的个人版）。记忆候选由「自动做梦」直接入库，
// 这里不再重复提议，避免噪音。

// MemorySuggestionView 是记忆候选（面板「记忆」建议卡片）。
type MemorySuggestionView struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Body        string   `json:"body"`
	Reason      string   `json:"reason"`
	Evidence    []string `json:"evidence"`
}

// SkillSuggestionView 是技能候选（面板「技能」建议卡片）。
type SkillSuggestionView struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Scope       string   `json:"scope"`
	Body        string   `json:"body"`
	Reason      string   `json:"reason"`
	Evidence    []string `json:"evidence"`
}

// MemorySuggestionsView 是记忆建议完整负载（与前端契约一致）。
type MemorySuggestionsView struct {
	Memories    []MemorySuggestionView `json:"memories"`
	Skills      []SkillSuggestionView  `json:"skills"`
	GeneratedAt string                 `json:"generatedAt"`
	Available   bool                   `json:"available"`
	Source      string                 `json:"source"`
}

// GaeaMemorySuggestions 返回记忆面板建议：技能候选来自 procedural 记忆
// 的主题聚类；记忆候选为空（自动做梦已直接入库，宁缺毋滥）。
func (a *App) GaeaMemorySuggestions() MemorySuggestionsView {
	view := MemorySuggestionsView{
		Memories:    []MemorySuggestionView{},
		Skills:      []SkillSuggestionView{},
		GeneratedAt: time.Now().Format(time.RFC3339),
		Source:      "自动做梦沉淀的记忆",
	}
	c := gaeaCtrl()
	if c == nil {
		return view
	}
	set := c.Memory()
	if set == nil {
		return view
	}
	view.Available = true
	view.Skills = suggestSkillsFromMemories(set.Store.List())
	return view
}

// skillNameWords 是技能名提炼时的停用词（避免「步骤/使用」这类通用词当主题）。
var skillNameWords = map[string]bool{
	"with": true, "from": true, "your": true, "this": true, "that": true,
	"into": true, "work": true, "step": true, "steps": true, "guide": true,
	"using": true, "use": true, "how": true, "the": true, "and": true,
	"for": true, "best": true, "practice": true, "practices": true,
}

// suggestSkillsFromMemories 从 procedural 记忆聚类主题词，返回技能候选：
// 多个方法论记忆共用同一主题词 → 提议沉淀为可复用技能（≥2 条才提）。
// 纯函数，便于单测。
func suggestSkillsFromMemories(ms []memory.Memory) []SkillSuggestionView {
	// 只取 procedural（规则/方法论）
	var procedural []memory.Memory
	for _, m := range ms {
		if m.Kind == memory.KindProcedural {
			procedural = append(procedural, m)
		}
	}
	if len(procedural) < 2 {
		return nil
	}

	// 主题词 → 记忆列表（从 name/title 提炼 ASCII 词干，长度 ≥4、非停用词）
	topicMemories := map[string][]memory.Memory{}
	wordSeen := map[string]bool{}
	for _, m := range procedural {
		words := strings.FieldsFunc(strings.ToLower(m.Name+" "+m.Title), func(r rune) bool {
			return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
		})
		for _, w := range words {
			if len(w) < 4 || skillNameWords[w] {
				continue
			}
			if !wordSeen[w] {
				wordSeen[w] = true
			}
			// 同一记忆多个词都含主题词时只计一次
			dup := false
			for _, em := range topicMemories[w] {
				if em.Name == m.Name {
					dup = true
					break
				}
			}
			if !dup {
				topicMemories[w] = append(topicMemories[w], m)
			}
		}
	}

	type cluster struct {
		topic string
		mems  []memory.Memory
	}
	var clusters []cluster
	for topic, mems := range topicMemories {
		if len(mems) >= 2 {
			clusters = append(clusters, cluster{topic: topic, mems: mems})
		}
	}
	sort.Slice(clusters, func(i, j int) bool {
		if len(clusters[i].mems) != len(clusters[j].mems) {
			return len(clusters[i].mems) > len(clusters[j].mems)
		}
		return clusters[i].topic < clusters[j].topic
	})
	if len(clusters) > 5 {
		clusters = clusters[:5]
	}

	out := make([]SkillSuggestionView, 0, len(clusters))
	for i, cl := range clusters {
		evidence := make([]string, 0, len(cl.mems))
		var body strings.Builder
		body.WriteString(fmt.Sprintf("# %s 工作流（自动沉淀）\n\n适用主题：%s\n\n", cl.topic, cl.topic))
		for _, m := range cl.mems {
			evidence = append(evidence, m.Name)
			label := displayTitleLocal(m.Title, m.Name)
			body.WriteString("\n## " + label + "\n")
			if strings.TrimSpace(m.Description) != "" {
				body.WriteString(m.Description + "\n")
			}
			if strings.TrimSpace(m.Body) != "" {
				body.WriteString(strings.TrimSpace(m.Body) + "\n")
			}
		}
		out = append(out, SkillSuggestionView{
			ID:          fmt.Sprintf("skill-%s-%d", cl.topic, i),
			Name:        "workflow-" + cl.topic,
			Description: fmt.Sprintf("沉淀 %d 条「%s」相关方法论为可复用技能", len(cl.mems), cl.topic),
			Scope:       "project",
			Body:        strings.TrimSpace(body.String()),
			Reason:      fmt.Sprintf("检测到 %d 条 procedural 记忆共用主题词 %s（多次同类任务 → 方法论自动候选）", len(cl.mems), cl.topic),
			Evidence:    evidence,
		})
	}
	return out
}

// displayTitleLocal 是 app 层的标题回退（与 memory 包 displayTitle 一致）。
func displayTitleLocal(title, name string) string {
	if t := strings.TrimSpace(title); t != "" {
		return t
	}
	return strings.ReplaceAll(name, "-", " ")
}

// GaeaAcceptMemorySuggestion 接受一条记忆建议：写入长期记忆（按 name 去重，
// 与自动做梦同一写入路径，source=explicit 落 dream 审计日志）。
func (a *App) GaeaAcceptMemorySuggestion(candidate interface{}) (string, error) {
	raw, err := json.Marshal(candidate)
	if err != nil {
		return "", err
	}
	var c MemorySuggestionView
	if err := json.Unmarshal(raw, &c); err != nil {
		return "", err
	}
	if strings.TrimSpace(c.Name) == "" {
		return "", fmt.Errorf("建议缺少 name")
	}
	ctrl := gaeaCtrl()
	if ctrl == nil {
		return "", fmt.Errorf("办公引擎未初始化")
	}
	n, err := ctrl.SaveDreamFacts("explicit", []memory.Memory{{
		Name:        c.Name,
		Title:       c.Title,
		Description: c.Description,
		Type:        memory.NormalizeType(c.Type),
		Kind:        memory.KindSemantic,
		Body:        c.Body,
	}})
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", fmt.Errorf("记忆建议内容为空，未写入")
	}
	return "saved:" + c.Name, nil
}

// GaeaAcceptSkillSuggestion 接受技能候选：固化为工作区技能并热加载
// （复用 GaeaCaptureSkill 的落盘 + 热加载通道）。
func (a *App) GaeaAcceptSkillSuggestion(candidate interface{}) (string, error) {
	raw, err := json.Marshal(candidate)
	if err != nil {
		return "", err
	}
	var c SkillSuggestionView
	if err := json.Unmarshal(raw, &c); err != nil {
		return "", err
	}
	if strings.TrimSpace(c.Name) == "" {
		return "", fmt.Errorf("建议缺少 name")
	}
	res, err := a.GaeaCaptureSkill(SkillCaptureInput{
		Name:        c.Name,
		Description: c.Description,
		Task:        c.Reason,
		Solution:    c.Body,
	})
	if err != nil {
		return "", err
	}
	return res.Path, nil
}
