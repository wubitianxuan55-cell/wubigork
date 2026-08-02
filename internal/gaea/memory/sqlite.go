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
	_, err := b.db.Exec(`
INSERT INTO facts(project, name, title, description, type, kind, tags, body, archived, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,0,?,?)
ON CONFLICT(project, name) DO UPDATE SET
  title=excluded.title, description=excluded.description,
  type=excluded.type, kind=excluded.kind, tags=excluded.tags,
  body=excluded.body, archived=0, updated_at=excluded.updated_at`,
		b.project, name, m.Title, m.Description,
		string(NormalizeType(string(m.Type))), string(NormalizeKind(string(m.Kind))),
		tags, m.Body, now, now)
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

func (b *sqliteBackend) List() []Memory {
	rows, err := b.db.Query(
		`SELECT name, title, description, type, kind, tags, body FROM facts WHERE project=? AND archived=0 ORDER BY name`,
		b.project)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Memory
	for rows.Next() {
		var m Memory
		var typ, kind, tags string
		if err := rows.Scan(&m.Name, &m.Title, &m.Description, &typ, &kind, &tags, &m.Body); err != nil {
			continue
		}
		m.Type = NormalizeType(typ)
		m.Kind = NormalizeKind(kind)
		m.Tags = parseTags(tags)
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

// Get returns one active fact by name (used by the memory_get tool and the
// controller when the backend is SQLite).
func (b *sqliteBackend) Get(name string) (Memory, bool) {
	name = slug(name)
	var m Memory
	var typ, kind, tags string
	err := b.db.QueryRow(
		`SELECT name, title, description, type, kind, tags, body FROM facts WHERE project=? AND name=? AND archived=0`,
		b.project, name).Scan(&m.Name, &m.Title, &m.Description, &typ, &kind, &tags, &m.Body)
	if err != nil {
		return Memory{}, false
	}
	m.Type = NormalizeType(typ)
	m.Kind = NormalizeKind(kind)
	m.Tags = parseTags(tags)
	return m, true
}

func parseRFC3339(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
