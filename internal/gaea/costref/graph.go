// v4.8 成本知识图谱组图器（纯函数，零 IO，与 ComputeAttribution 同范式）。
//
// 把成本域五类数据（成本条目 / 测算项目与明细 / 询价数据点 / 复盘笔记 / 分类树）
// 组装成一张可渲染的关联图：scope="tree"（默认）给分类树聚合总览（每分类一个
// 节点，Val=子树金额合计），scope="entry" 围绕 focus（分类路径或项目 ID）展开
// 条目与明细，并沿「明细→条目→询价/指标」「笔记→分类」补齐关联边。节点/边有
// 硬上限，超限置 Truncated（前端提示部分展示）。
//
// 匹配口径：
//   - 条目→分类：CategoryPath 前缀（a/b/c 归入 a、a/b、a/b/c 三级）；
//   - 明细→条目：EntryName 精确优先，标题归一化（costinquiry.MatchTitle）兜底，
//     边 Meta 记 matchedBy=entry_name|title；
//   - 明细→指标：指标由参考池（有版本留痕的项目明细）实时聚合（ComputeIndicators，
//     不落表，与 GaeaCostIndicators 同口径），按科目标题精确→归一化匹配；
//   - 询价→条目：标题精确→归一化匹配（suggests）；
//   - 笔记→分类：note.Category 等于分类名/路径/一级段（notes）。
package costref

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/costinquiry"
	"github.com/gaea/gaea/internal/gaea/costproject"
)

// 节点类型枚举（前端着色映射与此对齐）。
const (
	GraphNodeCategory  = "category"  // 分类树节点
	GraphNodeEntry     = "entry"     // 成本条目
	GraphNodeProject   = "project"   // 测算项目
	GraphNodeItem      = "item"      // 测算明细行
	GraphNodeIndicator = "indicator" // 造价参考指标
	GraphNodeInquiry   = "inquiry"   // 询价数据点
	GraphNodeNote      = "note"      // 复盘笔记
)

// 边类型枚举（source→target 方向固定）。
const (
	GraphEdgeBelongsTo  = "belongs_to" // category→entry 条目归属分类
	GraphEdgeContains   = "contains"   // project→item 项目含明细
	GraphEdgeReferences = "references" // item→entry 明细引用条目
	GraphEdgeBenchmarks = "benchmarks" // item→indicator 明细对标指标
	GraphEdgeSuggests   = "suggests"   // inquiry→entry 询价建议条目
	GraphEdgeNotes      = "notes"      // note→category 笔记沉淀分类
)

// 规模硬上限：节点默认 300（对应 600 边），最大 600；边硬上限 600。
// 超限截断并置 Truncated（宁少勿错，保证前端渲染规模可控）。
const (
	DefaultGraphNodes = 300
	MaxGraphNodes     = 600
	MaxGraphEdges     = 600
)

// GraphNode 图节点。Type 取节点类型枚举；Val 为该节点的金额量级（分类=子树
// 合计、条目=单价、项目=总合计、明细=金额、指标=中位数、询价=单价、笔记=引用数），
// 供前端按值定半径；Meta 是结构化明细（点击节点弹窗展示）。
type GraphNode struct {
	ID   string            `json:"id"`
	Name string            `json:"name"`
	Type string            `json:"type"`
	Desc string            `json:"desc"`
	Val  float64           `json:"val"`
	Meta map[string]string `json:"meta,omitempty"`
}

// GraphEdge 图边。Meta 携带匹配方式（matchedBy=entry_name|title）等溯源信息。
type GraphEdge struct {
	Source string            `json:"source"`
	Target string            `json:"target"`
	Type   string            `json:"type"`
	Weight float64           `json:"weight"`
	Meta   map[string]string `json:"meta,omitempty"`
}

// GraphStats 图规模统计（NodeCount/EdgeCount 为截断后的实际数量）。
type GraphStats struct {
	Truncated    bool           `json:"truncated"`
	NodeCount    int            `json:"nodeCount"`
	EdgeCount    int            `json:"edgeCount"`
	CountsByType map[string]int `json:"countsByType"`
}

