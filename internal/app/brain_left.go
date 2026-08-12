package app

import (
	"strings"

	"github.com/gaea/gaea/internal/gaea/memory"
)

// leftSource 左脑数据源（办公记忆 facts；测试可注入 fake）。
type leftSource interface {
	ListFacts() []memory.Memory
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
	terms := splitQueryTerms(query)
	var out []Hit
	for _, m := range l.facts() {
		name := displayName(m.Title, m.Name)
		text := factValue(m)
		for _, term := range terms {
			if matchAny(term, name, m.Name, text) {
				out = append(out, Hit{Brain: BrainLeft, Entity: name, Text: text})
				break
			}
		}
	}
	return out, nil
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
