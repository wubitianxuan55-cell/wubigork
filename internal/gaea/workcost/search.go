// search.go — 造价库混合检索：编码精确 → 关键词 → 向量 → 融合 → 候选多时本地精排。
//
// 一条查询词进来：
//   - 编码 / 自有码与查询词完全相同：定了就是它，只返回这些（恒排最前）；
//   - 关键词路：名称 / 编码 / 规格 / 章节 / 分类 / 项目特征等字段含查询词（SQL LIKE）；
//   - 向量路：查询词向量化一次，在类目的内存矩阵上取相似度最高的若干条——同义说法
//     （「挖机」对「挖掘机」、「砼」对「混凝土」）在这里被召回；
//   - 两路命中按 vecindex.Order 融合（0.55·向量 + 0.30·关键词 + 0.15·标题），候选超过
//     8 条且精排模型可用时再排一次；
//   - 每条结果标注命中方式（code / keyword / semantic）。
//
// 向量模型没开、索引没建、维度对不上——向量路一律静默让路，关键词路照常可用。
// 查询词为空 = 不检索，按原列表顺序返回（界面的浏览面不受影响）。
package workcost

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/gaea/gaea/internal/gaea/vecindex"
)

// 检索口径。
const (
	// DefaultSearchLimit 没给条数上限时的返回条数。
	DefaultSearchLimit = 50
	// searchKeywordCap 关键词路最多取多少条候选（再多的靠缩窄查询词）。
	searchKeywordCap = 300
	// searchRerankMin 候选超过这个数才值得请精排模型（少了开销不划算）。
	searchRerankMin = 8
	// searchRerankDocs 最多请精排模型看多少篇：耗时与篇数成正比（本机 bge-reranker-v2-m3
	// 实测约 12 毫秒/篇），融合序靠后的候选本来就排不进前几名。
	searchRerankDocs = 16
	// searchLoadChunk 按键批量取行时每批的键数。
	searchLoadChunk = 400
)

// 命中方式。
const (
	ViaCode     = "code"     // 编码精确命中
	ViaKeyword  = "keyword"  // 名称或其它字段含查询词
	ViaSemantic = "semantic" // 只有向量路命中（同义说法）
	ViaList     = "list"     // 没给查询词，按列表顺序
)

// SearchEmbedder / SearchReranker 混合检索的模型依赖，由 app 层注入（本包不认识具体客户端）。
// Rerank 返回 docs 的下标，按相关度从高到低。
type SearchEmbedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

type SearchReranker interface {
	Rerank(ctx context.Context, query string, docs []string, topN int) ([]int, error)
}

// Searcher 造价库混合检索。零值不可用；用 NewSearcher 构造。
type Searcher struct {
	store  *Store
	vec    *vecindex.Store
	embed  func() SearchEmbedder
	rerank func() SearchReranker
}

// NewSearcher 构造检索器。vs / embed / rerank 都可以为 nil（对应那一路不启用）；
// embed / rerank 是取值函数：每次检索取一次，返回 nil 表示模型当前不可用。
func NewSearcher(s *Store, vs *vecindex.Store, embed func() SearchEmbedder, rerank func() SearchReranker) *Searcher {
	return &Searcher{store: s, vec: vs, embed: embed, rerank: rerank}
}

// SearchQuery 一次检索的条件。
type SearchQuery struct {
	// Text 查询词；空 = 不检索，按列表顺序返回。
	Text string
	// Specialty 定额 / 清单的专业过滤（空或「全部」= 不过滤）。
	Specialty string
	// Kind 工料机的类型过滤（空或「全部」= 不过滤）。
	Kind string
	// Region 工料机的地区过滤（字典值；空= 不过滤）。工料机按地区分行，同一种
	// 工料机每个地区一行——只看某个地区的行、或排查「同身份跨地区」时用
	// （2026-11-26 造价反馈，百锦路 #4）。
	Region string
	// ProjectID 清单的项目过滤（0 = 跨项目）。
	ProjectID int64
	// Limit 返回条数上限（<=0 取 DefaultSearchLimit）。
	Limit int
	// Offset 列表模式（Text 为空）的翻页起点：跳过前 Offset 条再取 Limit 条。
	// 关键词模式忽略（那是融合排序，没有稳定全序）；窗口排序带编码兜底，翻页
	// 稳定可复现（2026-11-27 复盘 P2：无翻页只能前缀穷举，40+ 轮仍漏 1 条）。
	Offset int
}

