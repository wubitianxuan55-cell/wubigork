package knowledge

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// migrateMarker 是 Hephaestus.db profile 表中的知识库迁移标记。
const migrateMarker = "knowledge_migrated"

// sqliteBackend stores entries in the Hephaestus.db knowledge table.
type sqliteBackend struct {
	db *sql.DB
}

func (b *sqliteBackend) Save(e Entry) error {
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
	_, err := b.db.Exec(`
INSERT INTO knowledge(name, title, category, phase, discipline, tags, status, version, author, reviewer, source, body, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(name) DO UPDATE SET
  title=excluded.title, category=excluded.category, phase=excluded.phase,
  discipline=excluded.discipline, tags=excluded.tags, status=excluded.status,
  version=excluded.version, author=excluded.author, reviewer=excluded.reviewer,
  source=excluded.source, body=excluded.body, updated_at=excluded.updated_at`,
		e.Name, e.Title, e.Category, e.Phase, e.Discipline, tags, e.Status,
		strconv.Itoa(e.Version), e.Author, e.Reviewer, e.Source, e.Body,
		e.CreatedAt.Format(time.RFC3339), e.UpdatedAt.Format(time.RFC3339))
	return err
}

func (b *sqliteBackend) Get(name string) (*Entry, error) {
	var e Entry
	var tags, ver, created, updated string
	err := b.db.QueryRow(`
SELECT name, title, category, phase, discipline, tags, status, version, author, reviewer, source, body, created_at, updated_at
FROM knowledge WHERE name=?`, name).Scan(
		&e.Name, &e.Title, &e.Category, &e.Phase, &e.Discipline, &tags, &e.Status,
		&ver, &e.Author, &e.Reviewer, &e.Source, &e.Body, &created, &updated)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("entry %q not found", name)
		}
		return nil, err
	}
	e.Tags = parseTagsJSON(tags)
	e.Version, _ = strconv.Atoi(ver)
	e.CreatedAt, _ = time.Parse(time.RFC3339, created)
	e.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return &e, nil
}

func (b *sqliteBackend) Delete(name string) error {
	if _, err := b.db.Exec("DELETE FROM knowledge WHERE name=?", name); err != nil {
		return err
	}
	return nil
}

func (b *sqliteBackend) List() ([]EntrySummary, error) {
	rows, err := b.db.Query(
		`SELECT name, title, category, phase, tags, status, updated_at FROM knowledge ORDER BY updated_at DESC`)
	if err != nil {
		// 审计 P0#6 GA3-01：读失败不得吞成空切片（否则与「库空」不可区分）。
		return nil, fmt.Errorf("query knowledge list: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []EntrySummary
	for rows.Next() {
		var s EntrySummary
		var tags, updated string
		if err := rows.Scan(&s.Name, &s.Title, &s.Category, &s.Phase, &tags, &s.Status, &updated); err != nil {
			return nil, fmt.Errorf("scan knowledge list: %w", err)
		}
		s.Tags = parseTagsJSON(tags)
		s.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate knowledge list: %w", err)
	}
	return out, nil
}

func (b *sqliteBackend) Index() string {
	rows, err := b.db.Query(`SELECT name, title, category, status, updated_at FROM knowledge ORDER BY name`)
	if err != nil {
		// 索引不可用必须留痕（审计 GA3-13）：返回文案里写了「索引不可用」，
		// 但错误本身不能只停在字符串里。
		slog.Warn("knowledge: 索引渲染查询失败", "error", err)
		return "# 知识库索引\n\n（索引不可用）\n"
	}
	defer func() { _ = rows.Close() }()
	var buf strings.Builder
	buf.WriteString("# 知识库索引\n\n")
	type row struct {
		name, title, cat, status, updated string
	}
	var rowsList []row
	empty := true
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.name, &r.title, &r.cat, &r.status, &r.updated); err != nil {
			// 单行解析失败会让该条目从索引里消失：不能静默 continue。
			slog.Warn("knowledge: 索引跳过解析失败的行", "error", err)
			continue
		}
		rowsList = append(rowsList, r)
		empty = false
	}
	if err := rows.Err(); err != nil {
		// 迭代中断不得报成「暂无条目」：中文索引文案要区分「表空」与「读不出来」。
		return "# 知识库索引\n\n（索引不可用）\n"
	}
	if empty {
		buf.WriteString("（暂无条目。使用 knowledge_add 工具添加。）\n")
		return buf.String()
	}
	buf.WriteString("| 名称 | 标题 | 分类 | 状态 | 更新日期 |\n")
	buf.WriteString("|------|------|------|------|----------|\n")
	for _, r := range rowsList {
		dateStr := ""
		if t, err := time.Parse(time.RFC3339, r.updated); err == nil {
			dateStr = t.Format("2006-01-02")
		}
		fmt.Fprintf(&buf, "| %s | %s | %s | %s | %s |\n", r.name, r.title, r.cat, r.status, dateStr)
	}
	return buf.String()
}

