package types

import "fmt"

// ── 角色状态机 + 职业引用（共享契约）──────────────────────────
//
// 口径来源：docs/distill/05-character-career.md §7.1/§7.2/§7.3。
// 权威决策：
//   - 方案 C（handoff §2-1）：**不建 Career 数据模型**。职业树（3 主 + 2 副）
//     承载于 worldview.json 里 id:"careers" 的 section；角色只存
//     {career_id, stage} 引用，职业名记进 Character.Notes。
//     → 因此本文件只定义「引用」与「职业树 section 的载荷」，不定义 Career 主表。
//   - 水位守卫（handoff §3-1）：阶段推进必须带 UpdatedChapter 单调守卫，
//     重跑低章节不得重复升阶。MuMu 的职业推进独缺该守卫（心理/存活都有）。
//   - 亲密度（handoff §3-2）：最长匹配优先 + 优先采纳 LLM 直出的 DeltaHint，
//     禁止子串累加（MuMu 让「不信任」同时命中「信任」+10 与「不信任」-15）。
//   - member_count（handoff §3-3）：**派生值**，不落字段（MuMu 只增不减）。
//   - stage_progress（handoff §3-4）：MuMu 恒 0 且消费方是死组件 → 不实现。

// CharacterCareerRef 角色对职业的单条引用（主/副职业共用）。
//
// UpdatedChapter 是本契约的**关键增量**：职业阶段的水位守卫。
// 归还/推进阶段的实现必须以 `chapterNum >= UpdatedChapter` 为前提，
// 否则视为重放旧章节，直接跳过（MuMu 无此守卫 → 重跑低章节重复升阶）。
type CharacterCareerRef struct {
	CareerID string `json:"career_id"`
	Stage    int    `json:"stage"`
	// UpdatedChapter 该阶段最后一次推进的章号（水位）。0 = 未记录（视为最低水位）。
	UpdatedChapter int `json:"updated_chapter,omitempty"`
	// CareerName 冗余快照，便于前端在 worldview.json 缺失时仍可显示。
	CareerName string `json:"career_name,omitempty"`
}

// OrgMember 组织成员关系（取代只增不减的 member_count）。
type OrgMember struct {
	CharacterID string `json:"character_id"`
	Position    string `json:"position"`
	Rank        int    `json:"rank"`
	Status      string `json:"status"`              // active / retired / expelled / deceased
	Loyalty     int    `json:"loyalty"`             // 0..100，缺省语义 50
	JoinedAt    string `json:"joined_at,omitempty"` // "第N章"
	LeftAt      string `json:"left_at,omitempty"`
	// UpdatedChapter 成员状态水位（同职业守卫口径）。
	UpdatedChapter int    `json:"updated_chapter,omitempty"`
	Notes          string `json:"notes,omitempty"`
}

// RelChange 关系变更时间线的一条。
type RelChange struct {
	Chapter     int    `json:"chapter"`
	Description string `json:"description"`
}

// CareerDefinition 职业定义（方案 C 的承载形态）。
//
// ★ 这不是「新增 Career 数据模型」：它只是 worldview.json 中
// id:"careers" 那个 section 的 Content 文本的**结构化可选解析结果**，
// 用于把职业树注入角色生成 Prompt（P1 优先级）。
// 权威决策仍是：职业不随角色复制、不建独立文件、不建独立表。
type CareerDefinition struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	Type             string        `json:"type"` // main / sub
	Category         string        `json:"category,omitempty"`
	Description      string        `json:"description,omitempty"`
	Stages           []CareerStage `json:"stages,omitempty"` // 阶段名表
	MaxStage         int           `json:"max_stage,omitempty"`
	Requirements     string        `json:"requirements,omitempty"`
	SpecialAbilities string        `json:"special_abilities,omitempty"`
	WorldviewRules   string        `json:"worldview_rules,omitempty"`
	Source           string        `json:"source,omitempty"` // ai / manual
}