// CostGraphView 成本知识图谱视图（前端 SVG 渲染的输入）。
type CostGraphView struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
	Stats GraphStats  `json:"stats"`
}

// BuildGraph 组图器入口（纯函数）。scope="tree"（默认）= 分类树聚合 + 项目节点，
// 不展开明细；scope="entry" = 围绕 focus（分类路径或项目 ID）展开条目+明细+
// 匹配到的询价/指标/笔记，focus 为空时回退 tree 聚合（避免空屏）。limit 为节点
// 上限（<=0 取默认 300，>600 夹到 600），边硬上限 600。
func BuildGraph(
	projects []costproject.ProjectSummary,
	itemsByProject map[string][]costproject.Item,
	entries []cost.Summary,
	categories []cost.CategoryView,
	inquiries []costinquiry.Record,
	notes []Note,
	scope string,
	focus string,
	limit int,
) CostGraphView {
	nodeLimit := limit
	if nodeLimit <= 0 {
		nodeLimit = DefaultGraphNodes
	}
	if nodeLimit > MaxGraphNodes {
		nodeLimit = MaxGraphNodes
	}
	b := &graphBuilder{
		nodeLimit: nodeLimit,
		edgeLimit: MaxGraphEdges,
		nodeIDs:   map[string]bool{},
		edgeKeys:  map[string]bool{},
		counts:    map[string]int{},
	}

	// 确定性输入：项目按 ID 排序、条目按 Name 排序（同序去重取首）。
	ps := append([]costproject.ProjectSummary(nil), projects...)
	sort.Slice(ps, func(i, j int) bool { return ps[i].ID < ps[j].ID })
	es := append([]cost.Summary(nil), entries...)
	sort.Slice(es, func(i, j int) bool { return es[i].Name < es[j].Name })

	if scope == "entry" && strings.TrimSpace(focus) != "" {
		b.buildEntry(ps, itemsByProject, es, inquiries, notes, focus)
	} else {
		b.buildTree(ps, es, categories)
	}
	return CostGraphView{Nodes: b.nodes, Edges: b.edges, Stats: b.stats()}
}

// ── scope=tree：分类树聚合 + 项目节点（不展开明细）──────────────────────

func (b *graphBuilder) buildTree(projects []costproject.ProjectSummary, entries []cost.Summary, categories []cost.CategoryView) {
	// 子树聚合：每条目按分类路径逐级前缀累加（O(E×深度)），分类节点直接查表。
	type agg struct {
		count  int
		amount float64
	}
	aggs := map[string]*agg{}
	for _, e := range entries {
		path := normPath(e.CategoryPath)
		if path == "" {
			continue
		}
		segs := strings.Split(path, "/")
		for i := range segs {
			p := strings.Join(segs[:i+1], "/")
			a := aggs[p]
			if a == nil {
				a = &agg{}
				aggs[p] = a
			}
			a.count++
			a.amount += e.Price
		}
	}
	var walk func(nodes []*cost.CategoryView, parentPath string)
	walk = func(nodes []*cost.CategoryView, parentPath string) {
		for _, c := range nodes {
			if b.limitHit() {
				b.truncated = true
				return
			}
			path := c.Name
			if parentPath != "" {
				path = parentPath + "/" + c.Name
			}
			count, amount := 0, 0.0
			if a := aggs[path]; a != nil {
				count, amount = a.count, a.amount
			}
			if b.addNode(GraphNode{
				ID:   categoryNodeID(path),
				Name: c.Name,
				Type: GraphNodeCategory,
				Desc: fmt.Sprintf("%d 条", count),
				Val:  amount,
				Meta: map[string]string{
					"path":    path,
					"entries": strconv.Itoa(count),
					"amount":  fmtMoney(amount),
				},
			}) {
				walk(c.Children, path)
			}
			if b.limitHit() {
				b.truncated = true
				return
			}
		}
	}
	roots := make([]*cost.CategoryView, 0, len(categories))
	for i := range categories {
		roots = append(roots, &categories[i])
	}
	walk(roots, "")
	// 项目节点（不展开明细，Val=项目合计）。
	for _, p := range projects {
		if b.limitHit() {
			b.truncated = true
			return
		}
		b.addNode(projectNode(p))
	}
}

