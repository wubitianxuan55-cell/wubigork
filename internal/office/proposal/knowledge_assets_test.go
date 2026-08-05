package proposal

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/knowledge"
)

func newTestKnowledgeStore(t *testing.T) *knowledge.Store {
	t.Helper()
	st, err := knowledge.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func TestEnsureSpecAssets_SeedsAndSearch(t *testing.T) {
	st := newTestKnowledgeStore(t)
	svc := &Service{}
	svc.SetKnowledgeStoreForTest(st)
	if err := svc.EnsureSpecAssets(); err != nil {
		t.Fatalf("EnsureSpecAssets: %v", err)
	}
	// 幂等
	if err := svc.EnsureSpecAssets(); err != nil {
		t.Fatalf("EnsureSpecAssets 二次: %v", err)
	}
	res := svc.SearchSpecs("砷 筛选值")
	if len(res) == 0 {
		t.Fatal("SearchSpecs 无结果")
	}
	found := false
	for _, r := range res {
		if containsAny(r.Title, "GB 36600") && containsAny(r.Body, "砷") {
			found = true
		}
	}
	if !found {
		t.Fatalf("未命中 GB 36600 砷条文: %+v", res)
	}
}
