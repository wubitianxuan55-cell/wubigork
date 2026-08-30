package costref

// v4.8 成本知识图谱组图器（BuildGraph）表驱动测试：
//   - tree 聚合正确（分类子树金额/条目数 + 项目节点，无明细展开）；
//   - entry 展开与 limit 截断（节点/边硬上限，Truncated 置位）；
//   - EntryName 精确匹配优先于标题归一化（matchedBy 判定）；
//   - 无匹配不出错边（边两端节点必须在图中，无悬挂边）。

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/costinquiry"
	"github.com/gaea/gaea/internal/gaea/costproject"
)

// 测试夹具：两个案例项目 + 一条手动估价明细；三条成本条目（含同标题归一化冲突）；
// 两条询价；一条复盘笔记；一棵两级分类树。
func graphFixture() (
	[]costproject.ProjectSummary,
	map[string][]costproject.Item,
	[]cost.Summary,
	[]cost.CategoryView,
	[]costinquiry.Record,
	[]Note,
) {
	projects := []costproject.ProjectSummary{
		{Project: costproject.Project{ID: "p2", Name: "厂房 B", Status: "已沉淀"}, ItemCount: 1, Total: 600, VersionCount: 2},
		{Project: costproject.Project{ID: "p1", Name: "厂房 A", Status: "已保存版本"}, ItemCount: 2, Total: 970, VersionCount: 1},
	}
	itemsByProject := map[string][]costproject.Item{
		"p1": {
			{ID: 11, ProjectID: "p1", Name: "c30-concrete", Title: "C30 商品混凝土（泵送）", CategoryPath: "综合单价/土建", Unit: "m³", Quantity: 1, Price: 480, Amount: 480, EntryName: "c30"},
			{ID: 12, ProjectID: "p1", Name: "rebar", Title: "HRB400 钢筋", CategoryPath: "综合单价/土建", Unit: "t", Quantity: 2, Price: 245, Amount: 490, EntryName: "rebar"},
		},
		"p2": {
			// EntryName 为空 → 退化为标题归一化匹配（去括号：C30 商品混凝土（泵送）→ c30商品混凝土）
			{ID: 21, ProjectID: "p2", Name: "c30-2", Title: "C30 商品混凝土（防冻）", CategoryPath: "综合单价/土建", Unit: "m³", Quantity: 1, Price: 600, Amount: 600},
		},
	}
	entries := []cost.Summary{
		{Name: "rebar", Title: "HRB400 钢筋", CategoryPath: "综合单价/土建", Unit: "t", Price: 250, Status: "现行"},
		{Name: "c30", Title: "C30 商品混凝土（泵送）", CategoryPath: "综合单价/土建", Unit: "m³", Price: 480, Status: "现行"},
		{Name: "excavator", Title: "挖掘机台班", CategoryPath: "综合单价/机械", Unit: "台班", Price: 1500, Status: "现行"},
	}
	categories := []cost.CategoryView{
		{ID: 1, Name: "综合单价", Children: []*cost.CategoryView{
			{ID: 2, ParentID: 1, Name: "土建"},
			{ID: 3, ParentID: 1, Name: "机械"},
		}},
	}
	inquiries := []costinquiry.Record{
		{ID: 1, Title: "C30 商品混凝土", Unit: "m³", Price: 470, Source: "信息价", PriceDate: "2026-08"},
		{ID: 2, Title: "HRB400 钢筋", Unit: "t", Price: 2480, Source: "供应商比价"},
	}
	notes := []Note{
		{ID: 1, Title: "C30 泵送价区间", Category: "土建", Confidence: "高", Status: "已确认"},
	}
	return projects, itemsByProject, entries, categories, inquiries, notes
}

func nodeByID(t *testing.T, g CostGraphView, id string) GraphNode {
	t.Helper()
	for _, n := range g.Nodes {
		if n.ID == id {
			return n
		}
	}
	t.Fatalf("节点 %s 不存在（nodes=%+v）", id, g.Nodes)
	return GraphNode{}
}

