package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// sqliteBackend stores memories in the gaea.db facts table, scoped per project.
// It implements backend — the same interface as fileBackend — so the rest of
// the memory pipeline (index injection, remember tool, controller, search) is
// backend-agnostic.
type sqliteBackend struct {
	db      *sql.DB
	project string // slugified project working dir (per-project isolation)
}

func (b *sqliteBackend) Index() string {
	mems := b.List()
	if len(mems) == 0 {
		return ""
	}
	var buf strings.Builder
	buf.WriteString("# Memory\n\n")
	for _, m := range mems {
		kindTag := ""
		switch NormalizeKind(string(m.Kind)) {
		case KindEpisodic:
			kindTag = "[E] "
		case KindProcedural:
			kindTag = "[P] "
		}
		fmt.Fprintf(&buf, "- %s[%s](%s.md) — %s\n", kindTag, displayTitle(m.Title, m.Name), m.Name, oneLine(m.Description))
	}
	return strings.TrimSpace(buf.String())
}

// Path returns a logical reference to the fact. Model-facing tools that need
// full bodies use memory.Get (see memory_get tool) rather than a file path.
func (b *sqliteBackend) Path(name string) string {
	return fmt.Sprintf("hephaestus.db://%s/%s", b.project, slug(name))
}

// defaultFactSpace 是事实的空间缺省值（S1 双空间：写入端零值 = work，
// space.mode 开关由 S2 接线）。
const defaultFactSpace = "work"

func (b *sqliteBackend) Save(m Memory) (string, error) {
	name := slug(m.Name)
	if name == "" {
		return "", fmt.Errorf("memory needs a name")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	tags := "[]"
	if len(m.Tags) > 0 {
		if b, err := json.Marshal(m.Tags); err == nil {
			tags = string(b)
		}
	}
	space := m.Space
	if space == "" {
		space = defaultFactSpace
	}
	_, err := b.db.Exec(`
INSERT INTO facts(project, name, title, description, type, kind, tags, body, archived, created_at, updated_at, last_used_at, source_session, source_message, space_id)
VALUES(?,?,?,?,?,?,?,?,0,?,?,?,?,?,?)
ON CONFLICT(project, name) DO UPDATE SET
  title=excluded.title, description=excluded.description,
  type=excluded.type, kind=excluded.kind, tags=excluded.tags,
  body=excluded.body, archived=0, updated_at=excluded.updated_at,
  last_used_at=CASE WHEN excluded.last_used_at != '' THEN excluded.last_used_at ELSE facts.last_used_at END,
  source_session=excluded.source_session,
  source_message=excluded.source_message,
  space_id=excluded.space_id`,
		b.project, name, m.Title, m.Description,
		string(NormalizeType(string(m.Type))), string(NormalizeKind(string(m.Kind))),
		tags, m.Body, now, now, fmtTime(m.LastUsedAt), m.SourceSession, m.SourceMessage, space)
	if err != nil {
		return "", err
	}
	return b.Path(name), nil
}

// Archive flags a fact as archived (soft delete) — it stays traceable in the
// facts table with archived=1 and an updated_at timestamp. A missing fact is
// not an error.
func (b *sqliteBackend) Archive(name string) (string, error) {
	name = slug(name)
	if name == "" {
		return "", fmt.Errorf("memory needs a name")
	}
	res, err := b.db.Exec(
		`UPDATE facts SET archived=1, updated_at=? WHERE project=? AND name=? AND archived=0`,
		time.Now().UTC().Format(time.RFC3339), b.project, name)
	if err != nil {
		return "", err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return "", nil
	}
	return b.Path(name), nil
}

// Delete archives the fact (soft delete, same as file backend's Delete).
func (b *sqliteBackend) Delete(name string) error {
	_, err := b.Archive(name)
	return err
}

// Unarchive restores an archived fact back to active memory (reverse of
// Archive): clears archived=1 and bumps updated_at. A fact that is not
// currently archived (active, or already hard-deleted by CleanupArchived)
// is an error.
func (b *sqliteBackend) Unarchive(name string) error {
	name = slug(name)
	if name == "" {
		return fmt.Errorf("memory needs a name")
	}
	res, err := b.db.Exec(
		`UPDATE facts SET archived=0, updated_at=? WHERE project=? AND name=? AND archived=1`,
		time.Now().UTC().Format(time.RFC3339), b.project, name)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("memory %q 未归档或已被清理", name)
	}
	return nil
}