// ── scope=entry：focus 子树展开 ─────────────────────────────────────────

func (b *graphBuilder) buildEntry(
	projects []costproject.ProjectSummary,
	itemsByProject map[string][]costproject.Item,
	entries []cost.Summary,
	inquiries []costinquiry.Record,
	notes []Note,
	focus string,
) {
	focusKey := strings.Trim(strings.TrimSpace(focus), "/")
	projByID := map[string]costproject.ProjectSummary{}
	for _, p := range projects {
		projByID[p.ID] = p
	}
	entryByName, entryByTitle := entryIndexes(entries)
	indByKey, indByTitle := indicatorIndexes(projects, itemsByProject)
	inqExact, inqByTitle := inquiryIndexes(inquiries)

	// 项目 focus：项目→明细→条目（含指标/询价/笔记）。
	if p, ok := projByID[focusKey]; ok {
		b.expandProject(p, itemsByProject, entryByName, entryByTitle, indByKey, indByTitle, inqExact, inqByTitle, notes)
		return
	}
	// 分类 focus：子树分类+条目→引用明细（全项目）→指标/询价/笔记。
	b.expandCategory(focusKey, projects, itemsByProject, entries, entryByName, entryByTitle, indByKey, indByTitle, inqExact, inqByTitle, notes)
}

// expandProject 项目 focus 展开：contains/references/benchmarks/suggests/notes 五类边。
func (b *graphBuilder) expandProject(
	p costproject.ProjectSummary,
	itemsByProject map[string][]costproject.Item,
	entryByName map[string]cost.Summary,
	entryByTitle map[string]cost.Summary,
	indByKey, indByTitle map[string]Indicator,
	inqExact map[string][]costinquiry.Record,
	inqByTitle map[string][]costinquiry.Record,
	notes []Note,
) {
	if !b.addNode(projectNode(p)) {
		b.truncated = true
		return
	}
	// 明细节点 + contains 边；逐行匹配条目（references）与指标（benchmarks）。
	catPaths := map[string]bool{}
	seenEntry := map[string]bool{}
	for _, it := range sortedItems(itemsByProject[p.ID]) {
		if b.limitHit() {
			b.truncated = true
			break
		}
		if !b.addNode(itemNode(p.ID, it)) {
			continue
		}
		b.addEdge(GraphEdge{Source: projectNodeID(p.ID), Target: itemNodeID(p.ID, it), Type: GraphEdgeContains})
		if e, matchedBy, ok := matchEntry(it, entryByName, entryByTitle); ok {
			if b.addNode(entryNode(e)) {
				b.addEdge(GraphEdge{
					Source: itemNodeID(p.ID, it),
					Target: entryNodeID(e.Name),
					Type:   GraphEdgeReferences,
					Meta:   map[string]string{"matchedBy": matchedBy},
				})
				if !seenEntry[e.Name] {
					seenEntry[e.Name] = true
					b.referencedEntries = append(b.referencedEntries, e)
				}
				if path := normPath(e.CategoryPath); path != "" {
					catPaths[path] = true
				}
			}
		}
		b.addBenchmarks(itemNodeID(p.ID, it), it.Title, indByKey, indByTitle)
	}
	// 条目所属分类节点（笔记挂靠 + 图上下文）。
	b.addCategoryNodesForPaths(catPaths)
	for _, e := range b.referencedEntries {
		if b.limitHit() {
			b.truncated = true
			break
		}
		b.addSuggests(e, inqExact, inqByTitle)
	}
	b.addNotes(catPaths, notes)
}