func hasNode(g CostGraphView, id string) bool {
	for _, n := range g.Nodes {
		if n.ID == id {
			return true
		}
	}
	return false
}

func edgeOfType(g CostGraphView, typ, source, target string) *GraphEdge {
	for i := range g.Edges {
		e := &g.Edges[i]
		if e.Type == typ && e.Source == source && e.Target == target {
			return e
		}
	}
	return nil
}

// tree 聚合：分类子树金额/条目数前缀累加正确 + 项目节点；无明细/条目节点展开。
func TestBuildGraphTreeAggregation(t *testing.T) {
	projects, itemsByProject, entries, categories, inquiries, notes := graphFixture()
	g := BuildGraph(projects, itemsByProject, entries, categories, inquiries, notes, "tree", "", 0)

	if g.Stats.Truncated {
		t.Fatal("tree 规模内不应截断")
	}
	// 分类节点：根 + 两个子分类（未展开条目）。
	root := nodeByID(t, g, "cat:综合单价")
	// 子树合计 = 三条条目 480+250+1500 = 2230，条目数 3
	if root.Val != 2230 {
		t.Fatalf("根分类 Val = %v, want 2230", root.Val)
	}
	if root.Desc != "3 条" {
		t.Fatalf("根分类 Desc = %q, want 3 条", root.Desc)
	}
	civil := nodeByID(t, g, "cat:综合单价/土建")
	if civil.Val != 730 || civil.Desc != "2 条" { // 480+250
		t.Fatalf("土建聚合 = (%v, %q), want (730, 2 条)", civil.Val, civil.Desc)
	}
	machine := nodeByID(t, g, "cat:综合单价/机械")
	if machine.Val != 1500 || machine.Desc != "1 条" {
		t.Fatalf("机械聚合 = (%v, %q), want (1500, 1 条)", machine.Val, machine.Desc)
	}
	// 项目节点（Val=合计），且 tree 不展开明细/条目。
	proj := nodeByID(t, g, "proj:p1")
	if proj.Name != "厂房 A" || proj.Val != 970 {
		t.Fatalf("项目节点 = %+v", proj)
	}
	for _, n := range g.Nodes {
		if n.Type == GraphNodeItem || n.Type == GraphNodeEntry {
			t.Fatalf("tree 不应展开明细/条目节点: %+v", n)
		}
	}
	if g.Stats.EdgeCount != 0 {
		t.Fatalf("tree 无边，got %d", g.Stats.EdgeCount)
	}
	if g.Stats.NodeCount != len(g.Nodes) || g.Stats.EdgeCount != len(g.Edges) {
		t.Fatal("Stats 计数与实际不符")
	}
	if g.Stats.CountsByType[GraphNodeCategory] != 3 || g.Stats.CountsByType[GraphNodeProject] != 2 {
		t.Fatalf("CountsByType = %+v", g.Stats.CountsByType)
	}
}

