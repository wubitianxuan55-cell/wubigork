package app

// Herdsman 数字生命记忆联动（P5）：只读访问 digital-life/life.sqlite3——
// 虚拟人格角色（characters）、关系（relationships）、记忆摘要（memory_summaries）
// 与时间线/世界事件，供 gaea 记忆中枢展示。只读不写，绝不修改 Herdsman 数据。

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

// HerdsmanDigitalCharacter 虚拟人格角色概要（合并关系与记忆摘要）。
type HerdsmanDigitalCharacter struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Gender           string   `json:"gender"`
	Identity         string   `json:"identity"`
	Worldview        string   `json:"worldview"`
	TextModel        string   `json:"text_model"`
	Intimacy         int      `json:"intimacy"`
	Trust            int      `json:"trust"`
	Safety           int      `json:"safety"`
	Conflict         int      `json:"conflict"`
	LastInteractedAt string   `json:"last_interacted_at"`
	MemorySummary    string   `json:"memory_summary"`
	Highlights       []string `json:"highlights,omitempty"`
	MemoryEventCount int      `json:"memory_event_count"`
	Reinforcement    int      `json:"reinforcement"`
	UpdatedAt        string   `json:"updated_at"`
}

// HerdsmanDigitalEvent 时间线/世界事件条目。
type HerdsmanDigitalEvent struct {
	Type       string `json:"type"`
	Title      string `json:"title"`
	Summary    string `json:"summary"`
	OccurredAt string `json:"occurred_at"`
}

// HerdsmanDigitalLife 数字生命记忆总览。
type HerdsmanDigitalLife struct {
	Available       bool                       `json:"available"`
	Source          string                     `json:"source"`
	Error           string                     `json:"error,omitempty"`
	CharacterCount  int                        `json:"character_count"`
	TimelineEvents  int                        `json:"timeline_events"`
	StateCommits    int                        `json:"state_commits"`
	WorldEvents     int                        `json:"world_events"`
	MemoryEvents    int                        `json:"memory_events"`
	MemorySummaries int                        `json:"memory_summaries"`
	Relationships   int                        `json:"relationships"`
	TurnTraces      int                        `json:"turn_traces"`
	Characters      []HerdsmanDigitalCharacter `json:"characters"`
	RecentTimeline  []HerdsmanDigitalEvent     `json:"recent_timeline"`
	RecentWorld     []HerdsmanDigitalEvent     `json:"recent_world"`
}

type digitalCharacterJSON struct {
	Name           string `json:"name"`
	Gender         string `json:"gender"`
	Identity       string `json:"identity"`
	Worldview      string `json:"worldview"`
	ModelSelection struct {
		Text string `json:"text"`
	} `json:"model_selection"`
}

type digitalRelationshipJSON struct {
	Intimacy         int    `json:"intimacy"`
	Trust            int    `json:"trust"`
	Safety           int    `json:"safety"`
	Conflict         int    `json:"conflict"`
	LastInteractedAt string `json:"last_interacted_at"`
}

type digitalSummaryJSON struct {
	Summary       string   `json:"summary"`
	Highlights    []string `json:"highlights"`
	Reinforcement int      `json:"reinforcement"`
	EventCount    int      `json:"event_count"`
}

type digitalTimelineJSON struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

type digitalWorldJSON struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// openDigitalLifeDB 只读打开 life.sqlite3。
func openDigitalLifeDB(path string) (*sql.DB, error) {
	return sql.Open("sqlite", path+"?mode=ro&_busy_timeout=5000")
}

func digitalCount(db *sql.DB, table string) int {
	var n int
	_ = db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n)
	return n
}

func digitalJSON[T any](raw string) (T, bool) {
	var out T
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return out, false
	}
	return out, true
}

