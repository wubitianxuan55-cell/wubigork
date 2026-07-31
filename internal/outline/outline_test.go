package outline

import (
	"testing"

	"github.com/gaea/gaea/internal/types"
)

// ── FindNodeByID ──────────────────────────────────────────

func TestFindNodeByID_RootMatch(t *testing.T) {
	node := &types.OutlineNode{ID: "root", Title: "Root"}
	if got := FindNodeByID(node, "root"); got == nil || got.ID != "root" {
		t.Fatal("应找到根节点自身")
	}
}

func TestFindNodeByID_DeepChild(t *testing.T) {
	tree := &types.OutlineNode{
		ID: "A",
		Children: []types.OutlineNode{
			{ID: "B", Children: []types.OutlineNode{
				{ID: "C"},
			}},
		},
	}
	if got := FindNodeByID(tree, "C"); got == nil || got.ID != "C" {
		t.Fatal("应找到深度嵌套子节点")
	}
}

func TestFindNodeByID_NotFound(t *testing.T) {
	tree := &types.OutlineNode{
		ID: "A",
		Children: []types.OutlineNode{
			{ID: "B"},
		},
	}
	if got := FindNodeByID(tree, "Z"); got != nil {
		t.Fatalf("不存在的节点应返回 nil，却返回了 %v", got)
	}
}

func TestFindNodeByID_EmptyChildren(t *testing.T) {
	node := &types.OutlineNode{ID: "leaf"}
	if got := FindNodeByID(node, "leaf"); got == nil || got.ID != "leaf" {
		t.Fatal("叶子节点应能找到自身")
	}
	if got := FindNodeByID(node, "other"); got != nil {
		t.Fatal("叶子节点不应找到其他 ID")
	}
}

// ── sortOutlineNodes ──────────────────────────────────────

func TestSortOutlineNodes_AlreadySorted(t *testing.T) {
	nodes := []types.OutlineNode{
		{ID: "a", OrderIndex: 1},
		{ID: "b", OrderIndex: 2},
		{ID: "c", OrderIndex: 3},
	}
	sortOutlineNodes(nodes)
	for i, n := range nodes {
		if n.OrderIndex != i+1 {
			t.Fatalf("已排序列表不应被破坏: pos %d, order %d", i, n.OrderIndex)
		}
	}
}

func TestSortOutlineNodes_Unsorted(t *testing.T) {
	nodes := []types.OutlineNode{
		{ID: "c", OrderIndex: 3},
		{ID: "a", OrderIndex: 1},
		{ID: "b", OrderIndex: 2},
	}
	sortOutlineNodes(nodes)
	if nodes[0].ID != "a" || nodes[1].ID != "b" || nodes[2].ID != "c" {
		t.Fatalf("排序失败: %v %v %v", nodes[0].ID, nodes[1].ID, nodes[2].ID)
	}
}

func TestSortOutlineNodes_NestedChildren(t *testing.T) {
	nodes := []types.OutlineNode{
		{
			ID: "parent", OrderIndex: 1,
			Children: []types.OutlineNode{
				{ID: "c2", OrderIndex: 2},
				{ID: "c1", OrderIndex: 1},
			},
		},
	}
	sortOutlineNodes(nodes)
	children := nodes[0].Children
	if children[0].ID != "c1" || children[1].ID != "c2" {
		t.Fatalf("子节点排序失败: %v %v", children[0].ID, children[1].ID)
	}
}

// ── reindexOutlineNodes ────────────────────────────────────

func TestReindexOutlineNodes_NoParent(t *testing.T) {
	nodes := []types.OutlineNode{
		{ID: "x", OrderIndex: 99},
		{ID: "y", OrderIndex: 0},
	}
	reindexOutlineNodes(nodes, "")
	if nodes[0].OrderIndex != 1 || nodes[1].OrderIndex != 2 {
		t.Fatalf("重索引失败: [0]=%d [1]=%d", nodes[0].OrderIndex, nodes[1].OrderIndex)
	}
}

func TestReindexOutlineNodes_WithParent(t *testing.T) {
	nodes := []types.OutlineNode{
		{ID: "child", OrderIndex: 5, ParentID: "old"},
	}
	reindexOutlineNodes(nodes, "new-parent")
	if nodes[0].OrderIndex != 1 {
		t.Fatalf("重索引后 order 应为 1: %d", nodes[0].OrderIndex)
	}
	if nodes[0].ParentID != "new-parent" {
		t.Fatalf("ParentID 应更新为 new-parent: %q", nodes[0].ParentID)
	}
}

// ── updateNodeInTree ──────────────────────────────────────

