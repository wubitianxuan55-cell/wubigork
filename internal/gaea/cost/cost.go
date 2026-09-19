// Package cost — 成本库（记忆中枢扩展库）
//
// 成本条目：单价/单位/规格/来源，供方案测算与预结算复用。存储于
// Hephaestus.db cost_entries 表（schema V2）。与 knowledge 同模式：
// 显式、可编辑、可分类检索。
package cost

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/gaea/bm25"
	"github.com/gaea/gaea/internal/gaea/strutil"
)

// Entry 成本条目。
type Entry struct {
	Name         string
	Title        string
	Code         string // 定额编码/清单编码（归一化：半角大写、去空白；空=未录入）
	Category     string // 叶子分类名（兼容旧工具/展示）
	CategoryPath string // 完整分类路径：一级/二级/…/叶子（树形过滤与分组依据）
	Unit         string // 台班/吨/m³/工日…
	Price        float64
	// 人材机二级汇总（综合单价子目口径）：人工费/材料费/机械费 金额。
	// 与 Components 明细对应；组件金额合计与列值允许微小出入（导入原值保留）。
	LaborFee    float64
	MaterialFee float64
	MachineFee  float64
	// 费率（仅展示追溯，不参与计算）：管理费/利润/垫资为金额（元），税率为百分比。
	ManagementFee float64
	ProfitFee     float64
	AdvanceFee    float64
	TaxRate       float64
	Spec          string
	Source        string // 定额/市场询价/历史项目…
	Region        string // 地区（如 成都市区/上海）：价格三要素之一
	PriceDate     string // 价格时间/期数（如 2026-08 / 2026年第2期）
	PriceType     string // 价格口径：出厂价/到场价/安装综合价
	ValidUntil    string // 有效期至（YYYY-MM-DD，空=长期有效）
	SourceRow     int    // 导入原始工作表行号（0=手动录入未标注）
	Tags          []string
	Status        string // 现行/草稿/已归档
	Body          string
	Components    []Component // 人材机二级组成明细（综合单价子目内部）
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Component 综合单价子目的人材机组成明细行（二级）。
// Kind 取值 人工/材料/机械，或保留源文件合并段标签（如 人工+机械）。
// Note 保留原始行表达式（含损耗系数等），保证追溯。
type Component struct {
	Kind     string  // 人工/材料/机械（可合并标签）
	Title    string  // 资源/工作名称
	Unit     string  // 单位（可空）
	Quantity float64 // 含量/数量（0=未解析出）
	Price    float64 // 资源单价（0=未解析出）
	Amount   float64 // 金额（含量×单价，含损耗）
	Note     string  // 原始行表达式/备注
	Sort     int
}

// Summary 轻量视图（无 Body）。
type Summary struct {
	Name           string
	Title          string
	Code           string // 定额编码/清单编码（归一化；空=未录入）
	Category       string
	CategoryPath   string
	Unit           string
	Price          float64
	LaborFee       float64
	MaterialFee    float64
	MachineFee     float64
	ComponentCount int // 人材机组成行数（综合单价子目的二级明细规模）
	Spec           string
	Source         string
	Region         string
	PriceDate      string
	PriceType      string
	ValidUntil     string
	SourceRow      int
	Tags           []string
	Status         string
	UpdatedAt      time.Time
}

// Category 成本分类树节点（可任意层级）。
type Category struct {
	ID        int
	ParentID  int
	Name      string
	Sort      int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CategoryView 分类节点视图：直接归属该节点的条目数 + 子节点树。
type CategoryView struct {
	ID       int
	ParentID int
	Name     string
	Sort     int
	Count    int
	Children []*CategoryView
}

// Store 成本库存储（Hephaestus.db）。
type Store struct {
	db *sql.DB
}

// BM25 排序缓存（刀D v4.247，T7-3 原设计接通；包级——Store 即建即弃，
// openCostStore/hubCostStore 每次调用都 cost.Open 新实例）。key=db 池|数据
// 版本|category|status 过滤形态，语料=该形态下 SQL 全捞的条目集（name 序），
// 查询只对关键词命中子集取分。改前每查询对命中子集从零重建倒排（2000 条
// 库 20.6ms/22MB 分配）。写路径推进版本（bumpRankVersion/InvalidateRankers）
// 旧 Ranker 自然失效；map 超 16 项整体清空防任意过滤值撑大。
var (
	rankMu      sync.Mutex
	rankVersion atomic.Uint64
	rankers     = map[string]rankerEntry{}
)

type rankerEntry struct {
	ranker *bm25.Ranker
}

// bumpRankVersion 推进数据版本（写路径成功后调用，同包内直接访问）。
func bumpRankVersion() { rankVersion.Add(1) }

// InvalidateRankers 使全部 BM25 排序缓存失效（包外写路径用：app 层批量
// 导入直写 cost_entries 不经 Store.Save，提交后必须调用）。
func InvalidateRankers() { bumpRankVersion() }

// rankerFor 返回当前过滤形态与数据版本下的 BM25 打分器：语料=all（该
// category/status 过滤下 SQL 全捞的条目，name 序）。命中直接复用；版本
// 推进或 key 首见时构建。version 由调用方在捞语料**之前**快照传入——
// 语料与版本戳取自同一时点（2026-09-19 审计）：此前 rankerFor 内部再
// Load 一次版本，若写路径在捞语料与构 key 之间推进版本，旧语料会挂到
// 新版本 key 上一直用到下次写。
func rankerFor(db *sql.DB, version uint64, category, status string, all []Summary) *bm25.Ranker {
	key := fmt.Sprintf("%p|%d|%s|%s", db, version, category, status)
	rankMu.Lock()
	defer rankMu.Unlock()
	if e, ok := rankers[key]; ok {
		return e.ranker
	}
	if len(rankers) > 16 {
		rankers = map[string]rankerEntry{}
	}
	docs := make([]bm25.Doc, len(all))
	for i, e := range all {
		docs[i] = bm25.Doc{ID: i, Text: summaryDocText(e)}
	}
	r := bm25.NewRanker(docs)
	rankers[key] = rankerEntry{ranker: r}
	return r
}

// summaryDocText 把条目摘要拼成 BM25 文档串（名称/标题/编码/单位/规格/
// 来源/地区/价格形态/期数/标签）。
func summaryDocText(e Summary) string {
	return e.Name + " " + e.Title + " " + e.Code + " " + e.Unit + " " + e.Spec + " " + e.Source + " " + e.Region + " " + e.PriceType + " " + e.PriceDate + " " + strings.Join(e.Tags, " ")
}

// Open 打开成本库；gdb 为 nil 时返回不可用 store。
// 打开后播种默认分类树并执行幂等自愈（分类路径修复 + 价格元数据回填）。
func Open(gdb *sql.DB) *Store {
	s := &Store{db: gdb}
	s.EnsureDefaultCategories()
	s.SelfHeal()
	return s
}

// NormalizeCode 归一化条目编码：全角转半角、去全部空白、转大写。
// 定额/清单编码在不同表格中写作「A-1-12」「ａ1 12」「a1-12」等形态，
// 归一化后同码可精确命中；空串原样返回（未录入语义不变）。
func NormalizeCode(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == ' ' || r == '\u00a0' || r == '\u3000' || r == '\t':
			continue // 去空白（含全角空格/nbsp）
		case r >= '！' && r <= '～': // 全角 ASCII 区 → 半角
			r -= 0xFEE0
		}
		b.WriteRune(unicode.ToUpper(r))
	}
	return b.String()
}

