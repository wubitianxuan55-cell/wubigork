package app

import (
	"testing"

	"github.com/gaea/gaea/internal/office/proposal"
)

type fakeLeftSource struct{ ps []proposal.Proposal }

func (f *fakeLeftSource) ListProposals() ([]proposal.Proposal, error) { return f.ps, nil }

func TestLeftBrainSearchProposals(t *testing.T) {
	lb := &leftBrain{src: &fakeLeftSource{ps: []proposal.Proposal{
		{ID: "p1", Title: "土壤修复标书", Category: "环保工程", Requirements: "按保守报价原则编制"},
	}}}
	hits, err := lb.Search("保守报价")
	if err != nil || len(hits) == 0 {
		t.Fatalf("hits=%+v err=%v", hits, err)
	}
	if hits[0].Entity != "土壤修复标书" {
		t.Fatalf("entity = %q", hits[0].Entity)
	}
}
