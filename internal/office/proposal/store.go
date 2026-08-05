// Package proposal — 方案存储层（SQLite）
package proposal

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/google/uuid"
)

// Store 方案存储
type Store struct {
	db         *sql.DB
	root       string
	legacyDir  string
	filesDir   string
	exportsDir string
}

// NewStore 创建基于 SQLite 的存储实例
func NewStore(db *sql.DB, officeRoot string) *Store {
	return &Store{
		db:         db,
		root:       officeRoot,
		legacyDir:  filepath.Join(officeRoot, "proposals"),
		filesDir:   filepath.Join(officeRoot, "files"),
		exportsDir: filepath.Join(officeRoot, "exports"),
	}
}

// ExportDir 导出文件目录
func (s *Store) ExportDir() string { return s.exportsDir }

// FilesDir 上传文件目录
func (s *Store) FilesDir() string { return s.filesDir }

// LegacyDir 旧 JSON 数据目录（只读）
func (s *Store) LegacyDir() string { return s.legacyDir }

func (s *Store) errIfUnavailable() error {
	if s.db == nil {
		return fmt.Errorf("office.db 不可用")
	}
	return nil
}

// indexEntry 旧版索引条目（迁移用，保持旧 JSON 兼容）
type indexEntry struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Category  string `json:"category"`
	Template  string `json:"template"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updatedAt"`
}

// ─── 项目 ────────────────────────────────────────────────