// Hit 一条检索结果：条目 + 融合分 + 命中方式。
type Hit[T any] struct {
	Item  T
	Score float64
	Via   string
}

type (
	QuotaHit    = Hit[Quota]
	ResourceHit = Hit[Resource]
	BillItemHit = Hit[BillItem]
)

// kindSpec 一类条目在管线里的形状。
type kindSpec[T any] struct {
	indexKind  string
	keywordIDs func(s *Store, kw string, q SearchQuery) ([]string, error)
	// load 按键取行（ids 为 nil = 不按键过滤，按列表顺序取 limit 条）；同时套用 q 里的过滤条件。
	load func(s *Store, ids []string, q SearchQuery, limit int) ([]T, error)
	key  func(T) string
	// flags 从一行算命中标志：编码精确 / 名称含查询词 / 其它字段含查询词。
	flags func(T, string) (exact, title, field bool)
	text  func(T) string
}

// Quotas 检索消耗定额。
func (h *Searcher) Quotas(ctx context.Context, q SearchQuery) ([]QuotaHit, error) {
	return runSearch(ctx, h, quotaSpec, q)
}

// Resources 检索工料机。
func (h *Searcher) Resources(ctx context.Context, q SearchQuery) ([]ResourceHit, error) {
	return runSearch(ctx, h, resourceSpec, q)
}

// BillItems 检索清单行（跨项目；q.ProjectID 非 0 时只看该项目）。
func (h *Searcher) BillItems(ctx context.Context, q SearchQuery) ([]BillItemHit, error) {
	return runSearch(ctx, h, billSpec, q)
}

var quotaSpec = kindSpec[Quota]{
	indexKind: IndexKindQuota,
	keywordIDs: func(s *Store, kw string, q SearchQuery) ([]string, error) {
		like := "%" + kw + "%"
		sqlText := `SELECT code FROM gf_quotas WHERE (title LIKE ? OR code LIKE ? OR COALESCE(gcode,'') LIKE ?
  OR COALESCE(orig_code,'') LIKE ? OR chapter LIKE ? OR category_path LIKE ?)`
		args := []any{like, like, like, like, like, like}
		if sp := filterValue(q.Specialty); sp != "" {
			sqlText += ` AND specialty=?`
			args = append(args, sp)
		}
		sqlText += ` ORDER BY specialty, chapter, code LIMIT ?`
		args = append(args, searchKeywordCap)
		return queryKeys(s, sqlText, args...)
	},
	load: func(s *Store, ids []string, q SearchQuery, limit int) ([]Quota, error) {
		if ids == nil {
			return s.listQuotasPaged(filterValue(q.Specialty), q.Offset, limit)
		}
		return s.quotasByKeys(ids, filterValue(q.Specialty))
	},
	key: func(q Quota) string { return q.Code },
	flags: func(q Quota, kw string) (bool, bool, bool) {
		return eqAny(kw, q.Code, q.GCode, q.OrigCode), has(q.Title, kw),
			has(q.Chapter, kw) || has(q.CategoryPath, kw) || has(q.Specialty, kw)
	},
	text: func(q Quota) string {
		return QuotaIndexDoc(q.Code, q.GCode, q.Title, q.Specialty, q.Chapter, q.Unit, q.CategoryPath, "")
	},
}

