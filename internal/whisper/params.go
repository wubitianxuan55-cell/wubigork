// Package whisper — 全局参数常量（100% 对齐 ackem engine/ackemParams.ts）
package whisper

// ─── A组：情绪系统核心参数 ────────────────────────────────────

const (
	EmotionDecay    = 0.03  // A1: 每轮衰减率
	SingleTurnClamp = 10.0  // A2: 单轮变化上界
	EmotionCapDenom = 120.0 // A3: Cap Scale 分母

	LockAffHigh = 70.0  // A4: aff>70 锁高
	LockAffLow  = -50.0 // A5: aff<-50 锁低
	LockSecLow  = -60.0 // A6: sec<-60 锁低

	LockAffHighReduceNeg = 0.6 // A7: 锁高时负面折扣
	LockAffLowReducePos  = 0.5 // A8: 锁低时正面折扣
	LockSecLowReducePos  = 0.5 // A9: sec锁低正面折扣

	NoiseThresholdAbs = 80.0 // A10
	NoiseMax          = 0.5  // A11
)

// ─── L1 信任/裂痕 ─────────────────────────────────────────────

const (
	TrustPraise     = 1.5
	TrustApology    = 2.0
	TrustVulnerable = 1.0
	TrustTease      = 0.8
	TrustCold       = -1.5
	TrustHurtful    = -3.0
	TrustCasual     = 0.0
	TrustQuestion   = 0.0

	RiftHurtfulCooldown      = 2
	RiftRepairPositiveStreak = 4
	RiftModMin               = 0.3
	RiftModDecayPerRift      = 0.15

	StageWarmupTurns    = 10
	StageIntimateTrust  = 60.0
	StageIntimateEvents = 3
	StageDowngradeRifts = 5
	StageDowngradeTrust = 30.0

	MomentumAlpha           = 0.7
	AtmosphereWarmThreshold = 0.5
	AtmosphereCoolThreshold = -0.3
	TrustModMin             = 0.5
	TrustModMax             = 1.5

	StageWeightStranger = 0.8
	StageWeightFamiliar = 1.0
	StageWeightIntimate = 1.4
)

// ─── 沉默概率 ──────────────────────────────────────────────────

const (
	SilenceIntensityWeight  = 0.3
	SilenceRiftsWeight      = 0.2
	SilenceAroWeight        = 0.02
	SilenceThreshold        = 0.7
	SilenceSigmoidSteepness = 12.0
	AroExcessBaseline       = 50.0

	StageModifierStranger = 1.3
	StageModifierFamiliar = 1.0
	StageModifierIntimate = 0.7
)

// ─── 记忆回声 ──────────────────────────────────────────────────

const (
	MemoryEchoCap                = 2.0
	MemoryEchoAffWeight          = 0.5
	MemoryEchoSecPositive        = 0.3
	MemoryEchoSecNegative        = -0.3
	MemoryEchoAroIntensityWeight = 0.6 // P1新增: 对齐 ackem MEMORY_ECHO_ARO_INTENSITY_WEIGHT
	MemoryEchoDomTrustWeight     = 0.4 // P1新增: 对齐 ackem MEMORY_ECHO_DOM_TRUST_WEIGHT
	EffectiveTrustL1Weight       = 0.5
	EffectiveTrustMemWeight      = 0.5

	MoodCongruentValenceDiff      = 0.3
	MoodCongruentBoost            = 1.5
	MoodCongruentExtremeThreshold = 50.0
	MoodCongruentExtremeBoost     = 1.2

	MemoirTrustFloor = 25.0
)

// ─── L4 记忆系统 ──────────────────────────────────────────────

const (
	FactExtractionMaxPerTurn  = 8
	TierBCharBudget           = 8000
	MinConfidenceForInjection = 0.55
	AutoRetireCheckInterval   = 10

	FactDedupThreshold      = 0.42
	RecencyBoostWindowHours = 4.0
	RecencyBoostFactor      = 1.8

	WorkingMemoryMaxExchanges   = 6
	WorkingMemoryCharBudget     = 3000
	SemanticSearchTopK          = 5
	SemanticSearchMinSimilarity = 0.12

	ConsolidationIntervalTurns     = 30
	ConsolidationMinTurns          = 20
	ConsolidationMaxTurns          = 60
	ConsolidationMeaningfulDensity = 0.40
	ConsolidationMaxFactsInput     = 30
	ConsolidationInsightWeight     = 4.0
	ConsolidationMinFacts          = 6
	ConsolidationMaxInsights       = 4
)

// ─── 情节记忆 ──────────────────────────────────────────────────

