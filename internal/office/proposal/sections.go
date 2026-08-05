package proposal

import (
	"fmt"
	"strings"

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

// MoveSection 在同级内移动章节（delta=-1 上移，1 下移），自动重编号
func (s *Service) MoveSection(proposalID, sectionID string, delta int) (*Proposal, error) {
	p, err := s.store.Get(proposalID)
	if err != nil {
		return nil, err
	}
	if delta == 0 {
		return p, nil
	}
	flat := flattenSections(p.Sections)
	var target *ProposalSection
	for _, sec := range flat {
		if sec.ID == sectionID {
			target = sec
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("章节未找到: %s", sectionID)
	}
	siblings := siblingsOf(p.Sections, target.ParentID)
	idx := -1
	for i := range siblings {
		if siblings[i].ID == sectionID {
			idx = i
			break
		}
	}
	if idx < 0 || idx+delta < 0 || idx+delta >= len(siblings) {
		return p, nil
	}
	siblings[idx], siblings[idx+delta] = siblings[idx+delta], siblings[idx]
	reindexSections(p.Sections)
	p.UpdatedAt = now()
	if err := s.store.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

// siblingsOf 返回指定父节点下的子章节切片（顶层 parentID 为空）
func siblingsOf(ss []ProposalSection, parentID string) []ProposalSection {
	if parentID == "" {
		return ss
	}
	for i := range ss {
		if ss[i].ID == parentID {
			return ss[i].Children
		}
		if r := siblingsOf(ss[i].Children, parentID); r != nil {
			return r
		}
	}
	return nil
}

// ImportOutline 从 Markdown 标题解析章节树（#/##/###），替换现有大纲
func (s *Service) ImportOutline(proposalID, markdown string) (*Proposal, error) {
	p, err := s.store.Get(proposalID)
	if err != nil {
		return nil, err
	}
	var roots []*outlineNode
	var stack []*outlineNode
	idx := 0
	for _, line := range strings.Split(markdown, "\n") {
		trimmed := strings.TrimSpace(line)
		level := headingLevel(trimmed)
		if level == 0 {
			continue
		}
		title := strings.TrimSpace(trimmed[level:])
		n := &outlineNode{sec: ProposalSection{
			ID: uuid.New().String(), ProposalID: proposalID, Level: level,
			Index: idx, Title: title, Status: "pending",
		}}
		idx++
		for len(stack) >= level {
			stack = stack[:len(stack)-1]
		}
		if len(stack) > 0 {
			parent := stack[len(stack)-1]
			n.sec.ParentID = parent.sec.ID
			parent.children = append(parent.children, n)
		} else {
			roots = append(roots, n)
		}
		stack = append(stack, n)
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("未解析到任何 Markdown 标题")
	}
	sections := make([]ProposalSection, 0, len(roots))
	for _, r := range roots {
		sections = append(sections, materializeNode(r))
	}
	p.Sections = sections
	reindexSections(p.Sections)
	p.UpdatedAt = now()
	if err := s.store.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

// outlineNode 大纲导入用的内部节点树
type outlineNode struct {
	sec      ProposalSection
	children []*outlineNode
}

// materializeNode 把内部节点树转换为值树
func materializeNode(n *outlineNode) ProposalSection {
	sec := n.sec
	for _, c := range n.children {
		sec.Children = append(sec.Children, materializeNode(c))
	}
	return sec
}

func headingLevel(s string) int {
	if !strings.HasPrefix(s, "#") {
		return 0
	}
	n := 0
	for n < len(s) && s[n] == '#' {
		n++
	}
	if n > 3 || n >= len(s) || s[n] != ' ' {
		return 0
	}
	return n
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