func (b *sqliteBackend) ReadAll() ([]Entry, error) {
	rows, err := b.db.Query(`
SELECT name, title, category, phase, discipline, tags, status, version, author, reviewer, source, body, created_at, updated_at
FROM knowledge`)
	if err != nil {
		// 审计 P0#6 GA3-01：同 List，读失败如实上抛。
		return nil, fmt.Errorf("query knowledge entries: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Entry
	for rows.Next() {
		var e Entry
		var tags, ver, created, updated string
		if err := rows.Scan(&e.Name, &e.Title, &e.Category, &e.Phase, &e.Discipline, &tags, &e.Status,
			&ver, &e.Author, &e.Reviewer, &e.Source, &e.Body, &created, &updated); err != nil {
			return nil, fmt.Errorf("scan knowledge entries: %w", err)
		}
		e.Tags = parseTagsJSON(tags)
		e.Version, _ = strconv.Atoi(ver)
		e.CreatedAt, _ = time.Parse(time.RFC3339, created)
		e.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate knowledge entries: %w", err)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func parseTagsJSON(raw string) []string {
	var tags []string
	if strings.TrimSpace(raw) == "" || raw == "[]" {
		return nil
	}
	_ = json.Unmarshal([]byte(raw), &tags)
	return tags
}

// MigrateLegacyKnowledge 将旧 Markdown 知识库（dir 下的 *.md）幂等迁移到
// Hephaestus.db knowledge 表。完成后写 profile 标记跳过后续启动。旧文件保留。
//
// 标记的三种状态必须区分（审计 GA3-13）：行存在且非空=已迁移（返回 0）；
// sql.ErrNoRows=标记不存在（正常，本次迁移）；其余读取失败（profile 表缺失/
// 库损坏/锁超时）=异常，返回 error 而**不得**当作「未迁移」重跑——否则每次
// 启动都重复迁移，把真实故障掩盖成「一直在迁移」。
func MigrateLegacyKnowledge(gdb *sql.DB, dir string) (int, error) {
	if gdb == nil {
		return 0, nil
	}
	var marker string
	err := gdb.QueryRow("SELECT value FROM profile WHERE key = ?", migrateMarker).Scan(&marker)
	switch {
	case err == nil && marker != "":
		return 0, nil // 已迁移
	case err != nil && !errors.Is(err, sql.ErrNoRows):
		return 0, fmt.Errorf("读取迁移标记 %q 失败: %w", migrateMarker, err)
	}

	entries, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		// Glob 失败（模式非法/目录不可用）此前被 _ 丢弃，迁移静默变成「没条目」。
		return 0, fmt.Errorf("扫描旧知识库目录 %s 失败: %w", dir, err)
	}
	n, skipped := 0, 0
	for _, path := range entries {
		data, err := os.ReadFile(path)
		if err != nil {
			skipped++
			slog.Warn("knowledge: 旧知识库文件读取失败，已跳过", "path", path, "error", err)
			continue
		}
		e, err := ParseFrontmatter(string(data))
		if err != nil || e.Name == "" {
			skipped++
			slog.Warn("knowledge: 旧知识库文件无 frontmatter/名称，已跳过", "path", path, "error", err)
			continue
		}
		if err := (&sqliteBackend{db: gdb}).Save(*e); err != nil {
			return n, fmt.Errorf("migrate %s: %w", e.Name, err)
		}
		n++
	}
	if skipped > 0 {
		slog.Warn("knowledge: 旧知识库迁移完成但有条目被跳过（部分迁移）", "migrated", n, "skipped", skipped)
	}

	_, err = gdb.Exec("INSERT OR REPLACE INTO profile(key, value, source, confidence, updated_at) VALUES(?,?,?,?,datetime('now'))",
		migrateMarker, "done", "migration", 1.0)
	if err != nil {
		return n, fmt.Errorf("write migration marker: %w", err)
	}
	return n, nil
}
