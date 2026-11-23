package app

import (
	"github.com/gaea/gaea/internal/gaea/memory"
)

// mainBrain 主脑：全局画像（profile）。知识库 2026-11-23 退役后不再有 kb 源。
type mainBrain struct {
	profile *memory.ProfileStore
}

func (m *mainBrain) Read(entity string) ([]Fact, error) {
	var out []Fact
	if m.profile != nil {
		if mem, ok := m.profile.Get(entity); ok {
			out = append(out, Fact{Brain: BrainMain, Entity: mem.Name, Attribute: "profile", Value: mem.Body})
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
	return out, nil
}
