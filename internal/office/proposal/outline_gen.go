// Package proposal — 大纲生成（策略 + 字数预算）
package proposal

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// GenerateOutline AI 生成方案大纲（支持目录策略与字数预算）
func (s *Service) GenerateOutline(ctx context.Context, proposalID, requirements, strategy string, totalWords int) (*Proposal, error) {
	if s.ai == nil {
		return nil, fmt.Errorf("AI 客户端未初始化")
	}
	p, err := s.store.Get(proposalID)
	if err != nil {
		return nil, err
	}
	if strategy == "" {
		strategy = OutlineStrategyReference
	}
	if totalWords <= 0 {
		totalWords = FallbackTotalWords
		if p.BidSummary != nil && p.BidSummary.TotalWords > 0 {
			totalWords = p.BidSummary.TotalWords
		}
	}

	systemPrompt := enrichSoilPrompt(p.Template, `你是一位专业方案撰写顾问。根据用户需求和招标要求，生成三级结构化方案大纲。

返回纯 JSON，格式为：
{"title":"方案标题","sections":[
  {"title":"第1章 章节名","level":1,"children":[
    {"title":"1.1 节名","level":2,"children":[
      {"title":"1.1.1 小节名","level":3},
      {"title":"1.1.2 小节名","level":3}
    ]},
    {"title":"1.2 节名","level":2}
  ]},
  {"title":"第2章 章节名","level":1,"children":[...]}
]}

要求：
- 第1级（章）：5-10章，逻辑递进
- 第2级（节）：每章2-5节
- 第3级（小节）：每节1-3个小节（可选）
- 标题简洁有力
- 如果提供了评分标准，大纲必须覆盖所有评分项
- 目录策略：`+outlineStrategyPrompt(strategy))

	userMsg := fmt.Sprintf("模板类型：%s\n总字数目标：%d 字（依据招标文件要求，可调整）\n需求描述：%s", p.Template, totalWords, requirements)
	if p.BidSummary != nil {
		if len(p.BidSummary.TechScoring) > 0 {
			userMsg += "\n\n【评分标准（大纲必须覆盖）】"
			for _, item := range p.BidSummary.TechScoring {
				userMsg += fmt.Sprintf("\n- %s（%s分）：%s", item.Name, item.MaxScore, item.Requirement)
			}
		}
		if len(p.BidSummary.KeyRequirements) > 0 {
			userMsg += "\n\n【核心要求】"
			for _, req := range p.BidSummary.KeyRequirements {
				userMsg += "\n- " + req
			}
		}
		if p.BidSummary.Overview != "" {
			userMsg += "\n\n【项目概况】" + p.BidSummary.Overview
		}
	}
	reply, err := s.ai.ChatSimpleStream(ctx, "", systemPrompt, userMsg)
	if err != nil {
		return nil, fmt.Errorf("AI 生成失败: %w", err)
	}

	// 解析 AI 返回的 JSON（支持树形三级大纲）
	reply = extractJSON(reply)
	var outline struct {
		Title    string `json:"title"`
		Sections []struct {
			Title    string `json:"title"`
			Level    int    `json:"level"`
			Children []struct {
				Title    string `json:"title"`
				Level    int    `json:"level"`
				Children []struct {
					Title string `json:"title"`
					Level int    `json:"level"`
				} `json:"children"`
			} `json:"children"`
		} `json:"sections"`
	}
	if err := json.Unmarshal([]byte(reply), &outline); err != nil {
		return nil, fmt.Errorf("解析 AI 输出失败: %w\n原始输出: %s", err, truncate(reply, 500))
	}

	p.Title = outline.Title
	p.Requirements = requirements
	p.Status = "writing"
	p.Sections = buildSectionsTree(proposalID, outline.Sections)
	applyWordBudgetToProposal(p.Sections, totalWords)
	p.UpdatedAt = now()

	if err := s.store.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

// buildSectionsTree 把 AI 返回的三级大纲构建为章节树（保持原行为）
func buildSectionsTree(proposalID string, outlineSections []struct {
	Title    string `json:"title"`
	Level    int    `json:"level"`
	Children []struct {
		Title    string `json:"title"`
		Level    int    `json:"level"`
		Children []struct {
			Title string `json:"title"`
			Level int    `json:"level"`
		} `json:"children"`
	} `json:"children"`
}) []ProposalSection {
	var newSections []ProposalSection
	idx := 0
	for _, ch := range outlineSections {
		chSec := ProposalSection{
			ID: uuid.New().String(), ProposalID: proposalID,
			Index: idx, Level: ch.Level, Title: ch.Title, Status: "pending",
		}
		if ch.Level == 0 {
			chSec.Level = 1
		}
		idx++
		for _, sec := range ch.Children {
			secLevel := sec.Level
			if secLevel == 0 {
				secLevel = 2
			}
			subSec := ProposalSection{
				ID: uuid.New().String(), ProposalID: proposalID,
				ParentID: chSec.ID, Index: idx, Level: secLevel,
				Title: sec.Title, Status: "pending",
			}
			idx++
			for _, sub := range sec.Children {
				subLevel := sub.Level
				if subLevel == 0 {
					subLevel = 3
				}
				subSec.Children = append(subSec.Children, ProposalSection{
					ID: uuid.New().String(), ProposalID: proposalID,
					ParentID: subSec.ID, Index: idx, Level: subLevel,
					Title: sub.Title, Status: "pending",
				})
				idx++
			}
			chSec.Children = append(chSec.Children, subSec)
		}
		newSections = append(newSections, chSec)
	}
	return newSections
}
