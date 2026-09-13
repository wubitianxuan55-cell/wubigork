package types

// ── 伏笔（v2 共享契约）────────────────────────────────────────
//
// 口径来源：docs/distill/01-foreshadow-spec.md §2.1。
// 权威决策（docs/distill/09-impl-handoff.md §2-2/§2-3）：
//   - 存储 A1：主存储仍是 foreshadows.json，不引 SQLite 主表；
//     章号由 chapterNumOf() 从 PlantedIn 派生，本文件不存任何章号冗余字段。
//   - 状态机 = 并集 6 态，**wire 值保持 gaea 原生口径**：写入口径 = "revealed"
//     （前端回收率、后端统计、分析写回与既有测试全依赖该字符串）；MuMu 的
//     "resolved" 只作**读取别名**，读取时归一化为 "revealed"。
//     handoff §2-3/§4-7：把 revealed 改名或把读取归一化为 resolved 都是破坏性变更
//     （会让回收率/lint 计数静默归零，并让状态校验拒收新值）。
//   - description 字段名不改（不对齐 MuMu 的 content），避免存量 JSON/前端断裂。
//
// 零破坏迁移约束：本文件所有新增字段一律 `omitempty`，既有字段名/类型不动。
// 旧 foreshadows.json（只有 id/category/description/planted_in/revealed_in/
// status/is_long_term）必须原样解析成功——见 foreshadow_v2_test.go。

// ForeshadowStatus 伏笔状态。
//
// 并集 6 态（写入口径）：
//
//	pending            已规划未埋入
//	planted            已埋入
//	hinted             已暗示（gaea 原生，语义介于 pending/planted 之间）
//	revealed           已回收（gaea 原生 wire 值，写入统一用它）
//	partially_resolved 部分回收
//	abandoned          已废弃
//
// 另有 1 个**兼容别名**（仅用于读取外部数据，不得用于新写入）：
//
//	resolved           MuMu/外部数据的等价状态，读取时归一化为 revealed
type ForeshadowStatus string

const (
	// ForeshadowPending 已规划未埋入（MuMu 创建时的初始态）。
	ForeshadowPending ForeshadowStatus = "pending"
	// ForeshadowPlanted 已埋入。
	ForeshadowPlanted ForeshadowStatus = "planted"
	// ForeshadowHinted 已暗示（gaea 原生态，并集保留）。
	ForeshadowHinted ForeshadowStatus = "hinted"
	// ForeshadowRevealed 已回收（★ 写入口径：gaea v1 起的 wire 值）。
	//
	// 注意：该常量字面量必须保持 "revealed"——前端 ForeshadowPanel 回收率、
	// internal/stats 计数、internal/analysis 写回与既有测试都按该字符串比较，
	// 改名会让这些消费方静默读空（handoff §4-7）。
	ForeshadowRevealed ForeshadowStatus = "revealed"
	// ForeshadowResolved 兼容别名：外部数据（MuMu）用 "resolved" 表达同一状态。
	// 读取时用 NormalizeForeshadowStatus() 归一化；新写入不得再产出该值。
	ForeshadowResolved ForeshadowStatus = "resolved"
	// ForeshadowPartial 部分回收。
	ForeshadowPartial ForeshadowStatus = "partially_resolved"
	// ForeshadowAbandoned 已废弃。
	ForeshadowAbandoned ForeshadowStatus = "abandoned"
)

// AllForeshadowStatuses 并集 6 态（不含兼容别名 resolved），供 UI 排序/校验复用。
func AllForeshadowStatuses() []ForeshadowStatus {
	return []ForeshadowStatus{
		ForeshadowPending,
		ForeshadowPlanted,
		ForeshadowHinted,
		ForeshadowRevealed,
		ForeshadowPartial,
		ForeshadowAbandoned,
	}
}

// IsForeshadowStatusAlias 报告 s 是否为兼容别名（目前仅 resolved）。
func IsForeshadowStatusAlias(s ForeshadowStatus) bool {
	return s == ForeshadowResolved
}

// NormalizeForeshadowStatus 把读取到的状态归一化为写入口径。
//
// 只做一件事：resolved → revealed。其余值（含未知值）原样返回，不猜、不丢——
// 未知状态交由校验层报错，而不是在这里静默改写。
func NormalizeForeshadowStatus(s ForeshadowStatus) ForeshadowStatus {
	if s == ForeshadowResolved {
		return ForeshadowRevealed
	}
	return s
}