var resourceSpec = kindSpec[Resource]{
	indexKind: IndexKindResource,
	keywordIDs: func(s *Store, kw string, q SearchQuery) ([]string, error) {
		like := "%" + kw + "%"
		sqlText := `SELECT code FROM gf_resources WHERE (title LIKE ? OR spec LIKE ? OR code LIKE ?
  OR COALESCE(gcode,'') LIKE ? OR category_path LIKE ?)`
		args := []any{like, like, like, like, like}
		if k := filterValue(q.Kind); k != "" {
			sqlText += ` AND kind=?`
			args = append(args, k)
		}
		if rg := regionFilter(q.Region); rg != "" {
			sqlText += ` AND region=?`
			args = append(args, rg)
		}
		sqlText += ` ORDER BY kind, title, spec LIMIT ?`
		args = append(args, searchKeywordCap)
		return queryKeys(s, sqlText, args...)
	},
	load: func(s *Store, ids []string, q SearchQuery, limit int) ([]Resource, error) {
		var rows []Resource
		var err error
		if ids == nil {
			// 列表模式：地区条件必须进 SQL——offset 窗口先切后滤会把页切歪。
			rows, err = s.listResourcesPaged(filterValue(q.Kind), filterValue(q.Region), q.Offset, limit)
		} else {
			rows, err = s.resourcesByKeys(ids, filterValue(q.Kind))
		}
		if err != nil {
			return nil, err
		}
		return filterByRegion(rows, filterValue(q.Region)), nil
	},
	key: func(r Resource) string { return r.Code },
	flags: func(r Resource, kw string) (bool, bool, bool) {
		return eqAny(kw, r.Code, r.GCode), has(r.Title, kw),
			has(r.Spec, kw) || has(r.CategoryPath, kw) || has(r.Kind, kw)
	},
	text: func(r Resource) string {
		return ResourceIndexDoc(r.Code, r.GCode, r.Title, r.Spec, r.Kind, r.Unit, r.CategoryPath)
	},
}

var billSpec = kindSpec[BillItem]{
	indexKind: IndexKindBill,
	keywordIDs: func(s *Store, kw string, q SearchQuery) ([]string, error) {
		like := "%" + kw + "%"
		sqlText := `SELECT i.id FROM gf_bill_items i JOIN gf_projects p ON p.id=i.project_id
WHERE (i.title LIKE ? OR i.code LIKE ? OR COALESCE(i.gcode,'') LIKE ? OR i.feature LIKE ?)`
		args := []any{like, like, like, like}
		if q.ProjectID > 0 {
			sqlText += ` AND i.project_id=?`
			args = append(args, q.ProjectID)
		}
		if sp := filterValue(q.Specialty); sp != "" {
			sqlText += ` AND COALESCE(NULLIF(i.specialty,''), NULLIF(p.specialty,''), '')=?`
			args = append(args, sp)
		}
		sqlText += ` ORDER BY i.project_id, i.sort, i.id LIMIT ?`
		args = append(args, searchKeywordCap)
		return queryKeys(s, sqlText, args...)
	},
	load: func(s *Store, ids []string, q SearchQuery, limit int) ([]BillItem, error) {
		return s.billItemsByKeys(ids, q.ProjectID, filterValue(q.Specialty), limit)
	},
	key: func(it BillItem) string { return strconv.FormatInt(it.ID, 10) },
	flags: func(it BillItem, kw string) (bool, bool, bool) {
		return eqAny(kw, it.Code, it.GCode), has(it.Title, kw), has(it.Feature, kw) || has(it.Division, kw)
	},
	text: func(it BillItem) string {
		return BillIndexDoc(it.Code, it.GCode, it.Title, it.Feature, it.Unit, "", it.Specialty, it.Division)
	},
}

