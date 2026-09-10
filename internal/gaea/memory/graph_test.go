package memory

import (
	"database/sql"
	"reflect"
	"sort"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

// graphTestDB 开一个隔离的临时目录库（真迁移链，含 SchemaV18 图谱表）。
// 与 lifecycle_test 同一注入口径：t.TempDir() 隔离，绝不触碰真实用户库。
func graphTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { _ = db.CloseDatabase(dir) })
	return gdb
}

// seedProjectEvents 投影确定性测试用的固定事件序列（跨项目 + 全 op 覆盖）。
func seedProjectEvents() []Event {
	return []Event{
		{Seq: 1, At: 1000, Op: OpSave, Name: "cost-rule", Project: "proj-a", Space: "work",
			Kind: "semantic", Type: "feedback", Title: "造价规则", Desc: "组价先查历史价",
			Tags: []string{"造价"}, Refs: []string{"price-band", "ghost-key"}},
		{Seq: 2, At: 2000, Op: OpSave, Name: "price-band", Project: "proj-a", Space: "work",
			Kind: "semantic", Type: "project", Title: "价格带", Desc: "P25/中位/P75"},
		{Seq: 3, At: 3000, Op: OpTouch, Name: "price-band", Project: "proj-a", Space: "work"},
		{Seq: 4, At: 4000, Op: OpCite, Name: "price-band", Project: "proj-a", Space: "work"},
		{Seq: 5, At: 5000, Op: OpCite, Name: "ghost-key", Project: "proj-a", Space: "work", Refs: []string{"dangling"}},
		{Seq: 6, At: 6000, Op: OpArchive, Name: "cost-rule", Project: "proj-a"},
		{Seq: 7, At: 7000, Op: OpSave, Name: "novel-note", Project: "proj-b", Space: "play",
			Kind: "episodic", Type: "project", Title: "写作偏好", SourceSession: "session-x"},
		{Seq: 8, At: 8000, Op: OpCite, Name: "novel-note", Project: "proj-b", Space: "play"},
	}
}

// graphSnapshot 把图收敛成可比较形态（防御性拷贝，比较内容而不是别名）。
func graphSnapshot(g *Graph) Graph {
	return Graph{
		Nodes:    append([]GraphNode(nil), g.Nodes...),
		Edges:    append([]GraphEdge(nil), g.Edges...),
		Dangling: append([]string(nil), g.Dangling...),
		MaxSeq:   g.MaxSeq,
	}
}

// TestProjectEventsDeterministic 可重建性核心：同一份日志投影两次，输出逐字段
// 一致（出口判据「同一事件日志两次投影结果一致」）。
func TestProjectEventsDeterministic(t *testing.T) {
	events := seedProjectEvents()
	a := graphSnapshot(ProjectEvents(events))
	b := graphSnapshot(ProjectEvents(events))
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("同一日志两次投影不一致")
	}
	if a.MaxSeq != 8 {
		t.Fatalf("MaxSeq = %d, want 8", a.MaxSeq)
	}
}

// TestProjectEventsThreeWayEdges 三向边：source -produces→ event -affects→ entity。
func TestProjectEventsThreeWayEdges(t *testing.T) {
	g := ProjectEvents(seedProjectEvents())
	want := map[GraphEdge]bool{
		{Src: "src:proj-a/local", Tgt: "ev:1", EType: EdgeProduces}:     true,
		{Src: "ev:1", Tgt: "mem:proj-a/cost-rule", EType: EdgeAffects}:  true,
		{Src: "src:proj-b/session-x", Tgt: "ev:7", EType: EdgeProduces}: true,
		{Src: "ev:7", Tgt: "mem:proj-b/novel-note", EType: EdgeAffects}: true,
	}
	for _, e := range g.Edges {
		delete(want, e)
	}
	if len(want) > 0 {
		t.Fatalf("缺三向边：%+v", want)
	}
}

