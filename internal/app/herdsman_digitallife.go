package app

// Herdsman 数字生命记忆联动（P5）：只读访问 digital-life/life.sqlite3——
// 虚拟人格角色（characters）、关系（relationships）、记忆摘要（memory_summaries）
// 与时间线/世界事件，供 gaea 记忆中枢展示。只读不写，绝不修改 Herdsman 数据。
//
// 读取降级纪律（审计 P0 AP9-01）：Herdsman 升级后表名/列名变化、单行 data JSON
// 损坏，此前一律静默吞掉——面板显示「角色 0 个」或某角色凭空消失，与「真的
// 没有数据」不可区分。现在：可降级的故障逐条进 Warnings（前端展示）+ slog.Warn
// （含表名与错误原文）；只有在库整体不可读（计数表全失败，如库文件损坏）时才
// 返回致命 error，此时 out.Warnings 同样已填好，调用方与日志都拿得到原因。

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
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
	Available bool   `json:"available"`
	Source    string `json:"source"`
	Error     string `json:"error,omitempty"`
	// Warnings 是本次读取的降级说明（中文，人类可读；审计 P0 AP9-01 的契约
	// 字段，前端据此提示「数字生命库读取不完整」）。全部读取成功时为空切片。
	Warnings        []string                   `json:"warnings"`
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

// digitalWarn 记一条降级说明：进 Warnings（前端可见）并落 slog（含表名/错误
// 原文，便于定位 Herdsman 侧结构变化）。Warnings 是总览唯一的降级出口。
func (o *HerdsmanDigitalLife) digitalWarn(msg string) {
	o.Warnings = append(o.Warnings, msg)
	slog.Warn("数字生命读取降级", "source", o.Source, "warning", msg)
}

// digitalRowFailures 单行坏数据计数器：同一张表坏很多行时汇总为一条 Warnings
// （避免刷屏），同时保留首条示例错误让说明「不含糊」。
type digitalRowFailures struct {
	counts   map[string]int
	examples map[string]string
}

func newDigitalRowFailures() *digitalRowFailures {
	return &digitalRowFailures{counts: map[string]int{}, examples: map[string]string{}}
}

func (f *digitalRowFailures) note(table, detail string) {
	f.counts[table]++
	if _, has := f.examples[table]; !has {
		f.examples[table] = detail
	}
}

