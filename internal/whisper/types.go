// Package whisper — 轻语模块：AI gaea引擎（100% 对齐 ackem engine/types.ts）
package whisper

import "time"

// ─── 关系系统 ──────────────────────────────────────────────────

// RelationshipStage 关系三阶段
type RelationshipStage string

const (
	StageStranger RelationshipStage = "STRANGER"
	StageFamiliar RelationshipStage = "FAMILIAR"
	StageIntimate RelationshipStage = "INTIMATE"
)

// Atmosphere 氛围三元标签
type Atmosphere string

const (
	AtmoWarm    Atmosphere = "warm"
	AtmoNeutral Atmosphere = "neutral"
	AtmoCool    Atmosphere = "cool"
)

// ─── L0 事件系统 ──────────────────────────────────────────────

// EventType 事件分类（14种：10基础 + 4成人）
type EventType string

const (
	EvtPraise         EventType = "praise"
	EvtTease          EventType = "tease"
	EvtCasualChat     EventType = "casual_chat"
	EvtCold           EventType = "cold"
	EvtHurtful        EventType = "hurtful"
	EvtApology        EventType = "apology"
	EvtVulnerable     EventType = "vulnerable"
	EvtQuestion       EventType = "question"
	EvtExtremeRedline EventType = "extreme_redline"
	// 成人模式事件
	EvtAdultFlirt      EventType = "adult_flirt"
	EvtAdultDominant   EventType = "adult_dominant"
	EvtAdultSubmissive EventType = "adult_submissive"
	EvtAdultExplicit   EventType = "adult_explicit"
)

// Event L0 输出结构
type Event struct {
	Type             EventType `json:"type"`
	Intensity        float64   `json:"intensity"` // 0-1
	Sincerity        float64   `json:"sincerity"` // 0-1
	IsExtremeRedline bool      `json:"isExtremeRedline"`
	IsAdultContent   bool      `json:"isAdultContent"`
	AdultSubtype     string    `json:"adultSubtype,omitempty"` // flirt/dominant/submissive/explicit/romantic
}

// ─── L1 关系状态 ──────────────────────────────────────────────

// L1State 关系状态机
type L1State struct {
	Stage                    RelationshipStage `json:"stage"`
	Trust                    float64           `json:"trust"` // 0-100
	Rifts                    int               `json:"rifts"`
	AffectionMomentum        float64           `json:"affection_momentum"`
	Atmosphere               Atmosphere        `json:"atmosphere"`
	ConsecutivePositiveTurns int               `json:"consecutivePositiveTurns"`
	TurnsSinceLastRift       int               `json:"turnsSinceLastRift"`
	SharedEventsCount        int               `json:"sharedEventsCount"`
}

// Modulation L1 调制输出
type Modulation struct {
	TrustMod    float64    `json:"trustMod"`
	RiftMod     float64    `json:"riftMod"`
	StageWeight float64    `json:"stageWeight"`
	Atmosphere  Atmosphere `json:"atmosphere"`
}

// ExternalAtmosphere P1-4 外场气氛
type ExternalAtmosphere struct {
	Level float64    `json:"level"` // -1..1
	Label Atmosphere `json:"label"`
}

// ─── L2 情绪系统 ──────────────────────────────────────────────

// Emotion4D PAD 四维情绪
type Emotion4D struct {
	Aff float64 `json:"aff"` // affection 愉悦度
	Sec float64 `json:"sec"` // security 安全感
	Aro float64 `json:"aro"` // arousal 唤醒度
	Dom float64 `json:"dom"` // dominance 掌控感
}

// EmotionState 情绪完整状态（含锁机制）
type EmotionState struct {
	Aff          float64 `json:"aff"`
	Sec          float64 `json:"sec"`
	Aro          float64 `json:"aro"`
	Dom          float64 `json:"dom"`
	PrimaryLabel string  `json:"primaryLabel"`
	IsLocked     bool    `json:"isLocked"`
	// Mood 长期心境（v4.3d 新增）：4D 慢速 EWMA（α=MoodAlpha）向即时情绪靠拢，
	// 与即时情绪同量纲（-100..100），随 FullState 经 JSON 自动持久化；
	// 全 0 视为未播种（新会话），EmotionStep 以即时情绪首值播种。
	Mood [4]float64 `json:"mood"`
}