// IsForeshadowStatusValid 报告 s 是否属于并集 6 态（兼容别名不算合法写入值）。
func IsForeshadowStatusValid(s ForeshadowStatus) bool {
	switch s {
	case ForeshadowPending, ForeshadowPlanted, ForeshadowHinted,
		ForeshadowRevealed, ForeshadowPartial, ForeshadowAbandoned:
		return true
	}
	return false
}

// IsResolvedStatus 报告 s 是否表示「已回收」（兼容两套 wire 值）。
//
// 业务代码判断「是否已回收」一律走本助手，不要写字面量比较——
// 这样 gaea 存量（revealed）与外部导入（resolved）两条数据都判得对。
func IsResolvedStatus(s ForeshadowStatus) bool {
	return s == ForeshadowRevealed || s == ForeshadowResolved
}

// ForeshadowCategory 伏笔分类（对齐 MuMu schemas/foreshadow.py:23-31，7 值）。
//
// ★ 零破坏决策：Foreshadow.Category 的**字段类型保持 `string`**（v1 原样），
// 不改成命名类型 `ForeshadowCategory`。原因：改成命名类型会迫使
// `internal/analysis` 等域外消费方改动（它们直接赋 `string`），
// 与「零破坏存量、不改 out-of-scope 代码」冲突。
// 因此这里用**无类型字符串常量**表达值域：既可赋给 string 字段，也可比较。
//
// 读取需同时接受 gaea 存量 4 值（character / plot / world / relationship）与 v2 7 值。
const (
	ForeshadowIdentity     = "identity"     // 身世
	ForeshadowMystery      = "mystery"      // 悬念
	ForeshadowItem         = "item"         // 物品
	ForeshadowRelationship = "relationship" // 关系
	ForeshadowEvent        = "event"        // 事件
	ForeshadowAbility      = "ability"      // 能力
	ForeshadowProphecy     = "prophecy"     // 预言
)

// ForeshadowCategories 返回 v2 的 7 值分类（供校验/UI）。
func ForeshadowCategories() []string {
	return []string{
		ForeshadowIdentity, ForeshadowMystery, ForeshadowItem,
		ForeshadowRelationship, ForeshadowEvent, ForeshadowAbility, ForeshadowProphecy,
	}
}

// ForeshadowSourceType 伏笔来源，驱动清理语义。
type ForeshadowSourceType string

const (
	// ForeshadowSourceAnalysis 分析提取（可再生副产物，重分析时可批量删）。
	ForeshadowSourceAnalysis ForeshadowSourceType = "analysis"
	// ForeshadowSourceManual 手动登记（作者资产，只重置不删）。
	ForeshadowSourceManual ForeshadowSourceType = "manual"
)

// ResolveStatus 回收时机判定（四值枚举），由 UrgencyLevel 之前的分类逻辑产出。
type ResolveStatus string

const (
	ResolveMustNow ResolveStatus = "must_resolve_now" // 本章必须回收
	ResolveOverdue ResolveStatus = "overdue"          // 已超期
	ResolveNotYet  ResolveStatus = "not_yet"          // 计划在未来章，禁止提前回收
	ResolveNoPlan  ResolveStatus = "no_plan"          // 无明确回收计划
)

