// Package proposal — 工作流阶段与闸门
package proposal

import "fmt"

const (
	StageParse    = "parse"
	StageGenerate = "generate"
	StageCheck    = "check"
	StageFormat   = "format"
)

var stageRank = map[string]int{StageParse: 1, StageGenerate: 2, StageCheck: 3, StageFormat: 4}

// requireStage 检查阶段：有招标文件但未解析时阻止生成类操作
func (s *Service) requireStage(p *Proposal, need string) error {
	if need == StageGenerate && p.BidSummary != nil && len(p.BidSummary.RawFiles) > 0 && p.Stage == "" {
		return errStageGate(need)
	}
	return nil
}

// advanceStage 推进到更高阶段（幂等）
func (p *Proposal) advanceStage(next string) {
	if stageRank[next] > stageRank[p.Stage] {
		p.Stage = next
	}
}

func errStageGate(need string) error {
	if need == StageGenerate {
		return fmt.Errorf("阶段闸门：请先完成招标解析（AI 分析）后再生成大纲/章节")
	}
	return fmt.Errorf("阶段闸门：请先完成上一阶段（需要 %s）", need)
}