// EnsureDefaultProject 确保「未归档项目」存在（固定 ID=default）
func (s *Store) EnsureDefaultProject() (*Project, error) {
	if err := s.errIfUnavailable(); err != nil {
		return nil, err
	}
	p := &Project{ID: "default", Name: "未归档项目", Status: "active", CreatedAt: now(), UpdatedAt: now()}
	if _, err := s.db.Exec(
		"INSERT OR IGNORE INTO projects(id, name, category, client, status, created_at, updated_at) VALUES(?,?,?,?,?,?,?)",
		p.ID, p.Name, "", "", p.Status, p.CreatedAt, p.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return s.GetProject(p.ID)
}

// CreateProject 新建项目
func (s *Store) CreateProject(name, category, client string) (*Project, error) {
	if err := s.errIfUnavailable(); err != nil {
		return nil, err
	}
	if name == "" {
		name = "未命名项目"
	}
	p := &Project{ID: uuid.New().String(), Name: name, Category: category, Client: client, Status: "active", CreatedAt: now(), UpdatedAt: now()}
	if _, err := s.db.Exec(
		"INSERT INTO projects(id, name, category, client, status, created_at, updated_at) VALUES(?,?,?,?,?,?,?)",
		p.ID, p.Name, p.Category, p.Client, p.Status, p.CreatedAt, p.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return p, nil
}

// ListProjects 列出全部项目（按更新时间倒序）
func (s *Store) ListProjects() ([]Project, error) {
	if err := s.errIfUnavailable(); err != nil {
		return nil, err
	}
	rows, err := s.db.Query("SELECT id, name, category, client, status, created_at, updated_at FROM projects ORDER BY updated_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Category, &p.Client, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetProject 获取项目
func (s *Store) GetProject(id string) (*Project, error) {
	if err := s.errIfUnavailable(); err != nil {
		return nil, err
	}
	row := s.db.QueryRow("SELECT id, name, category, client, status, created_at, updated_at FROM projects WHERE id = ?", id)
	var p Project
	if err := row.Scan(&p.ID, &p.Name, &p.Category, &p.Client, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	return &p, nil
}

// DeleteProject 删除项目（级联删除方案/章节/文件/版本 + 清理磁盘文件目录）
func (s *Store) DeleteProject(id string) error {
	if err := s.errIfUnavailable(); err != nil {
		return err
	}
	rows, err := s.db.Query("SELECT id FROM proposals WHERE project_id = ?", id)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, pid)
	}
	rows.Close()
	if _, err := s.db.Exec("DELETE FROM projects WHERE id = ?", id); err != nil {
		return err
	}
	for _, pid := range ids {
		_ = os.RemoveAll(filepath.Join(s.filesDir, pid))
	}
	return nil
}

// ─── 方案 CRUD ───────────────────────────────────────────

// Create 创建方案（生成 ID/章节 ID，写入版本 1）
func (s *Store) Create(title, template, requirements, category, projectID string, sections []ProposalSection) (*Proposal, error) {
	if err := s.errIfUnavailable(); err != nil {
		return nil, err
	}
	id := uuid.New().String()
	ts := now()
	sections = normalizeSections(sections, id, "")
	p := &Proposal{
		ID: id, ProjectID: projectID, Title: title, Category: category, Template: template,
		Requirements: requirements, Status: "draft", Version: 1,
		Sections: sections, CreatedAt: ts, UpdatedAt: ts,
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := insertProposalTx(tx, p); err != nil {
		return nil, err
	}
	if err := replaceSectionsTx(tx, id, sections); err != nil {
		return nil, err
	}
	if _, err := tx.Exec("INSERT INTO versions(proposal_id, version, summary, created_at) VALUES(?,?,?,?)", id, 1, "创建", ts); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return p, nil
}

// normalizeSections 递归补齐章节 ID/ProposalID/ParentID/Status
func normalizeSections(ss []ProposalSection, proposalID, parentID string) []ProposalSection {
	for i := range ss {
		if ss[i].ID == "" {
			ss[i].ID = uuid.New().String()
		}
		ss[i].ProposalID = proposalID
		ss[i].ParentID = parentID
		if ss[i].Status == "" {
			ss[i].Status = "pending"
		}
		ss[i].Children = normalizeSections(ss[i].Children, proposalID, ss[i].ID)
	}
	return ss
}

// ImportLegacy 原样导入旧方案（保留 ID，版本 1，随附文件注册）
func (s *Store) ImportLegacy(p *Proposal, fileDocs []FileDoc) error {
	if err := s.errIfUnavailable(); err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := insertProposalTx(tx, p); err != nil {
		return err
	}
	if err := replaceSectionsTx(tx, p.ID, p.Sections); err != nil {
		return err
	}
	if _, err := tx.Exec("INSERT INTO versions(proposal_id, version, summary, created_at) VALUES(?,?,?,?)", p.ID, 1, "旧数据迁移", p.CreatedAt); err != nil {
		return err
	}
	for _, f := range fileDocs {
		if _, err := tx.Exec(
			"INSERT INTO files(id, proposal_id, kind, name, path, size, created_at) VALUES(?,?,?,?,?,?,?)",
			uuid.New().String(), p.ID, "attachment", f.Name, f.Path, f.Size, now(),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Get 获取单个方案（含章节树）
func (s *Store) Get(id string) (*Proposal, error) {
	if err := s.errIfUnavailable(); err != nil {
		return nil, err
	}
	row := s.db.QueryRow(`SELECT id, project_id, title, category, template, requirements, status, version, bid_summary, stage, created_at, updated_at FROM proposals WHERE id = ?`, id)
	var p Proposal
	var bs string
	if err := row.Scan(&p.ID, &p.ProjectID, &p.Title, &p.Category, &p.Template, &p.Requirements, &p.Status, &p.Version, &bs, &p.Stage, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	if bs != "" {
		var summary BidSummary
		if err := json.Unmarshal([]byte(bs), &summary); err == nil {
			p.BidSummary = &summary
		}
	}
	sections, err := s.loadSections(id)
	if err != nil {
		return nil, err
	}
	p.Sections = sections
	return &p, nil
}

// List 列出全部方案（按更新时间倒序，含章节树）
func (s *Store) List() ([]Proposal, error) {
	if err := s.errIfUnavailable(); err != nil {
		return nil, err
	}
	rows, err := s.db.Query("SELECT id FROM proposals ORDER BY updated_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var out []Proposal
	for _, id := range ids {
		p, err := s.Get(id)
		if err != nil {
			continue
		}
		out = append(out, *p)
	}
	return out, nil
}

// Update 更新方案（版本 +1，章节全量替换，写入版本快照）
func (s *Store) Update(p *Proposal) error {
	if err := s.errIfUnavailable(); err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if p.Version < 1 {
		p.Version = 1
	}
	p.Version++
	p.UpdatedAt = now()
	if err := upsertProposalTx(tx, p); err != nil {
		return err
	}
	if err := replaceSectionsTx(tx, p.ID, p.Sections); err != nil {
		return err
	}
	if _, err := tx.Exec("INSERT INTO versions(proposal_id, version, summary, created_at) VALUES(?,?,?,?)", p.ID, p.Version, "内容更新", p.UpdatedAt); err != nil {
		return err
	}
	return tx.Commit()
}

// Delete 删除方案（级联章节/文件/版本 + 清理磁盘文件目录）
func (s *Store) Delete(id string) error {
	if err := s.errIfUnavailable(); err != nil {
		return err
	}
	if _, err := s.db.Exec("DELETE FROM proposals WHERE id = ?", id); err != nil {
		return err
	}
	_ = os.RemoveAll(filepath.Join(s.filesDir, id))
	return nil
}

// HasProposal 判断方案是否已存在（迁移幂等用）
func (s *Store) HasProposal(id string) bool {
	if s.db == nil {
		return false
	}
	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM proposals WHERE id = ?", id).Scan(&n); err != nil {
		return false
	}
	return n > 0
}

// AddFile 登记附件，返回文件行 ID
func (s *Store) AddFile(proposalID, kind, name, path string, size int) (string, error) {
	if err := s.errIfUnavailable(); err != nil {
		return "", err
	}
	id := uuid.New().String()
	_, err := s.db.Exec(
		"INSERT INTO files(id, proposal_id, kind, name, path, size, created_at) VALUES(?,?,?,?,?,?,?)",
		id, proposalID, kind, name, path, size, now(),
	)
	if err != nil {
		return "", err
	}
	return id, nil
}

// SaveParseResults 全量替换方案解析结果（先删后插）
func (s *Store) SaveParseResults(proposalID string, items []ParseResultItem) error {
	if err := s.errIfUnavailable(); err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM parse_results WHERE proposal_id = ?", proposalID); err != nil {
		return err
	}
	for _, it := range items {
		if _, err := tx.Exec(
			`INSERT INTO parse_results(proposal_id, file_id, field, value, page, start, end, snippet, confidence, created_at)
			 VALUES(?,?,?,?,?,?,?,?,?,?)`,
			proposalID, it.FileID, it.Field, it.Value, it.Page, it.Start, it.End, it.Snippet, it.Confidence, now(),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ListParseResults 列出方案解析结果
func (s *Store) ListParseResults(proposalID string) ([]ParseResultItem, error) {
	if err := s.errIfUnavailable(); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(
		`SELECT file_id, field, value, page, start, end, snippet, confidence FROM parse_results WHERE proposal_id = ? ORDER BY id`,
		proposalID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ParseResultItem
	for rows.Next() {
		var it ParseResultItem
		if err := rows.Scan(&it.FileID, &it.Field, &it.Value, &it.Page, &it.Start, &it.End, &it.Snippet, &it.Confidence); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// GetProjectFacts 读取项目事实基线（JSON 对象）
func (s *Store) GetProjectFacts(projectID string) (map[string]string, error) {
	if err := s.errIfUnavailable(); err != nil {
		return nil, err
	}
	var raw string
	err := s.db.QueryRow("SELECT facts FROM project_facts WHERE project_id = ?", projectID).Scan(&raw)
	if err != nil {
		if err == sql.ErrNoRows {
			return map[string]string{}, nil
		}
		return nil, err
	}
	out := map[string]string{}
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &out)
	}
	return out, nil
}

// SaveProjectFacts 保存项目事实基线（全量覆盖）
func (s *Store) SaveProjectFacts(projectID string, facts map[string]string) error {
	if err := s.errIfUnavailable(); err != nil {
		return err
	}
	if facts == nil {
		facts = map[string]string{}
	}
	data, err := json.Marshal(facts)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO project_facts(project_id, facts, updated_at) VALUES(?,?,?)
		 ON CONFLICT(project_id) DO UPDATE SET facts=excluded.facts, updated_at=excluded.updated_at`,
		projectID, string(data), now(),
	)
	return err
}

// SeedTemplates 把默认模板幂等写入 templates 表
func (s *Store) SeedTemplates() error {
	if err := s.errIfUnavailable(); err != nil {
		return err
	}
	for _, t := range DefaultTemplates {
		data, err := json.Marshal(t.Sections)
		if err != nil {
			return err
		}
		if _, err := s.db.Exec(
			`INSERT INTO templates(id, name, description, sections, created_at, updated_at)
			 VALUES(?,?,?,?,?,?)
			 ON CONFLICT(id) DO UPDATE SET name=excluded.name, description=excluded.description, sections=excluded.sections, updated_at=excluded.updated_at`,
			t.ID, t.Name, t.Description, string(data), now(), now(),
		); err != nil {
			return err
		}
	}
	return nil
}

// ListTemplatesDB 从库中读取模板
func (s *Store) ListTemplatesDB() ([]Template, error) {
	if err := s.errIfUnavailable(); err != nil {
		return nil, err
	}
	rows, err := s.db.Query("SELECT id, name, description, sections FROM templates ORDER BY rowid")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Template
	for rows.Next() {
		var t Template
		var raw string
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &raw); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(raw), &t.Sections)
		out = append(out, t)
	}
	return out, rows.Err()
}

// SeedDarkRules 幂等写入内置暗标规则
func (s *Store) SeedDarkRules() error {
	if err := s.errIfUnavailable(); err != nil {
		return err
	}
	for _, r := range DefaultDarkRules() {
		enabled := 0
		if r.Enabled {
			enabled = 1
		}
		if _, err := s.db.Exec(
			`INSERT INTO dark_rules(id, name, description, options, enabled, created_at, updated_at)
			 VALUES(?,?,?,?,?,?,?)
			 ON CONFLICT(id) DO UPDATE SET name=excluded.name, description=excluded.description, options=excluded.options, enabled=excluded.enabled, updated_at=excluded.updated_at`,
			r.ID, r.Name, r.Description, marshalOptions(r.Options), enabled, now(), now(),
		); err != nil {
			return err
		}
	}
	return nil
}

// ListDarkRules 列出暗标规则
func (s *Store) ListDarkRules() ([]DarkRule, error) {
	if err := s.errIfUnavailable(); err != nil {
		return nil, err
	}
	rows, err := s.db.Query("SELECT id, name, description, options, enabled FROM dark_rules ORDER BY rowid")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DarkRule
	for rows.Next() {
		var r DarkRule
		var raw string
		var enabled int
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &raw, &enabled); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(raw), &r.Options)
		r.Enabled = enabled == 1
		out = append(out, r)
	}
	return out, rows.Err()
}

// SaveDarkRule 保存暗标规则（ID 为空自动生成）
func (s *Store) SaveDarkRule(r DarkRule) error {
	if err := s.errIfUnavailable(); err != nil {
		return err
	}
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	enabled := 0
	if r.Enabled {
		enabled = 1
	}
	_, err := s.db.Exec(
		`INSERT INTO dark_rules(id, name, description, options, enabled, created_at, updated_at)
		 VALUES(?,?,?,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET name=excluded.name, description=excluded.description, options=excluded.options, enabled=excluded.enabled, updated_at=excluded.updated_at`,
		r.ID, r.Name, r.Description, marshalOptions(r.Options), enabled, now(), now(),
	)
	return err
}

// DeleteDarkRule 删除暗标规则
func (s *Store) DeleteDarkRule(id string) error {
	if err := s.errIfUnavailable(); err != nil {
		return err
	}
	_, err := s.db.Exec("DELETE FROM dark_rules WHERE id = ?", id)
	return err
}

// ─── 内部方法 ────────────────────────────────────────────

func (s *Store) loadSections(proposalID string) ([]ProposalSection, error) {
	rows, err := s.db.Query(`SELECT id, proposal_id, parent_id, "index", level, title, content, status, sources, word_target, words FROM sections WHERE proposal_id = ? ORDER BY "index"`, proposalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byID := make(map[string]*ProposalSection)
	for rows.Next() {
		sec := &ProposalSection{}
		if err := rows.Scan(&sec.ID, &sec.ProposalID, &sec.ParentID, &sec.Index, &sec.Level, &sec.Title, &sec.Content, &sec.Status, &sec.Sources, &sec.WordTarget, &sec.Words); err != nil {
			return nil, err
		}
		byID[sec.ID] = sec
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var roots []*ProposalSection
	for _, sec := range byID {
		if sec.ParentID != "" {
			if parent, ok := byID[sec.ParentID]; ok {
				parent.Children = append(parent.Children, *sec)
				continue
			}
		}
		roots = append(roots, sec)
	}
	var sortTree func(ss []ProposalSection)
	sortTree = func(ss []ProposalSection) {
		sort.Slice(ss, func(i, j int) bool { return ss[i].Index < ss[j].Index })
		for i := range ss {
			sortTree(ss[i].Children)
		}
	}
	out := make([]ProposalSection, 0, len(roots))
	for _, r := range roots {
		out = append(out, *r)
	}
	sortTree(out)
	return out, nil
}

func insertProposalTx(tx *sql.Tx, p *Proposal) error {
	bs := ""
	if p.BidSummary != nil {
		data, err := json.Marshal(p.BidSummary)
		if err != nil {
			return err
		}
		bs = string(data)
	}
	_, err := tx.Exec(
		`INSERT INTO proposals(id, project_id, title, category, template, requirements, status, version, bid_summary, stage, created_at, updated_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.ID, p.ProjectID, p.Title, p.Category, p.Template, p.Requirements, p.Status, p.Version, bs, p.Stage, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func upsertProposalTx(tx *sql.Tx, p *Proposal) error {
	bs := ""
	if p.BidSummary != nil {
		data, err := json.Marshal(p.BidSummary)
		if err != nil {
			return err
		}
		bs = string(data)
	}
	_, err := tx.Exec(
		`INSERT INTO proposals(id, project_id, title, category, template, requirements, status, version, bid_summary, stage, created_at, updated_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET
		   project_id=excluded.project_id, title=excluded.title, category=excluded.category,
		   template=excluded.template, requirements=excluded.requirements, status=excluded.status,
		   version=excluded.version, bid_summary=excluded.bid_summary, stage=excluded.stage, updated_at=excluded.updated_at`,
		p.ID, p.ProjectID, p.Title, p.Category, p.Template, p.Requirements, p.Status, p.Version, bs, p.Stage, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func replaceSectionsTx(tx *sql.Tx, proposalID string, sections []ProposalSection) error {
	if _, err := tx.Exec("DELETE FROM sections WHERE proposal_id = ?", proposalID); err != nil {
		return err
	}
	var walk func(ss []ProposalSection) error
	walk = func(ss []ProposalSection) error {
		for _, sec := range ss {
			if _, err := tx.Exec(
				`INSERT INTO sections(id, proposal_id, parent_id, "index", level, title, content, status, sources, word_target, words, created_at, updated_at)
				 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
				sec.ID, proposalID, sec.ParentID, sec.Index, sec.Level, sec.Title, sec.Content, sec.Status, sec.Sources, sec.WordTarget, sec.Words, now(), now(),
			); err != nil {
				return err
			}
			if err := walk(sec.Children); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(sections)
}