// entry 项目展开：contains/references/benchmarks/suggests/notes 边齐备。
func TestBuildGraphEntryProjectExpansion(t *testing.T) {
	projects, itemsByProject, entries, categories, inquiries, notes := graphFixture()
	g := BuildGraph(projects, itemsByProject, entries, categories, inquiries, notes, "entry", "p1", 0)

	// 项目 + 两条明细 + 两条被引用条目 + 指标 + 询价 + 笔记。
	if nodeByID(t, g, "proj:p1").Type != GraphNodeProject {
		t.Fatal("缺项目中心节点")
	}
	if nodeByID(t, g, "item:p1:c30-concrete").Type != GraphNodeItem {
		t.Fatal("缺明细节点")
	}
	// contains：项目→明细
	if edgeOfType(g, GraphEdgeContains, "proj:p1", "item:p1:c30-concrete") == nil {
		t.Fatal("缺 contains 边")
	}
	// references：明细→条目（EntryName 精确 → matchedBy=entry_name）
	ref := edgeOfType(g, GraphEdgeReferences, "item:p1:c30-concrete", "entry:c30")
	if ref == nil || ref.Meta["matchedBy"] != "entry_name" {
		t.Fatalf("references 边错误: %+v", ref)
	}
	// benchmarks：明细→指标（指标由参考池实时聚合：C30 混凝土（泵送）+ HRB400 钢筋）
	bench := edgeOfType(g, GraphEdgeBenchmarks, "item:p1:c30-concrete", "ind:C30 商品混凝土（泵送）")
	if bench == nil {
		t.Fatalf("缺 benchmarks 边，指标节点: %+v", g.Stats.CountsByType)
	}
	// suggests：询价（标题归一化命中 C30 商品混凝土（泵送）→ C30 商品混凝土）→条目
	sug := edgeOfType(g, GraphEdgeSuggests, "inq:1", "entry:c30")
	if sug == nil {
		t.Fatal("缺 suggests 边")
	}
	// notes：笔记（Category=土建）→分类节点
	noteEdge := edgeOfType(g, GraphEdgeNotes, "note:1", "cat:综合单价/土建")
	if noteEdge == nil {
		t.Fatal("缺 notes 边（笔记应挂到条目所属分类）")
	}
	// p2 的明细不出现（focus=p1 只展开本项目）。
	for _, n := range g.Nodes {
		if n.Type == GraphNodeItem && n.ID == "item:p2:c30-2" {
			t.Fatal("focus=p1 不应出现 p2 明细")
		}
	}
}

// EntryName 精确优先：EntryName 命中时 matchedBy=entry_name；EntryName 未命中
// 时退化标题归一化（去括号）且 matchedBy=title。
func TestBuildGraphEntryNameExactFirst(t *testing.T) {
	projects, itemsByProject, entries, categories, inquiries, notes := graphFixture()
	g := BuildGraph(projects, itemsByProject, entries, categories, inquiries, notes, "entry", "p2", 0)

	// p2 明细 EntryName 为空：标题「C30 商品混凝土（防冻）」归一化去括号后命中
	// 「C30 商品混凝土（泵送）」→ matchedBy=title。
	ref := edgeOfType(g, GraphEdgeReferences, "item:p2:c30-2", "entry:c30")
	if ref == nil || ref.Meta["matchedBy"] != "title" {
		t.Fatalf("p2 应走标题归一化匹配: %+v", ref)
	}
	// 反例：若 EntryName 指向不存在条目，仍可由标题兜底（不出悬挂边）。
	projects2 := []costproject.ProjectSummary{
		{Project: costproject.Project{ID: "p3", Name: "厂房 C"}, ItemCount: 1, VersionCount: 1},
	}
	items3 := map[string][]costproject.Item{
		"p3": {{ID: 31, ProjectID: "p3", Title: "HRB400 钢筋", EntryName: "ghost-entry", Quantity: 1, Price: 240, Amount: 240}},
	}
	g2 := BuildGraph(projects2, items3, entries, categories, inquiries, notes, "entry", "p3", 0)
	ref2 := edgeOfType(g2, GraphEdgeReferences, "item:p3:31", "entry:rebar")
	if ref2 == nil || ref2.Meta["matchedBy"] != "title" {
		t.Fatalf("EntryName 未命中应退化标题匹配: %+v", ref2)
	}
}

