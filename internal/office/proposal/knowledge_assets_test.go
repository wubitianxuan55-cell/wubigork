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

func TestAssetsCRUD(t *testing.T) {
	st := newTestKnowledgeStore(t)
	svc := &Service{}
	svc.SetKnowledgeStoreForTest(st)
	if err := svc.AddAsset("某公司土壤修复业绩", []string{"业绩"}, "2023 年完成某化工厂污染场地修复 5 万 m³"); err != nil {
		t.Fatalf("AddAsset: %v", err)
	}
	if err := svc.AddAsset("项目经理张三", []string{"人员"}, "高级工程师，10 年修复经验"); err != nil {
		t.Fatal(err)
	}
	all := svc.ListAssets()
	if len(all) != 2 {
		t.Fatalf("ListAssets = %d, want 2", len(all))
	}
	hit := svc.SearchAssets("修复", "")
	if len(hit) == 0 {
		t.Fatal("SearchAssets 无结果")
	}
	byTag := svc.SearchAssets("", "人员")
	if len(byTag) != 1 || byTag[0].Title != "项目经理张三" {
		t.Fatalf("按标签检索异常: %+v", byTag)
	}
	if err := svc.RemoveAsset(all[0].Name); err != nil {
		t.Fatal(err)
	}
	if len(svc.ListAssets()) != 1 {
		t.Fatalf("删除后数量异常")
	}
}

func TestArchiveProposal_And_Reference(t *testing.T) {
	st := newTestKnowledgeStore(t)
	s := newServiceAt(t, t.TempDir(), &mockAI{def: "mock"})
	s.SetKnowledgeStoreForTest(st)
	proj, _ := s.store.EnsureDefaultProject()
	p, _ := s.store.Create("某修复投标方案", "soil-remediation-bid", "需求", "环保工程", proj.ID, []ProposalSection{
		{Title: "第一章", Level: 1, Content: "项目概况正文"},
	})
	name, err := s.ArchiveProposal(p.ID)
	if err != nil {
		t.Fatalf("ArchiveProposal: %v", err)
	}
	if name == "" {
		t.Fatal("归档名为空")
	}
	refs := s.SearchLegacyProposals("修复")
	if len(refs) == 0 {
		t.Fatal("历史方案检索无结果")
	}
	ref := s.legacyRefFor("soil-remediation-bid", "环保工程")
	if ref == "" {
		t.Fatal("同类型参考为空")
	}
	if !containsAny(ref, "某修复投标方案", "项目概况正文") {
		t.Fatalf("参考内容异常: %s", ref)
	}
}
