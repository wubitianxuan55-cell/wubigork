package proposal

import (
	"fmt"

	"github.com/google/uuid"
)

// AddSection 在指定章节下新增子章节（parentID 为空则新增顶级章节）
func (s *Service) AddSection(proposalID, parentID, title string) (*Proposal, error) {
	p, err := s.store.Get(proposalID)
	if err != nil {
		return nil, err
	}
	if title == "" {
		title = "新章节"
	}
	level := 1
	if parentID != "" {
		parent := findSection(p.Sections, parentID)
		if parent == nil {
			return nil, fmt.Errorf("父章节未找到: %s", parentID)
		}
		level = parent.Level + 1
		if level > 3 {
			level = 3
		}
	}
	sec := ProposalSection{
		ID: uuid.New().String(), ProposalID: proposalID, ParentID: parentID,
		Index: countSections(p.Sections), Level: level, Title: title, Status: "pending",
	}
	if parentID == "" {
		p.Sections = append(p.Sections, sec)
	} else {
		appendChild(p.Sections, parentID, sec)
	}
	if err := s.store.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

// RemoveSection 删除章节及其全部子章节
func (s *Service) RemoveSection(proposalID, sectionID string) (*Proposal, error) {
	p, err := s.store.Get(proposalID)
	if err != nil {
		return nil, err
	}
	before := countSections(p.Sections)
	p.Sections = removeSectionByID(p.Sections, sectionID)
	if countSections(p.Sections) == before {
		return nil, fmt.Errorf("章节未找到: %s", sectionID)
	}
	reindexSections(p.Sections)
	if err := s.store.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

// RenameSection 重命名章节（不改动内容与状态）
func (s *Service) RenameSection(proposalID, sectionID, title string) (*Proposal, error) {
	p, err := s.store.Get(proposalID)
	if err != nil {
		return nil, err
	}
	if title == "" {
		return nil, fmt.Errorf("章节标题不能为空")
	}
	found := false
	for _, sec := range flattenSections(p.Sections) {
		if sec.ID == sectionID {
			sec.Title = title
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("章节未找到: %s", sectionID)
	}
	if err := s.store.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

func findSection(ss []ProposalSection, id string) *ProposalSection {
	for _, sec := range flattenSections(ss) {
		if sec.ID == id {
			return sec
		}
	}
	return nil
}

func appendChild(ss []ProposalSection, parentID string, sec ProposalSection) bool {
	for i := range ss {
		if ss[i].ID == parentID {
			ss[i].Children = append(ss[i].Children, sec)
			return true
		}
		if appendChild(ss[i].Children, parentID, sec) {
			return true
		}
	}
	return false
}

func removeSectionByID(ss []ProposalSection, id string) []ProposalSection {
	out := ss[:0]
	for _, s := range ss {
		if s.ID == id {
			continue
		}
		s.Children = removeSectionByID(s.Children, id)
		out = append(out, s)
	}
	return out
}

// reindexSections 按深度优先顺序重排所有章节的 Index
func reindexSections(ss []ProposalSection) {
	i := 0
	var walk func(secs []ProposalSection)
	walk = func(secs []ProposalSection) {
		for j := range secs {
			secs[j].Index = i
			i++
			walk(secs[j].Children)
		}
	}
	walk(ss)
}
