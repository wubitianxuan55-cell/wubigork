package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ── 记忆事件日志（阶段五 5.1 首刀，docs/gaea-memory-graph-51-design-2026-09.md）──
//
// 事件日志是记忆语义图谱的唯一事实源：所有记忆写路径（remember 工具/桌面面板/
// 做梦/蒸馏合并/引用触达）经 sqliteBackend 落库成功后，同库 memory_events 表追加
// 一条事件（INSERT-only，绝不 UPDATE/DELETE）。图谱是日志的投影（ProjectEvents
// 纯函数），mem_graph_* 物化表可随时 DROP 后由日志整体重建——「日志即真相，删库
// 可重建」。日志不存全文（body 只留摘要 excerpt 供展示），内容真相仍在 facts 表；
// 日志存的是「何时/何空间/对哪条记忆/发生了什么」的事件真相与投影所需的元数据。

// Event 是事件日志中的一条记忆事件（与 db SchemaV18+V21 memory_events 列一一对应）。
type Event struct {
	Seq int64 `json:"seq"` // 日志单调序号（AUTOINCREMENT，投影按此定序）
	At  int64 `json:"at"`  // unix ms，事实时间：事件所述事实的发生时刻
	// RecordedAt 是事务时间（unix ms，SchemaV21）：本行写入日志的时刻，由
	// AppendEvent 服务端盖章——调用方可声明事实时间（At），不可伪造记录时间。
	// 0=V21 之前的旧行（当时两轴共用 at），读取端归一回落 At。
	RecordedAt int64    `json:"recordedAt,omitempty"`
	Op         string   `json:"op"`      // save / archive / unarchive / delete / touch / change_type / cite
	Name       string   `json:"name"`    // [MEM:name] 引用键（kebab-case slug）
	Project    string   `json:"project"` // 事实所属项目（slugified cwd）
	Space      string   `json:"space"`   // work / play（cite 可能空=不过滤）
	Kind       string   `json:"kind,omitempty"`
	Type       string   `json:"type,omitempty"`
	Title      string   `json:"title,omitempty"`
	Desc       string   `json:"description,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	// Refs 是本条事件携带的引用目标（save=body 里的 [MEM:x]/[[x]] 互引；
	// cite=dangling 未解析键——悬空引用不建边但必须留痕）。
	Refs          []string `json:"refs,omitempty"`
	Excerpt       string   `json:"excerpt,omitempty"` // body 摘要（展示用，非投影真相）
	SourceSession string   `json:"sourceSession,omitempty"`
	SourceMessage string   `json:"sourceMessage,omitempty"`
	Actor         string   `json:"actor,omitempty"` // 写入方粗记（dream/panel/model/…，尽力而为）
}

// 事件 op 常量（closed set，投影按 op 分派）。
const (
	OpSave       = "save"
	OpArchive    = "archive"
	OpUnarchive  = "unarchive"
	OpDelete     = "delete"
	OpTouch      = "touch"
	OpChangeType = "change_type"
	OpCite       = "cite"
	OpPin        = "pin"      // 固化（5.3 三态生命周期）
	OpUnpin      = "unpin"    // 解除固化
	OpFeedback   = "feedback" // 助手回答反馈（点赞/点踩，v4.238 能力层；投影只出事件节点不建实体）
)

// wikiRefRe 匹配 body 中的 wiki 式互引：[[name]] / [[name|别名]]。与 app 层
// GaeaMemoryGraph 的引用边同构；name 经 slug 归一后与记忆 Name 同构。
var wikiRefRe = regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`)

// ExtractRefNames 从 body 提取互引目标并归一（[MEM:x] 大小写不敏感归一小写 +
// [[x]] slug 归一），去重保序，剔除自引。两类引用都不解析链接目标是否存在——
// 存在性判定是投影（图）的事，日志只忠实记录「写了什么引用」。
func ExtractRefNames(body string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(name string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		out = append(out, name)
	}
	for _, m := range citationRe.FindAllStringSubmatch(body, -1) {
		add(strings.ToLower(m[1]))
	}
	for _, m := range wikiRefRe.FindAllStringSubmatch(body, -1) {
		add(slug(m[1]))
	}
	return out
}

// EventLog 是按 Hephaestus.db 打开的记忆事件日志（一个 userDir 一个库，
// 跨项目共用，事件行带 project 列）。
type EventLog struct {
	DB *sql.DB
}

// eventExcerptLimit 是事件里 body 摘要的截断上限（字节）。日志不是内容存储，
// 摘要只供事件列表/图节点 hover 展示；截断按字节与 evidence.ClampSummary 同风格，
// 宁可截进 UTF-8 中间也不为展示字节膨胀。
const eventExcerptLimit = 280

// AppendEvent 追加一条事件。日志是追加式：任何失败（库不可用/表未迁移）都
// 原样返回错误，由调用方决定是否吞掉——写路径挂钩处一律尽力而为（吞错不阻断
// 主写入，与 dream 审计同哲学），controller 的 cite 事件同理。
func (l *EventLog) AppendEvent(e Event) error {
	if l == nil || l.DB == nil {
		return nil
	}
	return appendEventExec(l.DB, e)
}

