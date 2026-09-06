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

// Task 任务/分组行（Level 缩进表达 WBS 层级，扁平数组按顺序渲染）。
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
}

// Assignment 分配（任务↔资源；(TaskID,ResourceID) 唯一，无独立 id）。
type Assignment struct {
	// TaskID 叶任务 id（分组行禁止分配，fail-closed）。
	TaskID string `json:"taskId"`
	// ResourceID 资源 id。
	ResourceID string `json:"resourceId"`
	// Units 工时资源投入强度（默认 1；成本=工期×units×费率）。
	Units *float64 `json:"units,omitempty"`
	// Quantity 材料固定总量（type=material：总量×单价，不随时长变）。
	Quantity float64 `json:"quantity,omitempty"`
	// Amount 成本资源金额（type=cost：该分配的固定金额，元）。
	Amount float64 `json:"amount,omitempty"`
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
