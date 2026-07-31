package context

import (
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
	"github.com/gaea/gaea/internal/util"
)

// ── Lorebook 2.0: 触发式上下文注入 + Token 预算 ──────────────

// InjectionRule 一条注入规则
type InjectionRule struct {
	Entry    types.LorebookEntry `json:"entry"`
	Priority int                 `json:"priority"` // 1=最高, 5=最低
}

// Budget 上下文 Token 预算追踪
type Budget struct {
	Capacity int            `json:"capacity"` // 总容量（模型上下文窗口）
	Used     int            `json:"used"`     // 已使用
	Sections []BudgetSection `json:"sections"` // 各分区用量
}

// BudgetSection 预算的一个分区
type BudgetSection struct {
	Name  string `json:"name"`
	Used  int    `json:"used"`
	Limit int    `json:"limit"` // 0 表示无上限
	Color string `json:"color"` // 可视化颜色
}

// NewBudget 创建 Token 预算追踪器
func NewBudget(capacity int) *Budget {
	return &Budget{
		Capacity: capacity,
		Sections: []BudgetSection{
			{Name: "系统提示", Color: "#60a5fa", Limit: capacity / 10},
			{Name: "当前场景", Color: "#4ade80", Limit: 0},
			{Name: "上文场景", Color: "#34d399", Limit: capacity / 5},
			{Name: "角色卡片", Color: "#f59e0b", Limit: capacity / 4},
			{Name: "Lorebook", Color: "#c084fc", Limit: capacity / 6},
			{Name: "故事记忆", Color: "#f87171", Limit: capacity / 8},
			{Name: "语义检索结果", Color: "#f472b6", Limit: capacity / 6},
		},
	}
}

// Track 记录一个分区的 token 用量
func (b *Budget) Track(sectionName string, tokens int) {
	for i := range b.Sections {
		if b.Sections[i].Name == sectionName {
			b.Sections[i].Used += tokens
			b.Used += tokens
			return
		}
	}
}

// Remaining 计算剩余可用 token
func (b *Budget) Remaining() int {
	return b.Capacity - b.Used
}

// UsagePercent 使用百分比
func (b *Budget) UsagePercent() float64 {
	if b.Capacity == 0 {
		return 0
	}
	return float64(b.Used) / float64(b.Capacity) * 100
}

// ── 触发式注入引擎 ──────────────────────────────────────────

// Engine Lorebook 注入引擎
type Engine struct {
	pm    *project.Manager
	rules []InjectionRule
}

// NewEngine 创建注入引擎
func NewEngine(pm *project.Manager) *Engine {
	return &Engine{pm: pm}
}

// Load 从项目加载 Lorebook 并构建注入规则
func (e *Engine) Load() error {
	lf, err := e.pm.ReadLorebook()
	if err != nil {
		return err
	}
	if lf == nil {
		return nil
	}

	e.rules = nil
	for _, entry := range lf.Entries {
		priority := 3
		switch entry.Category {
		case "character":
			priority = 1 // 角色信息优先级最高
		case "location":
			priority = 2
		case "item":
			priority = 3
		case "concept":
			priority = 4
		}

		e.rules = append(e.rules, InjectionRule{
			Entry:    entry,
			Priority: priority,
		})
	}

	// 按优先级排序
	sort.Slice(e.rules, func(i, j int) bool {
		return e.rules[i].Priority < e.rules[j].Priority
	})

	return nil
}

// FindTriggers 在文本中查找所有被触发的 Lorebook 词条
// 返回应注入的词条列表（按优先级排序，受 token 预算限制）
func (e *Engine) FindTriggers(text string, maxTokens int) []InjectionRule {
	var triggered []InjectionRule
	tokensUsed := 0

	for _, rule := range e.rules {
		if !strings.Contains(text, rule.Entry.Key) {
			continue
		}

		entryTokens := util.EstimateTokens(rule.Entry.Content)
		if tokensUsed+entryTokens > maxTokens {
			break
		}

		triggered = append(triggered, rule)
		tokensUsed += entryTokens
	}

	return triggered
}

// Inject 将触发的词条注入到系统提示中
// 返回注入后的系统提示和注入的词条列表
func (e *Engine) Inject(systemPrompt string, userText string, maxTokens int, budget *Budget) (string, []InjectionRule) {
	triggered := e.FindTriggers(userText, maxTokens)
	if len(triggered) == 0 {
		return systemPrompt, nil
	}

	var parts []string
	parts = append(parts, systemPrompt)
	parts = append(parts, "\n\n## 相关世界观设定（自动注入）")

	tokensUsed := 0
	for _, rule := range triggered {
		entryTokens := util.EstimateTokens(rule.Entry.Content)
		parts = append(parts, formatLorebookEntry(rule.Entry))
		tokensUsed += entryTokens
	}

	if budget != nil {
		budget.Track("Lorebook", tokensUsed)
	}

	return strings.Join(parts, "\n"), triggered
}

// BuildFullContext 构建完整的 AI 上下文
// 返回: (systemPrompt, userPrompt, budget)
func (e *Engine) BuildFullContext(opts BuildOptions) (string, string, *Budget) {
	budget := NewBudget(opts.ModelCapacity)

	// 系统提示（全量）
	systemPrompt := opts.SystemPrompt
	systemTokens := util.EstimateTokens(systemPrompt)
	budget.Track("系统提示", systemTokens)

	// 当前场景/章节
	currentScene := opts.CurrentScene
	sceneTokens := util.EstimateTokens(currentScene)
	budget.Track("当前场景", sceneTokens)

	// 角色信息（优先注入）
	charInfo := opts.CharacterInfo
	charTokens := util.EstimateTokens(charInfo)
	budget.Track("角色卡片", charTokens)

	// Lorebook 触发注入
	lorebookMax := opts.ModelCapacity / 6
	if lorebookMax > budget.Remaining() {
		lorebookMax = budget.Remaining()
	}
	systemPrompt, _ = e.Inject(systemPrompt, currentScene+" "+opts.PreviousScene, lorebookMax, budget)

	// 故事记忆
	memoryInfo := opts.MemoryInfo
	memTokens := util.EstimateTokens(memoryInfo)
	budget.Track("故事记忆", memTokens)

	// 构建最终 prompt
	finalSystem := systemPrompt + "\n\n" + charInfo + "\n\n" + memoryInfo
	userPrompt := currentScene
	if opts.PreviousScene != "" {
		userPrompt = "上文：\n" + opts.PreviousScene + "\n\n当前：\n" + currentScene
	}

	return finalSystem, userPrompt, budget
}

// BuildOptions 上下文构建选项
type BuildOptions struct {
	SystemPrompt   string
	CurrentScene   string
	PreviousScene  string
	CharacterInfo  string
	MemoryInfo     string
	ModelCapacity  int // 模型上下文窗口大小（默认 128000）
}

func formatLorebookEntry(entry types.LorebookEntry) string {
	return "---\n" +
		"词条: " + entry.Key + " [" + entry.Category + "]\n" +
		entry.Content
}

