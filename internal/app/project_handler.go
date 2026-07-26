package app

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/wubigork/wubigork/internal/project"
	"github.com/wubigork/wubigork/internal/util"
)

// ── 项目管理 ─────────────────────────────────────────────────

// CreateProject 新建小说项目
func (a *App) CreateProject(dir, title, genre, style string) (map[string]interface{}, error) {
	pm, err := project.Create(dir, title, genre, style, "")
	if err != nil {
		return nil, err
	}
	a.setPM(pm)
	a.initAgents()
	return map[string]interface{}{
		"title": pm.Meta.Title,
		"genre": pm.Meta.Genre,
		"style": pm.Meta.Style,
	}, nil
}

// BootstrapProject 一键引导：导入参考→AI生成世界观+角色+大纲
func (a *App) BootstrapProject(dir, title, genre, style, reference string) (map[string]interface{}, error) {
	// 1. 创建项目
	pm, err := project.Create(dir, title, genre, style, "")
	if err != nil {
		return nil, fmt.Errorf("创建项目失败: %w", err)
	}
	a.setPM(pm)
	a.initAgents()

	result := map[string]interface{}{
		"title": title,
	}

	// 2. 长素材先归纳压缩（避免超过 AI 上下文窗口）
	ref := reference
	if reference != "" && len([]rune(reference)) > util.RefLimit {
		summarized, err := a.summarizeReference(reference)
		if err != nil {
			slog.Warn("参考素材归纳失败，回退截断", "error", err)
			ref, _ = util.TruncateRef(reference)
		} else {
			ref = summarized
			result["referenceSummarized"] = true
		}
	}

	refHint := ""
	if ref != "" {
		refHint = fmt.Sprintf("\n\n【参考素材】\n%s\n\n请充分吸收以上参考素材。", ref)
	}

	// 3. 生成世界观（结构化 6 维度）
	wf, err := a.worldviewAgent.GenerateAllSections(a.ctx, genre, style, ref)
	if err != nil {
		return result, fmt.Errorf("世界观生成失败: %w", err)
	}
	result["worldview"] = wf.ToMarkdown()
	result["sectionCount"] = len(wf.Sections)

	// 4. 批量生成角色（传入世界观 + 参考素材上下文）
	wv, err := pm.ReadWorldview()
	if err != nil {
		slog.Warn("读取世界观失败", "error", err)
	}
	cf, err := a.characterAgent.BatchGenerate(a.ctx, 5, genre, wv+refHint)
	if err != nil {
		return result, fmt.Errorf("角色生成失败: %w", err)
	}
	a.characterAgent.Save(cf)
	result["charCount"] = len(cf.Characters)

	// 5. 生成初始大纲
	of, err := a.outlineAgent.Continue(a.ctx, 5)
	if err != nil {
		return result, fmt.Errorf("大纲生成失败: %w", err)
	}
	a.outlineAgent.Save(of)
	result["outlineCount"] = len(of.Nodes)

	return result, nil
}

// summarizeReference 用 AI 归纳压缩过长的参考素材
func (a *App) summarizeReference(reference string) (string, error) {
	tmpl := a.eng.Get("bootstrap-reference-summarize")
	if tmpl == nil {
		return "", fmt.Errorf("缺少 bootstrap-reference-summarize 模板文件")
	}

	systemPrompt := tmpl.BuildSystemPrompt("")
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"reference_material": reference,
	})

	reply, err := a.client.ChatSimpleStream(a.ctx, a.cfg.Model, systemPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("AI 归纳请求失败: %w", err)
	}

	jsonStr := util.ExtractJSON(reply)
	var summary struct {
		WorldviewElements string `json:"worldview_elements"`
		CharacterHints    string `json:"character_hints"`
		PlotIdeas         string `json:"plot_ideas"`
		StyleNotes        string `json:"style_notes"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &summary); err != nil {
		return "", fmt.Errorf("解析归纳结果失败: %w", err)
	}

	// 拼接为一段紧凑的参考文本
	return fmt.Sprintf(
		"世界观设定要点：%s\n\n角色设定提示：%s\n\n情节线索：%s\n\n风格基调：%s",
		summary.WorldviewElements,
		summary.CharacterHints,
		summary.PlotIdeas,
		summary.StyleNotes,
	), nil
}

// OpenProject 打开已有项目
func (a *App) OpenProject(dir string) (map[string]interface{}, error) {
	pm, err := project.Open(dir)
	if err != nil {
		return nil, err
	}
	a.setPM(pm)
	a.initAgents()
	return map[string]interface{}{
		"title": pm.Meta.Title,
		"genre": pm.Meta.Genre,
		"style": pm.Meta.Style,
	}, nil
}

// CloseProject 关闭当前项目
func (a *App) CloseProject() error {
	// closePM 内部已处理 nil 检查和写锁
	return a.closePM()
}

// GetProjectInfo 获取当前项目信息
func (a *App) GetProjectInfo() map[string]interface{} {
	pm := a.getPM()
	if pm == nil {
		return nil
	}
	return map[string]interface{}{
		"title": pm.Meta.Title,
		"genre": pm.Meta.Genre,
		"style": pm.Meta.Style,
		"path":  pm.Dir,
	}
}