func (b *sqliteBackend) ChangeType(name string, newType Type) error {
	name = slug(name)
	if name == "" {
		return fmt.Errorf("memory needs a name")
	}
	res, err := b.db.Exec(
		`UPDATE facts SET type=?, updated_at=? WHERE project=? AND name=? AND archived=0`,
		string(NormalizeType(string(newType))), time.Now().UTC().Format(time.RFC3339), b.project, name)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("memory %q not found", name)
	}
	return nil
}

// Touch 更新事实的 last_used_at（记录「最近一次被使用」用于高频排序）。
func (b *sqliteBackend) Touch(name string) error {
	name = slug(name)
	if name == "" {
		return fmt.Errorf("memory needs a name")
	}
	_, err := b.db.Exec(
		`UPDATE facts SET last_used_at=? WHERE project=? AND name=? AND archived=0`,
		time.Now().UTC().Format(time.RFC3339), b.project, name)
	return err
}

func (b *sqliteBackend) List() []Memory {
	return b.listInSpace("")
}

// ListInSpace 返回活跃事实，按空间过滤（S1 双空间读谓词）：space 为空不过滤
// （旧行为恒真，既有调用零变化），非空时仅返回该 space_id 下的行。
func (b *sqliteBackend) ListInSpace(space string) []Memory {
	return b.listInSpace(space)
}

func (b *sqliteBackend) listInSpace(space string) []Memory {
	query := `SELECT name, title, description, type, kind, tags, body, created_at, updated_at, last_used_at, source_session, source_message
		 FROM facts WHERE project=? AND archived=0`
	args := []any{b.project}
	if space != "" {
		query += ` AND space_id=?`
		args = append(args, space)
	}
	query += ` ORDER BY name`
	rows, err := b.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Memory
	for rows.Next() {
		var m Memory
		var typ, kind, tags, created, updated, lastUsed, srcSession, srcMessage string
		if err := rows.Scan(&m.Name, &m.Title, &m.Description, &typ, &kind, &tags, &m.Body,
			&created, &updated, &lastUsed, &srcSession, &srcMessage); err != nil {
			continue
		}
		m.Type = NormalizeType(typ)
		m.Kind = NormalizeKind(kind)
		m.Tags = parseTags(tags)
		m.SourceSession = srcSession
		m.SourceMessage = srcMessage
		m.UpdatedAt = parseRFC3339(updated)
		m.LastUsedAt = parseRFC3339(lastUsed)
		out = append(out, m)
	}
	return out
}

func (b *sqliteBackend) ListArchived() []ArchivedMemory {
	rows, err := b.db.Query(
		`SELECT name, title, description, type, kind, tags, body, updated_at FROM facts WHERE project=? AND archived=1 ORDER BY updated_at DESC`,
		b.project)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []ArchivedMemory
	for rows.Next() {
		var m Memory
		var typ, kind, tags, archivedAt string
		if err := rows.Scan(&m.Name, &m.Title, &m.Description, &typ, &kind, &tags, &m.Body, &archivedAt); err != nil {
			continue
		}
		m.Type = NormalizeType(typ)
		m.Kind = NormalizeKind(kind)
		m.Tags = parseTags(tags)
		out = append(out, ArchivedMemory{
			Memory:     m,
			Path:       b.Path(m.Name),
			ArchivedAt: parseRFC3339(archivedAt),
		})
	}
	return out
}

