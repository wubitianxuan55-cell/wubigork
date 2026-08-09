// Package proposal — 校验引擎：可插拔规则 + 统一检查报告
package proposal

import (
	"context"
)

// CheckItem 单条检查结果
// CheckLocation 原文定位：問題出現在哪一章、原文摘錄與起始偏移。
type CheckLocation struct {
	SectionID string `json:"sectionId,omitempty"`
	Excerpt   string `json:"excerpt,omitempty"`
	Offset    int    `json:"offset,omitempty"`
}

type CheckItem struct {
	Rule      string `json:"rule"`
	Severity  string `json:"severity"` // critical | warning | info
	Status    string `json:"status"`   // pass | fail | warn | skip | error
	SectionID string `json:"sectionId,omitempty"`
	Message   string `json:"message"`
	Evidence  string `json:"evidence,omitempty"`
	// Suggestion 是整改建議：指出該怎麼改，直接可照做。
	Suggestion string `json:"suggestion,omitempty"`
	// Locations 是原文定位：命中的章節 + 上下文摘錄，前端可一鍵跳轉。
	Locations []CheckLocation `json:"locations,omitempty"`
}

// CheckRule 校验规则接口
type CheckRule interface {
	Name() string
	Severity() string
	Run(ctx context.Context, p *Proposal) ([]CheckItem, error)
}

// ruleFunc 函数式规则适配器
type ruleFunc struct {
	name     string
	severity string
	fn       func(ctx context.Context, p *Proposal) ([]CheckItem, error)
}

func (r ruleFunc) Name() string     { return r.name }
func (r ruleFunc) Severity() string { return r.severity }
func (r ruleFunc) Run(ctx context.Context, p *Proposal) ([]CheckItem, error) {
	return r.fn(ctx, p)
}

// RunChecks 顺序执行全部规则，错误规则转为 error item
func RunChecks(ctx context.Context, p *Proposal, rules []CheckRule) []CheckItem {
	var out []CheckItem
	for _, rule := range rules {
		items, err := rule.Run(ctx, p)
		if err != nil {
			out = append(out, CheckItem{
				Rule: rule.Name(), Severity: rule.Severity(), Status: "error",
				Message: "规则执行失败: " + err.Error(),
			})
			continue
		}
		for i := range items {
			if items[i].Rule == "" {
				items[i].Rule = rule.Name()
			}
			if items[i].Severity == "" {
				items[i].Severity = rule.Severity()
			}
			out = append(out, items[i])
		}
	}
	return out
}

type ruleError struct{ msg string }

func (e *ruleError) Error() string { return e.msg }

// defaultRules 完整规则集：结构化 + AI 语义
func (s *Service) defaultRules() []CheckRule {
	return append(s.structuredRules(),
		ruleFunc{name: "评分覆盖检查", severity: "warning", fn: s.runCoverageRule})
}
