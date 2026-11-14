// Package schedule — 工程进度计划数据模型与引擎（Go 侧，v4.113.0 刀4）。
//
// 与前端 frontend/src/schedule/types.ts 同构（JSON 字段一一对应）：
// 一套数据模型驱动三种视图（横道图/单代号/双代号），前端引擎为权威口径，
// 本包是其忠实移植——供 agent 工具（schedule_get/apply/analyze）与
// 计划文件持久化使用。修改口径必须两侧同步（前端测试与 Go 测试互为镜像）。
//
// 日历口径：工期/时距均为工作日整数；日期由 startDate + 工作日历推导
// （默认周一~五，节假日例外）。
package schedule

// LinkType 搭接关系类型：完成-开始 / 开始-开始 / 完成-完成 / 开始-完成。
type LinkType string

const (
	FS LinkType = "FS"
	SS LinkType = "SS"
	FF LinkType = "FF"
	SF LinkType = "SF"
)

// Link 搭接关系（From 为前置任务，To 为后续任务）。
type Link struct {
	From string   `json:"from"`
	To   string   `json:"to"`
	Type LinkType `json:"type"`
	// Lag 时距（工作日，整数，可为负）。
	Lag int `json:"lag"`
}

// TaskMode 任务排程模式（对齐 Project）：auto=CPM 排程；manual=锁定开始不动。
type TaskMode string

const (
	ModeAuto   TaskMode = "auto"
	ModeManual TaskMode = "manual"
)

// DurationUnit 工期单位（v4.150 双工期口径刀1，镜像前端 types.ts）：
// 缺省/空=wd 工作日（现状口径，旧文件零迁移）；cd=日历天（自然日定时，
// 对齐 mspdi DurationFormat 8）。Duration 恒为「当前单位下的数值」。
type DurationUnit string

const (
	UnitWd DurationUnit = "wd"
	UnitCd DurationUnit = "cd"
)

// Task 任务/分组行（Level 缩进表达 WBS 层级，扁平数组按顺序渲染）。
// TaskStyle 单元格/横道外观（斑马 draw 参数子集；全字段可缺省，坏值 Validate 拒绝）。
type TaskStyle struct {
	// Color 文字颜色（#RRGGBB）。
	Color string `json:"color,omitempty"`
	// Fill 横道条填充色（#RRGGBB）。
	Fill string `json:"fill,omitempty"`
	// Font 字体族名（≤40 字符）。
	Font string `json:"font,omitempty"`
	// Size 字号 px（8-72）。
	Size int `json:"size,omitempty"`
}

type Task struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Duration int    `json:"duration"`
	// Level 大纲层级：0=分组，1=子任务（两级，足够工程口径）。
	Level int `json:"level"`
	// Progress 完成进度 0-100。
	Progress int `json:"progress"`
	// IsMilestone 里程碑（工期视为 0，菱形显示）。
	IsMilestone bool `json:"isMilestone,omitempty"`
	// Mode 排程模式（缺省 auto）；手动任务锁定开始、不参与关键线路。
	Mode TaskMode `json:"mode,omitempty"`
	// ManualStart 手动任务锁定开始（工作日序号，0=开工日）。
	ManualStart int `json:"manualStart,omitempty"`
	// FixedCost 任务固定成本（元，叶任务专属；v4.122 资源成本刀1。
	// 分组行禁止=汇总唯一口径为子孙求和）。
	FixedCost float64 `json:"fixedCost,omitempty"`
	// DurationUnit 工期单位（v4.150 双工期刀1：缺省/空=wd 工作日；cd=日历天，
	// 仅限叶任务非里程碑、数值上限 3650，见 Validate；v4.155 起 FS/SS/FF/SF
	// 搭接全类型放开）。
	DurationUnit DurationUnit `json:"durationUnit,omitempty"`
	// Category 工作分类（斑马复刻批5：group 字段独立化；分组维度 category 优先取它，
	// 缺省回落 WBS 父名。trim 上限 60，见 Validate；镜像前端 store.ts 归一）。
	Category string `json:"category,omitempty"`
	// Style 单元格/横道外观（斑马复刻批5：文字颜色/横道填充/字体族/字号；
	// 颜色=6 位 hex、字号 8-72、字体 ≤40，见 Validate——镜像前端 store.ts validStyle）。
	Style *TaskStyle `json:"style,omitempty"`
	// AskFinish 要求完成时间（斑马复刻 S5 预警：ask_stop_time 口径=上级下达的
	// 要求完成；工作日序号，0=开工日。缺省=未下达，预警「无级别」。负值 Validate
	// fail-closed，镜像前端 store.ts 归一）。**必须用 *int**：0 是合法值（要求
	// 开工日完成），int+omitempty 会把它在落盘时静默丢掉（前端 normalize 视 0
	// 为已设置）——指针形态 nil=未设置、0=合法已设置。
	AskFinish *int `json:"askFinish,omitempty"`
	// Custom 自定义字段值（v4.138 #14：固定 5 槽 text1..num2，镜像前端
	// customFields.ts 注册表；只存有值槽，值 string|number。旧 JSON 零迁移）。
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// ResourceType 资源类型（对齐 Project：工时/材料/成本三类）。
type ResourceType string

