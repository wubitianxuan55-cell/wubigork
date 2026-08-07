package app

import "strings"

// Brain 命名空间（与 1.x 记忆中枢三脑架构一致）。
const (
	BrainMain  = "brain.main"
	BrainLeft  = "brain.left"
	BrainRight = "brain.right"
)

// Fact 是跨脑统一事实。
type Fact struct {
	Brain     string `json:"brain"`
	Entity    string `json:"entity"`
	Attribute string `json:"attribute"`
	Value     string `json:"value"`
}

// Hit 是跨脑检索命中。
type Hit struct {
	Brain  string  `json:"brain"`
	Entity string  `json:"entity"`
	Text   string  `json:"text"`
	Score  float64 `json:"score"`
}

// Ref 是主脑跨脑关联引用。
type Ref struct {
	Brain string `json:"brain"`
	Ref   string `json:"ref"`
}

// brainAdapter 是单脑适配器的最小接口（测试可注入 fake）。
type brainAdapter interface {
	Read(entity string) ([]Fact, error)
	Write(entity, attribute, value string) error
	Search(query string) ([]Hit, error)
}

// BrainStore 三脑统一访问层。
type BrainStore struct {
	main  brainAdapter
	left  brainAdapter
	right brainAdapter
	links *LinkStore
}

func (b *BrainStore) Read(brain, entity string) ([]Fact, error) {
	ad := b.adapter(brain)
	if ad == nil {
		return []Fact{}, nil
	}
	return ad.Read(entity)
}

func (b *BrainStore) Write(brain, entity, attribute, value string) error {
	ad := b.adapter(brain)
	if ad == nil {
		return nil
	}
	return ad.Write(entity, attribute, value)
}

// Search 跨脑检索；brains 为空 = 三脑全搜。
func (b *BrainStore) Search(query string, brains ...string) ([]Hit, error) {
	names := brains
	if len(names) == 0 {
		names = []string{BrainMain, BrainLeft, BrainRight}
	}
	var out []Hit
	for _, n := range names {
		ad := b.adapter(n)
		if ad == nil {
			continue
		}
		hits, err := ad.Search(query)
		if err != nil {
			return nil, err
		}
		out = append(out, hits...)
	}
	return out, nil
}

func (b *BrainStore) Link(entity, brain, ref string) error {
	if b.links == nil {
		return nil
	}
	return b.links.Add(entity, brain, ref)
}

func (b *BrainStore) CrossRefs(entity string) ([]Ref, error) {
	if b.links == nil {
		return []Ref{}, nil
	}
	return b.links.ListByEntity(entity)
}

func (b *BrainStore) adapter(brain string) brainAdapter {
	switch brain {
	case BrainMain:
		return b.main
	case BrainLeft:
		return b.left
	case BrainRight:
		return b.right
	}
	return nil
}

// matchAny 供适配器做朴素关键词匹配。
func matchAny(query string, texts ...string) bool {
	for _, t := range texts {
		if strings.Contains(t, query) {
			return true
		}
	}
	return false
}
