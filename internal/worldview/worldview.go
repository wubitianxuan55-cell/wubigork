package worldview

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/wubigork/wubigork/internal/ai"
	"github.com/wubigork/wubigork/internal/config"
	"github.com/wubigork/wubigork/internal/project"
	"github.com/wubigork/wubigork/internal/prompt"
	"github.com/wubigork/wubigork/internal/types"
	"github.com/wubigork/wubigork/internal/util"
)

// Agent 世界观子代理 — 结构化构建、对话迭代、一致性校验
type Agent struct {
	client ai.LLMClient
	pm     *project.Manager
	cfg    *config.Config
	eng    *prompt.Engine
}

// New 创建世界观 Agent
func New(client ai.LLMClient, pm *project.Manager, cfg *config.Config, eng *prompt.Engine) *Agent {
	return &Agent{client: client, pm: pm, cfg: cfg, eng: eng}
}

// ── 对话 ────────────────────────────────────────────────────

// Chat 对话式编辑世界观（注入角色+大纲上下文）
func (a *Agent) Chat(ctx context.Context, userMsg string) (string, error) {
	wf, err := a.pm.ReadWorldviewFile()
	if err != nil {
		slog.Warn("世界观: 读取失败", "error", err)
	}
	currentWV := ""
	if wf != nil {
		currentWV = wf.ToMarkdown()
	}

	charsCtx := a.loadCharsContext()
	outlineCtx := a.loadOutlineContext()

	// 尝试加载 prompt 模板
	tmpl := a.eng.Get("worldview-agent")
	if tmpl == nil {
		return "", fmt.Errorf("缺少 worldview-agent 模板文件")
	}

	systemPrompt := tmpl.BuildSystemPrompt("")
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"user_idea":         userMsg,
		"current_worldview": currentWV,
		"characters":        charsCtx,
		"outlines":          outlineCtx,
	})

	return a.client.ChatSimpleStream(ctx, a.cfg.Model, systemPrompt, userPrompt)
}

// ChatSection 针对特定维度对话
func (a *Agent) ChatSection(ctx context.Context, sectionID, userMsg string) (string, error) {
	wf, err := a.pm.ReadWorldviewFile()
	if err != nil {
		slog.Warn("世界观: 读取失败", "error", err)
	}
	if wf == nil {
		return "", fmt.Errorf("无世界观数据")
	}

	// 找到目标 section
	var target *types.WorldviewSection
	for i := range wf.Sections {
		if wf.Sections[i].ID == sectionID {
			target = &wf.Sections[i]
			break
		}
	}
	if target == nil {
		return "", fmt.Errorf("未找到维度: %s", sectionID)
	}

	currentWV := wf.ToMarkdown()
	charsCtx := a.loadCharsContext()

	tmpl := a.eng.Get("worldview-chat-section")
	if tmpl == nil {
		return "", fmt.Errorf("缺少 worldview-chat-section 模板文件")
	}

	systemPrompt := tmpl.BuildSystemPrompt("")
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"target_section": fmt.Sprintf("维度「%s」\n%s", target.Title, target.Content),
		"full_worldview": currentWV,
		"characters":     charsCtx,
		"user_request":   userMsg,
	})

	return a.client.ChatSimpleStream(ctx, a.cfg.Model, systemPrompt, userPrompt)
}

// ── 保存 ────────────────────────────────────────────────────

// Save 保存世界观（AI 生成时更新，写入第一个有效 section）
func (a *Agent) Save(content string) error {
	// 写 markdown（向后兼容）
	if err := a.pm.WriteWorldview(content); err != nil {
		return err
	}
	// 同时更新 worldview.json
	wf, err := a.pm.ReadWorldviewFile()
	if err != nil {
		slog.Warn("世界观: 读取失败", "error", err)
	}
	if wf == nil {
		wf = &types.WorldviewFile{Sections: project.DefaultSections()}
	} else if len(wf.Sections) > 0 {
		// 找到第一个非 legacy 的 section 或创建
		found := false
		for i := range wf.Sections {
			if wf.Sections[i].ID != "legacy" {
				wf.Sections[i].Content = content
				found = true
				break
			}
		}
		if !found {
			wf.Sections = append(wf.Sections, types.WorldviewSection{
				ID: "era", Title: "时代背景", Content: content, Order: 1,
			})
		}
	}
	return a.pm.WriteWorldviewFile(wf)
}