// expandCategory 分类 focus 展开：子树分类+条目（belongs_to）+ 引用明细
// （contains/references）+ 指标/询价/笔记。
func (b *graphBuilder) expandCategory(
	focusPath string,
	projects []costproject.ProjectSummary,
	itemsByProject map[string][]costproject.Item,
	entries []cost.Summary,
	entryByName map[string]cost.Summary,
	entryByTitle map[string]cost.Summary,
	indByKey, indByTitle map[string]Indicator,
	inqExact map[string][]costinquiry.Record,
	inqByTitle map[string][]costinquiry.Record,
	notes []Note,
) {
	// 1) 子树条目：CategoryPath 精确或前缀命中；同时补齐分类节点链。
	// focus 分类节点先入图（中心节点），belongs_to 边才有源端。
	catPaths := map[string]bool{}
	subtreeEntries := map[string]bool{}
	b.addNode(categoryNode(focusPath))
	for _, e := range entries {
		if b.limitHit() {
			b.truncated = true
			break
		}
		path := normPath(e.CategoryPath)
		if path == "" || !underPath(path, focusPath) {
			continue
		}
		b.addCategoryChain(path, focusPath)
		catPaths[path] = true
		if b.addNode(entryNode(e)) {
			subtreeEntries[e.Name] = true
			b.addEdge(GraphEdge{
				Source: categoryNodeID(path),
				Target: entryNodeID(e.Name),
				Type:   GraphEdgeBelongsTo,
			})
			b.referencedEntries = append(b.referencedEntries, e)
		}
	}
	// 2) 明细：引用子树条目（精确/归一化）或自身分类路径在子树内（全项目扫描）。
	// 引用匹配只认子树条目——子树外的条目关联属于别的子树，不跨界拉入。
	for _, p := range projects {
		if b.limitHit() {
			b.truncated = true
			break
		}
		for _, it := range sortedItems(itemsByProject[p.ID]) {
			if b.limitHit() {
				b.truncated = true
				break
			}
			e, matchedBy, matched := matchEntry(it, entryByName, entryByTitle)
			ownCat := underPath(normPath(it.CategoryPath), focusPath)
			if (!matched || !subtreeEntries[e.Name]) && !ownCat {
				continue
			}
			if !b.addNode(itemNode(p.ID, it)) {
				continue
			}
			b.addNode(projectNode(p))
			b.addEdge(GraphEdge{Source: projectNodeID(p.ID), Target: itemNodeID(p.ID, it), Type: GraphEdgeContains})
			if matched {
				if b.addNode(entryNode(e)) {
					b.addEdge(GraphEdge{
						Source: itemNodeID(p.ID, it),
						Target: entryNodeID(e.Name),
						Type:   GraphEdgeReferences,
						Meta:   map[string]string{"matchedBy": matchedBy},
					})
					if path := normPath(e.CategoryPath); path != "" {
						b.addCategoryChain(path, focusPath)
						catPaths[path] = true
					}
					b.referencedEntries = append(b.referencedEntries, e)
				}
			}
			b.addBenchmarks(itemNodeID(p.ID, it), it.Title, indByKey, indByTitle)
		}
	}
	// 3) 询价（对去重后的条目）与笔记。
	seenEntry := map[string]bool{}
	dedup := make([]cost.Summary, 0, len(b.referencedEntries))
	for _, e := range b.referencedEntries {
		if seenEntry[e.Name] {
			continue
		}
		seenEntry[e.Name] = true
		dedup = append(dedup, e)
	}
	for _, e := range dedup {
		if b.limitHit() {
			b.truncated = true
			break
		}
		b.addSuggests(e, inqExact, inqByTitle)
	}
	b.addNotes(catPaths, notes)
}