// MemoryEcho 记忆回声
type MemoryEcho struct {
	Aff float64
	Sec float64
	Aro float64
	Dom float64
}

// ─── 人格系统 ──────────────────────────────────────────────────

// PersonalityDims TISOR 五维性格
type PersonalityDims struct {
	T float64 `json:"T"` // Tenderness 温柔度
	I float64 `json:"I"` // Initiative 主动性
	S float64 `json:"S"` // Submission 顺从度
	O float64 `json:"O"` // Originality 独特度
	R float64 `json:"R"` // Reserve 矜持度
}

// PersonalityPreset 人格预设
type PersonalityPreset struct {
	ID              string           `json:"id"`
	Label           string           `json:"label"`
	Gender          string           `json:"gender"` // male/female
	T, I, S, O, R   float64          `json:"-"`
	Dims            PersonalityDims  `json:"dims"`
	Tags            []string         `json:"tags,omitempty"`
	HiddenPersona   *PersonalityDims `json:"hiddenPersona,omitempty"`
	RequiresAdult18 bool             `json:"requiresAdult18,omitempty"`
	VoiceGuide      string           `json:"voiceGuide,omitempty"`
}

// PersonalityTemplate 人格详细模板（P2新增：对齐 ackem prompt/personality.ts）
type PersonalityTemplate struct {
	ID                string   `json:"id"`
	Label             string   `json:"label"`
	Gender            string   `json:"gender"`
	CoreContradiction string   `json:"coreContradiction"` // 核心矛盾
	SpeechPatterns    []string `json:"speechPatterns"`    // 常用语癖
	SpeakingStyle     string   `json:"speakingStyle"`     // 说话方式
	Prohibitions      []string `json:"prohibitions"`      // 人格专属禁止
	ExamplesLow       []string `json:"examplesLow"`       // 低亲密示例
	ExamplesMedium    []string `json:"examplesMedium"`    // 中亲密示例
	ExamplesHigh      []string `json:"examplesHigh"`      // 高亲密示例
}

// ─── 情绪涌现 ──────────────────────────────────────────────────

// EmergenceType 涌现类型
type EmergenceType string

const (
	EmergenceTimeReflection      EmergenceType = "timeReflection"
	EmergenceLateNightEmo        EmergenceType = "lateNightEmo"
	EmergenceExistentialWonder   EmergenceType = "existentialWonder"
	EmergenceAttachmentOverflow  EmergenceType = "attachmentOverflow"
	EmergenceVulnerabilityReveal EmergenceType = "vulnerabilityReveal"
	EmergenceDesireExpression    EmergenceType = "desireExpression"
)

// EmergenceState 涌现状态
type EmergenceState struct {
	Type          EmergenceType          `json:"type"`
	Intensity     float64                `json:"intensity"`
	Flavor        string                 `json:"flavor"`
	Phase         string                 `json:"phase"` // rising/sustained/fading/dissolved/broken
	StartedAt     time.Time              `json:"startedAt"`
	RoundsInPhase int                    `json:"roundsInPhase"`
	HasExpressed  bool                   `json:"hasExpressed"`
	Context       map[string]interface{} `json:"context"`
}

// EmergenceContext 涌现上下文
type EmergenceContext struct {
	Emotion                    EmotionState
	Stage                      RelationshipStage
	Trust                      float64
	Atmosphere                 string
	TimeOfDay                  string
	DaysSinceMet               int
	RecentAffHistory           []float64
	RecentEventTypes           []string
	ConsecutiveMeaningfulTurns int
	ConsecutiveVulnerableTurns int
	LastEmergence              *struct {
		Type string
		Turn int
	}
	LastSameTypeAt   *time.Time
	LastSameTypeTurn *int
	CurrentTurn      int
}

// EmergencePersistence 涌现持久化记录
type EmergencePersistence struct {
	Active  *EmergenceState         `json:"active"`
	History []EmergenceHistoryEntry `json:"history"`
}

type EmergenceHistoryEntry struct {
	Type              string    `json:"type"`
	LastTriggeredAt   time.Time `json:"lastTriggeredAt"`
	LastTriggeredTurn int       `json:"lastTriggeredTurn"`
}

// ─── 欲望栈 ────────────────────────────────────────────────────

