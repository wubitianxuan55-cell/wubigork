// Package costproject 测算项目与沉淀闭环（zaojia-database 蒸馏：我的项目 /
// 工程量清单 / 版本留痕 → 沉淀回成本库）。
//
// 测算项目 = 一次报价/测算工作的容器（项目类型/规模/工艺/状态）；测算明细行
// 引用 cost_entries 的单价（也可手动填价），数量×单价自动算金额；保存版本生成
// 不可变快照（JSON + 合计），支持回看/对比/恢复思路；「沉淀」把明细行 UPSERT
// 回 cost_entries，完成「沉淀即调用」闭环。存储于 Hephaestus.db（SchemaV10）。
package costproject

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

// idSeq 项目 id 生成用的进程内序号：与纳秒时间戳组合，保证同进程内不碰撞
// （纯时间戳在并发/负载下可能落到同一时钟刻度）。
var idSeq uint64

// Project 测算项目。
type Project struct {
	ID          string
	Name        string
	ProjectType string // 项目类型（房建/市政/安装/园林…）
	Scale       string // 规模（如 5 万 m²）
	Craft       string // 工艺/说明
	Status      string // 编制中 / 已保存版本 / 已沉淀
	Note        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ProjectSummary 项目列表视图（含条目数/合计/版本数，供列表与概览）。
type ProjectSummary struct {
	Project
	ItemCount    int
	Total        float64
	VersionCount int
}

// Item 测算明细行（工程量清单行）。
type Item struct {
	ID           int64
	ProjectID    string
	Name         string // kebab 稳定名（沉淀为 cost_entries.name）
	Title        string
	CategoryPath string
	Unit         string
	Quantity     float64
	Price        float64
	Amount       float64 // 数量×单价（保存时自动计算）
	EntryName    string  // 引用的成本条目 name（可空=手动估价）
	Source       string
	Note         string
	Sort         int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Version 不可变版本快照。
type Version struct {
	ID        int64
	ProjectID string
	Version   int
	Total     float64
	Snapshot  string // 明细行 JSON
	Note      string
	CreatedAt time.Time
}

// Store 测算项目存储（Hephaestus.db）。
type Store struct {
	db *sql.DB
}

// Open 打开测算项目存储；gdb 为 nil 时返回不可用 store。
func Open(gdb *sql.DB) *Store { return &Store{db: gdb} }

// Available 报告存储是否可用。
func (s *Store) Available() bool { return s.db != nil }

func (s *Store) requireDB() error {
	if s.db == nil {
		return fmt.Errorf("cost project store unavailable")
	}
	return nil
}

// SaveProject 新建/更新测算项目（id 为空时自动生成稳定 id）。
func (s *Store) SaveProject(p Project) (string, error) {
	if err := s.requireDB(); err != nil {
		return "", err
	}
	if strings.TrimSpace(p.Name) == "" {
		return "", fmt.Errorf("测算项目需要名称")
	}
	now := time.Now().UTC()
	if p.ID == "" {
		p.ID = fmt.Sprintf("proj-%d-%d", now.UnixNano(), atomic.AddUint64(&idSeq, 1))
		p.CreatedAt = now
	}
	if p.Status == "" {
		p.Status = "编制中"
	}
	p.UpdatedAt = now
	_, err := s.db.Exec(`
INSERT INTO cost_projects(id, name, project_type, scale, craft, status, note, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  name=excluded.name, project_type=excluded.project_type, scale=excluded.scale,
  craft=excluded.craft, status=excluded.status, note=excluded.note,
  updated_at=excluded.updated_at`,
		p.ID, p.Name, p.ProjectType, p.Scale, p.Craft, p.Status, p.Note,
		p.CreatedAt.Format(time.RFC3339), p.UpdatedAt.Format(time.RFC3339))
	return p.ID, err
}

// ListProjects 返回项目列表（按更新时间倒序），带条目数/合计/版本数。
func (s *Store) ListProjects() []ProjectSummary {
	if err := s.requireDB(); err != nil {
		return nil
	}
	rows, err := s.db.Query(`
SELECT p.id, p.name, p.project_type, p.scale, p.craft, p.status, p.note, p.created_at, p.updated_at,
  (SELECT COUNT(*) FROM cost_estimate_items i WHERE i.project_id = p.id),
  (SELECT COALESCE(SUM(i.amount), 0) FROM cost_estimate_items i WHERE i.project_id = p.id),
  (SELECT COUNT(*) FROM cost_estimate_versions v WHERE v.project_id = p.id)
FROM cost_projects p ORDER BY p.updated_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []ProjectSummary
	for rows.Next() {
		var s ProjectSummary
		var created, updated string
		if err := rows.Scan(&s.ID, &s.Name, &s.ProjectType, &s.Scale, &s.Craft, &s.Status, &s.Note,
			&created, &updated, &s.ItemCount, &s.Total, &s.VersionCount); err != nil {
			continue
		}
		s.CreatedAt, _ = time.Parse(time.RFC3339, created)
		s.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		out = append(out, s)
	}
	return out
}

// GetProject 按 id 读取项目。
func (s *Store) GetProject(id string) (*Project, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	var p Project
	var created, updated string
	err := s.db.QueryRow(`
SELECT id, name, project_type, scale, craft, status, note, created_at, updated_at
FROM cost_projects WHERE id=?`, id).Scan(
		&p.ID, &p.Name, &p.ProjectType, &p.Scale, &p.Craft, &p.Status, &p.Note, &created, &updated)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("测算项目 %q 不存在", id)
		}
		return nil, err
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, created)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return &p, nil
}

// DeleteProject 删除项目及其明细/版本（级联）。
func (s *Store) DeleteProject(id string) error {
	if err := s.requireDB(); err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, q := range []string{
		"DELETE FROM cost_estimate_items WHERE project_id=?",
		"DELETE FROM cost_estimate_versions WHERE project_id=?",
		"DELETE FROM cost_projects WHERE id=?",
	} {
		if _, err := tx.Exec(q, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SaveItem 新建/更新一条测算明细（id<=0 新建）；金额按数量×单价重算。
func (s *Store) SaveItem(i Item) (int64, error) {
	if err := s.requireDB(); err != nil {
		return 0, err
	}
	if strings.TrimSpace(i.Title) == "" {
		return 0, fmt.Errorf("明细行需要标题")
	}
	i.Amount = i.Quantity * i.Price
	now := time.Now().UTC()
	i.UpdatedAt = now
	if i.ID <= 0 {
		i.CreatedAt = now
		res, err := s.db.Exec(`
INSERT INTO cost_estimate_items(project_id, name, title, category_path, unit, quantity, price, amount, entry_name, source, note, sort, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			i.ProjectID, i.Name, i.Title, i.CategoryPath, i.Unit, i.Quantity, i.Price, i.Amount,
			i.EntryName, i.Source, i.Note, i.Sort,
			i.CreatedAt.Format(time.RFC3339), i.UpdatedAt.Format(time.RFC3339))
		if err != nil {
			return 0, err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return 0, err
		}
		i.ID = id
		return id, nil
	}
	_, err := s.db.Exec(`
UPDATE cost_estimate_items SET name=?, title=?, category_path=?, unit=?, quantity=?, price=?, amount=?, entry_name=?, source=?, note=?, sort=?, updated_at=?
WHERE id=?`,
		i.Name, i.Title, i.CategoryPath, i.Unit, i.Quantity, i.Price, i.Amount,
		i.EntryName, i.Source, i.Note, i.Sort, i.UpdatedAt.Format(time.RFC3339), i.ID)
	return i.ID, err
}

// ListItems 返回项目的全部明细行（按 sort,id 排序）。
func (s *Store) ListItems(projectID string) []Item {
	if err := s.requireDB(); err != nil {
		return nil
	}
	rows, err := s.db.Query(`
SELECT id, project_id, name, title, category_path, unit, quantity, price, amount, entry_name, source, note, sort, created_at, updated_at
FROM cost_estimate_items WHERE project_id=? ORDER BY sort, id`, projectID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Item
	for rows.Next() {
		var i Item
		var created, updated string
		if err := rows.Scan(&i.ID, &i.ProjectID, &i.Name, &i.Title, &i.CategoryPath, &i.Unit,
			&i.Quantity, &i.Price, &i.Amount, &i.EntryName, &i.Source, &i.Note, &i.Sort,
			&created, &updated); err != nil {
			continue
		}
		i.CreatedAt, _ = time.Parse(time.RFC3339, created)
		i.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		out = append(out, i)
	}
	return out
}

// DeleteItem 删除一条明细行。
func (s *Store) DeleteItem(id int64) error {
	if err := s.requireDB(); err != nil {
		return err
	}
	_, err := s.db.Exec("DELETE FROM cost_estimate_items WHERE id=?", id)
	return err
}

// SaveVersion 保存不可变版本快照：对当前明细行做 JSON 快照 + 合计。
// 版本号 = 该项目已有最大版本 + 1；无明细行时拒绝（避免空快照）。
func (s *Store) SaveVersion(projectID, note string) (*Version, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	items := s.ListItems(projectID)
	if len(items) == 0 {
		return nil, fmt.Errorf("项目没有明细行，无法保存版本")
	}
	snap, err := json.Marshal(items)
	if err != nil {
		return nil, err
	}
	total := 0.0
	for _, i := range items {
		total += i.Amount
	}
	var maxVer int
	_ = s.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM cost_estimate_versions WHERE project_id=?", projectID).Scan(&maxVer)
	v := &Version{
		ProjectID: projectID,
		Version:   maxVer + 1,
		Total:     total,
		Snapshot:  string(snap),
		Note:      note,
		CreatedAt: time.Now().UTC(),
	}
	res, err := s.db.Exec(`
INSERT INTO cost_estimate_versions(project_id, version, total, snapshot, note, created_at)
VALUES(?,?,?,?,?,?)`,
		v.ProjectID, v.Version, v.Total, v.Snapshot, v.Note, v.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	v.ID, _ = res.LastInsertId()
	return v, nil
}

// ListVersions 返回项目版本（新→旧）。
func (s *Store) ListVersions(projectID string) []Version {
	if err := s.requireDB(); err != nil {
		return nil
	}
	rows, err := s.db.Query(`
SELECT id, project_id, version, total, snapshot, note, created_at
FROM cost_estimate_versions WHERE project_id=? ORDER BY version DESC`, projectID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Version
	for rows.Next() {
		var v Version
		var created string
		if err := rows.Scan(&v.ID, &v.ProjectID, &v.Version, &v.Total, &v.Snapshot, &v.Note, &created); err != nil {
			continue
		}
		v.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, v)
	}
	return out
}