// TestProjectEventsReferenceAndDangling 互引：cost-rule→price-band 物化
// references 边；不存在的目标进 Dangling，不产生边也不产生节点（悬空拒写）。
func TestProjectEventsReferenceAndDangling(t *testing.T) {
	g := ProjectEvents(seedProjectEvents())
	ref := GraphEdge{Src: "mem:proj-a/cost-rule", Tgt: "mem:proj-a/price-band", EType: EdgeReferences}
	found := false
	for _, e := range g.Edges {
		if e == ref {
			found = true
		}
		if e.Src == "mem:proj-a/cost-rule" && e.Tgt == "mem:proj-a/ghost-key" {
			t.Fatalf("悬空引用被物化成边：%+v", e)
		}
	}
	if !found {
		t.Fatalf("缺互引边 %+v", ref)
	}
	if !reflect.DeepEqual(g.Dangling, []string{"ghost-key"}) {
		t.Fatalf("Dangling = %v, want [ghost-key]", g.Dangling)
	}
	for _, n := range g.Nodes {
		if n.ID == "mem:proj-a/ghost-key" {
			t.Fatalf("悬空键被建成实体节点（cite 不得节点化）")
		}
	}
}

// TestProjectEventsEntityFold 实体折算：last-write-wins——archive 后状态翻
// archived，save 事件的元数据（title/kind/tags）在场。
func TestProjectEventsEntityFold(t *testing.T) {
	g := ProjectEvents(seedProjectEvents())
	var cost *GraphNode
	for i := range g.Nodes {
		if g.Nodes[i].ID == "mem:proj-a/cost-rule" {
			cost = &g.Nodes[i]
		}
	}
	if cost == nil {
		t.Fatalf("实体节点缺失")
	}
	if cost.State != "archived" {
		t.Fatalf("State = %q, want archived（末态以最新事件为准）", cost.State)
	}
	if cost.Kind != "semantic" || cost.MemType != "feedback" {
		t.Fatalf("元数据丢失：kind=%s type=%s", cost.Kind, cost.MemType)
	}
	if cost.Name != "造价规则" {
		t.Fatalf("Name = %q, want 标题", cost.Name)
	}
}

// TestGraphRebuildFromLog 删库可重建端到端：写路径产生事件 → 物化 → 清空三表
// → EnsureGraphFresh 自愈重建 → 与重建前一致，且与纯函数投影一致（出口判据
// 「删库可重建」「两次投影一致」的物化形态）。
func TestGraphRebuildFromLog(t *testing.T) {
	gdb := graphTestDB(t)
	store := SQLiteStoreFor(gdb, t.TempDir(), t.TempDir())
	if _, err := store.Save(Memory{Name: "alpha", Title: "甲", Description: "第一条", Body: "参见 [[beta]]"}); err != nil {
		t.Fatalf("save alpha: %v", err)
	}
	if _, err := store.Save(Memory{Name: "beta", Title: "乙", Description: "第二条", Body: "被引用"}); err != nil {
		t.Fatalf("save beta: %v", err)
	}
	if err := store.Touch("beta"); err != nil {
		t.Fatalf("touch: %v", err)
	}
	if ok, err := EnsureGraphFresh(gdb); err != nil || !ok {
		t.Fatalf("EnsureGraphFresh = %v, %v；want 首次重建", ok, err)
	}
	before, err := MaterializedGraph(gdb)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(before.Nodes) == 0 {
		t.Fatalf("物化为空")
	}
	if len(before.Dangling) != 0 {
		t.Fatalf("Dangling = %v, want 空", before.Dangling)
	}
	// 再对账一次：水位已齐，不得重复重建。
	if ok, err := EnsureGraphFresh(gdb); err != nil || ok {
		t.Fatalf("第二次对账应跳过，got %v, %v", ok, err)
	}
	// 悬空引用 + [MEM:] 大小写归一。
	if _, err := store.Save(Memory{Name: "gamma", Body: "引用 [[ghost]] 与 [MEM:Alpha]"}); err != nil {
		t.Fatalf("save gamma: %v", err)
	}
	if ok, err := EnsureGraphFresh(gdb); err != nil || !ok {
		t.Fatalf("增量对账应重建，got %v, %v", ok, err)
	}
	after, err := MaterializedGraph(gdb)
	if err != nil {
		t.Fatalf("load 2: %v", err)
	}
	if !reflect.DeepEqual(after.Dangling, []string{"ghost"}) {
		t.Fatalf("Dangling = %v, want [ghost]", after.Dangling)
	}

	// 「删库」：清空物化三表 → 读前对账自愈 → 与删库前一致。
	for _, tbl := range []string{"mem_graph_nodes", "mem_graph_edges", "mem_graph_meta"} {
		if _, err := gdb.Exec(`DELETE FROM ` + tbl); err != nil {
			t.Fatalf("clear %s: %v", tbl, err)
		}
	}
	if ok, err := EnsureGraphFresh(gdb); err != nil || !ok {
		t.Fatalf("删库后应重建，got %v, %v", ok, err)
	}
	healed, err := MaterializedGraph(gdb)
	if err != nil {
		t.Fatalf("load healed: %v", err)
	}
	if !reflect.DeepEqual(graphSnapshot(healed), graphSnapshot(after)) {
		t.Fatalf("删库重建与删库前不一致")
	}
	events, err := (&EventLog{DB: gdb}).LoadEvents()
	if err != nil {
		t.Fatalf("load events: %v", err)
	}
	if dbg := graphSnapshot(ProjectEvents(events)); !reflect.DeepEqual(dbg, graphSnapshot(healed)) {
		for i := 0; i < len(dbg.Nodes) && i < len(healed.Nodes); i++ {
			if fmtNode(dbg.Nodes[i]) != fmtNode(healed.Nodes[i]) {
				t.Fatalf("NODE[%d] p=%+v h=%+v", i, dbg.Nodes[i], healed.Nodes[i])
			}
		}
		for i := 0; i < len(dbg.Edges) && i < len(healed.Edges); i++ {
			if dbg.Edges[i] != healed.Edges[i] {
				t.Fatalf("EDGE[%d] p=%+v h=%+v", i, dbg.Edges[i], healed.Edges[i])
			}
		}
		t.Fatalf("物化 ≠ 日志投影 (n=%d/%d e=%d/%d d=%v/%v s=%d/%d)",
			len(dbg.Nodes), len(healed.Nodes), len(dbg.Edges), len(healed.Edges),
			dbg.Dangling, healed.Dangling, dbg.MaxSeq, healed.MaxSeq)
	}
}

