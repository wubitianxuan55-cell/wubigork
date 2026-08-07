package app

import "github.com/gaea/gaea/internal/office/proposal"

// leftSource 左脑数据源（测试可注入 fake；App 装配时用真实实现）。
type leftSource interface {
	ListProposals() ([]proposal.Proposal, error)
}

type leftBrain struct {
	src leftSource
}

func (l *leftBrain) Read(entity string) ([]Fact, error) {
	ps, err := l.src.ListProposals()
	if err != nil {
		return nil, err
	}
	var out []Fact
	for _, p := range ps {
		if p.Title == entity || p.ID == entity {
			out = append(out, Fact{Brain: BrainLeft, Entity: p.Title, Attribute: "proposal", Value: p.Requirements})
		}
	}
	return out, nil
}

func (l *leftBrain) Write(entity, attribute, value string) error {
	return nil // 左脑办公记忆以现有业务写入为准，BrainStore 不做直写
}

func (l *leftBrain) Search(query string) ([]Hit, error) {
	ps, err := l.src.ListProposals()
	if err != nil {
		return nil, err
	}
	terms := splitQueryTerms(query)
	var out []Hit
	for _, p := range ps {
		for _, term := range terms {
			if matchAny(term, p.Title, p.Category, p.Requirements) {
				out = append(out, Hit{Brain: BrainLeft, Entity: p.Title, Text: p.Requirements})
				break
			}
		}
	}
	return out, nil
}

// proposalLeftSource 用 office 方案 Service 实现 leftSource。
type proposalLeftSource struct {
	svc *proposal.Service
}

func (p *proposalLeftSource) ListProposals() ([]proposal.Proposal, error) {
	if p.svc == nil {
		return nil, nil
	}
	return p.svc.List()
}
