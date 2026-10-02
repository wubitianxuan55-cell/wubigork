package app

// AP1-06 反向证据与语义钉子：findOutlineNode 共享递归（含 Children 分支）+
// 各站点谓词语义（认不认分支/按什么字段认）不被收敛抹平。审计第 19 批线 1。
// 把共享递归的 Children 分支短路（只扫顶层）→ TestFindOutlineNodeByNumNested
// （既有）与本文件 TestFindOutlineNode_DeepChildrenRecursion 红。

import (
	"testing"

	"github.com/gaea/gaea/internal/types"
)

// nestedOutline 卷→章两层的分卷大纲（含同名章号的主线/分支节点对）。
func nestedOutline() []types.OutlineNode {
	return []types.OutlineNode{
		{ID: "vol1", OrderIndex: 0, Title: "第一卷", Children: []types.OutlineNode{
			{ID: "ch3", OrderIndex: 3, Branch: "", Title: "第三章", ChapterFile: "003.md"},
			{ID: "ch3a", OrderIndex: 3, Branch: "a", Title: "第三章分支"},
		}},
		{ID: "ch9", OrderIndex: 9, Branch: "", Title: "第九章"},
	}
}

// TestFindOutlineNode_DeepChildrenRecursion AP1-06 钉子：共享递归必须下探
// Children——短路 Children 分支（只扫顶层）时本用例红。
func TestFindOutlineNode_DeepChildrenRecursion(t *testing.T) {
	nodes := nestedOutline()
	got := findOutlineNode(nodes, func(n *types.OutlineNode) bool {
		return n.OrderIndex == 3 && n.Branch == ""
	})
	if got == nil || got.ID != "ch3" {
		t.Fatalf("卷下主线节点 ch3 应命中，得到 %+v", got)
	}
}

// TestFindOutlineNodeByNum_BranchNotMatchedEvenDeep AP1-06 钉子：主线查找的
// 分支语义在谓词里——深层同名章号的分支节点不得命中（改前行为：返回 nil，
// 分支节点由 ensureChapterNode 定号，主线意图注入不认）。
func TestFindOutlineNodeByNum_BranchNotMatchedEvenDeep(t *testing.T) {
	// 树里唯一 OrderIndex==3 的节点是分支章（深层），主线查找必须落空。
	nodes := []types.OutlineNode{
		{ID: "vol1", OrderIndex: 0, Title: "第一卷", Children: []types.OutlineNode{
			{ID: "ch3a", OrderIndex: 3, Branch: "a", Title: "第三章分支"},
		}},
	}
	if got := findOutlineNodeByNum(nodes, 3); got != nil {
		t.Fatalf("深层分支节点不得被主线查找命中，得到 %+v", got)
	}
	if got := findOutlineNodeByNum(nestedOutline(), 3); got == nil || got.ID != "ch3" {
		t.Fatalf("同章号主线优先：应命中 ch3，得到 %+v", got)
	}
}

// TestGateOutlineIssues_OrderFallbackMatchesBranchNode AP1-06 钉子：写前闸
// 第二段匹配（OrderIndex 兜底）**不看 Branch**——ChapterFile 缺位时深层分支
// 节点也认（改前 findByOrder 行为，收敛时保留不抹平）。
func TestGateOutlineIssues_OrderFallbackMatchesBranchNode(t *testing.T) {
	a := newGateEmptyApp()
	pm := newGateProject(t)
	a.setPM(pm)

	if err := pm.WriteOutlines(&types.OutlineFile{Nodes: []types.OutlineNode{
		{ID: "vol1", OrderIndex: 0, Title: "第一卷", Children: []types.OutlineNode{
			{ID: "ch3a", OrderIndex: 3, Branch: "a"}, // 无 ChapterFile、空标题摘要
		}},
	}}); err != nil {
		t.Fatalf("写大纲: %v", err)
	}

	issues := gateOutlineIssues(pm, 3)
	if len(issues) == 0 {
		t.Fatal("OrderIndex 兜底应命中深层分支节点（不看 Branch），得到空表")
	}
}