func TestUpdateNodeInTree_RootLevel(t *testing.T) {
	nodes := []types.OutlineNode{
		{ID: "A", Title: "旧标题", OrderIndex: 1, Children: []types.OutlineNode{{ID: "A1"}}},
		{ID: "B", Title: "B 标题"},
	}
	updated := updateNodeInTree(nodes, types.OutlineNode{ID: "A", Title: "新标题", OrderIndex: 10})
	if !updated {
		t.Fatal("应返回 true")
	}
	if nodes[0].Title != "新标题" {
		t.Fatalf("标题未更新: %q", nodes[0].Title)
	}
	if nodes[0].OrderIndex != 10 {
		t.Fatalf("OrderIndex 未更新: %d", nodes[0].OrderIndex)
	}
	if len(nodes[0].Children) != 1 || nodes[0].Children[0].ID != "A1" {
		t.Fatal("子节点应被保留")
	}
}

func TestUpdateNodeInTree_Nested(t *testing.T) {
	nodes := []types.OutlineNode{
		{ID: "X", Children: []types.OutlineNode{
			{ID: "Y", Title: "旧 Y"},
		}},
	}
	updated := updateNodeInTree(nodes, types.OutlineNode{ID: "Y", Title: "新 Y"})
	if !updated {
		t.Fatal("应找到嵌套节点")
	}
	if nodes[0].Children[0].Title != "新 Y" {
		t.Fatalf("嵌套节点标题未更新: %q", nodes[0].Children[0].Title)
	}
}

func TestUpdateNodeInTree_NotFound(t *testing.T) {
	nodes := []types.OutlineNode{
		{ID: "only"},
	}
	if updateNodeInTree(nodes, types.OutlineNode{ID: "ghost"}) {
		t.Fatal("不存在的节点应返回 false")
	}
}

// ── addChildToNode ────────────────────────────────────────

func TestAddChildToNode_Direct(t *testing.T) {
	parent := &types.OutlineNode{ID: "root"}
	ok := addChildToNode(parent, "root", types.OutlineNode{ID: "new-child", Title: "子"})
	if !ok {
		t.Fatal("addChildToNode 应返回 true")
	}
	if len(parent.Children) != 1 || parent.Children[0].ID != "new-child" {
		t.Fatalf("子节点添加失败: %+v", parent.Children)
	}
}

func TestAddChildToNode_Nested(t *testing.T) {
	root := &types.OutlineNode{
		ID: "root",
		Children: []types.OutlineNode{
			{ID: "mid", Children: []types.OutlineNode{
				{ID: "leaf"},
			}},
		},
	}
	ok := addChildToNode(root, "mid", types.OutlineNode{ID: "sibling"})
	if !ok {
		t.Fatal("addChildToNode 应找到嵌套父节点")
	}
	mid := &root.Children[0]
	if len(mid.Children) != 2 || mid.Children[1].ID != "sibling" {
		t.Fatalf("嵌套添加失败: %+v", mid.Children)
	}
}

func TestAddChildToNode_ParentNotFound(t *testing.T) {
	root := &types.OutlineNode{ID: "root"}
	if addChildToNode(root, "no-such", types.OutlineNode{ID: "orphan"}) {
		t.Fatal("不存在的父节点应返回 false")
	}
}

// ── removeNodeFromList ─────────────────────────────────────

func TestRemoveNodeFromList_Leaf(t *testing.T) {
	nodes := []types.OutlineNode{
		{ID: "keep"},
		{ID: "remove"},
		{ID: "also-keep"},
	}
	result := removeNodeFromList(nodes, "remove")
	if len(result) != 2 {
		t.Fatalf("应剩余 2 节点: got %d", len(result))
	}
	if result[0].ID != "keep" || result[1].ID != "also-keep" {
		t.Fatalf("顺序错: %+v", result)
	}
}

func TestRemoveNodeFromList_RootWithChildren(t *testing.T) {
	nodes := []types.OutlineNode{
		{ID: "remove-me", Children: []types.OutlineNode{
			{ID: "child"},
		}},
		{ID: "sibling"},
	}
	result := removeNodeFromList(nodes, "remove-me")
	if len(result) != 1 || result[0].ID != "sibling" {
		t.Fatalf("移除失败: got %d nodes", len(result))
	}
}

func TestRemoveNodeFromList_NestedChild(t *testing.T) {
	nodes := []types.OutlineNode{
		{ID: "parent", Children: []types.OutlineNode{
			{ID: "remove-child"},
			{ID: "keep-child"},
		}},
	}
	result := removeNodeFromList(nodes, "remove-child")
	children := result[0].Children
	if len(children) != 1 || children[0].ID != "keep-child" {
		t.Fatalf("嵌套移除失败: %+v", children)
	}
}

func TestRemoveNodeFromList_NotFound(t *testing.T) {
	nodes := []types.OutlineNode{{ID: "only"}}
	result := removeNodeFromList(nodes, "ghost")
	if len(result) != 1 || result[0].ID != "only" {
		t.Fatalf("未找到时应保持原列表: got %d", len(result))
	}
}
