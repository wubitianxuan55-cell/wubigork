package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/memory"
)

// ── 记忆建议（方法论自动候选 P1-⑥）──────────────────────────────
// 记忆面板「建议」标签页的真实后端（auto-dream 已删 2026-11-20，用户拍板
// ——自动提炼建议质量差且干扰正常记忆运作，remember 工具+面板手写为正口）：
//   - skills：从已沉淀的 procedural 记忆（规则/方法论）聚类出「多次出现
//     同一主题词」的候选，供用户一键沉淀为可复用技能；
//   - merges：确定性重复检测（蒸馏合并，独立机制）。

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

// MemorySuggestionsView 是记忆建议完整负载（与前端契约一致；memories/
// overflow 随 auto-dream 删除而消亡）。
type MemorySuggestionsView struct {
	Skills []SkillSuggestionView `json:"skills"`
	// Merges 是蒸馏合并候选：确定性重复记忆，用户批准后归档较旧条。
	Merges      []MergeSuggestionView `json:"merges,omitempty"`
	GeneratedAt string                `json:"generatedAt"`
	Available   bool                  `json:"available"`
	Source      string                `json:"source"`
}

// GaeaMemorySuggestions 返回记忆面板建议：skills 来自 procedural 记忆的
// 主题聚类；merges 来自确定性重复检测。
func (a *App) GaeaMemorySuggestions() MemorySuggestionsView {
	view := MemorySuggestionsView{
		Skills:      []SkillSuggestionView{},
		GeneratedAt: time.Now().Format(time.RFC3339),
		Source:      "技能候选+重复合并候选",
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
	ms := set.Store.List()
	view.Skills = suggestSkillsFromMemories(ms)
	view.Merges = distillMergeViews(ms)
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


// acceptDetectConflicts 冲突比对（阶段九 9.3）：新记忆入库后与既有记忆比对——
// 同题旧记忆自动置失效（原文保留可恢复，绝不删除），BM25 疑似相似仅提示不改
// 状态（宁漏勿误）。返回人可读段；空串=无冲突。
func acceptDetectConflicts(set *memory.Set, m memory.Memory) string {
	rep := set.Store.DetectConflicts(m)
	for _, name := range rep.Confirmed {
		if err := set.Store.Invalidate(name, fmt.Sprintf("被新记忆 %q 推翻（同题）", m.Name)); err != nil {
			slog.Warn("记忆失效标记失败（保留原状）", "name", name, "error", err)
		}
	}
	return rep.Report()
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
