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
