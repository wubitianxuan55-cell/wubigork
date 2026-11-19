package app

// wenxue_funnel.go — 文学板块回写警告回执 + 最小事件账本 + 漏斗指标
// （2026-11-19 众神殿 pan_1795084662473 P0 施工，用户令「按论证实施」）。
//
// P0 = 本轮唯一施工项：wenxue_chapter_write 与 wenxue_candidate_promote 两条
// 正文入口（tools.go:560/:731 实锤）共用警告型回执——characters/timeline/
// unobserved 三态 + 仅 stale/dirty 时的 hint，替换两入口原有的无条件静态
// hint（裁决 disagreements 定案「先得到可核查信号，再验证用户价值」）。
// 硬拒、长章分页、章节上下文包、通用写板级围栏均为条件项，未取得实录或
// 误报数据前不进入 P0（裁决 conditions）。
//
// 事件账本授权形态（裁决 recommendation + 办公记忆写入确认红线）：默认关闭、
// 仅本地、不含正文、可删除。开关 = 书内 .gaea/stats/enable 标记文件（存在=
// 记账；创建/删除该文件即开/关），不引入全局 config 键；账本文件
// .gaea/stats/wenxue-funnel.jsonl 一行一事件、仅元数据，随书删除即清。
// 未启用时回执照常计算（characters/timeline 基于文件 mtime/时间线锚，无需
// 账本），unobserved 恒 unknown——只实现回执，不宣称漏斗完整。
//
// 观测语义（unobserved，ZCode r3 规格）：写第 N 章时读上一章当前正文 revision，
// 与账本中该章最后一条 body_write.revision 比对——无账本/无记录→unknown；
// 一致→clean；不一致→dirty（通用 write_file/edit_file 或外部编辑旁路的
// 低价检出，不改共享写面）。旁路仅标 dirty，不当成零（裁决 conditions）。

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// wenxueWriteback 回执三态快照（chapter_write / candidate_promote 两入口同构；
// 事件行 writeback 字段同构复用，供 M3/M6 计算）。
type wenxueWriteback struct {
	Characters string `json:"characters"`     // updated | stale | none
	Timeline   string `json:"timeline"`       // updated | stale | none
	Unobserved string `json:"unobserved"`     // clean | dirty | unknown
	Hint       string `json:"hint,omitempty"` // 仅 stale/dirty 时出现（警告型，不硬拒）
}

// wenxueFunnelEvent 事件行（一行一事件，仅元数据，无正文——裁决红线）。
type wenxueFunnelEvent struct {
	Type      string           `json:"type"`              // body_write | candidate_write | quality_check | final
	Ts        int64            `json:"ts"`                // UnixMilli
	Entry     string           `json:"entry,omitempty"`   // body_write：chapter_write | promote
	Kind      string           `json:"kind,omitempty"`    // body_write：create | rewrite
	Chapter   int              `json:"chapter,omitempty"` // 章号
	Path      string           `json:"path,omitempty"`    // 章节正文相对路径（body_write/final）/候选路径（candidate_write）
	Words     int              `json:"words,omitempty"`
	Delta     int              `json:"delta,omitempty"`
	Revision  string           `json:"revision,omitempty"`  // 正文 sha256（M6 旁路检出基线）
	Label     string           `json:"label,omitempty"`     // candidate_write：候选标签
	Scope     string           `json:"scope,omitempty"`     // quality_check："book"（全书体检）
	Chapters  int              `json:"chapters,omitempty"`  // quality_check：体检时章数
	Groups    []string         `json:"groups,omitempty"`    // quality_check：非空组名（频次证据，M4 只称频次不称覆盖率）
	Writeback *wenxueWriteback `json:"writeback,omitempty"` // body_write 附前一章回写快照（有前章才有；M3 分母）
}

// wenxueFunnelEnabled 事件账本开关：书内 .gaea/stats/enable 标记文件存在=记账。
// 默认不创建即默认关闭；删除该文件即关闭（未授权时只实现回执，不宣称漏斗完整）。
func wenxueFunnelEnabled(root string) bool {
	abs := filepath.Join(root, ".gaea", "stats", "enable")
	_, err := os.Stat(abs)
	return err == nil
}

// wenxueFunnelPath 账本文件路径（每书一份，随书可删）。
func wenxueFunnelPath(root string) string {
	return filepath.Join(root, ".gaea", "stats", "wenxue-funnel.jsonl")
}

