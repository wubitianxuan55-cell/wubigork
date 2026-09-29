package types

// ── 故事层骨架（长篇刀3，规格 进度计划/gaea-novel-story-spine-20260930.md）──
//
// 整书层结构化持续状态：主题论证（M12，报告制）/人物弧线水位（M3）/节拍位（M5）/
// 支线开关表（M6 简化）/未解问题池（M8 简化）。存 story_spine.json（项目根，
// 与 outline.json 平级）。全部可空——零 spine 是新书正常态（体检空报告、生成零注入）。

// StorySpine 故事脊椎（整书层骨架）。Version 供未来契约演进。
type StorySpine struct {
	Version       int               `json:"version"`
	Theme         StoryTheme        `json:"theme"`
	Arcs          []StoryArc        `json:"arcs,omitempty"`
	Beats         []StoryBeat       `json:"beats,omitempty"`
	Threads       []StoryThreadItem `json:"threads,omitempty"`
	OpenQuestions []StoryQuestion   `json:"open_questions,omitempty"`
}

// StoryTheme 主题论证（Truby 控制理念与反论）。M12 误报最高——只做报告不做闸门。
type StoryTheme struct {
	// ControllingIdea 控制理念：这本书在讲什么道理（一句话，正命题）。
	ControllingIdea string `json:"controlling_idea,omitempty"`
	// CounterIdea 反论最强陈述：对手/世界给这一命题的最强反驳（让主题有对抗性）。
	CounterIdea string `json:"counter_idea,omitempty"`
}

// StoryArc 人物弧线（want/need/misbelief + 水位点）。水位=misbelief 被动摇/强化的
// 进度，体检只提示覆盖度，不做单调闸。
type StoryArc struct {
	CharacterID string    `json:"character_id"` // 关联角色库 ID（可空=自由名）
	Name        string    `json:"name"`
	Want        string    `json:"want,omitempty"`      // 表层欲望（他以为自己要什么）
	Need        string    `json:"need,omitempty"`      // 深层需要（真正缺什么）
	Misbelief   string    `json:"misbelief,omitempty"` // 错误信念（弧线要推翻的东西）
	Beats       []ArcBeat `json:"beats,omitempty"`     // 水位点（按章推进）
}

// ArcBeat 弧线水位点。
type ArcBeat struct {
	Chapter int    `json:"chapter"`         // 发生章号（>0）
	Stage   string `json:"stage,omitempty"` // setup 确立 / provoke 初次反证 / escalate 反证递进（代价渐增）/ proof 行动推翻（终局）
	Note    string `json:"note,omitempty"`  // 这一水位发生了什么
}

// 弧线水位阶段枚举（校验用；空=未标不算错）。
const (
	ArcStageSetup    = "setup"
	ArcStageProvoke  = "provoke"
	ArcStageEscalate = "escalate"
	ArcStageProof    = "proof"
)

// StoryBeat 结构节拍位（三幕关键转折）。Chapter=0 表示未绑定；终局段估计以
// climax 节拍优先，无 climax 时按磁盘最大章回退。
type StoryBeat struct {
	ID      string `json:"id"`               // hook/first_plot_point/midpoint/second_plot_point/climax 或自定义 slug
	Name    string `json:"name"`             // 中文名（如「中点：价值翻转」）
	Chapter int    `json:"chapter"`          // 绑定章号（0=未定）
	Status  string `json:"status,omitempty"` // planned / hit
}

// 节拍状态枚举。
const (
	BeatStatusPlanned = "planned"
	BeatStatusHit     = "hit"
)

// 预置节拍 ID（提案初稿/终局段估计用；自定义节拍自由命名）。
const (
	BeatIDHook            = "hook"
	BeatIDFirstPlotPoint  = "first_plot_point"
	BeatIDMidpoint        = "midpoint"
	BeatIDSecondPlotPoint = "second_plot_point"
	BeatIDClimax          = "climax"
)

// StoryThreadItem 支线开关表条目（Kowal MICE 简化：嵌套 LIFO 与配额是后续刀，
// 本刀先管「开了没关=遗忘」）。
type StoryThreadItem struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	MICEType            string `json:"mice_type,omitempty"`             // query/character/event/place/thing
	OpenChapter         int    `json:"open_chapter"`                    // 开启章号（>0）
	LastAdvancedChapter int    `json:"last_advanced_chapter,omitempty"` // 最后推进章（体检判遗忘）
	Status              string `json:"status"`                          // active/parked/closed/abandoned
	CloseChapter        int    `json:"close_chapter,omitempty"`
	Note                string `json:"note,omitempty"`
}

// 支线状态枚举。
const (
	ThreadStatusActive    = "active"
	ThreadStatusParked    = "parked"
	ThreadStatusClosed    = "closed"
	ThreadStatusAbandoned = "abandoned"
)

// MICE 类型枚举（空=未标）。
const (
	MICEQuery     = "query"
	MICECharacter = "character"
	MICEEvent     = "event"
	MICEPlace     = "place"
	MICEThing     = "thing"
)

// StoryQuestion 未解问题池条目（读者视角的开放悬念）。
type StoryQuestion struct {
	ID            string `json:"id"`
	Question      string `json:"question"`
	RaisedChapter int    `json:"raised_chapter"`
	Status        string `json:"status"` // open/answered
	AnswerChapter int    `json:"answer_chapter,omitempty"`
}

// 问题状态枚举。
const (
	QuestionOpen     = "open"
	QuestionAnswered = "answered"
)

// StorySpineVersion 当前契约版本。
const StorySpineVersion = 1

// StoryHealthFinding 结构体检一条发现（确定性零 LLM；code 定位维度）。
type StoryHealthFinding struct {
	Code     string `json:"code"`     // thread_stale / question_overdue / beat_overdue / mid_sag / arc_thin / finale_open / theme_missing
	Severity string `json:"severity"` // S1 必须处理 / S2 建议处理 / S3 提示
	Message  string `json:"message"`
	Ref      string `json:"ref,omitempty"` // 定位（线程名/节拍名/章区间等）
}