const (
	ResWork     ResourceType = "work"
	ResMaterial ResourceType = "material"
	ResCost     ResourceType = "cost"
)

// Resource 资源（Project 资源工作表的工期制子集，v4.122 资源成本刀1）。
type Resource struct {
	ID   string       `json:"id"`
	Name string       `json:"name"`
	Type ResourceType `json:"type"`
	// Unit 材料计量单位（type=material 有意义：t/m³/…，展示与导出用）。
	Unit string `json:"unit,omitempty"`
	// StandardRate 标准费率（元）：work=元/工日；material=元/单位；cost 不用。
	StandardRate float64 `json:"standardRate,omitempty"`
	// CostPerUse 每次使用成本（元，每条分配计一次；work/material 可用）。
	CostPerUse float64 `json:"costPerUse,omitempty"`
	// MaxUnits 工时资源可用上限（默认 1；v1 仅承载与 mspdi 往返，不做平衡）。
	MaxUnits float64 `json:"maxUnits,omitempty"`
	// Calendar 资源级日历（v4.137 #13 镜像前端 types.ts SchedResource.calendar：
	// 个人周工作制+休假例外；缺省=跟随项目日历。影响使用视图可用性/超载口径，
	// 不改 CPM 排程——Go 侧不消费，仅承载与前端落盘往返，防保存链路静默擦除）。
	Calendar *Calendar `json:"calendar,omitempty"`
}

// Assignment 分配（任务↔资源；(TaskID,ResourceID) 唯一，无独立 id）。
//
// 数值字段一律指针三态（v4.166 契约对齐，审计 FE7-02）：前端 types.ts 三个
// 字段都是可选（undefined 合法），落盘权威在 Go——裸 float64 + omitempty 会把
// 「显式 0」与「未设置」合流（0 被 omitempty 抹成缺失，反过来缺失也读成 0），
// 而 0 在成本口径上与缺失分叉（units：显式 0=零成本，缺失=按缺省 1 计），
// applyDiff.sameScalar(0, undefined) 亦判为不等。nil=不落键，非 nil 的 0=落 0。
//
// 同类残留（本轮按「只在确实需要处改」未动，登记在案不静默）：Resource 数值
// 字段与 Task.ManualStart/FixedCost 仍是裸值+omitempty，0 会被抹成缺失；其中
// 只有 MaxUnits 存在真实分叉——前端 usage.ts 走 `res.maxUnits ?? 1`，显式
// maxUnits=0（=该资源完全不可用）经一次 Go 往返被放大为 1，超载判定随之偏移。
type Assignment struct {
	// TaskID 叶任务 id（分组行禁止分配，fail-closed）。
	TaskID string `json:"taskId"`
	// ResourceID 资源 id。
	ResourceID string `json:"resourceId"`
	// Units 工时资源投入强度（缺省 1，nil=未设置；成本=工期×units×费率）。
	Units *float64 `json:"units,omitempty"`
	// Quantity 材料固定总量（缺省 0，nil=未设置；type=material：总量×单价，
	// 不随时长变）。
	Quantity *float64 `json:"quantity,omitempty"`
	// Amount 成本资源金额（缺省 0，nil=未设置；type=cost：该分配的固定金额，元）。
	Amount *float64 `json:"amount,omitempty"`
}

