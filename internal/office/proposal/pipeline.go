// Package proposal — 一键流水线
package proposal

import (
	"context"
	"fmt"
)

// PipelineProgress 流水线进度回调
type PipelineProgress func(step, status, detail string)

// RunPipeline 按四阶段顺序执行可运行步骤：解析→大纲→批量生成→全面检查
func (s *Service) RunPipeline(ctx context.Context, proposalID string, onProgress PipelineProgress) (*Proposal, []CheckItem, error) {
	p, err := s.store.Get(proposalID)
	if err != nil {
		return nil, nil, err
	}
	if onProgress != nil {
		onProgress("parse", "start", "")
	}
	if p.BidSummary != nil && len(p.BidSummary.RawFiles) > 0 && p.Stage == "" {
		if _, err := s.ParseBidFileWithProgress(ctx, proposalID, func(stage, detail string) {
			if onProgress != nil {
				onProgress("parse", stage, detail)
			}
		}); err != nil {
			return nil, nil, fmt.Errorf("招标解析失败: %w", err)
		}
	}
	if onProgress != nil {
		onProgress("outline", "start", "")
	}
	p, _ = s.store.Get(proposalID)
	if len(p.Sections) == 0 {
		if _, err := s.GenerateOutline(ctx, proposalID, p.Requirements, "", 0); err != nil {
			return nil, nil, fmt.Errorf("大纲生成失败: %w", err)
		}
	}
	if onProgress != nil {
		onProgress("generate", "start", "")
	}
	if err := s.RunBatch(ctx, proposalID, func(cur, total int, sid, status string, words int) {
		if onProgress != nil {
			onProgress("generate", status, fmt.Sprintf("%d/%d", cur, total))
		}
	}); err != nil {
		return nil, nil, err
	}
	if onProgress != nil {
		onProgress("check", "start", "")
	}
	p, items, err := s.CheckAll(ctx, proposalID)
	if err != nil {
		return nil, nil, err
	}
	if onProgress != nil {
		onProgress("check", "done", fmt.Sprintf("%d 项", len(items)))
	}
	return p, items, nil
}