// addBenchmarks 明细→指标（标题精确→归一化），指标节点首次出现时补齐。
func (b *graphBuilder) addBenchmarks(itemID, title string, indByKey, indByTitle map[string]Indicator) {
	ind, ok := indByKey[strings.TrimSpace(title)]
	if !ok {
		if norm := costinquiry.MatchTitle(title); norm != "" {
			ind, ok = indByTitle[norm]
		}
	}
	if !ok {
		return
	}
	if b.addNode(indicatorNode(ind)) {
		b.addEdge(GraphEdge{Source: itemID, Target: indicatorNodeID(ind.Key), Type: GraphEdgeBenchmarks})
	}
}

// addSuggests 询价→条目（标题精确→归一化），询价节点首次出现时补齐。
func (b *graphBuilder) addSuggests(e cost.Summary, inqExact map[string][]costinquiry.Record, inqByTitle map[string][]costinquiry.Record) {
	var recs []costinquiry.Record
	if t := strings.TrimSpace(e.Title); t != "" {
		recs = inqExact[t]
	}
	if len(recs) == 0 {
		if norm := costinquiry.MatchTitle(e.Title); norm != "" {
			recs = inqByTitle[norm]
		}
	}
	for _, r := range recs {
		if b.limitHit() {
			b.truncated = true
			return
		}
		if b.addNode(inquiryNode(r)) {
			b.addEdge(GraphEdge{
				Source: inquiryNodeID(r.ID),
				Target: entryNodeID(e.Name),
				Type:   GraphEdgeSuggests,
				Meta:   map[string]string{"matchedBy": "title"},
			})
		}
	}
}

// addNotes 笔记→分类（note.Category 等于分类名/路径/一级段）。
func (b *graphBuilder) addNotes(catPaths map[string]bool, notes []Note) {
	for _, n := range notes {
		if b.limitHit() {
			b.truncated = true
			return
		}
		target := noteTargetPath(n.Category, catPaths)
		if target == "" {
			continue
		}
		if b.addNode(noteNode(n)) {
			b.addEdge(GraphEdge{Source: noteNodeID(n.ID), Target: categoryNodeID(target), Type: GraphEdgeNotes})
		}
	}
}

// noteTargetPath 在已加入的分类路径里找笔记归属：路径精确 > 叶名/一级段
// （同键取路径字典序最小者，保证确定性）。
func noteTargetPath(noteCat string, catPaths map[string]bool) string {
	nc := strings.TrimSpace(noteCat)
	if nc == "" || len(catPaths) == 0 {
		return ""
	}
	if catPaths[nc] {
		return nc
	}
	paths := make([]string, 0, len(catPaths))
	for p := range catPaths {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		segs := strings.Split(p, "/")
		if segs[len(segs)-1] == nc || segs[0] == nc {
			return p
		}
	}
	return ""
}

// ── 组图器：节点/边去重 + 截断 ──────────────────────────────────────────

type graphBuilder struct {
	nodeLimit int
	edgeLimit int
	nodes     []GraphNode
	edges     []GraphEdge
	nodeIDs   map[string]bool
	edgeKeys  map[string]bool
	counts    map[string]int
	truncated bool
	// referencedEntries 展开过程中命中过的条目（供询价 suggests 统一补边；
	// 调用方自行去重）。
	referencedEntries []cost.Summary
}

func (b *graphBuilder) limitHit() bool {
	return len(b.nodes) >= b.nodeLimit || len(b.edges) >= b.edgeLimit
}

// addNode 加入节点（已存在幂等返回 true）；超节点上限置 Truncated 并拒绝。
func (b *graphBuilder) addNode(n GraphNode) bool {
	if b.nodeIDs[n.ID] {
		return true
	}
	if len(b.nodes) >= b.nodeLimit {
		b.truncated = true
		return false
	}
	b.nodeIDs[n.ID] = true
	b.nodes = append(b.nodes, n)
	b.counts[n.Type]++
	return true
}