// wenxueFunnelAppend 追加一条事件（账本关闭时静默不写；写失败告警不阻断
// 写章主流程——账本是仪表，不是业务依赖）。行内绝不落正文。
func wenxueFunnelAppend(root string, ev wenxueFunnelEvent) {
	if !wenxueFunnelEnabled(root) {
		return
	}
	ev.Ts = time.Now().UnixMilli()
	line, err := json.Marshal(ev)
	if err != nil {
		slog.Warn("文学事件行序列化失败", "type", ev.Type, "error", err)
		return
	}
	p := wenxueFunnelPath(root)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		slog.Warn("文学事件账本目录创建失败", "path", p, "error", err)
		return
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		slog.Warn("文学事件账本打开失败", "path", p, "error", err)
		return
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Write(append(line, '\n')); err != nil {
		slog.Warn("文学事件账本写入失败", "path", p, "error", err)
	}
}

// wenxueFunnelLoad 读全量事件行（缺失/坏行跳过；供 unobserved 比对与指标计算）。
func wenxueFunnelLoad(root string) []wenxueFunnelEvent {
	raw, err := os.ReadFile(wenxueFunnelPath(root))
	if err != nil {
		return nil
	}
	var out []wenxueFunnelEvent
	for _, line := range splitJSONL(raw) {
		var ev wenxueFunnelEvent
		if err := json.Unmarshal(line, &ev); err == nil && ev.Type != "" {
			out = append(out, ev)
		}
	}
	return out
}

// splitJSONL 按行切分（容忍 CRLF 与空行）。
func splitJSONL(raw []byte) [][]byte {
	var out [][]byte
	start := 0
	for i, b := range raw {
		if b != '\n' {
			continue
		}
		line := raw[start:i]
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		if len(line) > 0 {
			out = append(out, line)
		}
		start = i + 1
	}
	if start < len(raw) {
		line := raw[start:]
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		if len(line) > 0 {
			out = append(out, line)
		}
	}
	return out
}

// wenxuePrevUnobserved 前一章旁路观测：账本中该章（按正文 path 匹配）最后
// 一条 body_write.revision vs 当前正文 sha256。账本关/无记录/读失败 → unknown。
func wenxuePrevUnobserved(root string, prev wenxueChapterMeta) string {
	if !wenxueFunnelEnabled(root) || prev.Path == "" {
		return "unknown"
	}
	events := wenxueFunnelLoad(root)
	var last *wenxueFunnelEvent
	for i := range events {
		ev := &events[i]
		if ev.Type == "body_write" && ev.Path == prev.Path {
			last = ev
		}
	}
	if last == nil || last.Revision == "" {
		return "unknown"
	}
	body, _, err := wenxueReadBody(root, prev.Path)
	if err != nil {
		return "unknown"
	}
	if wenxueRevision(body) == last.Revision {
		return "clean"
	}
	return "dirty"
}

// wenxueWritebackReceipt 共享警告回执：prev = 本次写入章在章节清单中的前一章
// （无前一章返回 nil，整个块省略——首章无回写待办可报，裁决定案）。
//
//	characters：前一章正文命中的角色卡中，是否有卡 mtime ≥ 前一章正文 mtime
//	  （写章后更新过）→ updated；有命中但均未更新 → stale；无角色目录/
//	  零卡/正文无命中/前一章正文读失败 → none（不告警）。
//	timeline：时间线存在前一章章号锚的事件行 → updated；有时间线但缺该章
//	  锚 → stale；无时间线目录/零事件/前一章章号不可解析（0）→ none。
//	unobserved：见 wenxuePrevUnobserved（账本授权后才可判）。
func wenxueWritebackReceipt(root string, prev wenxueChapterMeta) *wenxueWriteback {
	if prev.Path == "" {
		return nil
	}
	wb := &wenxueWriteback{Characters: "none", Timeline: "none", Unobserved: "unknown"}
	if body, _, err := wenxueReadBody(root, prev.Path); err == nil {
		updated, hit := false, false
		for _, c := range wenxueListCharacters(root) {
			name, _ := c["name"].(string)
			if name == "" || !wenxueContainsBody(body, name) {
				continue
			}
			hit = true
			if mt, _ := c["modTime"].(int64); mt >= prev.ModTime {
				updated = true
				break
			}
		}
		if hit {
			if updated {
				wb.Characters = "updated"
			} else {
				wb.Characters = "stale"
			}
		}
	}
	if prev.Number > 0 {
		entries := wenxueTimelineEntries(root)
		if len(entries) > 0 {
			found := false
			for _, e := range entries {
				if e.Chapter == prev.Number {
					found = true
					break
				}
			}
			if found {
				wb.Timeline = "updated"
			} else {
				wb.Timeline = "stale"
			}
		}
	}
	wb.Unobserved = wenxuePrevUnobserved(root, prev)
	var parts []string
	if wb.Characters == "stale" {
		parts = append(parts, fmt.Sprintf("第%d章出场角色的角色卡未见更新", prev.Number))
	}
	if wb.Timeline == "stale" {
		parts = append(parts, fmt.Sprintf("时间线未见第%d章事件", prev.Number))
	}
	if wb.Unobserved == "dirty" {
		parts = append(parts, fmt.Sprintf("第%d章正文在写作工具外被修改过", prev.Number))
	}
	if len(parts) > 0 {
		wb.Hint = strings.Join(parts, "；") + "——建议先补回写（wenxue_character_write / wenxue_timeline_append）再继续"
	}
	return wb
}

