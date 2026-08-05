// Package proposal — 批量生成与合并装配
package proposal

import (
	"context"
	"fmt"
	"strings"
)

// BatchUnit 批量生成单元
type BatchUnit struct {
	SectionID  string
	Title      string
	Level      int
	Index      int
	WordTarget int
}

// BuildBatchUnits 深度优先展开全部叶子章节为生成单元（有子级的章不单独生成）
func BuildBatchUnits(sections []ProposalSection) []BatchUnit {
	var out []BatchUnit
	var walk func(ss []ProposalSection)
	walk = func(ss []ProposalSection) {
		for _, sec := range ss {
			if len(sec.Children) == 0 {
				out = append(out, BatchUnit{
					SectionID: sec.ID, Title: sec.Title, Level: sec.Level,
					Index: sec.Index, WordTarget: sec.WordTarget,
				})
			}
			walk(sec.Children)
		}
	}
	walk(sections)
	return out
}

// BatchProgress 批量进度回调
type BatchProgress func(current, total int, sectionID, status string, words int)

// RunBatch 顺序批量生成未完成单元；已完成的跳过（断点续写）；失败标记 error 继续。
func (s *Service) RunBatch(ctx context.Context, proposalID string, progress BatchProgress) error {
	if s.ai == nil {
		return fmt.Errorf("AI 客户端未初始化")
	}
	p, err := s.store.Get(proposalID)
	if err != nil {
		return err
	}
	units := BuildBatchUnits(p.Sections)
	total := len(units)
	current := 0
	for _, u := range units {
		select {
		case <-ctx.Done():
			if progress != nil {
				progress(current, total, u.SectionID, "cancelled", 0)
			}
			return ctx.Err()
		default:
		}
		current++
		latest, err := s.store.Get(proposalID)
		if err != nil {
			return err
		}
		target := findSectionByID(latest.Sections, u.SectionID)
		if target == nil {
			continue
		}
		if target.Status == "completed" {
			if progress != nil {
				progress(current, total, u.SectionID, "skipped", countRunes(target.Content))
			}
			continue
		}
		updated, err := s.GenerateSection(ctx, proposalID, u.SectionID, "")
		if err != nil {
			if progress != nil {
				progress(current, total, u.SectionID, "error", 0)
			}
			continue
		}
		done := findSectionByID(updated.Sections, u.SectionID)
		words := 0
		if done != nil {
			words = countRunes(done.Content)
		}
		if progress != nil {
			progress(current, total, u.SectionID, "done", words)
		}
	}
	return nil
}

func findSectionByID(ss []ProposalSection, id string) *ProposalSection {
	for _, sec := range flattenSections(ss) {
		if sec.ID == id {
			return sec
		}
	}
	return nil
}

// Assemble 合并装配：统一编号（第N章 / N.M / N.M.K）+ 标题层级 + 正文
func Assemble(p *Proposal) string {
	var sb strings.Builder
	sb.WriteString("# " + p.Title + "\n\n")
	var walk func(ss []ProposalSection, prefix string, chapter int)
	walk = func(ss []ProposalSection, prefix string, chapter int) {
		idx := 0
		for _, sec := range ss {
			idx++
			num := ""
			markdownLevel := 0
			switch sec.Level {
			case 1:
				num = fmt.Sprintf("第%d章", idx)
				markdownLevel = 1
			case 2:
				num = fmt.Sprintf("%d.%d", chapter, idx)
				markdownLevel = 2
			default:
				num = fmt.Sprintf("%s.%d", prefix, idx)
				markdownLevel = 3
			}
			title := num + " " + sec.Title
			sb.WriteString(strings.Repeat("#", markdownLevel) + " " + title + "\n\n")
			if sec.Content != "" {
				sb.WriteString(sec.Content + "\n\n")
			} else {
				sb.WriteString("（待撰写）\n\n")
			}
			childChapter := idx
			if sec.Level != 1 {
				childChapter = chapter
			}
			walk(sec.Children, num, childChapter)
		}
	}
	walk(p.Sections, "", 0)
	return sb.String()
}