// addEdge 加入边（同 source+target+type 去重；两端节点必须在图中——不出悬挂边）；
// 超边上限置 Truncated 并拒绝。
func (b *graphBuilder) addEdge(e GraphEdge) {
	if !b.nodeIDs[e.Source] || !b.nodeIDs[e.Target] {
		return
	}
	key := e.Type + "\x00" + e.Source + "\x00" + e.Target
	if b.edgeKeys[key] {
		return
	}
	if len(b.edges) >= b.edgeLimit {
		b.truncated = true
		return
	}
	b.edgeKeys[key] = true
	b.edges = append(b.edges, e)
}

func (b *graphBuilder) stats() GraphStats {
	return GraphStats{
		Truncated:    b.truncated,
		NodeCount:    len(b.nodes),
		EdgeCount:    len(b.edges),
		CountsByType: b.counts,
	}
}

// ── 节点 ID / 视图构造 ──────────────────────────────────────────────────

func categoryNodeID(path string) string { return "cat:" + path }
func entryNodeID(name string) string    { return "entry:" + name }
func projectNodeID(id string) string    { return "proj:" + id }
func indicatorNodeID(k string) string   { return "ind:" + k }
func inquiryNodeID(id int64) string     { return "inq:" + strconv.FormatInt(id, 10) }
func noteNodeID(id int64) string        { return "note:" + strconv.FormatInt(id, 10) }

// itemNodeID 明细节点 ID：项目内优先用稳定名（name），无 name 用自增 ID。
func itemNodeID(projectID string, it costproject.Item) string {
	key := strings.TrimSpace(it.Name)
	if key == "" && it.ID > 0 {
		key = strconv.FormatInt(it.ID, 10)
	}
	if key == "" {
		key = costinquiry.MatchTitle(it.Title)
	}
	if key == "" {
		key = "row"
	}
	return "item:" + projectID + ":" + key
}

func projectNode(p costproject.ProjectSummary) GraphNode {
	return GraphNode{
		ID:   projectNodeID(p.ID),
		Name: p.Name,
		Type: GraphNodeProject,
		Desc: fmt.Sprintf("%d 条明细 · %d 版本", p.ItemCount, p.VersionCount),
		Val:  p.Total,
		Meta: map[string]string{
			"projectId":   p.ID,
			"projectType": p.ProjectType,
			"status":      p.Status,
			"items":       strconv.Itoa(p.ItemCount),
			"versions":    strconv.Itoa(p.VersionCount),
		},
	}
}

func entryNode(e cost.Summary) GraphNode {
	return GraphNode{
		ID:   entryNodeID(e.Name),
		Name: e.Title,
		Type: GraphNodeEntry,
		Desc: fmt.Sprintf("¥%s%s", fmtMoney(e.Price), unitSuffix(e.Unit)),
		Val:  e.Price,
		Meta: map[string]string{
			"name":   e.Name,
			"path":   e.CategoryPath,
			"unit":   e.Unit,
			"source": e.Source,
			"status": e.Status,
			"region": e.Region,
			"spec":   e.Spec,
		},
	}
}

func itemNode(projectID string, it costproject.Item) GraphNode {
	amount := it.Amount
	if amount <= 0 && it.Quantity > 0 {
		amount = it.Quantity * it.Price
	}
	name := strings.TrimSpace(it.Title)
	if name == "" {
		name = it.Name
	}
	return GraphNode{
		ID:   itemNodeID(projectID, it),
		Name: name,
		Type: GraphNodeItem,
		Desc: fmt.Sprintf("%s%s × ¥%s", fmtMoney(it.Quantity), unitSuffix(it.Unit), fmtMoney(it.Price)),
		Val:  amount,
		Meta: map[string]string{
			"projectId": projectID,
			"unit":      it.Unit,
			"quantity":  fmtMoney(it.Quantity),
			"price":     fmtMoney(it.Price),
			"entryName": it.EntryName,
		},
	}
}

