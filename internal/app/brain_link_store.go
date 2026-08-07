package app

import (
	"database/sql"
	"sort"
)

// LinkStore 主脑跨脑关联索引：db 非 nil 时落 Hephaestus.db brain_links 表
// （零迁移建表）；db 为 nil 时退化为内存模式（测试/未装配环境）。
type LinkStore struct {
	db *sql.DB

	mu    map[string][]Ref // 内存模式（db == nil）
	ready bool
}

func NewLinkStore(db *sql.DB) *LinkStore {
	ls := &LinkStore{db: db, mu: map[string][]Ref{}}
	if db == nil {
		ls.ready = true
		return ls
	}
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS brain_links (
		entity TEXT NOT NULL,
		brain TEXT NOT NULL,
		ref TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		PRIMARY KEY (entity, brain, ref)
	)`)
	ls.ready = err == nil
	return ls
}

func (l *LinkStore) Add(entity, brain, ref string) error {
	if l.db != nil && l.ready {
		_, err := l.db.Exec(
			`INSERT OR IGNORE INTO brain_links(entity, brain, ref) VALUES(?,?,?)`,
			entity, brain, ref)
		return err
	}
	if l.mu != nil {
		for _, r := range l.mu[entity] {
			if r.Brain == brain && r.Ref == ref {
				return nil
			}
		}
		l.mu[entity] = append(l.mu[entity], Ref{Brain: brain, Ref: ref})
	}
	return nil
}

func (l *LinkStore) ListByEntity(entity string) ([]Ref, error) {
	if l.db != nil && l.ready {
		rows, err := l.db.Query(`SELECT brain, ref FROM brain_links WHERE entity = ? ORDER BY brain, ref`, entity)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []Ref
		for rows.Next() {
			var r Ref
			if err := rows.Scan(&r.Brain, &r.Ref); err != nil {
				return nil, err
			}
			out = append(out, r)
		}
		return out, rows.Err()
	}
	out := append([]Ref(nil), l.mu[entity]...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Brain != out[j].Brain {
			return out[i].Brain < out[j].Brain
		}
		return out[i].Ref < out[j].Ref
	})
	return out, nil
}
