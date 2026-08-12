package app

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/memory"
)

type fakeLeftSource struct{ facts []memory.Memory }

func (f *fakeLeftSource) ListFacts() []memory.Memory { return f.facts }

func TestLeftBrainSearchOfficeFacts(t *testing.T) {
	lb := &leftBrain{src: &fakeLeftSource{facts: []memory.Memory{
		{Name: "soil-bid", Title: "土壤修复标书", Description: "按保守报价原则编制"},
	}}}
	hits, err := lb.Search("保守报价")
	if err != nil || len(hits) == 0 {
		t.Fatalf("hits=%+v err=%v", hits, err)
	}
	if hits[0].Entity != "土壤修复标书" {
		t.Fatalf("entity = %q", hits[0].Entity)
	}
}