func indicatorNode(ind Indicator) GraphNode {
	return GraphNode{
		ID:   indicatorNodeID(ind.Key),
		Name: ind.Key,
		Type: GraphNodeIndicator,
		Desc: fmt.Sprintf("样本 %d · 中位 ¥%s", ind.Samples, fmtMoney(ind.Median)),
		Val:  ind.Median,
		Meta: map[string]string{
			"samples": strconv.Itoa(ind.Samples),
			"min":     fmtMoney(ind.Min),
			"max":     fmtMoney(ind.Max),
			"mean":    fmtMoney(ind.Mean),
			"p25":     fmtMoney(ind.P25),
			"p75":     fmtMoney(ind.P75),
			"unit":    ind.Unit,
		},
	}
}

func inquiryNode(r costinquiry.Record) GraphNode {
	return GraphNode{
		ID:   inquiryNodeID(r.ID),
		Name: r.Title,
		Type: GraphNodeInquiry,
		Desc: fmt.Sprintf("%s · ¥%s%s", r.Source, fmtMoney(r.Price), dateSuffix(r.PriceDate)),
		Val:  r.Price,
		Meta: map[string]string{
			"source":     r.Source,
			"supplier":   r.Supplier,
			"region":     r.Region,
			"priceDate":  r.PriceDate,
			"validUntil": r.ValidUntil,
			"unit":       r.Unit,
			"spec":       r.Spec,
			"status":     r.Status,
		},
	}
}

func noteNode(n Note) GraphNode {
	return GraphNode{
		ID:   noteNodeID(n.ID),
		Name: n.Title,
		Type: GraphNodeNote,
		Desc: fmt.Sprintf("%s · %s · 引用 %d", n.Confidence, n.Status, n.RefCount),
		Val:  float64(n.RefCount),
		Meta: map[string]string{
			"confidence": n.Confidence,
			"status":     n.Status,
			"category":   n.Category,
			"boundary":   n.Boundary,
			"risk":       n.Risk,
			"evidence":   n.Evidence,
		},
	}
}

// ── 匹配索引 ────────────────────────────────────────────────────────────

// entryIndexes 条目双索引：name 精确 + 标题归一化（输入已按 Name 排序，同键取首）。
func entryIndexes(entries []cost.Summary) (map[string]cost.Summary, map[string]cost.Summary) {
	byName := map[string]cost.Summary{}
	byTitle := map[string]cost.Summary{}
	for _, e := range entries {
		if name := strings.TrimSpace(e.Name); name != "" {
			byName[name] = e
		}
		if key := costinquiry.MatchTitle(e.Title); key != "" {
			if _, dup := byTitle[key]; !dup {
				byTitle[key] = e
			}
		}
	}
	return byName, byTitle
}

// indicatorIndexes 指标双索引：参考池（有版本留痕的项目）明细实时聚合，
// 与 GaeaCostIndicators 同口径（VersionCount<=0 的临时工作稿不参与）。
func indicatorIndexes(projects []costproject.ProjectSummary, itemsByProject map[string][]costproject.Item) (map[string]Indicator, map[string]Indicator) {
	var pool []costproject.Item
	for _, p := range projects {
		if p.VersionCount <= 0 {
			continue
		}
		pool = append(pool, itemsByProject[p.ID]...)
	}
	exact, byTitle := map[string]Indicator{}, map[string]Indicator{}
	for _, ind := range ComputeIndicators(pool, "title") {
		if strings.TrimSpace(ind.Key) == "" {
			continue
		}
		exact[ind.Key] = ind
		if norm := costinquiry.MatchTitle(ind.Key); norm != "" {
			if _, dup := byTitle[norm]; !dup {
				byTitle[norm] = ind
			}
		}
	}
	return exact, byTitle
}

