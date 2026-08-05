// Package proposal — AI 语义覆盖规则
package proposal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// runCoverageRule 对照招标评分标准做语义覆盖检查（LLM）
func (s *Service) runCoverageRule(ctx context.Context, p *Proposal) ([]CheckItem, error) {
	if s.ai == nil || p.BidSummary == nil || len(p.BidSummary.TechScoring) == 0 {
		return nil, nil
	}
	var allContent strings.Builder
	for _, sec := range flattenSections(p.Sections) {
		if sec.Content != "" {
			allContent.WriteString(fmt.Sprintf("【%s】\n%s\n\n", sec.Title, sec.Content))
		}
	}
	var scoringList strings.Builder
	for _, item := range p.BidSummary.TechScoring {
		scoringList.WriteString(fmt.Sprintf("- %s（%s分）：%s\n", item.Name, item.MaxScore, item.Requirement))
	}
	sp := fmt.Sprintf("你是环保工程投标评审专家。对照评分标准检查方案：\n%s\n返回 JSON: [{\"name\":\"\",\"maxScore\":\"\",\"covered\":\"full|partial|none\",\"suggestion\":\"\"}]", scoringList.String())
	reply, err := s.ai.ChatSimpleStream(ctx, "", sp, allContent.String())
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(reply)
	if strings.HasPrefix(raw, "[") {
		reply = raw
	} else {
		reply = extractJSON(raw)
	}
	var results []struct {
		Name       string `json:"name"`
		MaxScore   string `json:"maxScore"`
		Covered    string `json:"covered"`
		Suggestion string `json:"suggestion"`
	}
	if err := json.Unmarshal([]byte(reply), &results); err != nil {
		// 兼容对象包裹 {"results":[...]}
		var wrapped struct {
			Results []struct {
				Name       string `json:"name"`
				MaxScore   string `json:"maxScore"`
				Covered    string `json:"covered"`
				Suggestion string `json:"suggestion"`
			} `json:"results"`
		}
		if err2 := json.Unmarshal([]byte(reply), &wrapped); err2 != nil {
			return nil, fmt.Errorf("解析覆盖检查失败: %w", err)
		}
		for _, r := range wrapped.Results {
			results = append(results, r)
		}
	}
	var out []CheckItem
	for _, r := range results {
		status := "pass"
		if r.Covered == "partial" {
			status = "warn"
		} else if r.Covered == "none" {
			status = "fail"
		}
		out = append(out, CheckItem{
			Status: status, Message: fmt.Sprintf("评分项「%s」（%s分）：%s", r.Name, r.MaxScore, r.Covered),
			Evidence: r.Suggestion,
		})
	}
	return out, nil
}
