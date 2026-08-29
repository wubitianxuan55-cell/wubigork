package app

import (
	"strings"

	"github.com/gaea/gaea/internal/gaea/memory"
)

// leftSource 左脑数据源（办公记忆 facts；测试可注入 fake）。
type leftSource interface {
	ListFacts() []memory.Memory
}

// spaceLeftSource 支持按空间取事实的数据源（S1.2 B 读端隔离器）：办公 facts
// 同库混存两空间（facts.space_id），读端按空间谓词收窄。实现为可选能力——
// fake 数据源可不实现（回退全量，与旧行为一致）。
type spaceLeftSource interface {
	ListFactsInSpace(space string) []memory.Memory
}

type leftBrain struct {
	src leftSource
}

func (l *leftBrain) Read(entity string) ([]Fact, error) {
	var out []Fact
	for _, m := range l.facts() {
		name := displayName(m.Title, m.Name)
		if name == entity || m.Name == entity {
			out = append(out, Fact{Brain: BrainLeft, Entity: name, Attribute: "fact", Value: factValue(m)})
		}
	}
	return out, nil
}

func (l *leftBrain) Write(entity, attribute, value string) error {
	return nil // 左脑办公记忆以现有业务写入为准，BrainStore 不做直写
}

func (l *leftBrain) Search(query string) ([]Hit, error) {
	return l.searchFacts(query, l.facts()), nil
}

// searchInSpace 是 Search 的空间限定版（S1.2 B）：facts 走 ListInSpace(space)
// 谓词（数据源实现 spaceLeftSource 时），空 space 与 Search 完全等价。空间
// 映射（设计 §勘误）：左脑办公 facts 同库混存两空间——work scope 只见 work
// 事实、play scope 只见 play 事实。
func (l *leftBrain) searchInSpace(query, space string) ([]Hit, error) {
	if space == "" {
		return l.Search(query)
	}
	ms := l.facts()
	if sp, ok := l.src.(spaceLeftSource); ok {
		ms = sp.ListFactsInSpace(space)
	}
	return l.searchFacts(query, ms), nil
}

// searchFacts 是左脑检索的匹配主体（Search / searchInSpace 共用）。
func (l *leftBrain) searchFacts(query string, ms []memory.Memory) []Hit {
	terms := splitQueryTerms(query)
	var out []Hit
	for _, m := range ms {
		name := displayName(m.Title, m.Name)
		text := factValue(m)
		for _, term := range terms {
			if matchAny(term, name, m.Name, text) {
				out = append(out, Hit{Brain: BrainLeft, Entity: name, Text: text})
				break
			}
		}
	}
	return out
}

func (l *leftBrain) facts() []memory.Memory {
	if l.src == nil {
		return nil
	}
	return l.src.ListFacts()
}

// factValue 描述优先、正文兜底（与主脑画像展示一致）。
func factValue(m memory.Memory) string {
	if strings.TrimSpace(m.Description) != "" {
		return m.Description
	}
	return m.Body
}

// officeFactLeftSource 用办公记忆（Hephaestus facts）实现 leftSource。
type officeFactLeftSource struct {
	store memory.Store
}

func (s *officeFactLeftSource) ListFacts() []memory.Memory {
	return s.store.List() // 零值 Store 为禁用 no-op，返回空
}

// ListFactsInSpace 实现 spaceLeftSource（S1.2 B）：facts 按空间谓词读取；
// 空 space = 全部（与 ListFacts 等价，旧行为）。
func (s *officeFactLeftSource) ListFactsInSpace(space string) []memory.Memory {
	return s.store.ListInSpace(space)
}
