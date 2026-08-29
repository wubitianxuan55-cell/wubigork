// Package whisper — memory_graph_subgraph_test.go
// v4.3b 子图召回 QuerySubgraph 测试

package whisper

import "testing"

// ─── 辅助 ──────────────────────────────────────────────────────

// findSubgraphNode 按 ID 查找子图节点,找不到即失败
func findSubgraphNode(t *testing.T, sg Subgraph, id string) GraphNode {
	t.Helper()
	for _, n := range sg.Nodes {
		if n.ID == id {
			return n
		}
	}
	t.Fatalf("子图中未找到节点 %q(全部节点: %+v)", id, sg.Nodes)
	return GraphNode{}
}

// findSubgraphEdge 按 From-Type-To 查找子图边,找不到即失败
func findSubgraphEdge(t *testing.T, sg Subgraph, from, typ, to string) GraphEdge {
	t.Helper()
	for _, e := range sg.Edges {
		if e.From == from && e.Type == typ && e.To == to {
			return e
		}
	}
	t.Fatalf("子图中未找到边 %q -%q→ %q(全部边: %+v)", from, typ, to, sg.Edges)
	return GraphEdge{}
}

// countSubgraphNode 统计 ID 匹配的节点数
func countSubgraphNode(sg Subgraph, id string) int {
	n := 0
	for _, x := range sg.Nodes {
		if x.ID == id {
			n++
		}
	}
	return n
}

// countSubgraphEdge 统计 From-Type-To 匹配的边数
func countSubgraphEdge(sg Subgraph, from, typ, to string) int {
	n := 0
	for _, e := range sg.Edges {
		if e.From == from && e.Type == typ && e.To == to {
			n++
		}
	}
	return n
}

// assertEdgesEndpointsInNodes 校验所有边的两端都在节点集合内
func assertEdgesEndpointsInNodes(t *testing.T, sg Subgraph) {
	t.Helper()
	in := map[string]bool{}
	for _, n := range sg.Nodes {
		in[n.ID] = true
	}
	for _, e := range sg.Edges {
		if !in[e.From] {
			t.Errorf("边 %+v 的 From %q 不在节点集合内", e, e.From)
		}
		if !in[e.To] {
			t.Errorf("边 %+v 的 To %q 不在节点集合内", e, e.To)
		}
	}
}

// ─── 用例 ──────────────────────────────────────────────────────

// TestQuerySubgraph_OneHop 一跳邻接:仅中心 + 直接邻居,两跳边/节点不出现
func TestQuerySubgraph_OneHop(t *testing.T) {
	kg := NewKnowledgeGraph()
	kg.Add("用户", "喜欢", "咖啡", 0.9, nil)
	kg.Add("用户", "住在", "北京", 0.8, nil)
	kg.Add("咖啡", "产于", "云南", 0.7, nil) // 两跳外,hops=1 不应出现

	sg := kg.QuerySubgraph("用户", 1)

	if len(sg.Nodes) != 3 {
		t.Fatalf("一跳子图应含 3 个节点(用户/咖啡/北京),实际 %d: %+v", len(sg.Nodes), sg.Nodes)
	}
	if len(sg.Edges) != 2 {
		t.Fatalf("一跳子图应含 2 条边,实际 %d: %+v", len(sg.Edges), sg.Edges)
	}

	center := findSubgraphNode(t, sg, "用户")
	if center.Name != "用户" || center.Weight <= 0 {
		t.Errorf("中心节点 Name/Weight 异常: %+v", center)
	}

	// 边权重沿用三元组 confidence
	if e := findSubgraphEdge(t, sg, "用户", "喜欢", "咖啡"); e.Weight != 0.9 {
		t.Errorf("边权重应为 confidence 0.9,实际 %v", e.Weight)
	}
	if e := findSubgraphEdge(t, sg, "用户", "住在", "北京"); e.Weight != 0.8 {
		t.Errorf("边权重应为 confidence 0.8,实际 %v", e.Weight)
	}

	// 两跳边与两跳节点不出现
	if countSubgraphEdge(sg, "咖啡", "产于", "云南") != 0 {
		t.Error("hops=1 时两跳边不应出现")
	}
	if countSubgraphNode(sg, "云南") != 0 {
		t.Error("hops=1 时两跳节点不应出现")
	}

	assertEdgesEndpointsInNodes(t, sg)
}

