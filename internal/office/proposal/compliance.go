package proposal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ComplianceItem 规范符合性检查项
type ComplianceItem struct {
	Standard string `json:"standard"`
	Clause   string `json:"clause"`
	Content  string `json:"content"`
	Status   string `json:"status"`
	Evidence string `json:"evidence"`
	Note     string `json:"note"`
}

// CheckCompliance 对照 HJ 25.4 等规范检查方案符合性
func (s *Service) CheckCompliance(ctx context.Context, proposalID string) (*Proposal, []ComplianceItem, error) {
	if s.ai == nil {
		return nil, nil, fmt.Errorf("AI 客户端未初始化")
	}
	p, err := s.store.Get(proposalID)
	if err != nil {
		return nil, nil, err
	}
	var allContent strings.Builder
	allContent.WriteString(fmt.Sprintf("# %s\n\n", p.Title))
	for _, sec := range flattenSections(p.Sections) {
		if sec.Content != "" {
			allContent.WriteString(fmt.Sprintf("## %s\n%s\n\n", sec.Title, truncate(sec.Content, 1500)))
		}
	}
	systemPrompt := fmt.Sprintf(`你是环保工程规范审查专家，精通 HJ 25.4。检查方案并返回 JSON：
[{"standard":"标准编号","clause":"条款号","content":"规范要求","status":"compliant|missing|partial","evidence":"方案中的对应描述","note":"补充建议"}]
%s`, SoilRemediationKB)
	reply, err := s.ai.ChatSimpleStream(ctx, "", systemPrompt, allContent.String())
	if err != nil {
		return nil, nil, fmt.Errorf("AI 检查失败: %w", err)
	}
	reply = extractJSON(reply)
	var items []ComplianceItem
	if err := json.Unmarshal([]byte(reply), &items); err != nil {
		return nil, nil, fmt.Errorf("解析失败: %w", err)
	}
	return p, items, nil
}