// wenxueContainsBody 正文子串命中（角色名短名误命中防护：名字至少 2 rune 才参与匹配）。
func wenxueContainsBody(body, name string) bool {
	if len([]rune(name)) < 2 {
		return false
	}
	return strings.Contains(body, name)
}

// ── 漏斗指标（纯函数；M2 30 天续写率留 P1，本刀不做）──────────────
//
// 口径全部来自裁决 conditions：
//   M1 每书每周正文写入章数：每章（path 维度）首个 body_write 计 1，重写不虚增；
//      words>=500 不是文学向统一有效章标准，不设字数门槛。
//   M3 下一章入口回写待办率：分母=带 writeback 快照的 body_write（有前章），
//      分子=characters 或 timeline 为 stale。
//   M4 质检调用频次：quality_check 次数 / 首写章数（只称频次，不称覆盖率）。
//   M5 候选提升使用率：entry=promote 的 body_write / candidate_write 次数；
//      零候选记 n/a（nil）。
//   M6 旁路检出率：分母=unobserved 可判定（clean/dirty）的 body_write，
//      分子=dirty；分母 0 记 n/a（nil）。

// wenxueFunnelRatio 分子/分母比（nil 分母=0 时由调用方落 n/a）。
type wenxueFunnelRatio struct {
	Num int `json:"num"`
	Den int `json:"den"`
}

// wenxueFunnelReport 指标快照（book 级；now 用于周分桶口径对齐）。
type wenxueFunnelReport struct {
	BodyChaptersPerWeek map[string]int     `json:"bodyChaptersPerWeek"` // ISO 周（如 2026-W47）→ 首写章数
	BodyChaptersTotal   int                `json:"bodyChaptersTotal"`
	WritebackPending    *wenxueFunnelRatio `json:"writebackPending"` // M3；无快照=nil
	QualityFrequency    *wenxueFunnelRatio `json:"qualityFrequency"` // M4；零章=nil
	PromoteUsage        *wenxueFunnelRatio `json:"promoteUsage"`     // M5；零候选=nil（n/a）
	BypassDirty         *wenxueFunnelRatio `json:"bypassDirty"`      // M6；无可判定=nil（n/a）
}

// wenxueFunnelCompute 从事件行算指标（纯函数，now 只参与周分桶键）。
func wenxueFunnelCompute(events []wenxueFunnelEvent, now time.Time) wenxueFunnelReport {
	rep := wenxueFunnelReport{BodyChaptersPerWeek: map[string]int{}}
	firstSeen := map[string]bool{}
	var m3n, m3d, qc, promoteN, candN, m6n, m6d int
	for _, ev := range events {
		switch ev.Type {
		case "body_write":
			if ev.Path != "" && !firstSeen[ev.Path] {
				firstSeen[ev.Path] = true
				rep.BodyChaptersTotal++
				year, week := time.UnixMilli(ev.Ts).ISOWeek()
				rep.BodyChaptersPerWeek[fmt.Sprintf("%d-W%02d", year, week)]++
			}
			if ev.Writeback != nil {
				m3d++
				if ev.Writeback.Characters == "stale" || ev.Writeback.Timeline == "stale" {
					m3n++
				}
				switch ev.Writeback.Unobserved {
				case "clean", "dirty":
					m6d++
					if ev.Writeback.Unobserved == "dirty" {
						m6n++
					}
				}
			}
			if ev.Entry == "promote" {
				promoteN++
			}
		case "candidate_write":
			candN++
		case "quality_check":
			qc++
		}
	}
	_ = now
	if m3d > 0 {
		rep.WritebackPending = &wenxueFunnelRatio{Num: m3n, Den: m3d}
	}
	if rep.BodyChaptersTotal > 0 {
		rep.QualityFrequency = &wenxueFunnelRatio{Num: qc, Den: rep.BodyChaptersTotal}
	}
	if candN > 0 {
		rep.PromoteUsage = &wenxueFunnelRatio{Num: promoteN, Den: candN}
	}
	if m6d > 0 {
		rep.BypassDirty = &wenxueFunnelRatio{Num: m6n, Den: m6d}
	}
	return rep
}
