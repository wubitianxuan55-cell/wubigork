// Package taskinbox 7.3-1 多入口统一任务收件箱的纯函数层（线 A）：任务卡
// （title + 来源 + space + 状态机）的形状校验、标题归一、空间过滤与收件箱排序。
// 四入口（意图中枢 palette / Ctrl+K / 语音 / 微信，外加收件箱面板手动新建 =
// inbox 兜底）存任务时统一经本包把关；状态文件读写归 App 层（线 B），视图归
// 前端（线 C）。
//
// 设计纪律（先例 internal/skilldistill、internal/routesuggest）：零 IO、零外部
// 依赖——全部数据由调用方组装，表驱动测试。ID 生成（crypto/rand）属 IO 侧不进
// 本包，仅 ParseTaskID 校验形状（残渣防御，先例 routesuggest.ParseSuggestionID）。
//
// V1 范围裁决（规格 进度计划/gaea-task-inbox-7-3-1-20260915.md §0）：任务 =
// 用户意图的持久卡，不是后台执行队列；收件箱是清单不是调度器（不自动执行）；
// space 维度必带（work/play 隔离沿 boards/space.ts，跨空间仅显式）。
package taskinbox

import (
	"errors"
	"sort"
	"strings"
)

const (
	// MaxTitleRunes 标题上限（rune 口径）：超长截断不报错——存任务不该被长度卡死。
	MaxTitleRunes = 120
	// MaxTasks 状态文件条目上限（防膨胀）：超限由 App 层 Save 拒绝新建（线 B）。
	MaxTasks = 500
)

// 四个合法状态值。不导出：外部一律经 ParseStatus 归一，防绕过状态机拼字符串。
const (
	statusPending   Status = "pending"   // 待处理
	statusDoing     Status = "doing"     // 进行中
	statusDone      Status = "done"      // 已完成（终态）
	statusAbandoned Status = "abandoned" // 已放弃（终态）
)

// ErrEmptyTitle NormalizeTitle 的固定错误：trim 后标题为空（App 层 errors.Is 判定）。
var ErrEmptyTitle = errors.New("taskinbox: 标题为空（trim 后）")

// Status 任务状态机四值：pending | doing | done | abandoned。
type Status string

// Task 一张任务卡：用户意图的持久快照。json 标签与状态文件契约一致（camelCase），
// 线 B 的 TaskInboxView 同字段同标签——此处以测试钉死防漂移。
type Task struct {
	ID        string `json:"id"`                // "ti-"+12hex（App 层生成，本包仅 ParseTaskID 校验）
	Title     string `json:"title"`             // NormalizeTitle 产物
	Space     string `json:"space"`             // "work"|"play"（必带，空间隔离判据）
	Status    Status `json:"status"`            // 四值状态机（CanTransition 把关）
	Source    string `json:"source"`            // ctrlk|palette|voice|weixin|inbox（来源审计链）
	Action    string `json:"action,omitempty"`  // 意图动作（navigate/…；手动新建为空）
	Target    string `json:"target,omitempty"`  // 意图目标（板块 id 等；「去板块」跳转用）
	Session   string `json:"session,omitempty"` // 源会话（审计链，V1 板块级跳转）
	Note      string `json:"note,omitempty"`    // 备注（手动补充）
	CreatedAt int64  `json:"createdAt"`         // unix ms（新建定死，更新不可变）
	UpdatedAt int64  `json:"updatedAt"`         // unix ms（每次变更刷新，排序口径）
}

// ParseStatus 把字符串归一为合法 Status：恰四值之一（大小写敏感、不 trim——
// 状态是程序写入的枚举而非用户输入）；未知值返回 ("", false) 由调用方拒绝。
func ParseStatus(s string) (Status, bool) {
	switch Status(s) {
	case statusPending, statusDoing, statusDone, statusAbandoned:
		return Status(s), true
	default:
		return "", false
	}
}

// CanTransition 状态机迁移合法性：pending→doing→done；pending|doing→abandoned。
// 终态（done/abandoned）不接受任何迁移；同值迁移（无变化）恒 false——由调用方
// 短路成功；空/未知值任何方向都 false（残渣防御）。
func CanTransition(from, to Status) bool {
	if from == to {
		return false // 同值迁移恒 false（无变化，调用方短路）
	}
	switch from {
	case statusPending:
		return to == statusDoing || to == statusAbandoned
	case statusDoing:
		return to == statusDone || to == statusAbandoned
	default:
		return false // done/abandoned 终态 + 未知残渣：一律不放行
	}
}

// NormalizeTitle 标题归一：TrimSpace（含全角空白）；trim 后为空返回 ErrEmptyTitle；
// 超 MaxTitleRunes 按 rune 截断（[]rune 口径——中文标题不得按字节腰斩）。
func NormalizeTitle(s string) (string, error) {
	t := strings.TrimSpace(s)
	if t == "" {
		return "", ErrEmptyTitle
	}
	if r := []rune(t); len(r) > MaxTitleRunes {
		return string(r[:MaxTitleRunes]), nil // 超长截断不报错：存任务不该被长度卡死
	}
	return t, nil
}

// ValidSpace 空间合法性：仅 "work"|"play"（大小写敏感）。任务必带 space，
// FilterBySpace 据此隔离——非法空间由 App 层 Save 拒绝。
func ValidSpace(s string) bool {
	return s == "work" || s == "play"
}

// ValidSource 来源合法性：仅 ctrlk|palette|voice|weixin|inbox（四入口 + 收件箱
// 面板手动新建兜底）；来源字段落库后不可变（审计链）。
func ValidSource(s string) bool {
	switch s {
	case "ctrlk", "palette", "voice", "weixin", "inbox":
		return true
	default:
		return false
	}
}

// ParseTaskID 校验任务 ID 形状："ti-" 前缀 + 恰 12 个小写 hex（共 15 字节）。
// 残渣防御先例 routesuggest.ParseSuggestionID：大写 hex、错前缀、长度偏差一律拒绝。
func ParseTaskID(id string) bool {
	if len(id) != 15 || !strings.HasPrefix(id, "ti-") {
		return false
	}
	for _, c := range id[3:] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// FilterBySpace 按空间过滤：space=="" 返回全部（GaeaTaskList 变参先例口径，
// 返回原切片不拷贝，调用方不得就地改写）；否则严格相等（隔离判据——play
// 不得混入 work），保持原序。
func FilterBySpace(tasks []Task, space string) []Task {
	if space == "" {
		return tasks
	}
	out := make([]Task, 0, len(tasks))
	for _, t := range tasks {
		if t.Space == space {
			out = append(out, t)
		}
	}
	return out
}

// Sort 收件箱排序（行动优先）：状态组序 pending<doing<done<abandoned，组内
// UpdatedAt 降序（最近变化在前），再 ID 升序（确定性）。返回新切片、不改入参
// （纯函数纪律）；未知状态残渣排全部合法组之后（不干扰合法组序）。
func Sort(tasks []Task) []Task {
	out := make([]Task, len(tasks))
	copy(out, tasks)
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := statusRank(out[i].Status), statusRank(out[j].Status)
		if ri != rj {
			return ri < rj
		}
		if out[i].UpdatedAt != out[j].UpdatedAt {
			return out[i].UpdatedAt > out[j].UpdatedAt
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// statusRank 状态组序：pending=0 < doing=1 < done=2 < abandoned=3；未知值 4 垫底。
func statusRank(s Status) int {
	switch s {
	case statusPending:
		return 0
	case statusDoing:
		return 1
	case statusDone:
		return 2
	case statusAbandoned:
		return 3
	default:
		return 4
	}
}