// Calendar 工作日历（对齐 Project 基准日历）：Workweek 为 JS getDay 口径 0=周日..6=周六。
type Calendar struct {
	// Workweek 视为工作日的星期集合。
	Workweek []int `json:"workweek"`
	// Holidays 节假日/停工例外（YYYY-MM-DD，命中即非工作日）。
	Holidays []string `json:"holidays"`
}

// Project 工程进度计划（计划文件 .gsched.json 的根对象）。
type Project struct {
	Name string `json:"name"`
	// StartDate 开工日期（YYYY-MM-DD，工作日历推算锚点）。
	StartDate string    `json:"startDate"`
	Tasks     []Task    `json:"tasks"`
	Links     []Link    `json:"links"`
	Calendar  *Calendar `json:"calendar,omitempty"`
	// Baseline 基线（v4.116 刀7：保存时的排程快照，缺省=尚未保存）。
	Baseline *Baseline `json:"baseline,omitempty"`
	// Baselines 基线槽位列表（v4.137 #11 多基线镜像前端 types.ts：FIFO 上限 3，
	// 支持签证 1→N 场景切换对比；缺省=单基线旧文件零迁移）。
	//
	// v4.166 契约对齐（审计 FE7-01）：此前 Go 无此字段，而 GaeaScheduleSave 是
	// 「Unmarshal 进本结构体 → MarshalIndent 整量覆盖写盘」，等于每次板块防抖
	// 自动保存都静默删掉用户第 2/3 条基线。本字段是**透传承载**：Go 侧不改写
	// 槽位顺序/条目，FIFO 裁剪与同名原位替换的唯一口径在前端 baseline.ts。
	Baselines []Baseline `json:"baselines,omitempty"`
	// Deadline 目标竣工日期（v4.117 刀8：YYYY-MM-DD，倒排校核用，缺省=未设）。
	Deadline string `json:"deadline,omitempty"`
	// Resources 资源表（v4.122 资源成本刀1：缺省=无资源维度，旧文件零迁移可读）。
	Resources []Resource `json:"resources,omitempty"`
	// Assignments 任务↔资源分配（(TaskID,ResourceID) 唯一）。
	Assignments []Assignment `json:"assignments,omitempty"`
	// AoaLayout 双代号手动布局 pins（v4.123 AOA 刀1：缺省=自动布局）。
	// schema 镜像 + 结构校验 pass-through：锚点键的业务合法性由前端渲染端
	// 裁决与清理，Go 无从验证也不验证（维持无 aoa.go）。
	AoaLayout *AoaLayout `json:"aoaLayout,omitempty"`
	// CustomLabels 自定义字段列名覆盖（v4.138 #14：key=text1..num2，空串/缺省
	// 回落注册表缺省 label；xlsx 导出列头与导入识别同口径）。
	CustomLabels map[string]string `json:"customLabels,omitempty"`
	// Reviews 检视序列（斑马复刻 S2 承载位：多期进度快照，缺省=旧文件零迁移）。
	// v1 只承载与透传：CPM 不消费、视图不消费，落盘前过 validateBlocks 的
	// fail-closed 闸；期间对比等语义在前端 reviews.ts 落地后才有消费者。
	Reviews []Review `json:"reviews,omitempty"`
	// Aoa 双代号点-箭持久图元（斑马复刻 N1 承载位：缺省=沿用前端 aoa.ts 派生口径）。
	// 一旦存在即视为权威图元表；Go 侧不生成也不改写，只做结构校验。
	Aoa *AoaGraph `json:"aoa,omitempty"`
	// NetworkStyle 图面参数（斑马复刻 N2 承载位：5 种显示模式 + 布图压实）。
	NetworkStyle *NetworkStyle `json:"networkStyle,omitempty"`
	// Views 视图快照列表（斑马复刻 S5 承载位：列+分组+筛选+排序+显示模式）。
	Views []View `json:"views,omitempty"`
}

