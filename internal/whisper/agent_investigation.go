// Package whisper — agent_investigation.go
// 100% 对齐 ackem desktop-agent/investigation/
// 调查系统：多步信息收集 → 综合 → 交付

package whisper

import "strings"

// ─── Investigation ────────────────────────────────────────────

// Investigation 调查任务
type Investigation struct {
	ID        string              `json:"id"`
	Question  string              `json:"question"`
	Status    string              `json:"status"` // pending/collecting/synthesizing/done
	Sources   []InvestigationSource `json:"sources"`
	Synthesis string              `json:"synthesis"`
	RoundCount int                `json:"roundCount"`
}

// InvestigationSource 调查来源
type InvestigationSource struct {
	ToolName string `json:"toolName"`
	Query    string `json:"query"`
	Result   string `json:"result"`
	Relevance float64 `json:"relevance"` // 0-1
}

// ─── Investigation Engine ─────────────────────────────────────

// NewInvestigation 创建调查任务
func NewInvestigation(question string) *Investigation {
	return &Investigation{
		ID:       genHexID(),
		Question: question,
		Status:   "pending",
	}
}

// StartCollection 开始收集阶段
func (inv *Investigation) StartCollection() {
	inv.Status = "collecting"
}

// AddSource 添加收集来源
func (inv *Investigation) AddSource(source InvestigationSource) {
	inv.Sources = append(inv.Sources, source)
	inv.RoundCount++
}

// Synthesize 综合收集结果
func (inv *Investigation) Synthesize() string {
	inv.Status = "synthesizing"

	// 按相关性排序
	var relevant []InvestigationSource
	for _, s := range inv.Sources {
		if s.Relevance > 0.3 {
			relevant = append(relevant, s)
		}
	}

	var sb strings.Builder
	sb.WriteString("【调查综合 · " + inv.Question + "】\n")

	if len(relevant) == 0 {
		sb.WriteString("未找到相关信息。\n")
	} else {
		for i, s := range relevant {
			sb.WriteString(itoa(i+1) + ". " + s.ToolName + " → " + truncStr(s.Result, 200) + "\n")
		}
	}

	inv.Synthesis = sb.String()
	inv.Status = "done"
	return inv.Synthesis
}

// ShouldContinueCollection 是否应继续收集
func (inv *Investigation) ShouldContinueCollection(maxRounds, minSources int) bool {
	if inv.RoundCount >= maxRounds {
		return false
	}
	if len(inv.Sources) >= minSources {
		// 检查最近来源的相关性
		if len(inv.Sources) > 0 {
			last := inv.Sources[len(inv.Sources)-1]
			if last.Relevance < 0.2 {
				return false
			}
		}
	}
	return inv.Status == "collecting"
}

// BuildInvestigationHint 构建调查提示
func (inv *Investigation) BuildInvestigationHint() string {
	if inv == nil || inv.Status == "done" {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("【调查中】\n")
	sb.WriteString("问题：" + inv.Question + "\n")
	sb.WriteString("已收集 " + itoa(len(inv.Sources)) + " 条信息\n")

	if len(inv.Sources) > 0 {
		sb.WriteString("最新发现：\n")
		for _, s := range inv.Sources {
			sb.WriteString("· [" + s.ToolName + "] " + truncStr(s.Result, 100) + "\n")
		}
	}
	return sb.String()
}