// runSearch 管线主体：关键词行 → 编码精确短路 → 向量行 → 融合 → 精排 → 标注。
func runSearch[T any](ctx context.Context, h *Searcher, spec kindSpec[T], q SearchQuery) ([]Hit[T], error) {
	limit := q.Limit
	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	if h == nil || h.store == nil || !h.store.Available() {
		return nil, fmt.Errorf("工料法存储不可用")
	}
	kw := strings.TrimSpace(q.Text)
	if kw == "" {
		rows, err := spec.load(h.store, nil, q, limit)
		if err != nil {
			return nil, err
		}
		out := make([]Hit[T], 0, min(len(rows), limit))
		for _, r := range rows {
			if len(out) >= limit {
				break
			}
			out = append(out, Hit[T]{Item: r, Via: ViaList})
		}
		return out, nil
	}

	// ── 关键词路 ──
	ids, err := spec.keywordIDs(h.store, kw, q)
	if err != nil {
		return nil, err
	}
	// 关键词路零命中就不去加载：load 把 nil 键列表当「列表模式」，不设防会把全库捞成候选。
	var rows []T
	if len(ids) > 0 {
		if rows, err = spec.load(h.store, ids, q, limit); err != nil {
			return nil, err
		}
	}
	cands := map[string]vecindex.Cand{}
	byKey := map[string]T{}
	var exact []Hit[T]
	for _, r := range rows {
		k := spec.key(r)
		ex, title, field := spec.flags(r, kw)
		if ex {
			exact = append(exact, Hit[T]{Item: r, Score: 1, Via: ViaCode})
			continue
		}
		cands[k] = vecindex.Cand{ID: k, Title: title, Field: field}
		byKey[k] = r
	}
	// 编码精确命中：定了就是它，不再补语义近似。
	if len(exact) > 0 {
		sortByKey(exact, spec.key)
		if len(exact) > limit {
			exact = exact[:limit]
		}
		return exact, nil
	}

	// ── 向量路 ──
	vecScore := map[string]float64{}
	var missing []string
	for _, hit := range h.vecHits(ctx, spec.indexKind, kw, limit) {
		if c, ok := cands[hit.ID]; ok {
			// 关键词路已经在候选里：把向量分并进去（两路都中，融合分最高）
			c.Vec = hit.Score
			cands[hit.ID] = c
			continue
		}
		if _, loaded := byKey[hit.ID]; loaded {
			continue
		}
		vecScore[hit.ID] = hit.Score
		missing = append(missing, hit.ID)
	}
	if len(missing) > 0 {
		vrows, err := spec.load(h.store, missing, q, len(missing))
		if err != nil {
			return nil, err
		}
		for _, r := range vrows {
			k := spec.key(r)
			if _, dup := cands[k]; dup {
				continue
			}
			cands[k] = vecindex.Cand{ID: k, Vec: vecScore[k]}
			byKey[k] = r
		}
	}

	// ── 融合 ──
	ordered := make([]vecindex.Cand, 0, len(cands))
	for _, c := range cands {
		ordered = append(ordered, c)
	}
	ordered = vecindex.Order(ordered)

	// ── 精排 ── 只请精排模型看融合序最靠前的若干篇，其余原序跟在后面。
	if len(ordered) > searchRerankMin && h.rerank != nil {
		if r := h.rerank(); r != nil {
			head := ordered
			if len(head) > searchRerankDocs {
				head = head[:searchRerankDocs]
			}
			docs := make([]string, len(head))
			for i, c := range head {
				docs[i] = spec.text(byKey[c.ID])
			}
			if rank, err := r.Rerank(ctx, kw, docs, len(docs)); err == nil {
				ordered = append(applyRank(head, rank), ordered[len(head):]...)
			} else {
				slog.Debug("workcost: 精排失败，按融合分返回", "kind", spec.indexKind, "error", err)
			}
		}
	}

	// ── 标注 ──
	out := make([]Hit[T], 0, min(len(ordered), limit))
	for _, c := range ordered {
		if len(out) >= limit {
			break
		}
		via := ViaSemantic
		if c.Keyword() {
			via = ViaKeyword
		}
		out = append(out, Hit[T]{Item: byKey[c.ID], Score: vecindex.Score(c), Via: via})
	}
	return out, nil
}

// vecHits 向量路：查询词向量化一次，在类目里取相似度最高的若干条。任何一步不可用都
// 返回 nil（静默让路，关键词路照常）。
func (h *Searcher) vecHits(ctx context.Context, kind, kw string, limit int) []vecindex.Hit {
	if h.vec == nil || !h.vec.Available() || h.embed == nil {
		return nil
	}
	// 类目里还没有向量（索引没建 / 模型从没开过）：直接让路，省掉一次查询词向量化。
	if meta, err := h.vec.Meta(kind); err != nil || meta.Count == 0 {
		return nil
	}
	emb := h.embed()
	if emb == nil {
		return nil
	}
	qvec, err := vecindex.Query(ctx, emb, kw)
	if err != nil || qvec == nil {
		slog.Debug("workcost: 查询词向量化失败，向量路让路", "kind", kind, "error", err)
		return nil
	}
	topN := limit * 4
	if topN < 30 {
		topN = 30
	}
	if topN > 200 {
		topN = 200
	}
	hits, err := h.vec.SearchVec(kind, qvec, topN, nil)
	if err != nil {
		slog.Debug("workcost: 向量检索不可用，向量路让路", "kind", kind, "error", err)
		return nil
	}
	return hits
}

