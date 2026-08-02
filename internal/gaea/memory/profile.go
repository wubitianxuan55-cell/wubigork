package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ProfileStore 读写 Hephaestus.db profile 表（主脑全局共享层）。
// 跨板块用户画像：办公 agent 记下的通用用户事实（Type=user）提升到这里，
// 未来任何 agent 都能读取；轻语 hermes.db 保留自己更精细的 user_dossier。
// value 列存 Memory 的 JSON 序列化，格式与 facts 表互通。
type ProfileStore struct {
	db *sql.DB
}

// NewProfileStore 构造画像存储；gdb 为 nil 时所有方法为 no-op。
func NewProfileStore(gdb *sql.DB) *ProfileStore {
	return &ProfileStore{db: gdb}
}

// Save 写入/更新一条用户画像事实。key = slug(name)（覆盖即更新）。
func (p *ProfileStore) Save(m Memory) error {
	if p.db == nil {
		return fmt.Errorf("profile store unavailable")
	}
	name := slug(m.Name)
	if name == "" {
		return fmt.Errorf("memory needs a name")
	}
	val, err := json.Marshal(m)
	if err != nil {
		return err
	}
	_, err = p.db.Exec(
		`INSERT INTO profile(key, value, source, confidence, updated_at) VALUES(?,?,?,?,datetime('now'))
		 ON CONFLICT(key) DO UPDATE SET value=excluded.value, source=excluded.source, updated_at=datetime('now')`,
		name, string(val), "hephaestus-remember", 1.0)
	return err
}

// Get 按名读取一条画像事实。
func (p *ProfileStore) Get(name string) (Memory, bool) {
	if p.db == nil {
		return Memory{}, false
	}
	var val string
	err := p.db.QueryRow("SELECT value FROM profile WHERE key = ?", slug(name)).Scan(&val)
	if err != nil {
		return Memory{}, false
	}
	var m Memory
	if err := json.Unmarshal([]byte(val), &m); err != nil {
		return Memory{}, false
	}
	return m, true
}

// All 返回全部画像事实（按 key 排序）。
func (p *ProfileStore) All() []Memory {
	if p.db == nil {
		return nil
	}
	rows, err := p.db.Query("SELECT key, value FROM profile WHERE key != ? ORDER BY key", migrateMarker)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Memory
	for rows.Next() {
		var key, val string
		if err := rows.Scan(&key, &val); err != nil {
			continue
		}
		var m Memory
		if err := json.Unmarshal([]byte(val), &m); err != nil {
			continue
		}
		if m.Name == "" {
			m.Name = key
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Delete 删除一条画像事实。
func (p *ProfileStore) Delete(name string) error {
	if p.db == nil {
		return nil
	}
	_, err := p.db.Exec("DELETE FROM profile WHERE key = ?", slug(name))
	return err
}

// Has 判断画像事实是否存在（去重用）。
func (p *ProfileStore) Has(name string) bool {
	_, ok := p.Get(name)
	return ok
}

// DetectConflicts 返回画像与办公 facts 中同名且描述不一致的冲突项。
// 主脑画像（新）与左脑遗留 facts（旧）对同一事实说法不同时，标记冲突，
// 供前端/调试展示，不自动覆盖（信任用户裁决）。
func (p *ProfileStore) DetectConflicts(facts []Memory) []string {
	if p.db == nil {
		return nil
	}
	var conflicts []string
	for _, f := range facts {
		if f.Type != TypeUser {
			continue
		}
		pf, ok := p.Get(f.Name)
		if !ok {
			continue
		}
		if strings.TrimSpace(pf.Description) != strings.TrimSpace(f.Description) {
			conflicts = append(conflicts, fmt.Sprintf("%s: 画像「%s」 vs facts「%s」", f.Name, oneLine(pf.Description), oneLine(f.Description)))
		}
	}
	sort.Strings(conflicts)
	return conflicts
}