const (
	EpisodeIntervalTurns             = 6
	EpisodeIntervalTurnsLow          = 10
	EpisodeEmotionIntensityThreshold = 0.5
	EpisodeSummaryMaxChars           = 200
	EpisodeRetrievalMax              = 3
	EpisodeCharBudget                = 1200
)

// ─── 分层记忆 ──────────────────────────────────────────────────

const (
	CoreMemoryWeightThreshold = 3.0
	CoreMemoryMaxCount        = 12
	CoreMemoryCharBudget      = 2000
	ActiveRecallMinInterval   = 8
	ActiveRecallProbability   = 0.15
)

// ─── 知识图谱 ──────────────────────────────────────────────────

const (
	KGQueryMaxTriples                = 8
	KGCharBudget                     = 800
	ContradictionSimilarityThreshold = 0.35
	ContradictionMinWeight           = 1.5
	MirrorCheckIntervalTurns         = 20
)

// ─── Embedding ─────────────────────────────────────────────────

const (
	VectorSearchTopK     = 6
	VectorSearchMinScore = 0.05
	EmbeddingSearchTopK  = 6
	EmbeddingMinScore    = 0.35
)

// ─── 性格漂移 ──────────────────────────────────────────────────

const (
	DriftMaxAbsolute   = 15.0
	DriftCheckInterval = 50
	DriftDelta         = 1.5
)

// ─── 破冰 ──────────────────────────────────────────────────────

const (
	IceBreakTrustThreshold     = 15.0
	IceBreakSincerityThreshold = 0.7
	IceBreakTrustBonus         = 3.0
)

// ─── 重聚 ──────────────────────────────────────────────────────

const (
	ReunionOfflineMinutes    = 30.0
	ReunionAffBoost          = 2.0
	ReunionSecBoost          = 1.5
	ReunionOfflineCapMinutes = 1440.0
)

// ─── 外场气氛 ──────────────────────────────────────────────────

const (
	ExternalMomentumAlpha = 0.95
	ExternalWarmThreshold = 0.4
	ExternalCoolThreshold = -0.2
)

// ─── 欲望栈 ────────────────────────────────────────────────────

const (
	DesireMaxSlots                  = 5
	DesireExpressThreshold          = 7.0
	DesireDecayPerTurn              = 0.3
	DesireIdleSettleTurns           = 8
	DesireExpressedSettleAfterTurns = 2
	DesireDormantUrgency            = 0.6
)

// ─── 基础 ──────────────────────────────────────────────────────

const (
	InitialTrust     = 50.0
	StateJSONVersion = "1.0"
)

// ─── OEG 创造者叙事 ───────────────────────────────────────────

const (
	OriginStreakExplore = 2
	OriginStreakDeep    = 4
	OriginStreakGuard   = 5
	OriginCooldownTurns = 8
)

// ─── 情绪涌现 ──────────────────────────────────────────────────

const (
	EmergenceIntensityThreshold       = 20.0
	EmergenceCooldownTurns            = 10
	ResponsiveEmergenceCooldownTurns  = 1
	SameTypeCooldownTurns             = 50
	SameTypeCooldownHours             = 72.0
	RisingMaxRounds                   = 3
	SustainedMaxRounds                = 10
	SustainedMinRounds                = 3
	FadingMaxRounds                   = 5
	AntiRepetitionSimilarityThreshold = 0.65
)

// ─── 欲望产生基础概率 ─────────────────────────────────────────

// ─── 欲望产生基础概率 ─────────────────────────────────────────

const DesireNewBaseChance = 0.08

// ─── 记忆增强参数 ──────────────────────────────────────────────

const (
	LongTermMinOccurrences      = 3
	LongTermBaseConfidence      = 0.6
	LongTermConfidenceCap       = 0.95
	DecayWeeksThreshold         = 4
	DecayWeeklyRate             = 0.1
	DecaySleepThreshold         = 0.4
	MaxShortTermPerDay          = 3
	LongTermConfidenceIncrement = 0.1
)

// ─── 记忆自编辑参数 ────────────────────────────────────────────

// ─── 记忆自编辑参数 ────────────────────────────────────────────

const (
	SelfEditReinforceWeightBoost = 0.3
	SelfEditLogMax               = 200
	SelfEditLogKeep              = 100
)

// ─── 关联冷启动参数 ────────────────────────────────────────────

const (
	ColdStartEdgeStrength    = 0.35
	ColdStartMinCosine       = 0.65
	ColdStartMaxOrphans      = 50
	ColdStartMaxPairsPerFact = 3
	ColdStartTextMinOverlap  = 2
)

// ─── 程序化习惯参数 ────────────────────────────────────────────

const (
	HabitMinOccurrences = 3
	HabitKeyMaxLen      = 48
)