// TestGraphNeighborsTraverse 记忆互引可图遍历：真实写路径 → 物化 → BFS，
// 1 跳达互引实体；hops 边界与未知起点。
func TestGraphNeighborsTraverse(t *testing.T) {
	gdb := graphTestDB(t)
	store := SQLiteStoreFor(gdb, t.TempDir(), t.TempDir())
	if _, err := store.Save(Memory{Name: "alpha", Title: "甲", Body: "参见 [[beta]]"}); err != nil {
		t.Fatalf("save alpha: %v", err)
	}
	if _, err := store.Save(Memory{Name: "beta", Title: "乙", Body: "参见 [[gamma]]"}); err != nil {
		t.Fatalf("save beta: %v", err)
	}
	if _, err := store.Save(Memory{Name: "gamma", Title: "丙", Body: "终点"}); err != nil {
		t.Fatalf("save gamma: %v", err)
	}
	if ok, err := EnsureGraphFresh(gdb); err != nil || !ok {
		t.Fatalf("EnsureGraphFresh = %v, %v", ok, err)
	}
	// 1 跳：alpha → beta（references）+ alpha 的落库事件 ev:1（三向边可回溯
	// 「这条记忆从哪来」——事件节点也是一跳邻居）。
	nb := GraphNeighbors(gdb, "alpha", 1) // [MEM:name] 键入参走 id 后缀解析
	if !containsID(nb, "/beta") || len(nb) != 2 {
		t.Fatalf("1 跳应达 beta + 落库事件，got %+v", nb)
	}
	viaRef := ""
	for _, n := range nb {
		if containsID([]Neighbor{n}, "/beta") {
			viaRef = n.Via
		}
	}
	if viaRef != EdgeReferences {
		t.Fatalf("到 beta 的经边 = %q, want references", viaRef)
	}
	// 2 跳：alpha → beta → gamma；完整 id 入参同解。
	nb2 := GraphNeighbors(gdb, "mem:"+projectPrefix(gdb)+"/alpha", 2)
	if !(containsID(nb2, "/beta") && containsID(nb2, "/gamma")) {
		t.Fatalf("2 跳应达 beta+gamma，got %+v", nb2)
	}
	if GraphNeighbors(gdb, "no-such-node", 2) != nil {
		t.Fatalf("未知起点应返回空")
	}
}

// projectPrefix 从物化实体 id 取 project 段（测试辅助）。
func projectPrefix(gdb *sql.DB) string {
	g, err := MaterializedGraph(gdb)
	if err != nil {
		return ""
	}
	for _, n := range g.Nodes {
		if n.NType == NodeEntity {
			rest := n.ID[4:] // 去掉 "mem:"
			for i := 0; i < len(rest); i++ {
				if rest[i] == '/' {
					return rest[:i]
				}
			}
		}
	}
	return ""
}

