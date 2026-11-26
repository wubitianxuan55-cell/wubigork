// index.go — 造价库语义索引的接入面：三类索引文档（消耗定额 / 工料机 / 分部分项清单）、
// 全量文档源（交给 internal/gaea/vecindex 的后台索引器做增量对账），以及写入打脏标记。
//
// 打脏标记不打在几十个写路径上，而是打在本包的库句柄上：Open 把句柄包成 dirtyDB /
// dirtyTx，写语句执行成功后按语句里的表名给对应类目调 vecindex.MarkDirty——现有与将来的
// 写路径一律覆盖，合并与重试由索引器负责。事务中途回滚后多打的标记无害（对账发现
// 没变化就什么都不做）。
//
// 索引正文只放「是什么」，不放会频繁变动的数（单价 / 工程量 / 基价）：调一次价不该
// 让一条工料机重嵌一遍。直接经 Store.DB() 写库的调用方不经这层拦截，漏打的写入由
// 索引器的定时兜底对账追上。
package workcost

import (
	"context"
	"database/sql"
	"regexp"
	"strconv"
	"strings"

	"github.com/gaea/gaea/internal/gaea/vecindex"
)

// 三类索引的类目名（vecindex.Store 里按它分表存向量）。
const (
	IndexKindQuota    = "wc_quota" // 消耗定额
	IndexKindResource = "wc_res"   // 工料机
	IndexKindBill     = "wc_bill"  // 分部分项清单
)

// IndexKinds 三类索引的类目名，固定顺序（对账按它串行）。
var IndexKinds = []string{IndexKindQuota, IndexKindResource, IndexKindBill}

// indexStreamBatch 文档源每批吐出的条数。
const indexStreamBatch = 512

// QuotaIndexDoc 定额的索引正文：编码 + 名称 + 专业 + 章节 + 单位 + 分类路径 + 含量行资源名。
// itemTitles 是含量行的资源名（已按稳定顺序拼好）；不含基价（核算数据，会变）。
func QuotaIndexDoc(code, gcode, title, specialty, chapter, unit, categoryPath, itemTitles string) string {
	return joinDoc(title, code, gcode, seg("专业", specialty), seg("章节", chapter), seg("单位", unit),
		categoryPath, seg("组成：", itemTitles))
}

// ResourceIndexDoc 工料机的索引正文：编码 + 名称 + 规格 + 类型 + 分类路径 + 单位。
// 不含单价（调价不该触发重嵌）。
func ResourceIndexDoc(code, gcode, title, spec, kind, unit, categoryPath string) string {
	return joinDoc(title, spec, kind, code, gcode, seg("单位", unit), categoryPath)
}

// BillIndexDoc 清单行的索引正文：编码 + 名称 + 项目特征 + 单位 + 项目名。
// 不含工程量 / 综合单价。
func BillIndexDoc(code, gcode, title, feature, unit, projectName, specialty, division string) string {
	return joinDoc(title, code, gcode, seg("单位", unit), seg("项目特征：", feature), seg("项目：", projectName),
		seg("专业", specialty), division)
}

// seg 文档的一段：值为空时整段不要（不留「专业」「单位」这类孤词）。
func seg(prefix, value string) string {
	if value = strings.TrimSpace(value); value == "" {
		return ""
	}
	return prefix + value
}

// joinDoc 把文档的各段拼成一句：去掉空段，用空格连接。
func joinDoc(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, " ")
}

// IndexSources 三类索引的全量文档源。vecindex.Source 必须把库里所有条目都吐出来
// （对账靠「全量」判断哪些向量过期），所以这里不加任何筛选。
func (s *Store) IndexSources() map[string]vecindex.Source {
	return map[string]vecindex.Source{
		IndexKindQuota:    s.quotaIndexSource(),
		IndexKindResource: s.resourceIndexSource(),
		IndexKindBill:     s.billIndexSource(),
	}
}