// entry 分类展开：子树条目 belongs_to + 引用明细（跨项目）+ 询价。
func TestBuildGraphEntryCategoryExpansion(t *testing.T) {
	projects, itemsByProject, entries, categories, inquiries, notes := graphFixture()
	g := BuildGraph(projects, itemsByProject, entries, categories, inquiries, notes, "entry", "综合单价/机械", 0)

	// 机械子树：1 条条目（挖掘机），无明细引用，无询价。
	if nodeByID(t, g, "cat:综合单价/机械").Type != GraphNodeCategory {
		t.Fatal("缺 focus 分类节点")
	}
	if nodeByID(t, g, "entry:excavator").Type != GraphNodeEntry {
		t.Fatal("缺子树条目节点")
	}
	if edgeOfType(g, GraphEdgeBelongsTo, "cat:综合单价/机械", "entry:excavator") == nil {
		t.Fatal("缺 belongs_to 边")
	}
	if hasNode(g, "entry:c30") {
		t.Fatal("土建条目不应出现在机械子树（不跨界拉入）")
	}
	// 土建子树：条目 + 跨项目明细引用 + 询价。
	g2 := BuildGraph(projects, itemsByProject, entries, categories, inquiries, notes, "entry", "综合单价/土建", 0)
	if edgeOfType(g2, GraphEdgeBelongsTo, "cat:综合单价/土建", "entry:c30") == nil {
		t.Fatal("土建缺 belongs_to 边")
	}
	if edgeOfType(g2, GraphEdgeContains, "proj:p1", "item:p1:c30-concrete") == nil ||
		edgeOfType(g2, GraphEdgeContains, "proj:p2", "item:p2:c30-2") == nil {
		t.Fatal("土建应展开引用明细（含 p2）")
	}
	if edgeOfType(g2, GraphEdgeSuggests, "inq:1", "entry:c30") == nil {
		t.Fatal("土建缺询价 suggests 边")
	}
}

// limit 截断：节点硬上限生效并置 Truncated；无匹配条目不产生错误边。
func TestBuildGraphLimitTruncation(t *testing.T) {
	projects, itemsByProject, entries, categories, inquiries, notes := graphFixture()
	// limit=4：focus=p1 → 项目+2 明细+1 条目 后触顶。
	g := BuildGraph(projects, itemsByProject, entries, categories, inquiries, notes, "entry", "p1", 4)
	if !g.Stats.Truncated {
		t.Fatal("limit=4 应置 Truncated")
	}
	if g.Stats.NodeCount != 4 || g.Stats.NodeCount != len(g.Nodes) {
		t.Fatalf("节点数应恰为 4, got %d", g.Stats.NodeCount)
	}
	// 每条边两端节点都必须在图中（截断不产生悬挂边）。
	ids := map[string]bool{}
	for _, n := range g.Nodes {
		ids[n.ID] = true
	}
	for _, e := range g.Edges {
		if !ids[e.Source] || !ids[e.Target] {
			t.Fatalf("悬挂边: %+v", e)
		}
	}
	// 未知 focus（既非项目也非分类）→ 无子树条目，只有 focus 节点本身，不出错。
	g2 := BuildGraph(projects, itemsByProject, entries, categories, inquiries, notes, "entry", "不存在的分类", 100)
	for _, n := range g2.Nodes {
		if n.Type != GraphNodeCategory {
			t.Fatalf("未知 focus 只应有 focus 分类节点, got %+v", n)
		}
	}
	// 无匹配不出错边：空输入 → 空图不 panic。
	g3 := BuildGraph(nil, nil, nil, nil, nil, nil, "entry", "", 0)
	if len(g3.Nodes) != 0 || len(g3.Edges) != 0 {
		t.Fatalf("空输入应得空图: %+v", g3)
	}
	// limit 夹取：>600 → 600。
	g4 := BuildGraph(projects, itemsByProject, entries, categories, inquiries, notes, "tree", "", 9999)
	if g4.Stats.Truncated {
		t.Fatal("夹取到 600 后小数据不应截断")
	}
}

// 边去重：同一 (source,target,type) 只出现一次。
func TestBuildGraphEdgeDedup(t *testing.T) {
	projects, itemsByProject, entries, categories, inquiries, notes := graphFixture()
	g := BuildGraph(projects, itemsByProject, entries, categories, inquiries, notes, "entry", "p1", 0)
	seen := map[string]bool{}
	for _, e := range g.Edges {
		k := e.Type + "\x00" + e.Source + "\x00" + e.Target
		if seen[k] {
			t.Fatalf("重复边: %+v", e)
		}
		seen[k] = true
	}
}