// loadHerdsmanDigitalLife 只读解析 life.sqlite3 为总览。
func loadHerdsmanDigitalLife(path string) (HerdsmanDigitalLife, error) {
	out := HerdsmanDigitalLife{Available: true, Source: "herdsman-digital-life"}
	db, err := openDigitalLifeDB(path)
	if err != nil {
		return out, fmt.Errorf("打开数字生命库失败: %w", err)
	}
	defer db.Close()

	out.CharacterCount = digitalCount(db, "characters")
	out.TimelineEvents = digitalCount(db, "life_timeline_events")
	out.StateCommits = digitalCount(db, "life_state_commits")
	out.WorldEvents = digitalCount(db, "world_events")
	out.MemoryEvents = digitalCount(db, "memory_events")
	out.MemorySummaries = digitalCount(db, "memory_summaries")
	out.Relationships = digitalCount(db, "relationships")
	out.TurnTraces = digitalCount(db, "turn_traces")

	// 角色 × 关系 × 记忆摘要。
	type rel struct {
		digitalRelationshipJSON
		updatedAt string
	}
	relByChar := map[string]rel{}
	if rows, err := db.Query("SELECT character_id, data, updated_at FROM relationships"); err == nil {
		for rows.Next() {
			var cid, data, up string
			if rows.Scan(&cid, &data, &up) == nil {
				if r, ok := digitalJSON[digitalRelationshipJSON](data); ok {
					relByChar[cid] = rel{digitalRelationshipJSON: r, updatedAt: up}
				}
			}
		}
		_ = rows.Close()
	}
	type summary struct {
		digitalSummaryJSON
		updatedAt string
	}
	sumByChar := map[string]summary{}
	if rows, err := db.Query("SELECT character_id, data, updated_at FROM memory_summaries"); err == nil {
		for rows.Next() {
			var cid, data, up string
			if rows.Scan(&cid, &data, &up) == nil {
				if s, ok := digitalJSON[digitalSummaryJSON](data); ok {
					cur, has := sumByChar[cid]
					if !has || up > cur.updatedAt {
						sumByChar[cid] = summary{digitalSummaryJSON: s, updatedAt: up}
					}
				}
			}
		}
		_ = rows.Close()
	}

	if rows, err := db.Query("SELECT id, data, updated_at FROM characters"); err == nil {
		for rows.Next() {
			var id, data, up string
			if rows.Scan(&id, &data, &up) != nil {
				continue
			}
			c, ok := digitalJSON[digitalCharacterJSON](data)
			if !ok {
				continue
			}
			item := HerdsmanDigitalCharacter{
				ID:        id,
				Name:      c.Name,
				Gender:    c.Gender,
				Identity:  c.Identity,
				Worldview: c.Worldview,
				TextModel: c.ModelSelection.Text,
				UpdatedAt: up,
			}
			if r, has := relByChar[id]; has {
				item.Intimacy = r.Intimacy
				item.Trust = r.Trust
				item.Safety = r.Safety
				item.Conflict = r.Conflict
				item.LastInteractedAt = r.LastInteractedAt
			}
			if s, has := sumByChar[id]; has {
				item.MemorySummary = excerpt(s.Summary, 600)
				item.Highlights = s.Highlights
				if len(item.Highlights) > 5 {
					item.Highlights = item.Highlights[:5]
				}
				item.MemoryEventCount = s.EventCount
				item.Reinforcement = s.Reinforcement
			}
			out.Characters = append(out.Characters, item)
		}
		_ = rows.Close()
	}
	sort.Slice(out.Characters, func(i, j int) bool { return out.Characters[i].Name < out.Characters[j].Name })

	// 最近时间线 / 世界事件。
	if rows, err := db.Query("SELECT category, data, occurred_at FROM life_timeline_events ORDER BY occurred_at DESC LIMIT 12"); err == nil {
		for rows.Next() {
			var cat, data, at string
			if rows.Scan(&cat, &data, &at) != nil {
				continue
			}
			e, _ := digitalJSON[digitalTimelineJSON](data)
			out.RecentTimeline = append(out.RecentTimeline, HerdsmanDigitalEvent{
				Type: cat, Title: e.Title, Summary: excerpt(e.Summary, 160), OccurredAt: at,
			})
		}
		_ = rows.Close()
	}
	if rows, err := db.Query("SELECT type, data, created_at FROM world_events ORDER BY created_at DESC LIMIT 12"); err == nil {
		for rows.Next() {
			var typ, data, at string
			if rows.Scan(&typ, &data, &at) != nil {
				continue
			}
			e, _ := digitalJSON[digitalWorldJSON](data)
			detail := e.Detail
			if detail == "" {
				detail = e.Title
			}
			out.RecentWorld = append(out.RecentWorld, HerdsmanDigitalEvent{
				Type: typ, Title: e.Title, Summary: excerpt(detail, 160), OccurredAt: at,
			})
		}
		_ = rows.Close()
	}
	return out, nil
}

func excerpt(s string, n int) string {
	s = strings.TrimSpace(s)
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}

// HerdsmanDigitalLife 返回 Herdsman 数字生命记忆总览（只读）。
func (a *App) HerdsmanDigitalLife() (HerdsmanDigitalLife, error) {
	dir := herdsmanDataDir()
	if dir == "" {
		return HerdsmanDigitalLife{Source: "herdsman-digital-life", Error: "无法定位 Herdsman 数据目录"}, fmt.Errorf("无法定位 Herdsman 数据目录")
	}
	path := filepath.Join(dir, "digital-life", "life.sqlite3")
	if _, err := os.Stat(path); err != nil {
		return HerdsmanDigitalLife{Source: "herdsman-digital-life", Error: "数字生命库不存在（未启用 Herdsman 数字生命）"}, err
	}
	return loadHerdsmanDigitalLife(path)
}