// Desire 单个欲望
type Desire struct {
	ID              string    `json:"id"`
	Topic           string    `json:"topic"`
	Category        string    `json:"category"` // curiosity/concern/share/tease/suggest
	Urgency         float64   `json:"urgency"`  // 0-10
	Status          string    `json:"status"`   // latent/active/expressed/settled
	SourceTurn      int       `json:"sourceTurn"`
	CreatedAt       time.Time `json:"createdAt"`
	ExpressedAtTurn *int      `json:"expressedAtTurn,omitempty"`
}

// DesireStack 欲望栈（最多5槽）
type DesireStack struct {
	Slots []*Desire `json:"slots"`
}

// OfflineThought P2-4 离线思维
type OfflineThought struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	Delivered bool      `json:"delivered"`
}

// ─── 表达参数 ──────────────────────────────────────────────────

// ExpressionParams L5 表达参数
type ExpressionParams struct {
	Mode      string `json:"mode"`      // NORMAL/SILENT_CANDIDATE
	Proximity string `json:"proximity"` // CLOSE/NEUTRAL/COOL/DEFENSIVE
	Tone      string `json:"tone"`
	Length    string `json:"length"` // SHORT/MEDIUM/LONG
}

// ─── OEG 创造者叙事 ───────────────────────────────────────────

// OriginExposureState 创造者曝光状态
type OriginExposureState string

const (
	OriginNormal        OriginExposureState = "NORMAL"
	OriginEntry         OriginExposureState = "ENTRY"
	OriginExplore       OriginExposureState = "EXPLORE"
	OriginDeep          OriginExposureState = "DEEP"
	OriginGuardCooldown OriginExposureState = "GUARD_COOLDOWN"
)

// OriginExposure 创造者曝光追踪
type OriginExposure struct {
	State             OriginExposureState `json:"state"`
	Streak            int                 `json:"streak"`
	CooldownUntilTurn int                 `json:"cooldownUntilTurn"`
}

// ─── 用户画像 ──────────────────────────────────────────────────

// UserProfile 用户画像（对齐 ackem userProfile）
type UserProfile struct {
	DominantArchetype   string  `json:"dominantArchetype"`
	SexualDirectness    float64 `json:"sexualDirectness"`
	DominancePreference float64 `json:"dominancePreference"`
	EmotionalNeediness  float64 `json:"emotionalNeediness"`
	TrustTrajectory     string  `json:"trustTrajectory"`
	LastUpdated         string  `json:"lastUpdated"`
	DetectedAtTurn      int     `json:"detectedAtTurn"`
}

// ─── 记忆绑定 ──────────────────────────────────────────────────

// EmotionalContext 情感上下文快照（L4→L1/L2 桥梁）
type EmotionalContext struct {
	Valence    float64           `json:"valence"`
	Intensity  float64           `json:"intensity"`
	RelStage   RelationshipStage `json:"relStage"`
	Trust      float64           `json:"trust"`
	Atmosphere Atmosphere        `json:"atmosphere"`
}

// MemoryAugmentedL1 记忆增强后的 L1 附加值
type MemoryAugmentedL1 struct {
	SharedEventsCount int `json:"sharedEventsCount"`
}

// ─── FullState 引擎完整状态 ───────────────────────────────────

// FullState 引擎完整状态（对齐 ackem FullState）
type FullState struct {
	Version              string                `json:"version"`
	Relationship         L1State               `json:"relationship"`
	Emotion              EmotionState          `json:"emotion"`
	Counters             StateCounters         `json:"counters"`
	LastActive           time.Time             `json:"lastActive"`
	ExternalAtmosphere   ExternalAtmosphere    `json:"externalAtmosphere"`
	PersonalityBaseline  *PersonalityDims      `json:"personalityBaseline,omitempty"`
	Personality          PersonalitySlice      `json:"personality"`
	UserProfile          *UserProfile          `json:"userProfile,omitempty"`
	DesireStack          DesireStack           `json:"desireStack"`
	OfflineThoughts      []OfflineThought      `json:"offlineThoughts"`
	EmergencePersistence *EmergencePersistence `json:"emergencePersistence,omitempty"`
	FirstMetDate         *time.Time            `json:"firstMetDate,omitempty"`
	AckemBirthday        *time.Time            `json:"ackemBirthday,omitempty"`
	OriginExposure       *OriginExposure       `json:"originExposure,omitempty"`
	// v4.3a: 会客厅关系记忆——关联索引/作息习惯/时间锚点随状态持久化。
	// 落库由 repos 装配进 memory_associations / user_habits / temporal_anchors 三表
	// （见 repos/memory_graph.go）；内存态 ↔ 本字段的双向同步见 memory_graph_persist.go。
	Associations    []Association    `json:"associations,omitempty"`
	Habits          []UserHabit      `json:"habits,omitempty"`
	TemporalAnchors []TemporalAnchor `json:"temporalAnchors,omitempty"`
}