// SaveSection 保存单个维度
func (a *Agent) SaveSection(sectionID, content string) error {
	wf, err := a.pm.ReadWorldviewFile()
	if err != nil {
		slog.Warn("世界观: 读取失败", "error", err)
	}
	if wf == nil {
		wf = &types.WorldviewFile{Sections: []types.WorldviewSection{}}
	}
	for i := range wf.Sections {
		if wf.Sections[i].ID == sectionID {
			wf.Sections[i].Content = content
			return a.pm.WriteWorldviewFile(wf)
		}
	}
	return fmt.Errorf("未找到维度: %s", sectionID)
}

// SaveAllSections 保存全部维度（前端批量提交）
func (a *Agent) SaveAllSections(sections []types.WorldviewSection) error {
	wf := &types.WorldviewFile{Sections: sections}
	return a.pm.WriteWorldviewFile(wf)
}

// ── 一键生成 ────────────────────────────────────────────────

// GenerateAllSections 让 AI 一次性填充所有 6 个世界观维度
func (a *Agent) GenerateAllSections(ctx context.Context, genre, style, reference string) (*types.WorldviewFile, error) {
	charsCtx := a.loadCharsContext()
	currentWV := ""
	wf, err := a.pm.ReadWorldviewFile()
	if err != nil {
		slog.Warn("世界观: 读取失败", "error", err)
	}
	if wf != nil {
		currentWV = wf.ToMarkdown()
	}

	// 防御性截断：调用方可能传入未归纳的原始素材
	ref, truncated := util.TruncateRef(reference)
	if truncated {
		slog.Warn("世界观生成: 参考素材过长已截断", "runes", len([]rune(reference)), "max", util.RefLimit)
	}

	refHint := ""
	if ref != "" {
		refHint = fmt.Sprintf("\n【参考素材】\n%s\n", ref)
	}

	tmpl := a.eng.Get("worldview-generate-all")
	if tmpl == nil {
		return nil, fmt.Errorf("缺少 worldview-generate-all 模板文件")
	}

	systemPrompt := tmpl.BuildSystemPrompt("")
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"genre_style_reference": fmt.Sprintf("题材: %s\n风格: %s%s", genre, style, refHint),
		"existing_worldview":    currentWV,
		"existing_characters":   charsCtx,
	})

	// ── 调用 LLM + JSON 解析重试 ──
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

	var result struct {
		Sections []types.WorldviewSection `json:"sections"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("解析 AI 生成的世界观 JSON 失败: %w", err)
	}

	if len(result.Sections) == 0 {
		return nil, fmt.Errorf("AI 未生成任何维度内容")
	}

	// 合并：AI 返回的 section 覆盖默认空模板，缺失的保留空模板
	merged := project.DefaultSections()
	for _, aiSec := range result.Sections {
		for i := range merged {
			if merged[i].ID == aiSec.ID && aiSec.Content != "" {
				merged[i].Content = aiSec.Content
				if aiSec.Title != "" {
					merged[i].Title = aiSec.Title
				}
			}
		}
	}

	// 保存
	wf = &types.WorldviewFile{Sections: merged}
	if err := a.pm.WriteWorldviewFile(wf); err != nil {
		return nil, err
	}
	return wf, nil
}

// ── 读取 ────────────────────────────────────────────────────

// GetCurrent 获取当前世界观 markdown（向后兼容旧接口）
func (a *Agent) GetCurrent() string {
	wf, err := a.pm.ReadWorldviewFile()
	if err != nil {
		slog.Warn("世界观: 读取失败", "error", err)
	}
	if wf == nil {
		return ""
	}
	return wf.ToMarkdown()
}

// GetSections 获取结构化世界观
func (a *Agent) GetSections() *types.WorldviewFile {
	wf, err := a.pm.ReadWorldviewFile()
	if err != nil || wf == nil {
		return &types.WorldviewFile{
			Sections: []types.WorldviewSection{
				{ID: "era", Title: "时代背景", Content: "", Order: 1},
				{ID: "geography", Title: "地理风貌", Content: "", Order: 2},
				{ID: "factions", Title: "势力格局", Content: "", Order: 3},
				{ID: "rules", Title: "规则体系", Content: "", Order: 4},
				{ID: "culture", Title: "文化习俗", Content: "", Order: 5},
				{ID: "history", Title: "历史事件", Content: "", Order: 6},
			},
		}
	}
	return wf
}

// ── 一致性检查 ─────────────────────────────────────────────

// CheckConsistency 扫描世界观+角色+大纲，检测逻辑矛盾
func (a *Agent) CheckConsistency(ctx context.Context) (*types.ConsistencyReport, error) {
	wf, err := a.pm.ReadWorldviewFile()
	if err != nil {
		slog.Warn("世界观: 读取失败", "error", err)
	}
	currentWV := ""
	if wf != nil {
		currentWV = wf.ToMarkdown()
	}
	if strings.TrimSpace(currentWV) == "" {
		return &types.ConsistencyReport{
			Issues:      []types.ConsistencyIssue{},
			OverallNote: "世界观为空，无法检查一致性。",
		}, nil
	}

	charsCtx := a.loadCharsContext()
	outlineCtx := a.loadOutlineContext()
	ff, err := a.pm.ReadForeshadows()
	if err != nil {
		slog.Warn("worldview: 读取伏笔失败", "error", err)
	}
	foreshadowCtx := ""
	if ff != nil && len(ff.Items) > 0 {
		b := util.MustMarshalCompact(ff.Items)
		foreshadowCtx = string(b)
	}

	tmpl := a.eng.Get("worldview-check-consistency")
	if tmpl == nil {
		return nil, fmt.Errorf("缺少 worldview-check-consistency 模板文件")
	}

	systemPrompt := tmpl.BuildSystemPrompt("")
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"worldview":   currentWV,
		"characters":  charsCtx,
		"outlines":    outlineCtx,
		"foreshadows": foreshadowCtx,
	})

	reply, err := a.client.ChatSimpleStream(ctx, a.cfg.Model, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("一致性检查失败: %w", err)
	}

	var report types.ConsistencyReport
	if err := json.Unmarshal([]byte(util.ExtractJSON(reply)), &report); err != nil {
		return nil, fmt.Errorf("解析检查报告失败: %w", err)
	}

	return &report, nil
}

// ── 自动保存 ───────────────────────────────────────────────

// ChatWithAutoSave 对话 + 从回复中提取更新并自动保存
func (a *Agent) ChatWithAutoSave(ctx context.Context, userMsg string) (string, error) {
	reply, err := a.Chat(ctx, userMsg)
	if err != nil {
		return "", fmt.Errorf("AI 调用失败: %w", err)
	}

	// 检查回复中是否包含维度更新标记
	if strings.Contains(reply, "---WORLDVIEW_SECTION_UPDATE---") {
		updates := extractSectionUpdates(reply)
		if len(updates) > 0 {
			wf, err := a.pm.ReadWorldviewFile()
			if err != nil {
				slog.Warn("世界观: 读取失败", "error", err)
			}
			if wf != nil {
				for id, content := range updates {
					for i := range wf.Sections {
						if wf.Sections[i].ID == id {
							wf.Sections[i].Content = content
						}
					}
				}
				if err := a.pm.WriteWorldviewFile(wf); err != nil {
					slog.Warn("世界观自动保存失败", "error", err)
				}
			}
		}
	}

	return reply, nil
}

// ── 内部辅助 ─────────────────────────────────────────────────

func (a *Agent) loadCharsContext() string {
	cf, err := a.pm.ReadCharacters()
	if err != nil {
		slog.Warn("世界观: 读取角色失败", "error", err)
	}
	if cf == nil || len(cf.Characters) == 0 {
		return "（暂无角色）"
	}
	b := util.MustMarshalCompact(cf)
	return string(b)
}

func (a *Agent) loadOutlineContext() string {
	of, err := a.pm.ReadOutlines()
	if err != nil {
		slog.Warn("世界观: 读取大纲失败", "error", err)
	}
	if of == nil || len(of.Nodes) == 0 {
		return "（暂无大纲）"
	}
	b := util.MustMarshalCompact(of)
	return string(b)
}

// extractSectionUpdates 从 AI 回复中提取维度更新
func extractSectionUpdates(reply string) map[string]string {
	updates := make(map[string]string)
	marker := "---WORLDVIEW_SECTION_UPDATE---"
	endMarker := "---END_UPDATE---"

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
			ID      string `json:"id"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal([]byte(strings.TrimSpace(jsonStr)), &update); err == nil {
			updates[update.ID] = update.Content
		}
	}
	return updates
}