// streamIndex 分批把一条查询的结果吐成文档；build 从当前行算出 (ID, 正文)，ok=false
// 表示这条不进索引（没有稳定键）。
func (s *Store) streamIndex(query string, build func(scan func(dest ...any) error) (string, string, bool)) vecindex.Source {
	return func(yield func(docs []vecindex.Doc) error) error {
		if !s.Available() {
			return nil
		}
		rows, err := s.db.Query(query)
		if err != nil {
			return err
		}
		defer func() { _ = rows.Close() }()
		docs := make([]vecindex.Doc, 0, indexStreamBatch)
		for rows.Next() {
			id, text, ok := build(rows.Scan)
			if !ok {
				continue
			}
			docs = append(docs, vecindex.Doc{ID: id, Text: text})
			if len(docs) >= indexStreamBatch {
				if err := yield(docs); err != nil {
					return err
				}
				docs = make([]vecindex.Doc, 0, indexStreamBatch)
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}
		if len(docs) > 0 {
			return yield(docs)
		}
		return nil
	}
}

// quotaIndexSource 定额全量。含量行资源名按 sort, id 排好序再拼——拼法不稳定的话
// 正文哈希会来回变，白白重嵌。含量行的资源名可能没填（只给了资源编码），回查资源表的名称。
// 定额与含量行分两步取、取含量行前先关掉定额游标：库句柄若只许一个连接，嵌套读会自己卡死。
func (s *Store) quotaIndexSource() vecindex.Source {
	return func(yield func(docs []vecindex.Doc) error) error {
		if !s.Available() {
			return nil
		}
		type qrow struct{ code, gcode, title, specialty, chapter, unit, cat string }
		var last string
		for {
			page := make([]qrow, 0, indexStreamBatch)
			rows, err := s.db.Query(`
SELECT code, COALESCE(gcode,''), title, specialty, chapter, unit, category_path
FROM gf_quotas WHERE code > ? ORDER BY code LIMIT ?`, last, indexStreamBatch)
			if err != nil {
				return err
			}
			for rows.Next() {
				var q qrow
				if err := rows.Scan(&q.code, &q.gcode, &q.title, &q.specialty, &q.chapter, &q.unit, &q.cat); err != nil {
					_ = rows.Close()
					return err
				}
				page = append(page, q)
			}
			if err := rows.Err(); err != nil {
				_ = rows.Close()
				return err
			}
			_ = rows.Close()
			if len(page) == 0 {
				return nil
			}
			last = page[len(page)-1].code
			codes := make([]string, len(page))
			for i := range page {
				codes[i] = page[i].code
			}
			items, err := s.quotaItemTitles(codes)
			if err != nil {
				return err
			}
			docs := make([]vecindex.Doc, 0, len(page))
			for _, q := range page {
				if q.code == "" {
					continue
				}
				docs = append(docs, vecindex.Doc{ID: q.code, Text: QuotaIndexDoc(q.code, q.gcode, q.title, q.specialty,
					q.chapter, q.unit, q.cat, strings.Join(items[q.code], "、"))})
			}
			if len(docs) > 0 {
				if err := yield(docs); err != nil {
					return err
				}
			}
			if len(page) < indexStreamBatch {
				return nil
			}
		}
	}
}

// quotaItemTitles 取一批定额的含量行资源名（按 quota_code, sort, id 排好）。
func (s *Store) quotaItemTitles(codes []string) (map[string][]string, error) {
	out := map[string][]string{}
	if len(codes) == 0 {
		return out, nil
	}
	ph := strings.TrimSuffix(strings.Repeat("?,", len(codes)), ",")
	args := make([]any, len(codes))
	for i, c := range codes {
		args[i] = c
	}
	rows, err := s.db.Query(`
SELECT i.quota_code, COALESCE(NULLIF(TRIM(i.title),''), r.title, '')
FROM gf_quota_items i LEFT JOIN gf_resources r ON r.code = i.resource_code
WHERE i.quota_code IN (`+ph+`) ORDER BY i.quota_code, i.sort, i.id`, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var code, title string
		if err := rows.Scan(&code, &title); err != nil {
			return nil, err
		}
		out[code] = append(out[code], title)
	}
	return out, rows.Err()
}

func (s *Store) resourceIndexSource() vecindex.Source {
	return s.streamIndex(`
SELECT code, COALESCE(gcode,''), title, spec, kind, unit, category_path
FROM gf_resources ORDER BY code`, func(scan func(dest ...any) error) (string, string, bool) {
		var code, gcode, title, spec, kind, unit, cat string
		if err := scan(&code, &gcode, &title, &spec, &kind, &unit, &cat); err != nil {
			return "", "", false
		}
		return code, ResourceIndexDoc(code, gcode, title, spec, kind, unit, cat), code != ""
	})
}

// billIndexSource 清单全量：清单行的键是行 ID（编码只在项目内唯一）；项目名取自项目表。
func (s *Store) billIndexSource() vecindex.Source {
	return s.streamIndex(`
SELECT b.id, b.code, COALESCE(b.gcode,''), b.title, b.feature, b.unit,
  COALESCE(p.name,''), COALESCE(b.specialty,''), b.division
FROM gf_bill_items b LEFT JOIN gf_projects p ON p.id=b.project_id
ORDER BY b.id`, func(scan func(dest ...any) error) (string, string, bool) {
		var id int64
		var code, gcode, title, feature, unit, project, specialty, division string
		if err := scan(&id, &code, &gcode, &title, &feature, &unit, &project, &specialty, &division); err != nil {
			return "", "", false
		}
		key := strconv.FormatInt(id, 10)
		return key, BillIndexDoc(code, gcode, title, feature, unit, project, specialty, division), id != 0
	})
}

// ── 写入打脏标记 ──────────────────────────────────────────────────────

// writeTableRe 从写语句里取出被写的表名（INSERT / REPLACE / UPDATE / DELETE）。
var writeTableRe = regexp.MustCompile(`(?i)(?:INSERT\s+(?:OR\s+\w+\s+)?INTO\s+|REPLACE\s+INTO\s+|UPDATE\s+(?:OR\s+\w+\s+)?|DELETE\s+FROM\s+)([A-Za-z_][A-Za-z0-9_]*)`)

// dirtyByTable 表 → 受它影响的索引类目。清单正文含项目名，所以项目表（改名）也算。
var dirtyByTable = map[string][]string{
	"gf_quotas":      {IndexKindQuota},
	"gf_quota_items": {IndexKindQuota},
	"gf_resources":   {IndexKindResource},
	"gf_bill_items":  {IndexKindBill},
	"gf_projects":    {IndexKindBill},
}

// dirtyKinds 一条写语句涉及的索引类目（去重，按固定顺序）；读语句与非本库表返回 nil。
func dirtyKinds(query string) []string {
	var out []string
	for _, kind := range IndexKinds {
		for _, m := range writeTableRe.FindAllStringSubmatch(query, -1) {
			if hit := dirtyByTable[strings.ToLower(m[1])]; containsKind(hit, kind) {
				out = append(out, kind)
				break
			}
		}
	}
	return out
}

func containsKind(list []string, kind string) bool {
	for _, k := range list {
		if k == kind {
			return true
		}
	}
	return false
}

// markDirtySQL 给一条已执行成功的写语句涉及的索引类目打脏标记。
func markDirtySQL(query string) {
	for _, kind := range dirtyKinds(query) {
		vecindex.MarkDirty(kind)
	}
}

// dirtyDB / dirtyTx / dirtyStmt 库句柄的薄包装：只拦写语句，读语句原样透传。
type dirtyDB struct{ *sql.DB }
type dirtyTx struct{ *sql.Tx }
type dirtyStmt struct {
	*sql.Stmt
	query string
}

func (d *dirtyDB) Exec(query string, args ...any) (sql.Result, error) {
	res, err := d.DB.Exec(query, args...)
	if err == nil {
		markDirtySQL(query)
	}
	return res, err
}

func (d *dirtyDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	res, err := d.DB.ExecContext(ctx, query, args...)
	if err == nil {
		markDirtySQL(query)
	}
	return res, err
}

func (d *dirtyDB) Prepare(query string) (*dirtyStmt, error) {
	st, err := d.DB.Prepare(query)
	if err != nil {
		return nil, err
	}
	return &dirtyStmt{Stmt: st, query: query}, nil
}

func (d *dirtyDB) PrepareContext(ctx context.Context, query string) (*dirtyStmt, error) {
	st, err := d.DB.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return &dirtyStmt{Stmt: st, query: query}, nil
}

func (d *dirtyDB) Begin() (*dirtyTx, error) {
	tx, err := d.DB.Begin()
	if err != nil {
		return nil, err
	}
	return &dirtyTx{Tx: tx}, nil
}

func (d *dirtyDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*dirtyTx, error) {
	tx, err := d.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &dirtyTx{Tx: tx}, nil
}

func (t *dirtyTx) Exec(query string, args ...any) (sql.Result, error) {
	res, err := t.Tx.Exec(query, args...)
	if err == nil {
		markDirtySQL(query)
	}
	return res, err
}

func (t *dirtyTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	res, err := t.Tx.ExecContext(ctx, query, args...)
	if err == nil {
		markDirtySQL(query)
	}
	return res, err
}

func (t *dirtyTx) Prepare(query string) (*dirtyStmt, error) {
	st, err := t.Tx.Prepare(query)
	if err != nil {
		return nil, err
	}
	return &dirtyStmt{Stmt: st, query: query}, nil
}

func (st *dirtyStmt) Exec(args ...any) (sql.Result, error) {
	res, err := st.Stmt.Exec(args...)
	if err == nil {
		markDirtySQL(st.query)
	}
	return res, err
}

func (st *dirtyStmt) ExecContext(ctx context.Context, args ...any) (sql.Result, error) {
	res, err := st.Stmt.ExecContext(ctx, args...)
	if err == nil {
		markDirtySQL(st.query)
	}
	return res, err
}