// applyRank 按精排给出的下标顺序重排；精排没点到的留在后面（保持原序）。
func applyRank(ordered []vecindex.Cand, rank []int) []vecindex.Cand {
	next := make([]vecindex.Cand, 0, len(ordered))
	seen := make(map[int]bool, len(ordered))
	for _, i := range rank {
		if i >= 0 && i < len(ordered) && !seen[i] {
			seen[i] = true
			next = append(next, ordered[i])
		}
	}
	for i := range ordered {
		if !seen[i] {
			next = append(next, ordered[i])
		}
	}
	return next
}

func sortByKey[T any](hits []Hit[T], key func(T) string) {
	// 条数很少（编码精确命中通常一两条），插入排序够用。
	for i := 1; i < len(hits); i++ {
		for j := i; j > 0 && key(hits[j].Item) < key(hits[j-1].Item); j-- {
			hits[j], hits[j-1] = hits[j-1], hits[j]
		}
	}
}

// filterValue 过滤值的规整：空或「全部」= 不过滤。
func filterValue(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || v == "全部" {
		return ""
	}
	return v
}

// regionFilter 地区过滤值规整：空或「全部」= 不过滤；其余归一到省级全称（「四川」
// 「四川省」都归一成「四川省」，见 NormalizeRegion）——地区限省后库里一律省级
// 全称，模型偶尔给简称时兜一下；归一不动就原样返回。
func regionFilter(v string) string {
	rg := filterValue(v)
	if rg == "" {
		return ""
	}
	if p, ok := NormalizeRegion(rg); ok {
		return p
	}
	return rg
}

// filterByRegion 按地区留下工料机行（region 已经 regionFilter 归一；空=原样返回）。
// 关键词/向量路按主键回表不写地区 SQL，在这里统一收口。
func filterByRegion(rows []Resource, region string) []Resource {
	rg := regionFilter(region)
	if rg == "" {
		return rows
	}
	out := rows[:0]
	for _, r := range rows {
		if strings.TrimSpace(r.Region) == rg {
			out = append(out, r)
		}
	}
	return out
}

func has(s, kw string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(kw))
}

func eqAny(kw string, vals ...string) bool {
	for _, v := range vals {
		if strings.EqualFold(strings.TrimSpace(v), kw) {
			return true
		}
	}
	return false
}

