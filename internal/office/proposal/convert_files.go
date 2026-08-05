package proposal

import (
	"context"
	"fmt"
	"strings"
)

// ConvertFiles 批量转换已上传文件（调用 MarkItDown），通过回调报告进度
func (s *Service) ConvertFiles(ctx context.Context, proposalID string, onProgress func(current, total int, filename, status string)) (*Proposal, error) {
	p, err := s.store.Get(proposalID)
	if err != nil {
		return nil, err
	}
	if p.BidSummary == nil || len(p.BidSummary.RawFiles) == 0 {
		return nil, fmt.Errorf("没有待转换的文件，请先上传")
	}
	total := len(p.BidSummary.RawFiles)
	for i := range p.BidSummary.RawFiles {
		f := &p.BidSummary.RawFiles[i]
		if onProgress != nil {
			onProgress(i+1, total, f.Name, "converting")
		}
		if f.Markdown != "" {
			if onProgress != nil {
				onProgress(i+1, total, f.Name, "skipped")
			}
			continue
		}
		md, usedOCR, err := s.convertFile(ctx, f.Path)
		if err != nil {
			f.Error = err.Error()
			if onProgress != nil {
				onProgress(i+1, total, f.Name, "failed: "+err.Error())
			}
			continue
		}
		f.Markdown = md
		f.Error = ""
		if usedOCR {
			f.OCRStatus = "ocr"
		} else {
			f.OCRStatus = ""
		}
		if onProgress != nil {
			onProgress(i+1, total, f.Name, "done")
		}
	}
	var merged strings.Builder
	merged.WriteString(fmt.Sprintf("# 招标文件汇编\n\n> 共 %d 个文件\n\n", total))
	for i, f := range p.BidSummary.RawFiles {
		merged.WriteString(fmt.Sprintf("## 文件 %d：%s\n\n", i+1, f.Name))
		if f.Markdown != "" {
			merged.WriteString(f.Markdown)
		} else {
			merged.WriteString("（转换失败）")
		}
		merged.WriteString("\n\n---\n\n")
	}
	p.BidSummary.RawMarkdown = merged.String()
	p.BidSummary.RawText = merged.String()
	p.UpdatedAt = now()
	if err := s.store.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}
