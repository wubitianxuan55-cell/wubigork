package app

import (
	"github.com/gaea/gaea/internal/gaea/knowledge"
	"github.com/gaea/gaea/internal/gaea/memory"
)

// mainBrain 主脑：全局画像（profile）+ 知识库（knowledge）。
type mainBrain struct {
	profile *memory.ProfileStore
	kb      *knowledge.Store
}

func (m *mainBrain) Read(entity string) ([]Fact, error) {
	var out []Fact
	if m.profile != nil {
		if mem, ok := m.profile.Get(entity); ok {
			out = append(out, Fact{Brain: BrainMain, Entity: mem.Name, Attribute: "profile", Value: mem.Body})
		}
	}
	if m.kb != nil {
		if e, err := m.kb.Get(entity); err == nil && e != nil {
			out = append(out, Fact{Brain: BrainMain, Entity: e.Name, Attribute: "knowledge", Value: e.Body})
		}
	}
	return out, nil
}

func (m *mainBrain) Write(entity, attribute, value string) error {
	if m.profile == nil {
		return nil
	}
	return m.profile.Save(memory.Memory{
		Name: entity, Title: attribute, Description: value, Type: "user", Body: value,
	})
}

func (m *mainBrain) Search(query string) ([]Hit, error) {
	var out []Hit
	if m.profile != nil {
		for _, mem := range m.profile.All() {
			if matchAny(query, mem.Name, mem.Title, mem.Description, mem.Body) {
				out = append(out, Hit{Brain: BrainMain, Entity: mem.Name, Text: mem.Body})
			}
		}
	}
	if m.kb != nil {
		for _, s := range m.kb.List() {
			if matchAny(query, s.Name, s.Title, s.Category) {
				out = append(out, Hit{Brain: BrainMain, Entity: s.Name, Text: s.Title})
			}
		}
	}
	return out, nil
}