// inquiryIndexes 询价双索引：标题精确 + 归一化（稳定序：ID 升序）。
func inquiryIndexes(inquiries []costinquiry.Record) (map[string][]costinquiry.Record, map[string][]costinquiry.Record) {
	sorted := append([]costinquiry.Record(nil), inquiries...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	exact, byTitle := map[string][]costinquiry.Record{}, map[string][]costinquiry.Record{}
	for _, r := range sorted {
		if t := strings.TrimSpace(r.Title); t != "" {
			exact[t] = append(exact[t], r)
		}
		if norm := costinquiry.MatchTitle(r.Title); norm != "" {
			byTitle[norm] = append(byTitle[norm], r)
		}
	}
	return exact, byTitle
}

// matchEntry 明细→条目匹配：EntryName 精确优先，标题归一化兜底；
// matchedBy ∈ entry_name|title。
func matchEntry(it costproject.Item, byName map[string]cost.Summary, byTitle map[string]cost.Summary) (cost.Summary, string, bool) {
	if name := strings.TrimSpace(it.EntryName); name != "" {
		if e, ok := byName[name]; ok {
			return e, "entry_name", true
		}
	}
	if key := costinquiry.MatchTitle(it.Title); key != "" {
		if e, ok := byTitle[key]; ok {
			return e, "title", true
		}
	}
	return cost.Summary{}, "", false
}

// ── 小工具 ──────────────────────────────────────────────────────────────

// normPath 分类路径归一：去首尾空白与首尾斜杠。
func normPath(p string) string {
	return strings.Trim(strings.TrimSpace(p), "/")
}

// underPath 报告 path 是否在 root 子树内（path==root 或 path 以 root/ 开头）。
func underPath(path, root string) bool {
	if path == "" || root == "" {
		return false
	}
	return path == root || strings.HasPrefix(path, root+"/")
}

// sortedItems 项目内明细按（ID, Name）稳定排序。
func sortedItems(items []costproject.Item) []costproject.Item {
	out := append([]costproject.Item(nil), items...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].ID != out[j].ID {
			return out[i].ID < out[j].ID
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// addCategoryChain 加入 path 自 focus 倒数第二段起的分类节点链（focus 自身由
// 条目路径覆盖），已存在幂等。
func (b *graphBuilder) addCategoryChain(path, root string) {
	segs := strings.Split(path, "/")
	start := 0
	if root != "" {
		rootSegs := strings.Split(root, "/")
		if len(segs) <= len(rootSegs) && path != root {
			return // 不在 root 之下（调用方已保证，防御）
		}
		if path == root {
			return
		}
		start = len(rootSegs) - 1 // root 的最后一段已作为子树根加入
	}
	for i := start; i < len(segs); i++ {
		if b.limitHit() {
			b.truncated = true
			return
		}
		b.addNode(categoryNode(strings.Join(segs[:i+1], "/")))
	}
}

// addCategoryNodesForPaths 为一组分类路径加节点（项目 focus 时补条目所属分类）。
func (b *graphBuilder) addCategoryNodesForPaths(paths map[string]bool) {
	list := make([]string, 0, len(paths))
	for p := range paths {
		if p != "" {
			list = append(list, p)
		}
	}
	sort.Strings(list)
	for _, p := range list {
		if b.limitHit() {
			b.truncated = true
			return
		}
		b.addNode(categoryNode(p))
	}
}

// categoryNode 由分类路径构造分类节点（Name=叶子段；展开场景作上下文节点，
// 金额填 0、Desc 给完整路径，与 tree 聚合节点的 Desc=条目数相区分）。
func categoryNode(path string) GraphNode {
	name := path
	if segs := strings.Split(path, "/"); len(segs) > 0 && segs[len(segs)-1] != "" {
		name = segs[len(segs)-1]
	}
	return GraphNode{
		ID:   categoryNodeID(path),
		Name: name,
		Type: GraphNodeCategory,
		Desc: path,
		Meta: map[string]string{"path": path},
	}
}

// fmtMoney 金额/数量去尾零展示（480 → "480"，480.5 → "480.5"）。
func fmtMoney(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func unitSuffix(unit string) string {
	if u := strings.TrimSpace(unit); u != "" {
		return "/" + u
	}
	return ""
}

func dateSuffix(d string) string {
	if s := strings.TrimSpace(d); s != "" {
		return " · " + s
	}
	return ""
}