// TestQuerySubgraph_TwoHops 两跳扩展
func TestQuerySubgraph_TwoHops(t *testing.T) {
	kg := NewKnowledgeGraph()
	kg.Add("A", "认识", "B", 0.5, nil)
	kg.Add("B", "认识", "C", 0.6, nil)

	sg := kg.QuerySubgraph("A", 2)

	if len(sg.Nodes) != 3 {
		t.Fatalf("两跳子图应含 A/B/C 3 个节点,实际 %d: %+v", len(sg.Nodes), sg.Nodes)
	}
	if len(sg.Edges) != 2 {
		t.Fatalf("两跳子图应含 2 条边,实际 %d: %+v", len(sg.Edges), sg.Edges)
	}
	findSubgraphNode(t, sg, "A")
	findSubgraphNode(t, sg, "B")
	findSubgraphNode(t, sg, "C")
	findSubgraphEdge(t, sg, "A", "认识", "B")
	findSubgraphEdge(t, sg, "B", "认识", "C")
}

// TestQuerySubgraph_HopsLEZero hops<=0 视为 1
func TestQuerySubgraph_HopsLEZero(t *testing.T) {
	kg := NewKnowledgeGraph()
	kg.Add("A", "认识", "B", 0.5, nil)
	kg.Add("B", "认识", "C", 0.6, nil)

	s0 := kg.QuerySubgraph("A", 0)
	s1 := kg.QuerySubgraph("A", 1)
	sNeg := kg.QuerySubgraph("A", -3)

	for name, sg := range map[string]Subgraph{"hops=0": s0, "hops=-3": sNeg} {
		if len(sg.Nodes) != 2 {
			t.Errorf("%s 应按 1 跳处理:期望 2 节点,实际 %d", name, len(sg.Nodes))
		}
		if len(sg.Edges) != 1 {
			t.Errorf("%s 应按 1 跳处理:期望 1 边,实际 %d", name, len(sg.Edges))
		}
		if countSubgraphNode(sg, "C") != 0 {
			t.Errorf("%s 不应包含两跳节点 C", name)
		}
	}

	if len(s0.Nodes) != len(s1.Nodes) || len(s0.Edges) != len(s1.Edges) {
		t.Error("hops=0 应与 hops=1 结果一致")
	}
}

// TestQuerySubgraph_IsolatedEntity 孤立实体(索引中存在但无关联三元组):仅自身节点
func TestQuerySubgraph_IsolatedEntity(t *testing.T) {
	// 直接构造内存态:实体在倒排索引中但无任何三元组
	kg := &KnowledgeGraph{entityIdx: map[string][]int{"孤": nil}}

	sg := kg.QuerySubgraph("孤", 2)
	if len(sg.Nodes) != 1 {
		t.Fatalf("孤立实体应仅含自身节点,实际 %d: %+v", len(sg.Nodes), sg.Nodes)
	}
	if sg.Nodes[0].ID != "孤" || sg.Nodes[0].Name != "孤" {
		t.Errorf("孤立实体节点 ID/Name 应为实体名: %+v", sg.Nodes[0])
	}
	if len(sg.Edges) != 0 {
		t.Errorf("孤立实体不应有边: %+v", sg.Edges)
	}
}

// TestQuerySubgraph_NonexistentEntity 不存在的实体返回空子图(非 nil 空 slice)
func TestQuerySubgraph_NonexistentEntity(t *testing.T) {
	kg := NewKnowledgeGraph()
	kg.Add("A", "认识", "B", 0.5, nil)

	sg := kg.QuerySubgraph("不存在", 3)
	if sg.Nodes == nil || sg.Edges == nil {
		t.Fatal("不存在的实体应返回非 nil 空 slice")
	}
	if len(sg.Nodes) != 0 || len(sg.Edges) != 0 {
		t.Errorf("应返回空子图,实际 nodes=%d edges=%d", len(sg.Nodes), len(sg.Edges))
	}
}

// TestQuerySubgraph_EmptyGraph 空图安全
func TestQuerySubgraph_EmptyGraph(t *testing.T) {
	kg := NewKnowledgeGraph()

	sg := kg.QuerySubgraph("A", 3)
	if sg.Nodes == nil || sg.Edges == nil {
		t.Fatal("空图查询应返回非 nil 空 slice")
	}
	if len(sg.Nodes) != 0 || len(sg.Edges) != 0 {
		t.Errorf("空图应返回空子图,实际 nodes=%d edges=%d", len(sg.Nodes), len(sg.Edges))
	}
}