// ReviewActual 单任务在某次检视上的实际数据（全字段可缺省=该期未回填）。
type ReviewActual struct {
	// Start 实际开始（工作日序号，相对开工日）。
	Start *int `json:"start,omitempty"`
	// Finish 实际完成（工作日序号，相对开工日）。
	Finish *int `json:"finish,omitempty"`
	// Percent 该期完成百分比 0-100（缺省=沿用任务 progress）。
	Percent *int `json:"percent,omitempty"`
}

// Review 一次检视（斑马「检视时间」，见逆向报告 §5.1）。
//
// 时间口径与 CpmResult 一致：At 为工作日序号；Date 只作展示与人工校核，
// 两者的一致性由前端 normalizeReviews 兜容错，落盘由 Validate 兜 fail-closed。
type Review struct {
	// At 检视时间（工作日序号，>=0，同一计划内唯一）。
	At int `json:"at"`
	// Date 对应 ISO 日期（YYYY-MM-DD）。
	Date string `json:"date"`
	// Frozen 冻结标记：拉直前锋线后为 true，禁止回改已完成/进行中。
	Frozen bool `json:"frozen,omitempty"`
	// Actuals 逐任务实际数据（键=叶任务 id；任务被删后仍保留，便于历史对比）。
	Actuals map[string]ReviewActual `json:"actuals"`
	// Note 备注。
	Note string `json:"note,omitempty"`
}

// AoaDot 双代号事件（点）持久图元。
type AoaDot struct {
	// ID 点身份（持久稳定，跨拓扑变更不变）。
	ID string `json:"id"`
	// Num 节点编号（1 起，拓扑序，保证箭线 i<j）。
	Num int `json:"num"`
	// Layer 层/行号（布图参数化用，缺省=自动）。
	Layer *int `json:"layer,omitempty"`
	// IsBegin 起点事件（唯一始）。
	IsBegin bool `json:"isBegin,omitempty"`
	// IsEnd 终点事件（唯一终）。
	IsEnd bool `json:"isEnd,omitempty"`
}

// AoaArrow 双代号箭线（实工作 / 虚工作 / 分组汇总）。
type AoaArrow struct {
	ID string `json:"id"`
	// From 尾事件 id（须存在于 Dots）。
	From string `json:"from"`
	// To 头事件 id（须存在于 Dots）。
	To string `json:"to"`
	// Kind task=实工作 / dummy=虚工作 / summary=分组汇总箭线。
	Kind string `json:"kind"`
	// TaskID kind=task/summary 时对应的任务 id。
	TaskID string `json:"taskId,omitempty"`
	// Lag 该箭线承载的时距（工作日，缺省 0；闸门 ±9999）。
	Lag int `json:"lag,omitempty"`
}

// AoaGraph 双代号点-箭持久图（对应前端 SchedAoaGraph）。
type AoaGraph struct {
	Dots   []AoaDot   `json:"dots"`
	Arrows []AoaArrow `json:"arrows"`
	// BeginDotID 起点事件 id（缺省=按 IsBegin 推断）。
	BeginDotID string `json:"beginDotId,omitempty"`
}