// TestGateOutlineIssues_ChapterFilePassFirst AP1-06 钉子：两段口径的先后——
// ChapterFile 前导章号命中的节点优先于 OrderIndex 兜底命中（即便后者在树上
// 更浅/更早）。
func TestGateOutlineIssues_ChapterFilePassFirst(t *testing.T) {
	a := newGateEmptyApp()
	pm := newGateProject(t)
	a.setPM(pm)

	if err := pm.WriteOutlines(&types.OutlineFile{Nodes: []types.OutlineNode{
		{ID: "ch3byOrder", OrderIndex: 3}, // OrderIndex 兜底候选（先出现在树上）
		{ID: "ch3byFile", OrderIndex: 7, ChapterFile: "003.md", Title: "第三章",
			Summary: "齐", KeyPoints: []string{"齐"}, Emotion: "齐"},
	}}); err != nil {
		t.Fatalf("写大纲: %v", err)
	}

	issues := gateOutlineIssues(pm, 3)
	if len(issues) != 0 {
		t.Fatalf("ChapterFile 命中的节点字段齐备，不应再兜底到缺字段节点：%+v", issues)
	}
}

// TestMarkOutlineDone_BranchParamRespected AP1-06 钉子：场景链的 mark 认调用方
// 传入的 branch——标分支章不得动主线节点，反之亦然（「Branch 可变」语义保留）。
func TestMarkOutlineDone_BranchParamRespected(t *testing.T) {
	a := newGateEmptyApp()
	pm := newGateProject(t)
	a.setPM(pm)
	of := &types.OutlineFile{Nodes: nestedOutline()}
	if err := pm.WriteOutlines(of); err != nil {
		t.Fatalf("写大纲: %v", err)
	}

	if err := a.markOutlineDone(pm, 3, "a"); err != nil {
		t.Fatalf("markOutlineDone(分支): %v", err)
	}
	of, err := pm.ReadOutlines()
	if err != nil {
		t.Fatalf("读大纲: %v", err)
	}
	vol := of.Nodes[0]
	if vol.ID != "vol1" || len(vol.Children) != 2 || vol.Children[0].ID != "ch3" || vol.Children[1].ID != "ch3a" {
		t.Fatalf("前置：大纲形状不符 vol1{ch3,ch3a}，得到 %+v", of.Nodes)
	}
	if vol.Children[0].Status == types.OutlineDone {
		t.Fatal("标分支章不得动主线节点")
	}
	if vol.Children[1].Status != types.OutlineDone {
		t.Fatalf("分支章应被标 done，得到 %v", vol.Children[1].Status)
	}

	if err := a.markOutlineDone(pm, 9, ""); err != nil {
		t.Fatalf("markOutlineDone(主线): %v", err)
	}
	of, err = pm.ReadOutlines()
	if err != nil {
		t.Fatalf("读大纲: %v", err)
	}
	if of.Nodes[1].ID != "ch9" {
		t.Fatalf("前置：Nodes[1] 应为 ch9，得到 %s", of.Nodes[1].ID)
	}
	if of.Nodes[1].Status != types.OutlineDone {
		t.Fatalf("主线章应被标 done，得到 %v", of.Nodes[1].Status)
	}
}

// TestSyncSceneRefs_MainLineOnly AP1-06 钉子：场景引用回写只认 Branch==""——
// 同章号分支节点不得收到 SceneRefs（改前行为，收敛时保留）。
func TestSyncSceneRefs_MainLineOnly(t *testing.T) {
	a := newGateEmptyApp()
	pm := newGateProject(t)
	a.setPM(pm)
	if err := pm.WriteOutlines(&types.OutlineFile{Nodes: nestedOutline()}); err != nil {
		t.Fatalf("写大纲: %v", err)
	}

	a.syncSceneRefs(pm, 3, []types.SceneMeta{{ID: "s1"}, {ID: "s2"}})
	of, err := pm.ReadOutlines()
	if err != nil {
		t.Fatalf("读大纲: %v", err)
	}
	vol := of.Nodes[0]
	main, branch := vol.Children[0], vol.Children[1]
	if main.ID != "ch3" {
		t.Fatalf("前置：Children[0] 应为主线节点，得到 %s", main.ID)
	}
	if len(main.SceneRefs) != 2 || main.SceneRefs[0] != "s1" || main.SceneRefs[1] != "s2" {
		t.Fatalf("主线节点应收到 SceneRefs [s1 s2]，得到 %v", main.SceneRefs)
	}
	if len(branch.SceneRefs) != 0 {
		t.Fatalf("分支节点不得收到 SceneRefs，得到 %v", branch.SceneRefs)
	}
}
