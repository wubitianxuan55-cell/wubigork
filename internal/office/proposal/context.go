// Package proposal — 章节生成上下文统一构建
package proposal

import (
	"context"
	"fmt"
	"strings"
)

// SectionContext 章节生成的完整上下文
type SectionContext struct {
	Proposal     *Proposal
	Target       *ProposalSection
	SystemPrompt string
	UserPrompt   string
	PrevContent  string
	NextTitle    string
	WordTarget   int
}

// SectionContext 构建章节生成上下文（大纲/招标要点/项目事实/字数目标/前后锚点）
func (s *Service) SectionContext(ctx context.Context, proposalID, sectionID string) (*SectionContext, error) {
	if s.ai == nil {
		return nil, fmt.Errorf("AI 客户端未初始化")
	}
	p, err := s.store.Get(proposalID)
	if err != nil {
		return nil, err
	}
	if err := s.requireStage(p, StageGenerate); err != nil {
		return nil, err
	}
	flat := flattenSections(p.Sections)
	var target *ProposalSection
	var prevContent string
	for _, sec := range flat {
		if sec.ID == sectionID {
			target = sec
			break
		}
		if sec.Content != "" {
			prevContent = sec.Content
		}
	}
	if target == nil {
		return nil, fmt.Errorf("章节未找到: %s", sectionID)
	}
	nextTitle := ""
	for i, sec := range flat {
		if sec.ID == sectionID && i+1 < len(flat) {
			nextTitle = flat[i+1].Title
		}
	}
	wordTarget := target.WordTarget
	if wordTarget <= 0 {
		wordTarget = 800
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("方案标题：%s", p.Title))
	parts = append(parts, fmt.Sprintf("方案类型：%s", p.Template))
	if p.Requirements != "" {
		parts = append(parts, "需求描述："+p.Requirements)
	}
	parts = append(parts, "方案大纲：")
	var walk func(ss []ProposalSection, depth int)
	walk = func(ss []ProposalSection, depth int) {
		for _, sec := range ss {
			marker := ""
			if sec.Status == "completed" {
				marker = " ✓"
			}
			parts = append(parts, fmt.Sprintf("%s%d. %s%s", strings.Repeat("  ", depth), sec.Index+1, sec.Title, marker))
			walk(sec.Children, depth+1)
		}
	}
	walk(p.Sections, 0)
	if prevContent != "" {
		parts = append(parts, "\n前一章节内容参考：\n"+truncate(prevContent, 1500))
	}
	if nextTitle != "" {
		parts = append(parts, "下一章节标题："+nextTitle+"（结尾应自然衔接，不提前展开）")
	}
	if p.BidSummary != nil {
		bs := p.BidSummary
		if len(bs.TechScoring) > 0 {
			parts = append(parts, "\n【招标评分标准】")
			for _, item := range bs.TechScoring {
				parts = append(parts, fmt.Sprintf("- %s（%s分）：%s", item.Name, item.MaxScore, item.Requirement))
			}
		}
		if len(bs.KeyRequirements) > 0 {
			parts = append(parts, "\n【核心要求】")
			for _, req := range bs.KeyRequirements {
				parts = append(parts, "- "+req)
			}
		}
		if len(bs.RedLines) > 0 {
			parts = append(parts, "\n【废标条款（严禁违反）】")
			for _, r := range bs.RedLines {
				parts = append(parts, "- "+r)
			}
		}
		if len(bs.Format) > 0 {
			parts = append(parts, "\n【格式要求】")
			for _, it := range bs.Format {
				parts = append(parts, "- "+it.Name+"："+it.Content)
			}
		}
		if len(bs.DarkRules) > 0 {
			parts = append(parts, "\n【暗标要求（必须遵守）】")
			for _, it := range bs.DarkRules {
				parts = append(parts, "- "+it.Name+"："+it.Content)
			}
		}
		if bs.Overview != "" {
			parts = append(parts, "\n【项目概况】"+bs.Overview)
		}
		if bs.Duration != "" {
			parts = append(parts, "\n【工期】"+bs.Duration)
		}
	}
	if p.ProjectID != "" {
		if facts, err := s.store.GetProjectFacts(p.ProjectID); err == nil && len(facts) > 0 {
			parts = append(parts, "\n【项目事实基线（全篇保持一致）】")
			for k, v := range facts {
				parts = append(parts, "- "+k+"："+v)
			}
		}
	}
	if p.Template != "" || p.Category != "" {
		if ref := s.legacyRefFor(p.Template, p.Category); ref != "" {
			parts = append(parts, "\n【历史方案参考（同类型，仅供结构/措辞参考，不得抄袭）】\n"+ref)
		}
	}
	parts = append(parts, fmt.Sprintf("\n本章节字数目标：%d 字", wordTarget))

	systemPrompt := enrichSoilPrompt(p.Template, fmt.Sprintf(`你是一位专业的环保工程投标方案撰写专家，精通土壤修复领域。现在撰写投标技术方案中的「%s」章节。
要求：
- 语言专业、条理清晰，符合投标文件规范
- 使用 Markdown 格式
- 字数约 %d 字（核心章节应更详细）
- 紧扣本章标题和招标评分标准
- 尽可能引用场地污染数据，体现实地调研深度
- 如果上下文中有评分标准，本章应尽量覆盖相关评分点
- 对于技术描述，引用 HJ 25.4 等规范标准
- 如果上下文有暗标要求，必须遵守
直接输出章节正文，不需要标题。`, target.Title, wordTarget))

	return &SectionContext{
		Proposal:     p,
		Target:       target,
		SystemPrompt: systemPrompt,
		UserPrompt:   strings.Join(parts, "\n"),
		PrevContent:  prevContent,
		NextTitle:    nextTitle,
		WordTarget:   wordTarget,
	}, nil
}

// GenerateSection AI 撰写章节内容（基于统一上下文）
func (s *Service) GenerateSection(ctx context.Context, proposalID, sectionID, instruction string) (*Proposal, error) {
	sc, err := s.SectionContext(ctx, proposalID, sectionID)
	if err != nil {
		return nil, err
	}
	userMsg := sc.UserPrompt
	if instruction != "" {
		userMsg += "\n\n【额外要求】" + instruction
	}
	reply, err := s.ai.ChatSimpleStream(ctx, "", sc.SystemPrompt, userMsg)
	if err != nil {
		return nil, fmt.Errorf("AI 生成失败: %w", err)
	}
	sc.Target.Content = reply
	sc.Target.Status = "completed"
	sc.Target.Words = countRunes(reply)
	sc.Proposal.UpdatedAt = now()
	if err := s.store.Update(sc.Proposal); err != nil {
		return nil, err
	}
	return sc.Proposal, nil
}