// flush 按传入的表序输出汇总说明（顺序固定，输出稳定，便于测试与用户阅读）。
func (f *digitalRowFailures) flush(out *HerdsmanDigitalLife, tables []string, format string) {
	for _, table := range tables {
		if n := f.counts[table]; n > 0 {
			out.digitalWarn(fmt.Sprintf(format, table, n, f.examples[table]))
		}
	}
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

// digitalCount 统计单表行数。审计 P0 AP9-01：不再吞掉 Scan 错误——表名/列名
// 一变时返回错误（此前静默返回 0，与「真的没有数据」不可区分）。
func digitalCount(db *sql.DB, table string) (int, error) {
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// digitalJSON 解析单条 data JSON。审计 P0 AP9-01：返回底层错误而不仅是 ok
// 布尔——降级说明要能说清「坏在哪」（invalid character 'x' ...），而不是只
// 报一句无法定位的「解析失败」。
func digitalJSON[T any](raw string) (T, error) {
	var out T
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return out, err
	}
	return out, nil
}

// digitalLifeTables 汇总 Warnings 时的固定表序（读端出现的全部明细表）。
var digitalLifeTables = []string{"relationships", "memory_summaries", "characters", "life_timeline_events", "world_events"}

// loadHerdsmanDigitalLife 只读解析 life.sqlite3 为总览（降级见文件头纪律）。
func loadHerdsmanDigitalLife(path string) (HerdsmanDigitalLife, error) {
	out := HerdsmanDigitalLife{Available: true, Source: "herdsman-digital-life", Warnings: []string{}}
	db, err := openDigitalLifeDB(path)
	if err != nil {
		return out, fmt.Errorf("打开数字生命库失败: %w", err)
	}
	defer func() { _ = db.Close() }()

	// 计数表：单表读失败（表改名/缺表）降级为 Warnings 并保持 0，让面板能
	// 同时显示「有数据」的部分与「读不到」的部分；全部失败（库文件损坏等）
	// 才致命——此时 out.Warnings 已带齐逐表原因。
	countTables := []struct {
		name string
		dst  *int
	}{
		{"characters", &out.CharacterCount},
		{"life_timeline_events", &out.TimelineEvents},
		{"life_state_commits", &out.StateCommits},
		{"world_events", &out.WorldEvents},
		{"memory_events", &out.MemoryEvents},
		{"memory_summaries", &out.MemorySummaries},
		{"relationships", &out.Relationships},
		{"turn_traces", &out.TurnTraces},
	}
	countFailures := 0
	var firstCountErr error
	for _, ct := range countTables {
		n, err := digitalCount(db, ct.name)
		if err != nil {
			countFailures++
			if firstCountErr == nil {
				firstCountErr = err
			}
			out.digitalWarn(fmt.Sprintf("读取 %s 表失败：%v", ct.name, err))
			continue
		}
		*ct.dst = n
	}
	if countFailures == len(countTables) {
		return out, fmt.Errorf("数字生命库不可读（%d 张表全部读取失败，首个错误: %v）", countFailures, firstCountErr)
	}

	scanFails := newDigitalRowFailures()
	jsonFails := newDigitalRowFailures()

	// 角色 × 关系 × 记忆摘要。
	type rel struct {
		digitalRelationshipJSON
		updatedAt string
	}
	relByChar := map[string]rel{}
	if rows, err := db.Query("SELECT character_id, data, updated_at FROM relationships"); err != nil {
		out.digitalWarn(fmt.Sprintf("读取 relationships 表明细失败：%v", err))
	} else {
		for rows.Next() {
			var cid, data, up string
			if err := rows.Scan(&cid, &data, &up); err != nil {
				scanFails.note("relationships", err.Error())
				slog.Warn("数字生命记录读取失败", "table", "relationships", "character_id", cid, "error", err)
				continue
			}
			r, err := digitalJSON[digitalRelationshipJSON](data)
			if err != nil {
				jsonFails.note("relationships", err.Error())
				slog.Warn("数字生命记录 JSON 解析失败", "table", "relationships", "character_id", cid, "error", err)
				continue
			}
			relByChar[cid] = rel{digitalRelationshipJSON: r, updatedAt: up}
		}
		if err := rows.Err(); err != nil {
			out.digitalWarn(fmt.Sprintf("遍历 relationships 表中断：%v", err))
		}
		_ = rows.Close()
	}
	type summary struct {
		digitalSummaryJSON
		updatedAt string
	}
	sumByChar := map[string]summary{}
	if rows, err := db.Query("SELECT character_id, data, updated_at FROM memory_summaries"); err != nil {
		out.digitalWarn(fmt.Sprintf("读取 memory_summaries 表明细失败：%v", err))
	} else {
		for rows.Next() {
			var cid, data, up string
			if err := rows.Scan(&cid, &data, &up); err != nil {
				scanFails.note("memory_summaries", err.Error())
				slog.Warn("数字生命记录读取失败", "table", "memory_summaries", "character_id", cid, "error", err)
				continue
			}
			s, err := digitalJSON[digitalSummaryJSON](data)
			if err != nil {
				jsonFails.note("memory_summaries", err.Error())
				slog.Warn("数字生命记录 JSON 解析失败", "table", "memory_summaries", "character_id", cid, "error", err)
				continue
			}
			cur, has := sumByChar[cid]
			if !has || up > cur.updatedAt {
				sumByChar[cid] = summary{digitalSummaryJSON: s, updatedAt: up}
			}
		}
		if err := rows.Err(); err != nil {
			out.digitalWarn(fmt.Sprintf("遍历 memory_summaries 表中断：%v", err))
		}
		_ = rows.Close()
	}

	if rows, err := db.Query("SELECT id, data, updated_at FROM characters"); err != nil {
		out.digitalWarn(fmt.Sprintf("读取 characters 表明细失败：%v", err))
	} else {
		for rows.Next() {
			var id, data, up string
			if err := rows.Scan(&id, &data, &up); err != nil {
				scanFails.note("characters", err.Error())
				slog.Warn("数字生命记录读取失败", "table", "characters", "id", id, "error", err)
				continue
			}
			c, err := digitalJSON[digitalCharacterJSON](data)
			if err != nil {
				// 单行坏 JSON 不再让该角色静默消失：计入汇总 + 日志留痕。
				jsonFails.note("characters", err.Error())
				slog.Warn("数字生命记录 JSON 解析失败", "table", "characters", "id", id, "error", err)
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
		if err := rows.Err(); err != nil {
			out.digitalWarn(fmt.Sprintf("遍历 characters 表中断：%v", err))
		}
		_ = rows.Close()
	}
	sort.Slice(out.Characters, func(i, j int) bool { return out.Characters[i].Name < out.Characters[j].Name })

	// 最近时间线 / 世界事件。
	if rows, err := db.Query("SELECT category, data, occurred_at FROM life_timeline_events ORDER BY occurred_at DESC LIMIT 12"); err != nil {
		out.digitalWarn(fmt.Sprintf("读取 life_timeline_events 表明细失败：%v", err))
	} else {
		for rows.Next() {
			var cat, data, at string
			if err := rows.Scan(&cat, &data, &at); err != nil {
				scanFails.note("life_timeline_events", err.Error())
				slog.Warn("数字生命记录读取失败", "table", "life_timeline_events", "error", err)
				continue
			}
			e, err := digitalJSON[digitalTimelineJSON](data)
			if err != nil {
				jsonFails.note("life_timeline_events", err.Error())
				slog.Warn("数字生命记录 JSON 解析失败", "table", "life_timeline_events", "category", cat, "error", err)
				continue
			}
			out.RecentTimeline = append(out.RecentTimeline, HerdsmanDigitalEvent{
				Type: cat, Title: e.Title, Summary: excerpt(e.Summary, 160), OccurredAt: at,
			})
		}
		if err := rows.Err(); err != nil {
			out.digitalWarn(fmt.Sprintf("遍历 life_timeline_events 表中断：%v", err))
		}
		_ = rows.Close()
	}
	if rows, err := db.Query("SELECT type, data, created_at FROM world_events ORDER BY created_at DESC LIMIT 12"); err != nil {
		out.digitalWarn(fmt.Sprintf("读取 world_events 表明细失败：%v", err))
	} else {
		for rows.Next() {
			var typ, data, at string
			if err := rows.Scan(&typ, &data, &at); err != nil {
				scanFails.note("world_events", err.Error())
				slog.Warn("数字生命记录读取失败", "table", "world_events", "error", err)
				continue
			}
			e, err := digitalJSON[digitalWorldJSON](data)
			if err != nil {
				jsonFails.note("world_events", err.Error())
				slog.Warn("数字生命记录 JSON 解析失败", "table", "world_events", "type", typ, "error", err)
				continue
			}
			detail := e.Detail
			if detail == "" {
				detail = e.Title
			}
			out.RecentWorld = append(out.RecentWorld, HerdsmanDigitalEvent{
				Type: typ, Title: e.Title, Summary: excerpt(detail, 160), OccurredAt: at,
			})
		}
		if err := rows.Err(); err != nil {
			out.digitalWarn(fmt.Sprintf("遍历 world_events 表中断：%v", err))
		}
		_ = rows.Close()
	}

	// 单行坏数据汇总（固定表序；条数 + 首条示例错误，既不静默也不刷屏）。
	scanFails.flush(&out, digitalLifeTables, "%s 表 %d 行读取失败已跳过（示例：%s）")
	jsonFails.flush(&out, digitalLifeTables, "%s 表 %d 条记录 data JSON 解析失败已跳过（示例：%s）")
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
		return HerdsmanDigitalLife{Source: "herdsman-digital-life", Warnings: []string{}, Error: "无法定位 Herdsman 数据目录"}, fmt.Errorf("无法定位 Herdsman 数据目录")
	}
	path := filepath.Join(dir, "digital-life", "life.sqlite3")
	if _, err := os.Stat(path); err != nil {
		return HerdsmanDigitalLife{Source: "herdsman-digital-life", Warnings: []string{}, Error: "数字生命库不存在（未启用 Herdsman 数字生命）"}, err
	}
	return loadHerdsmanDigitalLife(path)
}
