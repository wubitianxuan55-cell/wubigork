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
	Name      string
	Title     string
	Category  string // 机械/材料/人工/运输/检测/其他
	Unit      string // 台班/吨/m³/工日…
	Price     float64
	Spec      string
	Source    string // 定额/市场询价/历史项目…
	Tags      []string
	Status    string // 现行/草稿/已归档
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Summary 轻量视图（无 Body）。
type Summary struct {
	Name      string
	Title     string
	Category  string
	Unit      string
	Price     float64
	Spec      string
	Source    string
	Tags      []string
	Status    string
	UpdatedAt time.Time
}

// Store 成本库存储（Hephaestus.db）。
type Store struct {
	db *sql.DB
}

// Open 打开成本库；gdb 为 nil 时返回不可用 store。
func Open(gdb *sql.DB) *Store {
	return &Store{db: gdb}
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
INSERT INTO cost_entries(name, title, category, unit, price, spec, source, tags, status, body, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(name) DO UPDATE SET
  title=excluded.title, category=excluded.category, unit=excluded.unit,
  price=excluded.price, spec=excluded.spec, source=excluded.source,
  tags=excluded.tags, status=excluded.status, body=excluded.body,
  updated_at=excluded.updated_at`,
		e.Name, e.Title, e.Category, e.Unit, e.Price, e.Spec, e.Source,
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
SELECT name, title, category, unit, price, spec, source, tags, status, body, created_at, updated_at
FROM cost_entries WHERE name=?`, name).Scan(
		&e.Name, &e.Title, &e.Category, &e.Unit, &e.Price, &e.Spec, &e.Source,
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
		conds = append(conds, "category = ?")
		args = append(args, category)
	}
	if strings.TrimSpace(status) != "" && status != "all" {
		conds = append(conds, "status = ?")
		args = append(args, status)
	}
	sqlText := "SELECT name, title, category, unit, price, spec, source, tags, status, updated_at FROM cost_entries"
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
		if err := rows.Scan(&sm.Name, &sm.Title, &sm.Category, &sm.Unit, &sm.Price, &sm.Spec, &sm.Source, &tags, &sm.Status, &updated); err != nil {
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
			hay := strings.ToLower(e.Name + "\x00" + e.Title + "\x00" + e.Unit + "\x00" + e.Spec + "\x00" + e.Source + "\x00" + strings.Join(e.Tags, " "))
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
