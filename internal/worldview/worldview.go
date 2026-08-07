package worldview

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/prompt"
	"github.com/gaea/gaea/internal/types"
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

// Chat 对话式编辑世界观（直接使用前端传来的当前设定文本）
func (a *Agent) Chat(ctx context.Context, userMsg string, currentContent string) (string, error) {
	// 尝试加载 prompt 模板
	tmpl := a.eng.Get("worldview-agent")
	if tmpl == nil {
		return "", fmt.Errorf("缺少 worldview-agent 模板文件")
	}

	systemPrompt := tmpl.BuildSystemPrompt("")
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"user_idea":         userMsg,
		"current_worldview": currentContent,
	})

	return a.chat(ctx, systemPrompt, userPrompt)
}

// ── 保存 ────────────────────────────────────────────────────
// ── 保存 ────────────────────────────────────────────────────
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

// ── 读取 ────────────────────────────────────────────────────
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

// ChatWithAutoSave 对话 + 从回复中提取更新并自动保存
func (a *Agent) ChatWithAutoSave(ctx context.Context, userMsg string, currentContent string) (string, error) {
	reply, err := a.Chat(ctx, userMsg, currentContent)
	if err != nil {
		return "", fmt.Errorf("AI 调用失败: %w", err)
	}

	saved := false

	// 旧格式兼容：维度更新标记
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
				} else {
					saved = true
				}
			}
		}
	}

	// 新格式：提取 ```markdown 代码块中的完整设定文本
	if !saved {
		if content := extractMarkdownBlock(reply); content != "" {
			if err := a.Save(content); err != nil {
				slog.Warn("世界观自动保存失败", "error", err)
			}
		}
	}

	return reply, nil
}

// ── 内部辅助 ─────────────────────────────────────────────────

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

// extractMarkdownBlock 从 AI 回复中提取 ```markdown 代码块内容
func extractMarkdownBlock(reply string) string {
	start := strings.Index(reply, "```markdown")
	if start == -1 {
		return ""
	}
	// 跳过 "```markdown" 和紧随的换行
	nl := strings.Index(reply[start:], "\n")
	if nl == -1 {
		return ""
	}
	bodyStart := start + nl + 1
	end := strings.Index(reply[bodyStart:], "```")
	if end == -1 {
		return ""
	}
	return strings.TrimSpace(reply[bodyStart : bodyStart+end])
}

// featureModel 小说功能级模型（持久化绑定 func_novel，运行中切换即时生效；空=全局）
func (a *Agent) featureModel() (engine, model string) {
	return a.cfg.GetFeatureModel("novel")
}

// chat 功能级对话：带 novel 引擎覆盖
func (a *Agent) chat(ctx context.Context, system, user string) (string, error) {
	eng, model := a.featureModel()
	// 未绑定（model 为空）时留空，由客户端按活跃引擎解析默认模型（等价 routeModel 全局路径），
	// 避免把全局 cfg.Model 发给非 xAI 引擎导致 404（E03）。
	return a.client.ChatSimpleStreamWithOptions(ctx, model, system, user, ai.ChatSimpleOptions{EngineID: eng})
}