// Foreshadow 伏笔追踪条目 v2。
//
// 字段增删遵循两条硬约束（handoff §2-3 / §4-4）：
//   - 既有字段名与类型不改动（id/category/description/planted_in/revealed_in/status/is_long_term）；
//   - 新增字段一律可选（omitempty），旧 JSON 缺省即零值。
type Foreshadow struct {
	// ── 标识（沿用 gaea 原生）──
	ID       string `json:"id"`       // stable_id = sha256(category+chapterFile+description)[:8]
	Category string `json:"category"` // ★ 字段类型保持 string（v1 兼容）；v2 值域扩为 7 值

	// ── 文本（MuMu: title / content / hint_text / resolution_text）──
	Title          string `json:"title,omitempty"`           // ≤200 字
	Description    string `json:"description"`               // ★ 既有字段名不改，= MuMu 的 content
	HintText       string `json:"hint_text,omitempty"`       // 埋入时的暗示文本（原文摘录）
	ResolutionText string `json:"resolution_text,omitempty"` // 回收时的揭示文本（原文摘录）

	// ── 来源 ──
	SourceType     ForeshadowSourceType `json:"source_type,omitempty"`         // analysis / manual
	SourceMemoryID string               `json:"source_memory_id,omitempty"`    // = 稳定 ID（幂等去重键）
	SourceAnalysis string               `json:"source_analysis_ref,omitempty"` // 分析记录引用（可选）

	// ── 章节关联（用章节文件名，非 ID —— 与 gaea 一致）──
	PlantedIn       string `json:"planted_in"`                  // ★ 既有，埋入章文件名
	TargetResolveIn string `json:"target_resolve_in,omitempty"` // 计划回收章文件名（★ 本域最大增量）
	RevealedIn      string `json:"revealed_in,omitempty"`       // ★ 既有，实际回收章文件名

	// ── 状态 ──
	Status ForeshadowStatus `json:"status"`

	// ── 评分（新增）──
	Importance float64 `json:"importance,omitempty"` // 0.0-1.0，缺省语义为 0.5
	Strength   int     `json:"strength,omitempty"`   // 1-10，缺省语义为 5
	Subtlety   int     `json:"subtlety,omitempty"`   // 1-10，缺省语义为 5

	// ── 长线 ──
	IsLongTerm bool `json:"is_long_term"` // ★ 既有；口径 = 跨越 10 章（与 stale 阈值同源，不另立第二套）

	// ── 关联（新增）──
	RelatedCharacters  []string `json:"related_characters,omitempty"`
	RelatedForeshadows []string `json:"related_foreshadow_ids,omitempty"` // 伏笔链
	Tags               []string `json:"tags,omitempty"`

	// ── 备注（新增）──
	Notes           string `json:"notes,omitempty"`            // 创作备注（仅作者可见）
	ResolutionNotes string `json:"resolution_notes,omitempty"` // 回收方式说明

	// ── 注入控制（新增；bool 用指针以区分「未设置」与「显式 false」）──
	AutoRemind           *bool `json:"auto_remind,omitempty"`            // 缺省语义 true
	RemindBeforeChapters int   `json:"remind_before_chapters,omitempty"` // 缺省语义 5，1..20
	IncludeInContext     *bool `json:"include_in_context,omitempty"`     // 缺省语义 true

	// ── 时间戳（新增，ISO8601 字符串，与 gaea 其它文件一致用 string 不用 time.Time）──
	CreatedAt  string `json:"created_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
	PlantedAt  string `json:"planted_at,omitempty"`
	ResolvedAt string `json:"resolved_at,omitempty"`
}

// ForeshadowUrgency 超期紧急度的运行时投影。
//
// 权威决策（spec §A5）：urgency **不落库**，由后端按「计划 vs 实际」偏差实时计算；
// 落库需在每次章节推进后全量重写，脆。前端只渲染后端结果，不再自算章号（修正 D7）。
type ForeshadowUrgency struct {
	// Level 0=不紧急 1=需关注 2=急需回收 3=已超期。
	Level int `json:"level"`
	// RemainingChapters = 计划回收章 − 当前章（可为负）。
	RemainingChapters int `json:"remaining_chapters"`
	// OverdueChapters 已超期章数（未超期为 0）。
	OverdueChapters int `json:"overdue_chapters,omitempty"`
	// ResolveStatus 回收时机判定（must_resolve_now / overdue / not_yet / no_plan）。
	ResolveStatus ResolveStatus `json:"resolve_status"`
	// MustResolve 本条目是否进入「本章必须回收」集合。
	//
	// ★ 显式字段，不得省略：MuMu 服务层返回该键但响应模型无此字段，被静默丢弃
	// （handoff §3-8）。gaea 必须把结果显式带出。
	MustResolve bool `json:"must_resolve"`
}

// ForeshadowAmbiguity 按描述内容匹配到多个候选时的歧义记录。
//
// 权威决策（handoff §3-9 相邻约束）：引用失效时**禁止回落内容匹配**
// （宁可漏收不可错收），因此歧义只做上报，不自动选择。
type ForeshadowAmbiguity struct {
	RefID      string   `json:"ref_id"`
	Title      string   `json:"title,omitempty"`
	Candidates []string `json:"candidates"`
	Message    string   `json:"message"`
}

// ForeshadowFile foreshadows.json 完整结构。
//
// v1 兼容：Items 字段名不变。SchemaVersion 新增且 omitempty：旧文件缺省为 0，
// 读取层应把 0 视为 v1、2 视为 v2，不得因缺该字段而拒绝加载。
type ForeshadowFile struct {
	SchemaVersion int          `json:"schema_version,omitempty"` // v2 = 2；缺省（0）视为 v1
	Items         []Foreshadow `json:"items"`
}