// StateCounters 状态计数器
type StateCounters struct {
	TotalTurns                 int  `json:"totalTurns"`
	SharedEventsCount          int  `json:"sharedEventsCount"`
	ConsecutiveMeaningfulTurns int  `json:"consecutiveMeaningfulTurns"`
	LastConsolidationTurn      *int `json:"lastConsolidationTurn,omitempty"`
	LastMirrorCheckTurn        *int `json:"lastMirrorCheckTurn,omitempty"`
}

// PersonalitySlice 人格切片（嵌入 FullState）
type PersonalitySlice struct {
	PresetID    string  `json:"presetId"`
	HiddenRatio float64 `json:"hiddenRatio,omitempty"`
	T           float64 `json:"T"`
	I           float64 `json:"I"`
	S           float64 `json:"S"`
	O           float64 `json:"O"`
	R           float64 `json:"R"`
}

// ─── TurnTrace 轮次追踪 ──────────────────────────────────────

// TurnTrace 单轮完整追踪
type TurnTrace struct {
	Turn      int          `json:"turn"`
	L0        TurnTraceL0  `json:"l0"`
	L1        TurnTraceL1  `json:"l1"`
	L2        TurnTraceL2  `json:"l2"`
	L3        TurnTraceL3  `json:"l3"`
	L4        TurnTraceL4  `json:"l4"`
	L5        *TurnTraceL5 `json:"l5,omitempty"`
	Timestamp *time.Time   `json:"timestamp,omitempty"`
}

type TurnTraceL0 struct {
	Type      EventType `json:"type"`
	Intensity float64   `json:"intensity"`
	Sincerity float64   `json:"sincerity,omitempty"`
}

type TurnTraceL1 struct {
	Trust      float64           `json:"trust"`
	Rifts      int               `json:"rifts"`
	Stage      RelationshipStage `json:"stage"`
	Atmosphere Atmosphere        `json:"atmosphere,omitempty"`
}

type TurnTraceL2 struct {
	Aff   float64 `json:"aff"`
	Sec   float64 `json:"sec"`
	Aro   float64 `json:"aro"`
	Dom   float64 `json:"dom"`
	Label string  `json:"label"`
}

type TurnTraceL3 struct {
	Silent        bool          `json:"silent"`
	TierBChars    int           `json:"tierBChars"`
	FactsUsed     int           `json:"factsUsed,omitempty"`
	EmbeddingHits int           `json:"embeddingHits,omitempty"`
	TopicSource   string        `json:"topicSource,omitempty"`
	EmergenceType EmergenceType `json:"emergenceType,omitempty"`
}

type TurnTraceL4 struct {
	Wrote bool `json:"wrote"`
}

// TurnTraceL5 工具调用追踪
type TurnTraceL5 struct {
	ToolCalls []string `json:"toolCalls,omitempty"`
}

// ─── 节奏引擎 ──────────────────────────────────────────────────

// RhythmMode 节奏模式
type RhythmMode string

const (
	RhythmChatter   RhythmMode = "chatter"
	RhythmMonologue RhythmMode = "monologue"
	RhythmDefault   RhythmMode = "default"
)

// RhythmDecision 节奏决策
type RhythmDecision struct {
	Mode           RhythmMode `json:"mode"`
	Count          int        `json:"count"`
	Separator      string     `json:"separator"`
	MaxCharsPerMsg int        `json:"maxCharsPerMsg"`
	Instruction    string     `json:"instruction"`
}

// ─── 记忆系统类型 ──────────────────────────────────────────────