// TestEventLogAppendLoad 事件日志基本盘：追加→读回保序；缺 op/name 拒收。
func TestEventLogAppendLoad(t *testing.T) {
	gdb := graphTestDB(t)
	l := &EventLog{DB: gdb}
	if err := l.AppendEvent(Event{Op: OpSave, Name: "a"}); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := l.AppendEvent(Event{Op: OpCite, Name: "b", Space: "play"}); err != nil {
		t.Fatalf("append 2: %v", err)
	}
	if err := l.AppendEvent(Event{Op: "", Name: "x"}); err == nil {
		t.Fatalf("缺 op 应拒收")
	}
	events, err := l.LoadEvents()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(events) != 2 || events[0].Name != "a" || events[1].Name != "b" {
		t.Fatalf("读回不符：%+v", events)
	}
	if events[1].Space != "play" {
		t.Fatalf("space 丢失")
	}
	if max, _ := l.MaxEventSeq(); max != 2 {
		t.Fatalf("MaxSeq = %d, want 2", max)
	}
}

// TestSQLiteBackendEventHooks 写路径挂钩：Save/Touch/Archive 逐 op 留痕；
// 未命中触达不落事件；save 事件带互引 refs 与来源。
func TestSQLiteBackendEventHooks(t *testing.T) {
	gdb := graphTestDB(t)
	store := SQLiteStoreFor(gdb, t.TempDir(), t.TempDir())
	if _, err := store.Save(Memory{Name: "hooked", Body: "正文 [MEM:other] 引用", Space: "play", SourceSession: "s-1"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := store.Touch("hooked"); err != nil {
		t.Fatalf("touch: %v", err)
	}
	if _, err := store.Archive("hooked"); err != nil {
		t.Fatalf("archive: %v", err)
	}
	if err := store.Touch("hooked"); err != nil { // 归档后触达不命中，不落事件
		t.Fatalf("touch archived: %v", err)
	}
	events, err := (&EventLog{DB: gdb}).LoadEvents()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	var ops []string
	for _, e := range events {
		ops = append(ops, e.Op)
	}
	want := []string{OpSave, OpTouch, OpArchive}
	if !reflect.DeepEqual(ops, want) {
		t.Fatalf("ops = %v, want %v", ops, want)
	}
	sv := events[0]
	if sv.Project == "" || sv.Space != "play" || sv.SourceSession != "s-1" {
		t.Fatalf("save 事件元数据缺失：%+v", sv)
	}
	if !reflect.DeepEqual(sv.Refs, []string{"other"}) {
		t.Fatalf("refs = %v, want [other]", sv.Refs)
	}
	_ = store.Touch("absent") // 未命中不落事件
	events2, _ := (&EventLog{DB: gdb}).LoadEvents()
	if len(events2) != 3 {
		t.Fatalf("未命中触达不应落事件，events=%d", len(events2))
	}
}

// TestAppendCiteEvents cite 事件：命中与悬空各留一条；nil DB 安全。
func TestAppendCiteEvents(t *testing.T) {
	gdb := graphTestDB(t)
	l := &EventLog{DB: gdb}
	l.AppendCiteEvents([]string{"hit"}, []string{"ghost"}, "work")
	events, _ := l.LoadEvents()
	if len(events) != 2 {
		t.Fatalf("want 2 events, got %d", len(events))
	}
	sort.Slice(events, func(i, j int) bool { return events[i].Name < events[j].Name })
	if events[0].Name != "ghost" || len(events[0].Refs) != 1 || events[0].Refs[0] != "dangling" {
		t.Fatalf("悬空 cite 事件形态不符：%+v", events[0])
	}
	if events[1].Name != "hit" || len(events[1].Refs) != 0 {
		t.Fatalf("命中 cite 事件形态不符：%+v", events[1])
	}
	// nil DB 不炸。
	(&EventLog{}).AppendCiteEvents([]string{"x"}, nil, "")
}

// containsID 邻居命中断言（id 后缀匹配，规避跨机临时目录 project 段差异）。
func containsID(nb []Neighbor, suffix string) bool {
	for _, n := range nb {
		if len(n.ID) >= len(suffix) && n.ID[len(n.ID)-len(suffix):] == suffix {
			return true
		}
	}
	return false
}

func fmtNode(n GraphNode) string {
	return n.ID + "|" + n.NType + "|" + n.Name + "|" + n.Desc + "|" + n.State + "|" + n.Kind + "|" + n.MemType + "|" + n.Space + "|" + n.Project
}