// queryKeys 跑一条只取键的查询。
func queryKeys(s *Store, sqlText string, args ...any) ([]string, error) {
	rows, err := s.db.Query(sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// chunksOf 把键列表切成若干批（SQL 的 IN 有长度上限）。
func chunksOf(keys []string) [][]string {
	var out [][]string
	for i := 0; i < len(keys); i += searchLoadChunk {
		end := i + searchLoadChunk
		if end > len(keys) {
			end = len(keys)
		}
		out = append(out, keys[i:end])
	}
	return out
}

func inPlaceholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func keysArgs(keys []string) []any {
	args := make([]any, len(keys))
	for i, k := range keys {
		args[i] = k
	}
	return args
}

// ── 按键取行 ──────────────────────────────────────────────────────────

// quotasByKeys 按编码取定额（列与 ListQuotas 相同，含量行不随行带出）。
func (s *Store) quotasByKeys(codes []string, specialty string) ([]Quota, error) {
	var out []Quota
	for _, chunk := range chunksOf(codes) {
		q := `
SELECT id, code, COALESCE(gcode,''), COALESCE(orig_code,''), title, specialty, chapter, unit, category_path,
  base_labor, base_material, base_machine, source, region, price_date, note, status, created_at, updated_at,
  COALESCE(derived_from,''),
  (SELECT COUNT(*) FROM gf_bill_items b WHERE b.quota_code=gf_quotas.code)
FROM gf_quotas WHERE code IN (` + inPlaceholders(len(chunk)) + `)`
		args := keysArgs(chunk)
		if specialty != "" {
			q += ` AND specialty=?`
			args = append(args, specialty)
		}
		rows, err := s.db.Query(q, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var it Quota
			if e := rows.Scan(&it.ID, &it.Code, &it.GCode, &it.OrigCode, &it.Title, &it.Specialty, &it.Chapter, &it.Unit,
				&it.CategoryPath, &it.BaseLabor, &it.BaseMaterial, &it.BaseMachine, &it.Source,
				&it.Region, &it.PriceDate, &it.Note, &it.Status, &it.CreatedAt, &it.UpdatedAt, &it.DerivedFrom, &it.UsageCount); e != nil {
				_ = rows.Close()
				return nil, e
			}
			out = append(out, it)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, err
		}
		_ = rows.Close()
	}
	return out, nil
}

// resourcesByKeys 按编码取工料机。
func (s *Store) resourcesByKeys(codes []string, kind string) ([]Resource, error) {
	var out []Resource
	for _, chunk := range chunksOf(codes) {
		q := resourceCols + ` WHERE code IN (` + inPlaceholders(len(chunk)) + `)`
		args := keysArgs(chunk)
		if kind != "" {
			q += ` AND kind=?`
			args = append(args, kind)
		}
		rows, err := s.db.Query(q, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			r, e := scanResource(rows)
			if e != nil {
				continue
			}
			out = append(out, *r)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, err
		}
		_ = rows.Close()
	}
	return out, nil
}

// billItemsByKeys 按行 ID 取清单行；ids 为 nil = 跨项目按列表顺序取 limit 条。
func (s *Store) billItemsByKeys(ids []string, projectID int64, specialty string, limit int) ([]BillItem, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	var out []BillItem
	if ids == nil {
		q := billItemCols + ` WHERE 1=1`
		args := []any{}
		if projectID > 0 {
			q += ` AND i.project_id=?`
			args = append(args, projectID)
		}
		if specialty != "" {
			q += ` AND COALESCE(NULLIF(i.specialty,''), NULLIF(p.specialty,''), '')=?`
			args = append(args, specialty)
		}
		q += ` ORDER BY i.project_id DESC, i.sort, i.id LIMIT ?`
		args = append(args, limit)
		rows, err := s.db.Query(q, args...)
		if err != nil {
			return nil, err
		}
		out, err = scanBillItemsWithLinks(s, rows)
		if err != nil {
			return nil, err
		}
		return out, nil
	}
	for _, chunk := range chunksOf(ids) {
		q := billItemCols + ` WHERE i.id IN (` + inPlaceholders(len(chunk)) + `)`
		args := keysArgs(chunk)
		if projectID > 0 {
			q += ` AND i.project_id=?`
			args = append(args, projectID)
		}
		if specialty != "" {
			q += ` AND COALESCE(NULLIF(i.specialty,''), NULLIF(p.specialty,''), '')=?`
			args = append(args, specialty)
		}
		q += ` ORDER BY i.project_id, i.sort, i.id`
		rows, err := s.db.Query(q, args...)
		if err != nil {
			return nil, err
		}
		part, err := scanBillItemsWithLinks(s, rows)
		if err != nil {
			return nil, err
		}
		out = append(out, part...)
	}
	return out, nil
}

// scanBillItemsWithLinks 读出清单行并装上挂接定额（与 BillItems 同口径）。
func scanBillItemsWithLinks(s *Store, rows *sql.Rows) ([]BillItem, error) {
	defer func() { _ = rows.Close() }()
	out := []BillItem{}
	var ids []int64
	fallback := map[int64]string{}
	for rows.Next() {
		it, err := scanBillItem(rows)
		if err != nil {
			continue
		}
		ids = append(ids, it.ID)
		fallback[it.ID] = it.QuotaCode
		out = append(out, *it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	links := s.quotaLinkRowsByItems(ids, fallback)
	for i := range out {
		out[i].attachLinks(links[out[i].ID])
	}
	return out, nil
}
