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
// 记忆面板「建议」标签页的真实后端：
//   - memories：自动做梦（suggest 模式）的待确认建议——提炼结果先入队
//     （dream-pending.json），用户逐条接受才入库（v4.377 建议制口径）；
//   - skills：从已沉淀的 procedural 记忆（规则/方法论）聚类出「多次出现
//     同一主题词」的候选，供用户一键沉淀为可复用技能。

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
	Memories []MemorySuggestionView `json:"memories"`
	Skills   []SkillSuggestionView  `json:"skills"`
	// Merges 是蒸馏合并候选（做梦 2.0 第一刀）：确定性重复记忆，用户批准后
	// 归档较旧条。可选项——旧前端/旧 mock 不读此字段不受影响。
	Merges      []MergeSuggestionView `json:"merges,omitempty"`
	GeneratedAt string                `json:"generatedAt"`
	Available   bool                  `json:"available"`
	Source      string                `json:"source"`
}

// GaeaMemorySuggestions 返回记忆面板建议：memories 来自自动做梦待确认队列
//（suggest 模式，空间过滤）；skills 来自 procedural 记忆的主题聚类。
func (a *App) GaeaMemorySuggestions() MemorySuggestionsView {
	view := MemorySuggestionsView{
		Memories:    []MemorySuggestionView{},
		Skills:      []SkillSuggestionView{},
		GeneratedAt: time.Now().Format(time.RFC3339),
		Source:      "自动做梦待确认建议（接受才入库）",
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
	view.Memories = dreamPendingViews(set.UserDir, gaeaEffectiveSpace())
	ms := set.Store.List()
	view.Skills = suggestSkillsFromMemories(ms)
	view.Merges = distillMergeViews(ms)
	return view
}

// dreamPendingViews 把待确认队列转成面板建议视图（note 型 Type="note"，
// 接受时走 QuickAdd）。空队列返回空切片（不返回 nil——前端 JSON 序列化
// 契约，v4.354 同坑）。
func dreamPendingViews(userDir, space string) []MemorySuggestionView {
	items := dreamPendingList(userDir, space)
	out := make([]MemorySuggestionView, 0, len(items))
	for _, it := range items {
		if it.Kind == "note" {
			scope := strings.TrimSpace(it.NoteScope)
			if scope == "" {
				scope = "local"
			}
			out = append(out, MemorySuggestionView{
				ID:          it.ID,
				Name:        "note-" + it.ID,
				Description: "项目笔记建议（" + scope + " 作用域）",
				Type:        "note",
				Body:        it.Note,
				Reason:      "自动做梦提炼的零散经验，接受后追加到记忆文档",
			})
			continue
		}
		out = append(out, MemorySuggestionView{
			ID:          it.ID,
			Name:        it.Name,
			Title:       it.Title,
			Description: it.Description,
			Type:        it.Type,
			Body:        it.Body,
			Reason:      "自动做梦提炼的事实，接受后写入长期记忆（未接受不入库）",
		})
	}
	return out
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
// source=explicit 落 dream 审计日志）。待确认队列命中的建议按队列项内容
// 写入并出队（note 型走 QuickAdd 追加记忆文档）；非队列建议（旧路径）按
// 传入内容写入。
func (a *App) GaeaAcceptMemorySuggestion(candidate interface{}) (string, error) {
	raw, err := json.Marshal(candidate)
	if err != nil {
		return "", err
	}
	var c MemorySuggestionView
	if err := json.Unmarshal(raw, &c); err != nil {
		return "", err
	}
	if strings.TrimSpace(c.Name) == "" && strings.TrimSpace(c.ID) == "" {
		return "", fmt.Errorf("建议缺少 name")
	}
	ctrl := gaeaCtrl()
	if ctrl == nil {
		return "", fmt.Errorf("办公引擎未初始化")
	}
	set := ctrl.Memory()
	if set == nil {
		return "", fmt.Errorf("记忆未就绪")
	}
	// 待确认队列命中（v4.377 建议制主路径）：按队列项写入并出队。出队失败
	// 不得静默落入旧路径——那会把未经确认的内容写进记忆且队列残留。
	if it, ok, terr := dreamPendingTake(set.UserDir, c.ID); terr != nil {
		return "", fmt.Errorf("移除待确认建议失败: %w", terr)
	} else if ok {
		if it.Kind == "note" {
			if _, err := ctrl.QuickAdd(parseScope(it.NoteScope), it.Note); err != nil {
				return "", err
			}
			return "saved:note:" + it.ID, nil
		}
		n, err := ctrl.SaveDreamFacts(it.Space, "explicit", []memory.Memory{{
			Name:        it.Name,
			Title:       it.Title,
			Description: it.Description,
			Type:        memory.NormalizeType(it.Type),
			Kind:        memory.NormalizeKind(it.MemoryKind),
			Body:        it.Body,
		}})
		if err != nil {
			return "", err
		}
		if n == 0 {
			return "", fmt.Errorf("记忆建议内容为空，未写入")
		}
		return "saved:" + it.Name, nil
	}
	// 非队列建议（旧路径，行为保持）：type 缺省 reference、kind 语义记忆。
	// S1.2 A：显式接受路径与自动做梦同点盖章——按勘察锚点取 gaeaEffectiveSpace()
	//（配置生效空间；mode=off 得 ""，SaveDreamFacts 写侧 Normalize 兜底 work），
	// source=explicit 落 dream 审计日志（含 Space 列）。
	n, err := ctrl.SaveDreamFacts(gaeaEffectiveSpace(), "explicit", []memory.Memory{{
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

// GaeaDismissMemorySuggestion 忽略一条待确认建议：出队即丢弃，不写记忆。
func (a *App) GaeaDismissMemorySuggestion(id string) error {
	ctrl := gaeaCtrl()
	if ctrl == nil {
		return fmt.Errorf("办公引擎未初始化")
	}
	set := ctrl.Memory()
	if set == nil {
		return fmt.Errorf("记忆未就绪")
	}
	_, ok, err := dreamPendingTake(set.UserDir, id)
	if err != nil {
		return fmt.Errorf("移除待确认建议失败: %w", err)
	}
	if !ok {
		return fmt.Errorf("建议不存在或已处理")
	}
	return nil
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
