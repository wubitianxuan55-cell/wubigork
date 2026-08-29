package app

import (
	"strings"

	"github.com/gaea/gaea/internal/gaea/spaces"
)

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

// SearchInSpace 是 Search 的空间限定版（S1.2 B 读端隔离器，统一检索 brain 组
// 使用）。空间映射（设计 §检索面地图 + §勘误）：
//   - brain.right（轻语 whisper）= play 专属整域：work scope 整体丢弃、play
//     scope 可见、""=全部（旧行为）；
//   - brain.left（办公 facts）：同库混存两空间，按 space 走 ListInSpace 谓词
//     （数据源实现 spaceLeftSource 时生效）；
//   - brain.main（profile + knowledge）：共享面不过滤——画像与工程知识库无
//     空间维度，两空间同可见。
//
// space 为空与 Search 完全等价（既有调用零行为变化）。
func (b *BrainStore) SearchInSpace(query, space string, brains ...string) ([]Hit, error) {
	if space == "" {
		return b.Search(query, brains...)
	}
	names := brains
	if len(names) == 0 {
		names = []string{BrainMain, BrainLeft, BrainRight}
	}
	var out []Hit
	for _, n := range names {
		// whisper 右脑 play 专属：work scope 整体丢弃（隔离红线）。
		if n == BrainRight && space != spaces.SpacePlay {
			continue
		}
		ad := b.adapter(n)
		if ad == nil {
			continue
		}
		var hits []Hit
		var err error
		if n == BrainLeft {
			if lb, ok := ad.(*leftBrain); ok {
				hits, err = lb.searchInSpace(query, space)
			} else {
				hits, err = ad.Search(query)
			}
		} else {
			hits, err = ad.Search(query)
		}
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
