package app

import (
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/context"
	"github.com/gaea/gaea/internal/memory"
)

// ── 上下文智能 API ──────────────────────────────────────────

// SearchMemories 语义检索相关记忆
func (a *App) SearchMemories(query string, maxResults int) ([]map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	idx, err := memory.BuildFromProject(pm)
	if err != nil {
		return nil, err
	}

	if maxResults <= 0 {
		maxResults = 5
	}

	results := idx.Search(query, maxResults)
	var output []map[string]interface{}
	for _, m := range results {
		output = append(output, map[string]interface{}{
			"id":           m.ID,
			"chapter_num":  m.ChapterNum,
			"text":         m.Text,
			"category":     m.Category,
			"tokens":       m.Tokens,
			"score":        m.Score,
		})
	}
	return output, nil
}

// InjectMemories 将相关记忆注入到当前上下文
func (a *App) InjectMemories(currentContext string, maxMemories int, maxTokens int) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	idx, err := memory.BuildFromProject(pm)
	if err != nil {
		return nil, err
	}

	if maxMemories <= 0 {
		maxMemories = 5
	}
	if maxTokens <= 0 {
		maxTokens = 2000
	}

	enrichedContext, injected := idx.InjectIntoContext(currentContext, maxMemories, maxTokens)

	var injectedList []map[string]interface{}
	for _, m := range injected {
		injectedList = append(injectedList, map[string]interface{}{
			"id":          m.ID,
			"chapter_num": m.ChapterNum,
			"text":        m.Text,
			"score":       m.Score,
		})
	}

	return map[string]interface{}{
		"enriched_context": enrichedContext,
		"injected":         injectedList,
		"injected_count":   len(injected),
	}, nil
}

// FindLorebookTriggers 查找文本中触发的 Lorebook 词条
func (a *App) FindLorebookTriggers(text string) ([]map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	engine := context.NewEngine(pm)
	if err := engine.Load(); err != nil {
		return nil, err
	}

	triggered := engine.FindTriggers(text, 5000)
	var result []map[string]interface{}
	for _, rule := range triggered {
		result = append(result, map[string]interface{}{
			"key":       rule.Entry.Key,
			"content":   rule.Entry.Content,
			"category":  rule.Entry.Category,
			"priority":  rule.Priority,
		})
	}
	return result, nil
}

// BuildContextBudget 构建上下文并返回 Token 预算
func (a *App) BuildContextBudget(systemPrompt string, currentScene string, previousScene string, characterInfo string, memoryInfo string, modelCapacity int) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	if modelCapacity <= 0 {
		modelCapacity = 128000 // 默认 Grok 上下文窗口
	}

	engine := context.NewEngine(pm)
	engine.Load()

	sys, usr, budget := engine.BuildFullContext(context.BuildOptions{
		SystemPrompt:   systemPrompt,
		CurrentScene:   currentScene,
		PreviousScene:  previousScene,
		CharacterInfo:  characterInfo,
		MemoryInfo:     memoryInfo,
		ModelCapacity:  modelCapacity,
	})

	// 构建分区的 JSON 表示
	var sections []map[string]interface{}
	for _, sec := range budget.Sections {
		sections = append(sections, map[string]interface{}{
			"name":  sec.Name,
			"used":  sec.Used,
			"limit": sec.Limit,
			"color": sec.Color,
		})
	}

	return map[string]interface{}{
		"system_prompt":  sys,
		"user_prompt":    usr,
		"capacity":       budget.Capacity,
		"used":           budget.Used,
		"remaining":      budget.Remaining(),
		"usage_percent":  budget.UsagePercent(),
		"sections":       sections,
	}, nil
}

// GetAllEntityNames 获取所有实体名（用于 @-mention）
func (a *App) GetAllEntityNames() ([]map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	// 从角色获取
	chars, err := pm.ReadCharacters()
	var result []map[string]interface{}

	if err == nil && chars != nil {
		for _, ch := range chars.Characters {
			result = append(result, map[string]interface{}{
				"name": ch.Name,
				"type": "character",
				"id":   ch.ID,
			})
		}
		for _, org := range chars.Organizations {
			result = append(result, map[string]interface{}{
				"name": org.Name,
				"type": "organization",
				"id":   org.ID,
			})
		}
	}

	// 从 Lorebook 获取
	lorebook, err := pm.ReadLorebook()
	if err == nil && lorebook != nil {
		for _, entry := range lorebook.Entries {
			result = append(result, map[string]interface{}{
				"name": entry.Key,
				"type": entry.Category,
				"id":   "lorebook:" + entry.Key,
			})
		}
	}

	return result, nil
}

// BuildRichContext 一键构建富上下文（语义记忆 + Lorebook 注入）
func (a *App) BuildRichContext(systemPrompt string, userText string) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	var enriched strings.Builder
	enriched.WriteString(systemPrompt)

	// 1. 语义记忆检索
	memIdx, err := memory.BuildFromProject(pm)
	if err == nil {
		enrichedCtx, injected := memIdx.InjectIntoContext(userText, 5, 3000)
		if len(injected) > 0 {
			enriched.WriteString("\n\n" + enrichedCtx[len(systemPrompt):]) // 只追加新增部分
		}
	}

	// 2. Lorebook 触发注入
	engine := context.NewEngine(pm)
	if err := engine.Load(); err == nil {
		injectedSys, _ := engine.Inject(enriched.String(), userText, 3000, nil)
		enriched.Reset()
		enriched.WriteString(injectedSys)
	}

	return map[string]interface{}{
		"enriched_system_prompt": enriched.String(),
	}, nil
}
