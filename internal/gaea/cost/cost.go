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
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/gaea/gaea/internal/gaea/bm25"
)

// Entry 成本条目。
type Entry struct {
	Name         string
	Title        string
	Category     string // 叶子分类名（兼容旧工具/展示）
	CategoryPath string // 完整分类路径：一级/二级/…/叶子（树形过滤与分组依据）
	Unit         string // 台班/吨/m³/工日…
	Price        float64
	Spec         string
	Source       string // 定额/市场询价/历史项目…
	Tags         []string
	Status       string // 现行/草稿/已归档
	Body         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Summary 轻量视图（无 Body）。
type Summary struct {
	Name         string
	Title        string
	Category     string
	CategoryPath string
	Unit         string
	Price        float64
	Spec         string
	Source       string
	Tags         []string
	Status       string
	UpdatedAt    time.Time
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

// Open 打开成本库；gdb 为 nil 时返回不可用 store。
func Open(gdb *sql.DB) *Store {
	s := &Store{db: gdb}
	s.EnsureDefaultCategories()
	return s
}

// Available 报告存储是否可用。
func (s *Store) Available() bool { return s.db != nil }

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
	_, err := s.db.Exec(`
INSERT INTO cost_entries(name, title, category, category_path, unit, price, spec, source, tags, status, body, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(name) DO UPDATE SET
  title=excluded.title, category=excluded.category, category_path=excluded.category_path, unit=excluded.unit,
  price=excluded.price, spec=excluded.spec, source=excluded.source,
  tags=excluded.tags, status=excluded.status, body=excluded.body,
  updated_at=excluded.updated_at`,
		e.Name, e.Title, e.Category, e.CategoryPath, e.Unit, e.Price, e.Spec, e.Source,
		tags, e.Status, e.Body, e.CreatedAt.Format(time.RFC3339), e.UpdatedAt.Format(time.RFC3339))
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
SELECT name, title, category, category_path, unit, price, spec, source, tags, status, body, created_at, updated_at
FROM cost_entries WHERE name=?`, name).Scan(
		&e.Name, &e.Title, &e.Category, &e.CategoryPath, &e.Unit, &e.Price, &e.Spec, &e.Source,
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
	return &e, nil
}

// Delete 删除条目。
func (s *Store) Delete(name string) error {
	if s.db == nil {
		return nil
	}
	_, err := s.db.Exec("DELETE FROM cost_entries WHERE name=?", name)
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
	sqlText := "SELECT name, title, category, category_path, unit, price, spec, source, tags, status, updated_at FROM cost_entries"
	if len(conds) > 0 {
		sqlText += " WHERE " + strings.Join(conds, " AND ")
	}
	sqlText += " ORDER BY name"

	rows, err := s.db.Query(sqlText, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Summary
	for rows.Next() {
		var sm Summary
		var tags, updated string
		if err := rows.Scan(&sm.Name, &sm.Title, &sm.Category, &sm.CategoryPath, &sm.Unit, &sm.Price, &sm.Spec, &sm.Source, &tags, &sm.Status, &updated); err != nil {
			continue
		}
		sm.Tags = parseTagsJSON(tags)
		sm.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		out = append(out, sm)
	}
	// 关键词在 Go 侧做包含过滤：按词拆分（词间 AND、字段间 OR），
	// 精确子串匹配。刻意不在 SQL 里拼 6 列 OR LIKE 链——modernc/sqlite
	// 对特定形状的长 OR 链存在返回空集的怪癖（单列 LIKE 正常）。
	q := strings.ToLower(strings.TrimSpace(query))
	if q != "" {
		terms := strings.Fields(q)
		filtered := out[:0]
		for _, e := range out {
			hay := strings.ToLower(e.Name + "\x00" + e.Title + "\x00" + e.Category + "\x00" + e.CategoryPath + "\x00" + e.Unit + "\x00" + e.Spec + "\x00" + e.Source + "\x00" + strings.Join(e.Tags, " "))
			ok := true
			for _, term := range terms {
				if !strings.Contains(hay, term) {
					ok = false
					break
				}
			}
			if ok {
				filtered = append(filtered, e)
			}
		}
		out = filtered
		// BM25 本地排序（零 token）：命中词越多/密度越高排越前，
		// 未命中 BM25 的纯子串命中条目保持原顺序排在后面。
		if len(out) > 1 {
			docs := make([]bm25.Doc, len(out))
			for i, e := range out {
				docs[i] = bm25.Doc{ID: i, Text: e.Name + " " + e.Title + " " + e.Unit + " " + e.Spec + " " + e.Source + " " + strings.Join(e.Tags, " ")}
			}
			scored := bm25.NewRanker(docs).Rank(query)
			if len(scored) > 0 {
				seen := make(map[int]bool, len(scored))
				ranked := make([]Summary, 0, len(out))
				for _, s := range scored {
					if s.ID >= 0 && s.ID < len(out) {
						ranked = append(ranked, out[s.ID])
						seen[s.ID] = true
					}
				}
				for i, e := range out {
					if !seen[i] {
						ranked = append(ranked, e)
					}
				}
				out = ranked
			}
		}
	}
	// 有查询词时保留 BM25/精排后的相关度顺序；空查询保持 name 排序
	//（SQL 已 ORDER BY name，此处仅为兜底保证确定性）。
	if q == "" {
		sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
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
// 父节点必须先于子节点）。对齐造价行业「费用要素 × 信息价专业分类」组织：
// 一级为费用要素（人工/材料/机械/运输/检测/综合单价/其他），
// 材料按信息价体系细分到三级（土建/安装/周转 → 钢材/水泥/电线电缆…）。
// 名称全局唯一；用户可随时增删改。
var defaultCategories = [][2]string{
	{"", "人工"}, {"", "材料"}, {"", "机械"}, {"", "运输"}, {"", "检测"}, {"", "综合单价"}, {"", "其他"},
	{"人工", "普工"}, {"人工", "技工"}, {"人工", "特殊工种"},
	{"材料", "土建材料"}, {"材料", "安装材料"}, {"材料", "周转材料"}, {"材料", "辅助材料"}, {"材料", "市政绿化材料"},
	{"土建材料", "水泥及水泥制品"}, {"土建材料", "砖瓦灰砂石"}, {"土建材料", "钢材"},
	{"土建材料", "木材及竹木制品"}, {"土建材料", "防水材料"}, {"土建材料", "保温吸声材料"},
	{"土建材料", "装饰石材"}, {"土建材料", "墙面天棚及屋面饰面材料"},
	{"安装材料", "电线电缆"}, {"安装材料", "管材管件"}, {"安装材料", "阀门"}, {"安装材料", "灯具照明"}, {"安装材料", "消防器材"},
	{"周转材料", "模板"}, {"周转材料", "脚手架"}, {"周转材料", "扣件"},
	{"机械", "土方机械"}, {"机械", "桩基机械"}, {"机械", "起重机械"}, {"机械", "运输机械"}, {"机械", "混凝土机械"}, {"机械", "钢筋机械"},
	{"运输", "场内运输"}, {"运输", "场外运输"},
	{"检测", "材料检测"}, {"检测", "实体检测"},
	{"综合单价", "土方"}, {"综合单价", "混凝土"}, {"综合单价", "钢筋"}, {"综合单价", "装饰装修"},
	{"其他", "管理费"}, {"其他", "税费"}, {"其他", "措施费"},
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
	if len(nodes) == 0 {
		return nil
	}

	// 解析每节点完整路径（父链递归 + memo）。
	pathMemo := map[int]string{}
	var resolve func(id int) string
	resolve = func(id int) string {
		if p, ok := pathMemo[id]; ok {
			return p
		}
		parent := parentOf[id]
		if parent == 0 {
			pathMemo[id] = names[id]
			return names[id]
		}
		pathMemo[id] = resolve(parent) + "/" + names[id]
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
//   - id > 0：更新名称/排序；改名时同步重写该子树下成本条目的 category_path。
func (s *Store) SaveCategory(parentID int, name string, sort int, id int) (int, error) {
	if s.db == nil {
		return 0, fmt.Errorf("cost store unavailable")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("分类名称不能为空")
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
	if _, err := s.db.Exec("UPDATE cost_categories SET name=?, parent_id=?, sort=?, updated_at=? WHERE id=?",
		name, parentID, sort, now, id); err != nil {
		return 0, fmt.Errorf("更新分类失败: %w", err)
	}
	// 改名时重写该子树下条目路径：旧路径前缀 → 新路径前缀。
	if name != oldName {
		oldPath := s.CategoryPathOf(oldParent, oldName)
		newPath := s.CategoryPathOf(parentID, name)
		if oldPath != "" && newPath != "" {
			_, _ = s.db.Exec(
				`UPDATE cost_entries SET category_path = ? || substr(category_path, ?)
				 WHERE category_path = ? OR category_path LIKE ? ESCAPE '\'`,
				newPath, len(oldPath)+1, oldPath, escapeLike(oldPath)+"/%")
		}
	}
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
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(title)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevDash = false
		} else if !prevDash {
			b.WriteRune('-')
			prevDash = true
		}
	}
	name := strings.Trim(b.String(), "-")
	if name == "" {
		name = "cost"
	}
	if runes := []rune(name); len(runes) > 60 {
		name = string(runes[:60])
	}
	return name
}