// NetworkStyle 图面参数（斑马 network 状态块的 gaea 子集，见逆向报告 §4.1–§4.2）。
//
// 布尔用指针保留三态（false 与"未设置"语义不同：前者显式关闭错层/隐藏线）。
type NetworkStyle struct {
	// Mode timescaled / timescaledLogic / logic / pdm / ganttNetwork。
	Mode string `json:"mode,omitempty"`
	// DotStyle circle / rect。
	DotStyle string `json:"dotStyle,omitempty"`
	// ArrowStyle solid / hollow。
	ArrowStyle string `json:"arrowStyle,omitempty"`
	// NamePos auto / horizontal / vertical。
	NamePos string `json:"namePos,omitempty"`
	// NodeRadius 节点半径（px，4..200）。
	NodeRadius *int `json:"nodeRadius,omitempty"`
	// HeadHeight 箭头头部高度（px，2..60）。
	HeadHeight *int `json:"headHeight,omitempty"`
	// LayerHeight1/2 两层高度（px，20..2000）。
	LayerHeight1 *int `json:"layerHeight1,omitempty"`
	LayerHeight2 *int `json:"layerHeight2,omitempty"`
	// Stagger 错层排布（斑马 is_stagger）。
	Stagger *bool `json:"stagger,omitempty"`
	// HideUnimportantLink 隐藏非重要关系线。
	HideUnimportantLink *bool `json:"hideUnimportantLink,omitempty"`
	// TickUnit yearMonth / monthDay / dayHour。
	TickUnit string `json:"tickUnit,omitempty"`
	// TickLines none / small / large。
	TickLines string `json:"tickLines,omitempty"`
	// Density compact / normal / loose。
	Density string `json:"density,omitempty"`
	// AspectMax 纵横比上限（0.1..20）。
	AspectMax *float64 `json:"aspectMax,omitempty"`
	// MaxPages 页数上限（1..1000）。
	MaxPages *int `json:"maxPages,omitempty"`
}

// ViewSort 视图排序。
type ViewSort struct {
	Field string `json:"field"`
	Dir   string `json:"dir"`
}

// View 视图快照（斑马 views_list，见逆向报告 §6.1）。
type View struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Kind gantt / pdm / aoa / usage / dcma / monte / evm。
	Kind string `json:"kind"`
	// Cols 可见列键集（缺省=全部）。
	Cols []string `json:"cols,omitempty"`
	// Hide 隐藏列键集。
	Hide []string `json:"hide,omitempty"`
	// GroupBy 多级分组字段。
	GroupBy []string `json:"groupBy,omitempty"`
	// Filter 筛选表达式（缺省=不过滤）。
	Filter string `json:"filter,omitempty"`
	// Sort 排序。
	Sort *ViewSort `json:"sort,omitempty"`
	// RenderMode 网络图显示模式（kind=aoa/pdm 时有效）。
	RenderMode string `json:"renderMode,omitempty"`
}

// AoaPt 双代号手动布点（整数像素）。
type AoaPt struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// AoaLayout 双代号布局覆盖：pin 按事件锚点键匹配（混合锚定）。
type AoaLayout struct {
	Pins map[string]AoaPt `json:"pins"`
}

// BaselineRow 基线行快照：单任务保存基线时的排程结果（叶任务专属）。
type BaselineRow struct {
	// Name 任务名（任务被移除后仍可读）。
	Name string `json:"name"`
	// ES/EF 基线最早开始/完成（工作日序号）。
	ES int `json:"es"`
	EF int `json:"ef"`
	// Dur 有效工期（里程碑 0）。
	Dur int `json:"dur"`
	// Critical 保存时是否关键。
	Critical bool `json:"critical"`
}

// Baseline 基线（对齐 Project「设置基线」：快照排程结果，供漂移对比）。
type Baseline struct {
	// Name 基线名（缺省「基线」）。
	Name string `json:"name"`
	// SavedAt 保存时间（YYYY-MM-DD HH:mm，由调用方标注）。
	SavedAt string `json:"savedAt"`
	// Duration 保存时总工期（工作日）。
	Duration int `json:"duration"`
	// Rows 叶任务基线行。
	Rows map[string]BaselineRow `json:"rows"`
}

// TaskCpm 单任务 CPM 计算结果（天数，ES/EF 为相对开工日的天偏移）。
type TaskCpm struct {
	ES int `json:"es"`
	EF int `json:"ef"`
	LS int `json:"ls"`
	LF int `json:"lf"`
	// TF 总时差。
	TF int `json:"tf"`
	// FF 自由时差。
	FF int `json:"ff"`
	// Critical 关键工作（TF==0 且非手动）。
	Critical bool `json:"critical"`
}

// CpmResult CPM 整体计算结果。
type CpmResult struct {
	OK    bool               `json:"ok"`
	Error string             `json:"error,omitempty"`
	Cycle []string           `json:"cycle,omitempty"`
	Rows  map[string]TaskCpm `json:"rows"`
	// Duration 总工期（天）= max(EF)。
	Duration int `json:"duration"`
}