// Available 报告存储是否可用。
func (s *Store) Available() bool { return s.db != nil }

// DB 暴露底层数据库句柄(委托)。仅供 app 层直写「快照类型归属 app、
// 不宜反向依赖 cost 包」的伴生表(如 v4.158.0 组价确认留痕 cost_compose_records);
// 成本库自身表(cost_entries/cost_entry_components 等)仍应走本包方法,
// 本包不做通用 SQL 网关。db 为 nil(不可用)时返回 nil,调用方自行判空。
func (s *Store) DB() *sql.DB { return s.db }

// Save 写入/更新一条成本条目（同名 UPSERT）。
func (s *Store) Save(e Entry) error {
	if s.db == nil {
		return fmt.Errorf("cost store unavailable")
	}
	if strings.TrimSpace(e.Name) == "" {
		return fmt.Errorf("cost entry needs a name")
	}
	if strings.TrimSpace(e.CategoryPath) == "" {
		e.CategoryPath = strings.TrimSpace(e.Category)
	}
	if strings.TrimSpace(e.Category) == "" {
		e.Category = leafOfPath(e.CategoryPath)
	}
	e.Code = NormalizeCode(e.Code)
	now := time.Now().UTC()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	e.UpdatedAt = now
	tags := "[]"
	if len(e.Tags) > 0 {
		if b, err := json.Marshal(e.Tags); err == nil {
			tags = string(b)
		}
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`
INSERT INTO cost_entries(name, title, code, category, category_path, unit, price, labor_fee, material_fee, machine_fee, management_fee, profit_fee, advance_fee, tax_rate, spec, source, region, price_date, price_type, valid_until, source_row, tags, status, body, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(name) DO UPDATE SET
  title=excluded.title, code=excluded.code, category=excluded.category, category_path=excluded.category_path, unit=excluded.unit,
  price=excluded.price, labor_fee=excluded.labor_fee, material_fee=excluded.material_fee,
  machine_fee=excluded.machine_fee, management_fee=excluded.management_fee,
  profit_fee=excluded.profit_fee, advance_fee=excluded.advance_fee, tax_rate=excluded.tax_rate,
  spec=excluded.spec, source=excluded.source,
  region=excluded.region, price_date=excluded.price_date, price_type=excluded.price_type,
  valid_until=excluded.valid_until, source_row=excluded.source_row,
  tags=excluded.tags, status=excluded.status, body=excluded.body,
  updated_at=excluded.updated_at`,
		e.Name, e.Title, e.Code, e.Category, e.CategoryPath, e.Unit, e.Price,
		e.LaborFee, e.MaterialFee, e.MachineFee,
		e.ManagementFee, e.ProfitFee, e.AdvanceFee, e.TaxRate,
		e.Spec, e.Source,
		e.Region, e.PriceDate, e.PriceType, e.ValidUntil, e.SourceRow,
		tags, e.Status, e.Body, e.CreatedAt.Format(time.RFC3339), e.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return err
	}
	// 人材机二级组成：整组替换（同名 UPSERT 语义一致）。
	if _, err := tx.Exec("DELETE FROM cost_entry_components WHERE entry_name=?", e.Name); err != nil {
		return err
	}
	for i, c := range e.Components {
		if strings.TrimSpace(c.Title) == "" {
			continue
		}
		if _, err := tx.Exec(`
INSERT INTO cost_entry_components(entry_name, kind, title, unit, quantity, price, amount, note, sort, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
			e.Name, c.Kind, c.Title, c.Unit, c.Quantity, c.Price, c.Amount, c.Note, i, now.Format(time.RFC3339), now.Format(time.RFC3339)); err != nil {
			return err
		}
	}
	err = tx.Commit()
	if err == nil {
		bumpRankVersion()
	}
	return err
}

// Get 按名读取完整条目。
func (s *Store) Get(name string) (*Entry, error) {
	if s.db == nil {
		return nil, fmt.Errorf("cost store unavailable")
	}
	var e Entry
	var tags, created, updated string
	err := s.db.QueryRow(`
SELECT name, title, code, category, category_path, unit, price, labor_fee, material_fee, machine_fee, management_fee, profit_fee, advance_fee, tax_rate, spec, source, region, price_date, price_type, valid_until, source_row, tags, status, body, created_at, updated_at
FROM cost_entries WHERE name=?`, name).Scan(
		&e.Name, &e.Title, &e.Code, &e.Category, &e.CategoryPath, &e.Unit, &e.Price,
		&e.LaborFee, &e.MaterialFee, &e.MachineFee,
		&e.ManagementFee, &e.ProfitFee, &e.AdvanceFee, &e.TaxRate,
		&e.Spec, &e.Source,
		&e.Region, &e.PriceDate, &e.PriceType, &e.ValidUntil, &e.SourceRow,
		&tags, &e.Status, &e.Body, &created, &updated)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("cost entry %q not found", name)
		}
		return nil, err
	}
	e.Tags = parseTagsJSON(tags)
	e.CreatedAt, _ = time.Parse(time.RFC3339, created)
	e.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	e.Components = s.componentsOf(name)
	return &e, nil
}

// componentsOf 读取条目的全部人材机组成行（按 sort 排序）。
func (s *Store) componentsOf(name string) []Component {
	rows, err := s.db.Query(`
SELECT kind, title, unit, quantity, price, amount, note, sort
FROM cost_entry_components WHERE entry_name=? ORDER BY sort, id`, name)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Component
	for rows.Next() {
		var c Component
		if err := rows.Scan(&c.Kind, &c.Title, &c.Unit, &c.Quantity, &c.Price, &c.Amount, &c.Note, &c.Sort); err != nil {
			continue
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		slog.Warn("cost: 组成行迭代中断，返回部分数据", "entry", name, "error", err)
	}
	return out
}

// Delete 删除条目。
func (s *Store) Delete(name string) error {
	if s.db == nil {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM cost_entry_components WHERE entry_name=?", name); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM cost_entries WHERE name=?", name); err != nil {
		return err
	}
	err = tx.Commit()
	if err == nil {
		bumpRankVersion()
	}
	return err
}

// List 返回全部摘要（按 name 排序）。
func (s *Store) List() []Summary {
	return s.Search("", "", "")
}

// Search 检索成本条目：关键词匹配名称/标题/规格/来源/标签/正文，
// category/status 过滤。
func (s *Store) Search(query, category, status string) []Summary {
	if s.db == nil {
		return nil
	}
	var conds []string
	var args []interface{}
	if strings.TrimSpace(category) != "" && category != "all" {
		// 分类参数支持三种形态：完整路径（含子树）、叶子名（兼容旧数据）、
		// 以及 category_path 为空但 category 精确匹配的旧条目。
		conds = append(conds, "(category_path = ? OR category_path LIKE ? ESCAPE '\\' OR (category_path = '' AND category = ?))")
		args = append(args, category, escapeLike(category)+"/%", category)
	}
	if strings.TrimSpace(status) != "" && status != "all" {
		conds = append(conds, "status = ?")
		args = append(args, status)
	}
	sqlText := "SELECT name, title, code, category, category_path, unit, price, labor_fee, material_fee, machine_fee, COALESCE(cc.cnt, 0), spec, source, region, price_date, price_type, valid_until, source_row, tags, status, updated_at FROM cost_entries LEFT JOIN (SELECT entry_name, COUNT(*) AS cnt FROM cost_entry_components GROUP BY entry_name) cc ON cc.entry_name = cost_entries.name"
	if len(conds) > 0 {
		sqlText += " WHERE " + strings.Join(conds, " AND ")
	}
	sqlText += " ORDER BY name"

	// 语料版本快照（先于 SQL 捞取）：写路径在捞语料期间推进版本时，本查询
	// 语料按旧版本挂 key，下次写路径推进后自然失效——语料与版本戳原子。
	corpusVer := rankVersion.Load()
	rows, err := s.db.Query(sqlText, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var all []Summary
	for rows.Next() {
		var sm Summary
		var tags, updated string
		if err := rows.Scan(&sm.Name, &sm.Title, &sm.Code, &sm.Category, &sm.CategoryPath, &sm.Unit, &sm.Price,
			&sm.LaborFee, &sm.MaterialFee, &sm.MachineFee, &sm.ComponentCount, &sm.Spec, &sm.Source,
			&sm.Region, &sm.PriceDate, &sm.PriceType, &sm.ValidUntil, &sm.SourceRow, &tags, &sm.Status, &updated); err != nil {
			continue
		}
		sm.Tags = parseTagsJSON(tags)
		sm.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		all = append(all, sm)
	}
	if err := rows.Err(); err != nil {
		slog.Warn("cost: 检索语料迭代中断，返回部分数据", "error", err)
	}
	// 关键词在 Go 侧做包含过滤：按词拆分（词间 AND、字段间 OR），
	// 精确子串匹配。刻意不在 SQL 里拼 6 列 OR LIKE 链——modernc/sqlite
	// 对特定形状的长 OR 链存在返回空集的怪癖（单列 LIKE 正常）。
	var out []Summary
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		out = all
		// 空查询保持 name 排序（SQL 已 ORDER BY name，此处仅为兜底保证确定性）。
		sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
		return out
	}
	terms := strings.Fields(q)
	gis := make([]int, 0, len(all)) // 命中条目在 all（name 序语料）中的下标
	for gi, e := range all {
		hay := strings.ToLower(e.Name + "\x00" + e.Title + "\x00" + e.Code + "\x00" + e.Category + "\x00" + e.CategoryPath + "\x00" + e.Unit + "\x00" + e.Spec + "\x00" + e.Source + "\x00" + e.Region + "\x00" + e.PriceType + "\x00" + e.PriceDate + "\x00" + strings.Join(e.Tags, " "))
		ok := true
		for _, term := range terms {
			if !strings.Contains(hay, term) {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, e)
			gis = append(gis, gi)
		}
	}
	// BM25 本地排序（零 token）：命中词越多/密度越高排越前，未命中 BM25
	// 的纯子串命中条目保持原顺序排在后面。打分器按「db 池+数据版本+过滤
	// 形态」缓存（语料=all 全量，改前=命中子集从零重建）；命中子集按其
	// 语料下标取分，同分按语料序（=name 序，与改前 tie-break 一致）。
	if len(out) > 1 {
		if r := rankerFor(s.db, corpusVer, category, status, all); r != nil {
			scored := r.Rank(query)
			if len(scored) > 0 {
				scoreOf := make(map[int]float64, len(scored))
				for _, sc := range scored {
					scoreOf[sc.ID] = sc.Score
				}
				type hitEntry struct {
					gi int
					sc float64
					e  Summary
				}
				hits := make([]hitEntry, 0, len(out))
				rest := make([]Summary, 0, len(out))
				for i, e := range out {
					if sc, ok := scoreOf[gis[i]]; ok {
						hits = append(hits, hitEntry{gi: gis[i], sc: sc, e: e})
					} else {
						rest = append(rest, e)
					}
				}
				sort.Slice(hits, func(a, b int) bool {
					if d := hits[a].sc - hits[b].sc; d > 0.0001 {
						return true
					} else if d < -0.0001 {
						return false
					}
					return hits[a].gi < hits[b].gi
				})
				ranked := make([]Summary, 0, len(out))
				for _, h := range hits {
					ranked = append(ranked, h.e)
				}
				out = append(ranked, rest...)
			}
		}
	}
	return out
}

func parseTagsJSON(raw string) []string {
	var tags []string
	if strings.TrimSpace(raw) == "" || raw == "[]" {
		return nil
	}
	_ = json.Unmarshal([]byte(raw), &tags)
	return tags
}

// ── 多级分类树（按分类分级保存）──────────────────────────────────

// defaultCategories 默认分类树（{父名, 名称}，父名为空=一级；顺序即层级，
// 父节点必须先于子节点）。
//
// 结构（2026-08-19 数据库梳理后与真实库对齐）：树承载两类内容——
//   - 资源库层：人工（普工/技工/特殊工种）、材料（土建/安装/周转/辅助/市政绿化）、
//     机械（土方/桩基/起重/运输/混凝土/钢筋）、运输、检测、其他（管理费/税费/措施费/
//     处置/服务）——信息价/材料价/人工费/机械费条目按此归类；
//   - 综合单价层（用户定调，对标《市政成本测算手册》）：一级=综合单价，二级=专业
//     （道路/交通/绿化/电力/给水/暖气/雨污/照明/房建/其他工程），三级=分部；
//     房建工程 收纳房建成本测算手册条目（给排水/电气/通风空调/采暖/弱电/土建/单方指标）。
//
// 名称全局唯一（同父同名唯一索引兜底）；用户可随时增删改。
var defaultCategories = [][2]string{
	// 资源库层
	{"", "人工"}, {"", "材料"}, {"", "机械"}, {"", "运输"}, {"", "检测"},
	{"", "综合单价"}, {"", "其他"},
	{"人工", "普工"}, {"人工", "技工"}, {"人工", "特殊工种"},
	{"材料", "土建材料"}, {"材料", "安装材料"}, {"材料", "周转材料"}, {"材料", "辅助材料"}, {"材料", "市政绿化材料"},
	{"机械", "土方机械"}, {"机械", "桩基机械"}, {"机械", "起重机械"}, {"机械", "运输机械"}, {"机械", "混凝土机械"}, {"机械", "钢筋机械"},
	{"运输", "场内运输"}, {"运输", "场外运输"},
	{"检测", "材料检测"}, {"检测", "实体检测"},
	{"其他", "管理费"}, {"其他", "税费"}, {"其他", "措施费"}, {"其他", "处置"}, {"其他", "服务"},
	{"土建材料", "水泥及水泥制品"}, {"土建材料", "砖瓦灰砂石"}, {"土建材料", "钢材"},
	{"土建材料", "木材及竹木制品"}, {"土建材料", "防水材料"}, {"土建材料", "保温吸声材料"},
	{"土建材料", "装饰石材"}, {"土建材料", "墙面天棚及屋面饰面材料"}, {"土建材料", "玻璃及玻璃制品"},
	{"安装材料", "电线电缆"}, {"安装材料", "管材管件"}, {"安装材料", "阀门"}, {"安装材料", "灯具照明"},
	{"安装材料", "消防器材"}, {"安装材料", "通风空调"}, {"安装材料", "电气配件"},
	{"周转材料", "模板"}, {"周转材料", "脚手架"}, {"周转材料", "扣件"},
	{"辅助材料", "临建设施"}, {"辅助材料", "燃料火工"}, {"辅助材料", "土工合成材料"},

	// 综合单价层：一级 = 综合单价
	{"综合单价", "土方"}, {"综合单价", "混凝土"}, {"综合单价", "钢筋"},
	{"综合单价", "装饰装修"}, {"综合单价", "修复处置"},
	{"综合单价", "道路工程"}, {"综合单价", "交通工程"}, {"综合单价", "绿化工程"},
	{"综合单价", "电力工程"}, {"综合单价", "给水工程"}, {"综合单价", "暖气工程"},
	{"综合单价", "雨污工程"}, {"综合单价", "照明工程"}, {"综合单价", "其他工程"},
	{"综合单价", "房建工程"},
	// 二级 = 专业 → 三级 = 分部
	{"道路工程", "土方工程"}, {"道路工程", "地基处理"}, {"道路工程", "机动车道"},
	{"道路工程", "非机动车道"}, {"道路工程", "人行道"}, {"道路工程", "附属构造"}, {"道路工程", "拆除工程"},
	{"交通工程", "标识标牌"}, {"交通工程", "标线"}, {"交通工程", "信号灯"},
	{"绿化工程", "土方工程"}, {"绿化工程", "乔木"}, {"绿化工程", "灌木"}, {"绿化工程", "地被"},
	{"电力工程", "土方工程"}, {"电力工程", "管沟与井室"}, {"电力工程", "电缆敷设"}, {"电力工程", "设备安装"},
	{"给水工程", "土方工程"}, {"给水工程", "管道铺设"}, {"给水工程", "井室及附件"},
	{"暖气工程", "土方工程"}, {"暖气工程", "管道铺设"}, {"暖气工程", "井室及附件"},
	{"雨污工程", "土方工程"}, {"雨污工程", "管道铺设"}, {"雨污工程", "检查井及雨水口"},
	{"照明工程", "基础工程"}, {"照明工程", "灯杆灯具安装"}, {"照明工程", "电缆敷设"},
	{"其他工程", "拆除工程"}, {"其他工程", "临时设施"},
	// 房建工程：房建成本测算手册章节
	{"房建工程", "给排水工程"}, {"房建工程", "电气工程"}, {"房建工程", "通风空调工程"},
	{"房建工程", "采暖工程"}, {"房建工程", "弱电工程"}, {"房建工程", "土建工程"}, {"房建工程", "单方指标"},
}

// EnsureDefaultCategories 幂等播种默认分类树（已存在节点跳过）。
func (s *Store) EnsureDefaultCategories() {
	if s.db == nil {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	ids := map[string]int{"": 0}
	for _, c := range defaultCategories {
		parent, ok := ids[c[0]]
		if !ok {
			continue
		}
		var id int
		err := s.db.QueryRow("SELECT id FROM cost_categories WHERE parent_id=? AND name=?", parent, c[1]).Scan(&id)
		if err == sql.ErrNoRows {
			if res, e := s.db.Exec("INSERT OR IGNORE INTO cost_categories(parent_id, name, sort, created_at, updated_at) VALUES(?,?,?,?,?)",
				parent, c[1], 0, now, now); e == nil && res != nil {
				if last, e := res.LastInsertId(); e == nil {
					id = int(last)
				}
			}
			if id == 0 {
				_ = s.db.QueryRow("SELECT id FROM cost_categories WHERE parent_id=? AND name=?", parent, c[1]).Scan(&id)
			}
		}
		if id > 0 {
			// 子节点以其自身名称登记，父节点查找时按父名命中（默认分类名全局唯一）。
			ids[c[1]] = id
		}
	}
}

// Categories 返回完整分类树（含每节点直接条目数与子树）。
func (s *Store) Categories() []CategoryView {
	if s.db == nil {
		return nil
	}
	rows, err := s.db.Query("SELECT id, parent_id, name, sort FROM cost_categories ORDER BY sort, id")
	if err != nil {
		return nil
	}
	defer rows.Close()

	type node struct {
		view   CategoryView
		parent int
	}
	var nodes []node
	names := map[int]string{}
	parentOf := map[int]int{}
	for rows.Next() {
		var n node
		if err := rows.Scan(&n.view.ID, &n.view.ParentID, &n.view.Name, &n.view.Sort); err != nil {
			continue
		}
		n.parent = n.view.ParentID
		names[n.view.ID] = n.view.Name
		parentOf[n.view.ID] = n.view.ParentID
		nodes = append(nodes, n)
	}
	if err := rows.Err(); err != nil {
		slog.Warn("cost: 分类树读取迭代中断", "error", err)
	}
	if len(nodes) == 0 {
		return nil
	}

	// 解析每节点完整路径（父链递归 + memo）。onPath 环保护（2026-09-19
	// 审计）：memo 只记完成态，环中节点互等永远等不到——数据带环时此处
	// 无限递归栈溢出，遇环诚实截断。
	pathMemo := map[int]string{}
	onPath := map[int]bool{}
	var resolve func(id int) string
	resolve = func(id int) string {
		if p, ok := pathMemo[id]; ok {
			return p
		}
		if onPath[id] {
			pathMemo[id] = names[id]
			return names[id]
		}
		parent := parentOf[id]
		if parent == 0 {
			pathMemo[id] = names[id]
			return names[id]
		}
		onPath[id] = true
		pathMemo[id] = resolve(parent) + "/" + names[id]
		delete(onPath, id)
		return pathMemo[id]
	}

	// 直接条目计数（category_path 精确等于节点路径）。
	counts := map[string]int{}
	if crow, e := s.db.Query("SELECT category_path, COUNT(*) FROM cost_entries WHERE category_path != '' GROUP BY category_path"); e == nil {
		for crow.Next() {
			var p string
			var n int
			if crow.Scan(&p, &n) == nil {
				counts[p] = n
			}
		}
		if err := crow.Err(); err != nil {
			slog.Warn("cost: 条目计数迭代中断", "error", err)
		}
		crow.Close()
	}

	byParent := map[int][]*CategoryView{}
	for i := range nodes {
		v := &nodes[i].view
		v.Count = counts[resolve(v.ID)]
		byParent[v.ParentID] = append(byParent[v.ParentID], v)
	}
	for _, list := range byParent {
		sort.Slice(list, func(i, j int) bool {
			if list[i].Sort != list[j].Sort {
				return list[i].Sort < list[j].Sort
			}
			return list[i].ID < list[j].ID
		})
	}
	for i := range nodes {
		nodes[i].view.Children = byParent[nodes[i].view.ID]
	}
	var roots []CategoryView
	for _, v := range byParent[0] {
		roots = append(roots, *v)
	}
	return roots
}

// CategoryPath 返回分类节点的完整路径（"一级/二级/…/名称"）。
func (s *Store) CategoryPath(id int) string {
	if s.db == nil || id <= 0 {
		return ""
	}
	var parts []string
	cur := id
	seen := map[int]bool{}
	for cur > 0 && !seen[cur] {
		seen[cur] = true
		var name string
		var parent int
		if err := s.db.QueryRow("SELECT name, parent_id FROM cost_categories WHERE id=?", cur).Scan(&name, &parent); err != nil {
			return ""
		}
		parts = append([]string{name}, parts...)
		cur = parent
	}
	return strings.Join(parts, "/")
}

// SaveCategory 新建/更新分类节点。
//   - id <= 0：新建（同父同名幂等，冲突时返回既有 id）；
//   - id > 0：更新名称/父节点/排序；改名或换父时同步重写该子树下成本条目的
//     category_path（2026-09-19 审计 P0：原实现只在改名时重写，且用 Go 字节
//     数喂 SQLite 按字符计数的 substr——中文路径下子树后缀整段截断/错位；
//     现改为 rune 计数偏移 + 精确行同步叶子名，节点与条目两写包同一事务）。
func (s *Store) SaveCategory(parentID int, name string, sort int, id int) (int, error) {
	if s.db == nil {
		return 0, fmt.Errorf("cost store unavailable")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("分类名称不能为空")
	}
	// 名称是路径拼装的分隔符语义字符：入库即破坏 CategoryPath 树解析
	// （"A/B" 名让 LIKE 前缀重写命中 newPath 自身），创建/改名一律拒绝。
	if strings.Contains(name, "/") {
		return 0, fmt.Errorf("分类名称不能包含 /")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if id <= 0 {
		var existing int
		err := s.db.QueryRow("SELECT id FROM cost_categories WHERE parent_id=? AND name=?", parentID, name).Scan(&existing)
		if err == nil {
			return existing, nil
		}
		res, err := s.db.Exec("INSERT INTO cost_categories(parent_id, name, sort, created_at, updated_at) VALUES(?,?,?,?,?)",
			parentID, name, sort, now, now)
		if err != nil {
			return 0, fmt.Errorf("新建分类失败: %w", err)
		}
		id64, _ := res.LastInsertId()
		return int(id64), nil
	}

	var oldName string
	var oldParent int
	if err := s.db.QueryRow("SELECT name, parent_id FROM cost_categories WHERE id=?", id).Scan(&oldName, &oldParent); err != nil {
		return 0, fmt.Errorf("分类不存在: %w", err)
	}
	// 环检测：新父不能是自己或自己的祖先链上的节点（挂到后代下会让
	// Categories 的 resolve 递归无 memo 可命中而栈溢出，CategoryPath 的
	// seen 只能自保）。
	if parentID == id {
		return 0, fmt.Errorf("不能把分类挂到自己名下")
	}
	for cur, seen := parentID, map[int]bool{}; cur > 0 && !seen[cur]; seen[cur] = true {
		var parent int
		if err := s.db.QueryRow("SELECT parent_id FROM cost_categories WHERE id=?", cur).Scan(&parent); err != nil {
			break // 父链断裂（悬空引用）：不再向上追究
		}
		if parent == id {
			return 0, fmt.Errorf("不能把分类挂到自己的子孙节点下（会形成环）")
		}
		cur = parent
	}

	// 改名或换父时重写子树条目路径。oldPath/newPath 在节点 UPDATE 前算好：
	// oldPath 读旧树，newPath 的父链不受本次 UPDATE 影响（环已拒绝）。
	rewrite := name != oldName || parentID != oldParent
	var oldPath, newPath string
	if rewrite {
		oldPath = s.CategoryPathOf(oldParent, oldName)
		newPath = s.CategoryPathOf(parentID, name)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("开启分类更新事务失败: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec("UPDATE cost_categories SET name=?, parent_id=?, sort=?, updated_at=? WHERE id=?",
		name, parentID, sort, now, id); err != nil {
		return 0, fmt.Errorf("更新分类失败: %w", err)
	}
	if rewrite && oldPath != "" && newPath != "" {
		// 直接子条目：路径精确替换 + 叶子名同步（category 列存直接父名）。
		if _, err := tx.Exec(`UPDATE cost_entries SET category_path=?, category=? WHERE category_path=?`,
			newPath, name, oldPath); err != nil {
			return 0, fmt.Errorf("重写子树条目路径失败: %w", err)
		}
		// 更深子树条目：仅重写路径前缀。偏移按 rune 计数（SQLite substr 对
		// 文本按 UTF-8 字符计数，Go len 是字节数——中文路径下字节偏移会把
		// 后缀截断/错位）。
		runeOff := utf8.RuneCountInString(oldPath) + 1
		if _, err := tx.Exec(
			`UPDATE cost_entries SET category_path = ? || substr(category_path, ?)
			 WHERE category_path LIKE ? ESCAPE '\'`,
			newPath, runeOff, escapeLike(oldPath)+"/%"); err != nil {
			return 0, fmt.Errorf("重写子树条目路径失败: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("提交分类更新事务失败: %w", err)
	}
	bumpRankVersion()
	return id, nil
}

// CategoryPathOf 由父节点与名称拼出完整路径（无需数据库节点）。
func (s *Store) CategoryPathOf(parentOf int, nameOf string) string {
	if p := s.CategoryPath(parentOf); p != "" {
		return p + "/" + nameOf
	}
	return nameOf
}

// DeleteCategory 删除分类节点：存在子节点或条目时拒绝（提示先处理）。
func (s *Store) DeleteCategory(id int) error {
	if s.db == nil {
		return nil
	}
	var childCount int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM cost_categories WHERE parent_id=?", id).Scan(&childCount); err != nil {
		return err
	}
	if childCount > 0 {
		return fmt.Errorf("分类下还有 %d 个子分类，请先删除或移走子分类", childCount)
	}
	path := s.CategoryPath(id)
	if path != "" {
		var entryCount int
		if err := s.db.QueryRow(
			`SELECT COUNT(*) FROM cost_entries WHERE category_path = ? OR category_path LIKE ? ESCAPE '\'`,
			path, escapeLike(path)+"/%").Scan(&entryCount); err != nil {
			return err
		}
		if entryCount > 0 {
			return fmt.Errorf("分类「%s」下还有 %d 条成本条目，请先移动或删除", path, entryCount)
		}
	}
	_, err := s.db.Exec("DELETE FROM cost_categories WHERE id=?", id)
	if err == nil {
		bumpRankVersion()
	}
	return err
}

func leafOfPath(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// SlugName 由标题确定性生成唯一键（稳定 UPSERT）：保留中文/字母/数字，其余
// 折叠为连字符，小写截断。同名标题重复保存会覆盖更新而非新增。
// cost_save 工具与文件导入共用此规则，保证同一标题的条目键一致。
func SlugName(title string) string {
	name := strutil.TitleSlug(title)
	if name == strutil.TitleSlugFallback && !hasTitleSlugRune(title) {
		name = "cost"
	}
	return name
}

// hasTitleSlugRune distinguishes TitleSlug's "entry" fallback from a real title
// whose canonical slug happens to be "entry"; only the former keeps cost's
// historical "cost" fallback.
func hasTitleSlugRune(title string) bool {
	for _, r := range title {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}