// TestQuerySubgraph_DedupMultiplePaths 同一节点经多条路径到达只出现一次
func TestQuerySubgraph_DedupMultiplePaths(t *testing.T) {
	kg := NewKnowledgeGraph()
	kg.Add("A", "认识", "B", 0.5, nil)
	kg.Add("A", "认识", "C", 0.4, nil)
	kg.Add("B", "认识", "D", 0.3, nil)
	kg.Add("C", "认识", "D", 0.2, nil) // D 经 B、C 两条路径到达

	sg := kg.QuerySubgraph("A", 2)

	if len(sg.Nodes) != 4 {
		t.Fatalf("应含 A/B/C/D 共 4 个节点,实际 %d: %+v", len(sg.Nodes), sg.Nodes)
	}
	if countSubgraphNode(sg, "D") != 1 {
		t.Errorf("节点 D 应只出现一次,实际 %d 次", countSubgraphNode(sg, "D"))
	}
	if len(sg.Edges) != 4 {
		t.Fatalf("应含 4 条边,实际 %d: %+v", len(sg.Edges), sg.Edges)
	}
	if countSubgraphEdge(sg, "B", "认识", "D") != 1 || countSubgraphEdge(sg, "C", "认识", "D") != 1 {
		t.Error("去重后 B-D 与 C-D 边应各出现一次")
	}

	// 节点权重 = 关联边(去重后)最大权重
	if n := findSubgraphNode(t, sg, "D"); n.Weight != 0.3 {
		t.Errorf("节点 D 权重应取关联边最大权重 0.3,实际 %v", n.Weight)
	}
}

// TestQuerySubgraph_EdgeDedupSameFromToType 同 From-To-Type 边去重并取最大权重
func TestQuerySubgraph_EdgeDedupSameFromToType(t *testing.T) {
	kg := NewKnowledgeGraph()
	kg.Add("A", "认识", "B", 0.5, nil)
	kg.Add("A", "认识", "B", 0.9, nil) // 同一 From-To-Type,取最大权重

	sg := kg.QuerySubgraph("A", 1)

	if len(sg.Edges) != 1 {
		t.Fatalf("同 From-To-Type 边应去重为 1 条,实际 %d: %+v", len(sg.Edges), sg.Edges)
	}
	if e := findSubgraphEdge(t, sg, "A", "认识", "B"); e.Weight != 0.9 {
		t.Errorf("去重边权重应取最大 confidence 0.9,实际 %v", e.Weight)
	}
	if len(sg.Nodes) != 2 {
		t.Errorf("应含 A/B 两个节点,实际 %d: %+v", len(sg.Nodes), sg.Nodes)
	}
}

// TestQuerySubgraph_EdgesBothEndpointsInNodes 边只保留两端都在节点集合内的
func TestQuerySubgraph_EdgesBothEndpointsInNodes(t *testing.T) {
	kg := NewKnowledgeGraph()
	kg.Add("A", "认识", "B", 0.5, nil)
	kg.Add("B", "认识", "C", 0.6, nil)

	sg := kg.QuerySubgraph("A", 1)

	if len(sg.Nodes) != 2 {
		t.Fatalf("hops=1 应含 A/B 两个节点,实际 %d: %+v", len(sg.Nodes), sg.Nodes)
	}
	if len(sg.Edges) != 1 {
		t.Fatalf("hops=1 应只含 A-B 一条边,实际 %d: %+v", len(sg.Edges), sg.Edges)
	}
	// B-C 边一端(C)不在节点集合内,应被排除
	if countSubgraphEdge(sg, "B", "认识", "C") != 0 {
		t.Error("两端不在节点集合内的边应被排除")
	}
	assertEdgesEndpointsInNodes(t, sg)
}

// TestQuerySubgraph_CaseInsensitive 索引小写匹配,节点保留原始写法;边保持三元组方向
func TestQuerySubgraph_CaseInsensitive(t *testing.T) {
	kg := NewKnowledgeGraph()
	kg.Add("Alice", "likes", "Bob", 0.8, nil)

	sg := kg.QuerySubgraph("alice", 1)

	if len(sg.Nodes) != 2 {
		t.Fatalf("应含 2 个节点,实际 %d: %+v", len(sg.Nodes), sg.Nodes)
	}
	// 中心节点采用图谱中存储的原始写法
	center := findSubgraphNode(t, sg, "Alice")
	if center.Name != "Alice" {
		t.Errorf("中心节点应为原始写法 Alice: %+v", center)
	}
	// 边方向保留三元组原方向(Alice→Bob)
	findSubgraphEdge(t, sg, "Alice", "likes", "Bob")

	// 从 Bob 查询:方向仍为 Alice→Bob,不因查询实体翻转
	sg2 := kg.QuerySubgraph("Bob", 1)
	if countSubgraphEdge(sg2, "Alice", "likes", "Bob") != 1 {
		t.Error("从 Bob 查询时边方向不应翻转")
	}
	if len(sg2.Nodes) != 2 {
		t.Errorf("Bob 一跳应含 2 个节点,实际 %d: %+v", len(sg2.Nodes), sg2.Nodes)
	}
}