// MemoryFact 核心记忆事实
type MemoryFact struct {
	ID              string    `json:"id"`
	Domain          string    `json:"domain"`
	Subcategory     string    `json:"subcategory"`
	Subject         string    `json:"subject"`
	Summary         string    `json:"summary"`
	Weight          float64   `json:"weight"`
	Confidence      float64   `json:"confidence"`
	Status          string    `json:"status"` // active/retired
	SelfRelevance   float64   `json:"selfRelevance"`
	Triggers        []string  `json:"triggers"`
	SourceSessionID string    `json:"sourceSessionId"`
	SourceTurnIndex int       `json:"sourceTurnIndex"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	Tier            string    `json:"tier,omitempty"`         // core/archival
	Sensitivity     string    `json:"sensitivity,omitempty"`  // normal/avoid
	PrivacyLevel    string    `json:"privacyLevel,omitempty"` // normal/intimate/explicit
	// P1新增字段（对齐 ackem MemoryFact）
	EmotionalContext *EmotionalContext `json:"emotionalContext,omitempty"` // 写入时的情感快照
	UpdateTrail      []string          `json:"updateTrail,omitempty"`      // 合并时间轨迹
	DerivedFrom      []string          `json:"derivedFrom,omitempty"`      // consolidated 溯源 ID
	FactLayer        string            `json:"factLayer,omitempty"`        // raw/consolidated
	AgeMeta          *AgeMeta          `json:"ageMeta,omitempty"`          // 年龄元数据
}

// AgeMeta 年龄元数据（对齐 ackem types AgeMeta）
type AgeMeta struct {
	Age          int    `json:"age"`
	BirthdayMMDD string `json:"birthdayMMDD,omitempty"`
	BirthYear    int    `json:"birthYear,omitempty"`
	RecordedAt   string `json:"recordedAt,omitempty"`
	IsEstimate   bool   `json:"isEstimate"`
}

// Episode 情节记忆片段
type Episode struct {
	ID                 string    `json:"id"`
	Summary            string    `json:"summary"`
	EmotionalIntensity float64   `json:"emotionalIntensity"`
	DominantEmotion    string    `json:"dominantEmotion"`
	Keywords           []string  `json:"keywords"`
	PrevEpisodeID      *string   `json:"prevEpisodeId,omitempty"`
	SourceSessionID    string    `json:"sourceSessionId"`
	StartTurn          int       `json:"startTurn"`
	EndTurn            int       `json:"endTurn"`
	CreatedAt          time.Time `json:"createdAt"`
}

// ─── 习惯系统 ──────────────────────────────────────────────────

// UserHabit 用户习惯槽
type UserHabit struct {
	ID              string  `json:"id"`
	Type            string  `json:"type"`              // dnd/suppress_type/health_reminder
	Scope           string  `json:"scope"`             // short_term/long_term
	Weekday         *int    `json:"weekday,omitempty"` // 0=Sun..6=Sat, nil=每天
	HourStart       int     `json:"hourStart"`
	HourEnd         int     `json:"hourEnd"`
	Confidence      float64 `json:"confidence"`
	OccurrenceCount int     `json:"occurrenceCount"`
	FirstSeenAt     int64   `json:"firstSeenAt"` // unix ms
	LastConfirmedAt int64   `json:"lastConfirmedAt"`
	ExpiresAt       *int64  `json:"expiresAt,omitempty"`
	Source          string  `json:"source"` // explicit/detected
	SuppressTarget  string  `json:"suppressTarget,omitempty"`
	Note            string  `json:"note,omitempty"`
	CreatedAt       int64   `json:"createdAt"`
	UpdatedAt       int64   `json:"updatedAt"`
}

// ─── 关联索引 ──────────────────────────────────────────────────

// Association 记忆关联边
type Association struct {
	ID              string  `json:"id"`
	FactIDA         string  `json:"factIdA"`
	FactIDB         string  `json:"factIdB"`
	AssociationType string  `json:"associationType"` // temporal/entity/event_chain/emotion_peak/self_reference/thematic
	Strength        float64 `json:"strength"`        // 0-1
	CreatedAt       int64   `json:"createdAt"`
	LastActivatedAt int64   `json:"lastActivatedAt"`
}

// ─── 检索类型 ──────────────────────────────────────────────────

// TemporalContext 时间上下文
type TemporalContext struct {
	TimeOfDay string // morning/afternoon/evening/late_night
	IsWeekend bool
	Month     int
	Season    string // spring/summer/autumn/winter
	Hour      int
	Weekday   int
	GapHours  float64
	LocalDate string
}

// ─── v5.41: Context 系统类型 ─────────────────────────────────

// UserEngagementLevel 用户参与度级别
type UserEngagementLevel string

const (
	EngagementActiveNow      UserEngagementLevel = "active_now"
	EngagementRecentlyActive UserEngagementLevel = "recently_active"
	EngagementIdle           UserEngagementLevel = "idle"
	EngagementLikelyAway     UserEngagementLevel = "likely_away"
)

// CompanionPresenceMode 陪伴在场模式
type CompanionPresenceMode string

const (
	CompanionActive   CompanionPresenceMode = "active"
	CompanionQuiet    CompanionPresenceMode = "quiet"
	CompanionSleeping CompanionPresenceMode = "sleeping"
)

// UserRuntimeContext 用户运行时上下文
type UserRuntimeContext struct {
	LastActiveAt         string              `json:"lastActiveAt"`
	MinutesSinceLastChat int                 `json:"minutesSinceLastChat"`
	Engagement           UserEngagementLevel `json:"engagement"`
	RecentUserSnippets   []string            `json:"recentUserSnippets"`
}

// CompanionRuntimeContext 陪伴运行时上下文
type CompanionRuntimeContext struct {
	Mode              CompanionPresenceMode `json:"mode"`
	IdleDurationMs    int64                 `json:"idleDurationMs"`
	LastInteractionMs int64                 `json:"lastInteractionMs"`
}

// TimeRuntimeContext 本地时钟与时段
type TimeRuntimeContext struct {
	LocalDate string `json:"localDate"`
	LocalTime string `json:"localTime"`
	TimeOfDay string `json:"timeOfDay"`
	Hour      int    `json:"hour"`
	Minute    int    `json:"minute"`
	IsWeekend bool   `json:"isWeekend"`
}

// UserActivityCategory 生活场景大类
type UserActivityCategory string

const (
	ActivityRest          UserActivityCategory = "rest"
	ActivityWork          UserActivityCategory = "work"
	ActivityStudy         UserActivityCategory = "study"
	ActivityTravel        UserActivityCategory = "travel"
	ActivitySocial        UserActivityCategory = "social"
	ActivityEntertainment UserActivityCategory = "entertainment"
	ActivityDaily         UserActivityCategory = "daily"
	ActivityHealth        UserActivityCategory = "health"
	ActivityUnknown       UserActivityCategory = "unknown"
)

// ActivityTense 场景时态
type ActivityTense string

const (
	TenseFuture  ActivityTense = "future"
	TensePresent ActivityTense = "present"
	TensePast    ActivityTense = "past"
)

// UserActivityContext 用户生活场景推断
type UserActivityContext struct {
	Category   UserActivityCategory `json:"category"`
	Tense      ActivityTense        `json:"tense"`
	Label      string               `json:"label"`
	Confidence float64              `json:"confidence"`
	Source     []string             `json:"source"`
}

// ForegroundScene 前台窗口场景
type ForegroundScene string

const (
	SceneMeeting      ForegroundScene = "meeting"
	ScenePresentation ForegroundScene = "presentation"
	SceneFocus        ForegroundScene = "focus"
	SceneOther        ForegroundScene = "other"
)

// ForegroundSnapshot 前台窗口快照
type ForegroundSnapshot struct {
	Enabled              bool            `json:"enabled"`
	Title                string          `json:"title"`
	Scene                ForegroundScene `json:"scene"`
	ShouldSuppressHealth bool            `json:"shouldSuppressHealth"`
	UpdatedAt            int64           `json:"updatedAt"`
}

// RuntimeContext 统一运行时上下文
type RuntimeContext struct {
	CapturedAt string                  `json:"capturedAt"`
	SessionID  string                  `json:"sessionId"`
	User       UserRuntimeContext      `json:"user"`
	Companion  CompanionRuntimeContext `json:"companion"`
	Time       TimeRuntimeContext      `json:"time"`
	Activity   UserActivityContext     `json:"activity"`
}
