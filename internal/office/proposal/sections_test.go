package proposal

import (
	"testing"
)

func testTree() []ProposalSection {
	return []ProposalSection{
		{ID: "c1", Title: "第1章", Level: 1, Index: 0, Children: []ProposalSection{
			{ID: "c1-1", Title: "1.1", Level: 2, Index: 1, Children: []ProposalSection{
				{ID: "c1-1-1", Title: "1.1.1", Level: 3, Index: 2},
			}},
			{ID: "c1-2", Title: "1.2", Level: 2, Index: 3},
		}},
		{ID: "c2", Title: "第2章", Level: 1, Index: 4},
	}
}

func TestFlattenSections(t *testing.T) {
	tree := testTree()
	flat := flattenSections(tree)
	if len(flat) != 5 {
		t.Fatalf("flatten 应返回 5 个节点，实际 %d", len(flat))
	}
	want := []string{"c1", "c1-1", "c1-1-1", "c1-2", "c2"}
	for i, f := range flat {
		if f.ID != want[i] {
			t.Errorf("顺序 %d: 期望 %s 实际 %s", i, want[i], f.ID)
		}
	}
	// 指针必须指向原切片元素，原地修改应生效
	flat[0].Title = "改名"
	if tree[0].Title != "改名" {
		t.Error("flatten 返回的指针未指向原数据")
	}
}

func TestRemoveSectionByID(t *testing.T) {
	tree := testTree()
	tree = removeSectionByID(tree, "c1-1")
	if len(tree[0].Children) != 1 || tree[0].Children[0].ID != "c1-2" {
		t.Fatalf("删除子章节失败: %+v", tree[0].Children)
	}
	if countSections(tree) != 3 {
		t.Fatalf("删除后应剩 3 个节点，实际 %d", countSections(tree))
	}
	// 删除整章
	tree = removeSectionByID(tree, "c1")
	if len(tree) != 1 || tree[0].ID != "c2" {
		t.Fatalf("删除整章失败: %+v", tree)
	}
}

func TestAddSectionThroughService(t *testing.T) {
	svc := NewService(t.TempDir(), nil)
	p, err := svc.Create("测试方案", "blank", "需求", "其他")
	if err != nil {
		t.Fatal(err)
	}
	// 新增顶级章节
	p, err = svc.AddSection(p.ID, "", "第一章")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Sections) != 1 || p.Sections[0].Title != "第一章" || p.Sections[0].Level != 1 {
		t.Fatalf("新增顶级章节失败: %+v", p.Sections)
	}
	parentID := p.Sections[0].ID
	// 新增子章节
	p, err = svc.AddSection(p.ID, parentID, "1.1")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Sections[0].Children) != 1 || p.Sections[0].Children[0].Level != 2 {
		t.Fatalf("新增子章节失败: %+v", p.Sections[0].Children)
	}
	childID := p.Sections[0].Children[0].ID
	// 重命名子章节
	p, err = svc.RenameSection(p.ID, childID, "1.1 改名")
	if err != nil {
		t.Fatal(err)
	}
	if p.Sections[0].Children[0].Title != "1.1 改名" {
		t.Fatalf("重命名失败: %+v", p.Sections[0].Children[0])
	}
	// 删除子章节
	p, err = svc.RemoveSection(p.ID, childID)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Sections[0].Children) != 0 || countSections(p.Sections) != 1 {
		t.Fatalf("删除子章节失败: %+v", p.Sections)
	}
}