// ListArchivedPaged 返回归档事实的分页视图（updated_at 倒序，最新在前）：
// 总量 + 当前页条目，防止全量返回拖垮前端/接口。limit 钳制到 [1, 200]
// （默认 50），offset < 0 按 0 处理。文件不存在/查询失败返回错误。
func (b *sqliteBackend) ListArchivedPaged(limit, offset int) ([]ArchivedMemory, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	var total int
	if err := b.db.QueryRow(
		`SELECT COUNT(*) FROM facts WHERE project=? AND archived=1`, b.project).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := b.db.Query(
		`SELECT name, title, description, type, kind, tags, body, updated_at FROM facts WHERE project=? AND archived=1 ORDER BY updated_at DESC, name LIMIT ? OFFSET ?`,
		b.project, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []ArchivedMemory
	for rows.Next() {
		var m Memory
		var typ, kind, tags, archivedAt string
		if err := rows.Scan(&m.Name, &m.Title, &m.Description, &typ, &kind, &tags, &m.Body, &archivedAt); err != nil {
			return nil, 0, err
		}
		m.Type = NormalizeType(typ)
		m.Kind = NormalizeKind(kind)
		m.Tags = parseTags(tags)
		out = append(out, ArchivedMemory{
			Memory:     m,
			Path:       b.Path(m.Name),
			ArchivedAt: parseRFC3339(archivedAt),
		})
	}
	return out, total, rows.Err()
}

// CleanupArchived 硬删除归档超过 cutoff 时间点的事实（生命周期清理，
// T6-8.2）：返回被删除的归档行（含溯源字段），供调用方写审计/日志。
// 已归档但 updated_at 为空（异常数据）不删，避免误伤。
func (b *sqliteBackend) CleanupArchived(cutoff time.Time) ([]ArchivedMemory, error) {
	cut := cutoff.UTC().Format(time.RFC3339)
	rows, err := b.db.Query(
		`SELECT name, title, description, type, kind, tags, body, updated_at, source_session, source_message FROM facts WHERE project=? AND archived=1 AND updated_at != '' AND updated_at < ?`,
		b.project, cut)
	if err != nil {
		return nil, err
	}
	var doomed []ArchivedMemory
	for rows.Next() {
		var m Memory
		var typ, kind, tags, archivedAt, srcSession, srcMessage string
		if err := rows.Scan(&m.Name, &m.Title, &m.Description, &typ, &kind, &tags, &m.Body, &archivedAt, &srcSession, &srcMessage); err != nil {
			rows.Close()
			return nil, err
		}
		m.Type = NormalizeType(typ)
		m.Kind = NormalizeKind(kind)
		m.Tags = parseTags(tags)
		m.SourceSession = srcSession
		m.SourceMessage = srcMessage
		doomed = append(doomed, ArchivedMemory{
			Memory:     m,
			Path:       b.Path(m.Name),
			ArchivedAt: parseRFC3339(archivedAt),
		})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	if len(doomed) == 0 {
		return nil, nil
	}
	for _, am := range doomed {
		if _, err := b.db.Exec(
			`DELETE FROM facts WHERE project=? AND name=? AND archived=1`, b.project, slug(am.Name)); err != nil {
			return doomed, err
		}
	}
	return doomed, nil
}

// Get returns one active fact by name (used by the memory_get tool and the
// controller when the backend is SQLite).
func (b *sqliteBackend) Get(name string) (Memory, bool) {
	name = slug(name)
	var m Memory
	var typ, kind, tags, lastUsed, srcSession, srcMessage string
	err := b.db.QueryRow(
		`SELECT name, title, description, type, kind, tags, body, last_used_at, source_session, source_message
		 FROM facts WHERE project=? AND name=? AND archived=0`,
		b.project, name).Scan(&m.Name, &m.Title, &m.Description, &typ, &kind, &tags, &m.Body,
		&lastUsed, &srcSession, &srcMessage)
	if err != nil {
		return Memory{}, false
	}
	m.Type = NormalizeType(typ)
	m.Kind = NormalizeKind(kind)
	m.Tags = parseTags(tags)
	m.SourceSession = srcSession
	m.SourceMessage = srcMessage
	m.LastUsedAt = parseRFC3339(lastUsed)
	return m, true
}

// fmtTime 把 time.Time 格式化为 RFC3339（零值输出空串，保持列默认）。
func fmtTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func parseRFC3339(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