// execer 是 *sql.DB 与 *sql.Tx 的公共子集：事件插入必须落在调用方的事务
// 连接上（刀B v4.246 写路径事务化）——Hephaestus.db 是单连接池（MaxOpen
// Conns(1)），事务持有连接期间任何 l.DB.Exec 都会等连接而互相死锁。
type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// appendEventExec 是 AppendEvent 的执行载体（exec 决定落库还是落事务）。
func appendEventExec(x execer, e Event) error {
	if x == nil {
		return nil
	}
	if e.Op == "" || e.Name == "" {
		return fmt.Errorf("event needs op and name")
	}
	if e.At == 0 {
		e.At = time.Now().UnixMilli()
	}
	// 事务时间（SchemaV21）由日志侧盖章：无论调用方传什么，写入时刻以本处
	// 时钟为准（审计锚点不可被调用方伪造）；fact time 与 transaction time
	// 从此两轴独立。
	e.RecordedAt = time.Now().UnixMilli()
	tags, refs := "[]", "[]"
	if b, err := json.Marshal(e.Tags); err == nil && len(e.Tags) > 0 {
		tags = string(b)
	}
	if b, err := json.Marshal(e.Refs); err == nil && len(e.Refs) > 0 {
		refs = string(b)
	}
	if len(e.Excerpt) > eventExcerptLimit {
		e.Excerpt = e.Excerpt[:eventExcerptLimit]
	}
	_, err := x.Exec(`
INSERT INTO memory_events(at, op, name, project, space, kind, type, title, description, tags, refs, excerpt, source_session, source_message, actor, recorded_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		e.At, e.Op, e.Name, e.Project, e.Space, e.Kind, e.Type, e.Title, e.Desc,
		tags, refs, e.Excerpt, e.SourceSession, e.SourceMessage, e.Actor, e.RecordedAt)
	return err
}

// LoadEvents 读出全部事件（seq 升序 = 投影输入序）。单行损坏（历史演进/手改）
// 跳过不阻断——投影要求确定性，不要求每一行都活着。
func (l *EventLog) LoadEvents() ([]Event, error) {
	if l == nil || l.DB == nil {
		return nil, nil
	}
	rows, err := l.DB.Query(`SELECT seq, at, op, name, project, space, kind, type, title, description, tags, refs, excerpt, source_session, source_message, actor, recorded_at FROM memory_events ORDER BY seq`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		var tags, refs string
		if err := rows.Scan(&e.Seq, &e.At, &e.Op, &e.Name, &e.Project, &e.Space, &e.Kind, &e.Type,
			&e.Title, &e.Desc, &tags, &refs, &e.Excerpt, &e.SourceSession, &e.SourceMessage, &e.Actor, &e.RecordedAt); err != nil {
			continue
		}
		// 旧行归一（V21 前写入的行 recorded_at=0=没有记录时间真相）：回落
		// 事实时间，保证消费端（投影/视图）拿到的时间轴恒非零。
		if e.RecordedAt == 0 {
			e.RecordedAt = e.At
		}
		_ = json.Unmarshal([]byte(tags), &e.Tags)
		_ = json.Unmarshal([]byte(refs), &e.Refs)
		if len(e.Tags) == 0 {
			e.Tags = nil // 归一 nil 口径：投影输入两条路（内存构造/日志读回）同形
		}
		if len(e.Refs) == 0 {
			e.Refs = nil
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// MaxEventSeq 返回日志当前最大 seq（空日志返回 0）。物化水位对账用。
func (l *EventLog) MaxEventSeq() (int64, error) {
	if l == nil || l.DB == nil {
		return 0, nil
	}
	var max sql.NullInt64
	if err := l.DB.QueryRow(`SELECT MAX(seq) FROM memory_events`).Scan(&max); err != nil {
		return 0, err
	}
	return max.Int64, nil
}

// AppendCiteEvents 把一次回合的引用解析结果落成 cite 事件：每个解析命中的键
// 一条（Touch 触达已有挂钩记 touch，这里补「回复引用了它」这一事件事实），
// 每个悬空键也一条（Refs 带 dangling 标记语义：投影不给悬空键建实体节点，
// 但事件与来源边保留——悬空可追溯，这就是「悬空拒写」的留痕侧）。尽力而为：
// 日志不可用静默跳过，绝不影响回合。
func (l *EventLog) AppendCiteEvents(resolved, dangling []string, space string) {
	if l == nil || l.DB == nil {
		return
	}
	now := time.Now().UnixMilli()
	for _, name := range resolved {
		_ = l.AppendEvent(Event{At: now, Op: OpCite, Name: name, Space: space})
	}
	for _, name := range dangling {
		_ = l.AppendEvent(Event{At: now, Op: OpCite, Name: name, Space: space, Refs: []string{"dangling"}})
	}
}