// CareerStage 职业的一个阶段。注意不含 stage_progress（MuMu 死字段，不实现）。
type CareerStage struct {
	Level       int    `json:"level"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// CareersSectionID 方案 C 中承载职业树的 worldview section id。
const CareersSectionID = "careers"

// CareerMainLabel 渲染主职业标签，如「剑修·3阶」。
//
// 回灌通道（05 §7.4）用：MuMu 渲染「剑修(3/10阶)」带最大阶；gaea 方案 C 不建
// Career 主表，worldview careers section 是 markdown 文本——对它做死解析不如
// 不做，故不渲染最大阶。MainCareerID 无独立名称快照（SubCareers 才有
// CareerName），值域即「名称式 ID」直渲染。无主职业返回空串。
func (c Character) CareerMainLabel() string {
	if c.MainCareerID == "" {
		return ""
	}
	if c.MainCareerStage <= 0 {
		return c.MainCareerID
	}
	return fmt.Sprintf("%s·%d阶", c.MainCareerID, c.MainCareerStage)
}

// CareerSubLabels 渲染副职业标签列表（「炼丹师·2阶」）。CareerName 冗余快照
// 优先，缺失退 career_id；两者皆空跳过（防御空引用行）。
func (c Character) CareerSubLabels() []string {
	out := make([]string, 0, len(c.SubCareers))
	for _, ref := range c.SubCareers {
		name := ref.CareerName
		if name == "" {
			name = ref.CareerID
		}
		if name == "" {
			continue
		}
		if ref.Stage > 0 {
			out = append(out, fmt.Sprintf("%s·%d阶", name, ref.Stage))
		} else {
			out = append(out, name)
		}
	}
	return out
}

// ── 章节分析 → 角色域差分（ApplyChapterDiff 的输入）──────────

// CharacterStateDiff 一次章节分析对单个角色的净变更。
type CharacterStateDiff struct {
	Name       string              `json:"name"`
	NewState   string              `json:"new_state,omitempty"` // 心理/处境新值
	Status     string              `json:"status,omitempty"`    // 存活状态新值 Alive/Dead/Missing/Transformed
	KeyEvent   string              `json:"key_event,omitempty"`
	DeltaHint  int                 `json:"delta_hint,omitempty"` // LLM 直出的亲密度增量提示
	Reason     string              `json:"reason,omitempty"`
	CareerIncs []CareerStageChange `json:"career_increments,omitempty"`
}

// CareerStageChange 职业阶段推进请求。
type CareerStageChange struct {
	CareerID string `json:"career_id"`
	IsMain   bool   `json:"is_main"` // true = 主职业，false = 副职业
	NewStage int    `json:"new_stage"`
	Reason   string `json:"reason,omitempty"`
}

// OrgStateDiff 一次章节分析对单个组织的净变更。
type OrgStateDiff struct {
	OrgName          string            `json:"org_name"`
	PowerValue       *int              `json:"power_value,omitempty"` // 0..100
	Destroyed        *bool             `json:"destroyed,omitempty"`
	DestroyedChapter int               `json:"destroyed_chapter,omitempty"`
	Members          []OrgMemberChange `json:"member_changes,omitempty"`
}

// OrgMemberChange 组织成员的加入/离开/状态变更。
//
// DeltaHint / LoyaltyHint 为 gaea 新增（handoff §3-2 的延伸）：
// 让 LLM 直接给数值，不再靠关键词词典猜。
type OrgMemberChange struct {
	CharacterID string `json:"character_id"`
	ChangeType  string `json:"change_type"` // joined / left / expelled / betrayed / promoted / status
	Position    string `json:"position,omitempty"`
	Rank        int    `json:"rank,omitempty"`
	Status      string `json:"status,omitempty"`
	LoyaltyHint *int   `json:"loyalty_hint,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

// OrgMemberChangeType 组织成员变更类型常量。
const (
	OrgMemberJoined   = "joined"
	OrgMemberLeft     = "left"
	OrgMemberExpelled = "expelled"
	OrgMemberBetrayed = "betrayed"
	OrgMemberPromoted = "promoted"
	OrgMemberStatus   = "status"
)

// RelationshipChange 一次章节分析对单条关系的净变更。
type RelationshipChange struct {
	FromName   string `json:"from_name"`
	ToName     string `json:"to_name"`
	ChangeDesc string `json:"change_desc"`          // 关系变化描述（关键词回退时用它算增量）
	DeltaHint  int    `json:"delta_hint,omitempty"` // LLM 直出的亲密度增量（优先采纳）
	Status     string `json:"status,omitempty"`     // active / broken / past / complicated
	NewType    string `json:"new_type,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

// ChapterDiffResult 一次章节差分应用的净产出（UI toast / 日志 / 幂等校验）。
//
// 与 05 §7.2 的 Outcome 同名同义；放在共享契约层供 analysis / 未来 characterstate
// 共同引用，避免各域各定义一份漂移。
type ChapterDiffResult struct {
	StateUpdated     int      `json:"state_updated"`
	RelCreated       int      `json:"rel_created"`
	RelUpdated       int      `json:"rel_updated"`
	OrgMemberUpdated int      `json:"org_member_updated"`
	OrgStateUpdated  int      `json:"org_state_updated"`
	CareerUpdated    int      `json:"career_updated"`
	Changes          []string `json:"changes,omitempty"`
	// Skipped 因水位守卫被拒绝的变更（重放低章节的证据，便于测试断言）。
	Skipped []string `json:"skipped,omitempty"`
}

// ── 派生值与守卫 helper（纯函数，无副作用，供各域直接复用）────

// ActiveMemberCount 派生活跃成员数。
//
// ★ member_count 一律由本函数派生，**不得**存储累加字段：
// MuMu 只在新建成员行时 +1、离开不减（handoff §3-3），单调虚高后靠两个
// 补丁 API 兜底。派生值天然免疫该缺陷。
func ActiveMemberCount(list []OrgMember) int {
	n := 0
	for _, m := range list {
		if m.Status == "" || m.Status == "active" {
			n++
		}
	}
	return n
}

// CareerStageAdvanceAllowed 职业阶段推进的水位守卫。
//
// 返回 true 仅当 chapterNum >= ref.UpdatedChapter（同章重跑允许幂等覆盖，
// 低章节重放拒绝）。MuMu 的职业推进缺该守卫（handoff §3-1）。
func CareerStageAdvanceAllowed(ref CharacterCareerRef, chapterNum int) bool {
	if chapterNum <= 0 {
		return false
	}
	return chapterNum >= ref.UpdatedChapter
}

// SurvivalStatusAdvanceAllowed 存活状态推进的水位守卫（对应 MuMu :214 有守卫的语义）。
func SurvivalStatusAdvanceAllowed(c Character, chapterNum int) bool {
	if chapterNum <= 0 {
		return false
	}
	return chapterNum >= c.StatusChangedChapter
}

// StateAdvanceAllowed 心理/处境状态推进的水位守卫（对应 MuMu :293 有守卫的语义）。
func StateAdvanceAllowed(c Character, chapterNum int) bool {
	if chapterNum <= 0 {
		return false
	}
	return chapterNum >= c.StateUpdatedChapter
}

// ClampIntimacy 把亲密度钳制到 [-100, 100]。
func ClampIntimacy(v int) int {
	if v < -100 {
		return -100
	}
	if v > 100 {
		return 100
	}
	return v
}

// ClampDeltaHint 把 LLM 直出的增量提示钳制到 ±30（与 05 §7.3 一致）。
func ClampDeltaHint(v int) int {
	if v < -30 {
		return -30
	}
	if v > 30 {
		return 30
	}
	return v
}
