package app

import (
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
)

// 设置中枢批「文学提示词收编」（2026-11-19）：wenxue_system_prompt /
// wenxue_system_prompt_file 双键此前无任何绑定（配置面=手改 ga.cfg），设置页
// 文学组补读/写绑对。走 CoreB 直名门面（proxy 按名路由，无需 gaeaToGaea 映射）。
// 语义与 wenxueSystemPrompt() 一致：file 键优先，双空 = 代码默认
// DefaultWenxueSystemPrompt；写经 gaeaApplyCfg 落盘并跟随重建图文引擎
// （wenxueReloadIfInit），未初始化时首次 WenxueEngineInit 自然取新配置。

// WenxuePromptView 文学分板身份提示词：双配置键原文 + 当前生效值（file 优先、
// 双空=板块默认），effective 供设置页只读展示「实际出声的人格」。
type WenxuePromptView struct {
	Prompt    string `json:"prompt"`
	File      string `json:"file"`
	Effective string `json:"effective"`
}

// WenxuePromptGet 读文学身份提示词双键与生效值。
func (a *App) WenxuePromptGet() (WenxuePromptView, error) {
	ga.mu.Lock()
	defer ga.mu.Unlock()
	if ga.cfg == nil {
		// ga.cfg 未就绪（测试/早期构建）：与 wenxueSystemPrompt() 同语义，
		// 恒不回落办公人格，直接报板块默认。
		return WenxuePromptView{Effective: gaeaConfig.DefaultWenxueSystemPrompt}, nil
	}
	eff, err := ga.cfg.ResolveWenxueSystemPrompt()
	if err != nil {
		return WenxuePromptView{}, err
	}
	return WenxuePromptView{
		Prompt:    ga.cfg.Agent.WenxueSystemPrompt,
		File:      ga.cfg.Agent.WenxueSystemPromptFile,
		Effective: eff,
	}, nil
}

// WenxuePromptSet 写双键（空串=清空该键回落默认）并重建。
func (a *App) WenxuePromptSet(prompt string, file string) error {
	return a.gaeaApplyCfg(func(cfg *gaeaConfig.Config) {
		cfg.Agent.WenxueSystemPrompt = prompt
		cfg.Agent.WenxueSystemPromptFile = file
	})
}
