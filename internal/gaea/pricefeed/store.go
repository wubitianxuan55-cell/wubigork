package pricefeed

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Store 价格源/抓取记录/价格历史的 SQLite 存储（Hephaestus.db SchemaV4）。
type Store struct {
	db *sql.DB
}

// Open 打开价格源存储；gdb 为 nil 时返回不可用 store。
func Open(gdb *sql.DB) *Store { return &Store{db: gdb} }

// Available 报告存储是否可用。
func (s *Store) Available() bool { return s.db != nil }

// ── 价格源 ────────────────────────────────────────────────────────

// ListSources 返回全部价格源。
func (s *Store) ListSources() []Source {
	if s.db == nil {
		return nil
	}
	rows, err := s.db.Query(`SELECT id,name,url,parser,frequency_hours,area,headers,enabled,last_fetch_at,created_at FROM price_sources ORDER BY created_at`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Source
	for rows.Next() {
		var src Source
		var headers string
		var enabled int
		if err := rows.Scan(&src.ID, &src.Name, &src.URL, &src.Parser, &src.FrequencyHours, &src.Area, &headers, &enabled, &src.LastFetchAt, &src.CreatedAt); err != nil {
			continue
		}
		_ = json.Unmarshal([]byte(headers), &src.Headers)
		src.Enabled = enabled != 0
		out = append(out, src)
	}
	return out
}

// GetSource 按 ID 读取价格源。
func (s *Store) GetSource(id string) (Source, bool) {
	for _, src := range s.ListSources() {
		if src.ID == id {
			return src, true
		}
	}
	return Source{}, false
}

// SaveSource 写入/更新价格源（UPSERT by id）。
func (s *Store) SaveSource(src Source) error {
	if s.db == nil {
		return fmt.Errorf("price store unavailable")
	}
	if strings.TrimSpace(src.ID) == "" {
		src.ID = fmt.Sprintf("src-%d", time.Now().UnixNano())
	}
	if strings.TrimSpace(src.URL) == "" {
		return fmt.Errorf("价格源 URL 不能为空")
	}
	if src.CreatedAt == "" {
		src.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	headers := "{}"
	if len(src.Headers) > 0 {
		if b, err := json.Marshal(src.Headers); err == nil {
			headers = string(b)
		}
	}
	enabled := 0
	if src.Enabled {
		enabled = 1
	}
	_, err := s.db.Exec(`
INSERT INTO price_sources(id,name,url,parser,frequency_hours,area,headers,enabled,last_fetch_at,created_at)
VALUES(?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  name=excluded.name, url=excluded.url, parser=excluded.parser,
  frequency_hours=excluded.frequency_hours, area=excluded.area,
  headers=excluded.headers, enabled=excluded.enabled`,
		src.ID, src.Name, src.URL, src.Parser, src.FrequencyHours, src.Area,
		headers, enabled, src.LastFetchAt, src.CreatedAt)
	return err
}

// TouchSource 更新最近抓取时间。
func (s *Store) TouchSource(id, at string) error {
	if s.db == nil {
		return nil
	}
	_, err := s.db.Exec("UPDATE price_sources SET last_fetch_at=? WHERE id=?", at, id)
	return err
}

// DeleteSource 删除价格源（保留历史抓取记录）。
func (s *Store) DeleteSource(id string) error {
	if s.db == nil {
		return nil
	}
	_, err := s.db.Exec("DELETE FROM price_sources WHERE id=?", id)
	return err
}

// ── 抓取记录 ──────────────────────────────────────────────────────

// FetchRecord 是一次抓取的持久化记录（summary 为候选 JSON）。
type FetchRecord struct {
	ID         string      `json:"id"`
	SourceID   string      `json:"sourceId"`
	SourceName string      `json:"sourceName"`
	URL        string      `json:"url"`
	Period     string      `json:"period"`
	FetchedAt  string      `json:"fetchedAt"`
	Status     string      `json:"status"` // pending / applied / ignored
	Candidates []Candidate `json:"candidates"`
}

// SaveFetch 写入抓取记录（新记录默认 pending）。
func (s *Store) SaveFetch(f FetchRecord) error {
	if s.db == nil {
		return fmt.Errorf("price store unavailable")
	}
	if f.ID == "" {
		f.ID = fmt.Sprintf("fetch-%d", time.Now().UnixNano())
	}
	if f.FetchedAt == "" {
		f.FetchedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if f.Status == "" {
		f.Status = "pending"
	}
	sum, _ := json.Marshal(f.Candidates)
	_, err := s.db.Exec(`
INSERT INTO price_fetch(id,source_id,source_name,url,period,fetched_at,status,summary)
VALUES(?,?,?,?,?,?,?,?)`,
		f.ID, f.SourceID, f.SourceName, f.URL, f.Period, f.FetchedAt, f.Status, string(sum))
	return err
}

// ListFetches 返回抓取记录（新→旧，默认 limit 20）。
func (s *Store) ListFetches(limit int) []FetchRecord {
	if s.db == nil {
		return nil
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(`SELECT id,source_id,source_name,url,period,fetched_at,status,summary FROM price_fetch ORDER BY fetched_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []FetchRecord
	for rows.Next() {
		var f FetchRecord
		var sum string
		if err := rows.Scan(&f.ID, &f.SourceID, &f.SourceName, &f.URL, &f.Period, &f.FetchedAt, &f.Status, &sum); err != nil {
			continue
		}
		_ = json.Unmarshal([]byte(sum), &f.Candidates)
		out = append(out, f)
	}
	return out
}

// SetFetchStatus 更新抓取记录状态（applied/ignored）。
func (s *Store) SetFetchStatus(id, status string) error {
	if s.db == nil {
		return nil
	}
	_, err := s.db.Exec("UPDATE price_fetch SET status=? WHERE id=?", status, id)
	return err
}

// ── 价格历史 ──────────────────────────────────────────────────────

// History 是一条已发布的价格快照。
type History struct {
	Name      string  `json:"name"`
	Title     string  `json:"title"`
	Unit      string  `json:"unit"`
	Price     float64 `json:"price"`
	Source    string  `json:"source"`
	Period    string  `json:"period"`
	Region    string  `json:"region,omitempty"`    // 发布时条目所在地区
	PriceType string  `json:"priceType,omitempty"` // 发布时条目价格口径
	FetchedAt string  `json:"fetchedAt"`
	Note      string  `json:"note"`
}

// AddHistory 写入价格历史快照。
func (s *Store) AddHistory(h History) error {
	if s.db == nil {
		return fmt.Errorf("price store unavailable")
	}
	if h.FetchedAt == "" {
		h.FetchedAt = time.Now().UTC().Format(time.RFC3339)
	}
	_, err := s.db.Exec(`
INSERT INTO cost_price_history(name,title,unit,price,source,period,region,price_type,fetched_at,note)
VALUES(?,?,?,?,?,?,?,?,?,?)`,
		h.Name, h.Title, h.Unit, h.Price, h.Source, h.Period, h.Region, h.PriceType, h.FetchedAt, h.Note)
	return err
}

// ListHistory 返回某条目的价格历史（新→旧）。
func (s *Store) ListHistory(name string, limit int) []History {
	if s.db == nil {
		return nil
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(`SELECT name,title,unit,price,source,period,region,price_type,fetched_at,note FROM cost_price_history WHERE name=? ORDER BY fetched_at DESC LIMIT ?`, name, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []History
	for rows.Next() {
		var h History
		if err := rows.Scan(&h.Name, &h.Title, &h.Unit, &h.Price, &h.Source, &h.Period, &h.Region, &h.PriceType, &h.FetchedAt, &h.Note); err != nil {
			continue
		}
		out = append(out, h)
	}
	return out
}
