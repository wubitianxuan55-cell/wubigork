package app

import (
	"github.com/gaea/gaea/internal/whisper"
	whisperdb "github.com/gaea/gaea/internal/whisper/db/repos"
)

// rightBrain 右脑：轻语 hermes.db 记忆事实（统一读写，不迁移数据）。
type rightBrain struct {
	dataRoot string
}

func (r *rightBrain) Read(entity string) ([]Fact, error) {
	var out []Fact
	for _, f := range whisperdb.LoadFactsFromDB(r.dataRoot) {
		if f.Subject == entity {
			out = append(out, Fact{Brain: BrainRight, Entity: f.Subject, Attribute: f.Subcategory, Value: f.Summary})
		}
	}
	return out, nil
}

func (r *rightBrain) Write(entity, attribute, value string) error {
	return whisperdb.InsertFact(r.dataRoot, whisper.MemoryFact{
		Domain:      "brain",
		Subcategory: attribute,
		Subject:     entity,
		Summary:     value,
		Weight:      1,
		Confidence:  1,
		Status:      "active",
		FactLayer:   "brain",
	})
}

func (r *rightBrain) Search(query string) ([]Hit, error) {
	terms := splitQueryTerms(query)
	var out []Hit
	for _, f := range whisperdb.LoadFactsFromDB(r.dataRoot) {
		for _, term := range terms {
			if matchAny(term, f.Subject, f.Summary, f.Subcategory) {
				out = append(out, Hit{Brain: BrainRight, Entity: f.Subject, Text: f.Summary})
				break
			}
		}
	}
	return out, nil
}

// splitQueryTerms 按空白切分查询词。
func splitQueryTerms(query string) []string {
	var terms []string
	start := -1
	for i, r := range query {
		if r == ' ' || r == '\t' || r == '\n' {
			if start >= 0 {
				terms = append(terms, query[start:i])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		terms = append(terms, query[start:])
	}
	return terms
}
