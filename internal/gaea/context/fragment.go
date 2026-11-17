package context

import (
	"fmt"
	"strings"
)

// 上下文碎片契约（codex 蒸馏 T6，context-fragments 同构）：所有注入上下文的
// 统一形状。价值不在「能包标记」，而在把「谁写的这段」变成结构化字段——
// Guardian 授权判定、压缩精确替换、章节化防重复注入都读它的字段。
//
// 三条纪律（照抄 codex，不自作聪明）：
//  1. 归属由生产者声明，绝不从渲染文本反推；
//  2. 工具相关上下文即使由框架代写，归属仍是 ProvTool，不洗成 ProvHarness；
//  3. 只有文本才包标记（本契约 Body 恒为文本；图片等结构化内容不走此形状）。

// Provenance 是内容出处，独立于消息角色（role=user ≠ 用户说的）。
type Provenance int

const (
	// ProvUnknown 是零值：历史数据或缺字段时不猜来源。
	ProvUnknown Provenance = iota
	// ProvDeveloperInstructions 是开发者指令（系统提示词的规则段）。
	ProvDeveloperInstructions
	// ProvSkills 是技能目录/用法/选中技能内容。
	ProvSkills
	// ProvAgentsMD 是 AGENTS.md 载入的指令。
	ProvAgentsMD
	// ProvHarness 是框架托管的内建上下文（用户配置之外）。
	ProvHarness
	// ProvHarnessHook 是 hook 输出——用户配置的脚本，可信度与内建不同，单独一档。
	ProvHarnessHook
	// ProvTool 是工具结果/工具上下文（即使框架代写）。
	ProvTool
	// ProvUser 专指真实用户输入，不是「带 user 角色的注入片段」。
	ProvUser
)

// String 返回归属的可读名（日志与事件用）。
func (p Provenance) String() string {
	switch p {
	case ProvDeveloperInstructions:
		return "developer_instructions"
	case ProvSkills:
		return "skills"
	case ProvAgentsMD:
		return "agents_md"
	case ProvHarness:
		return "harness"
	case ProvHarnessHook:
		return "harness_hook"
	case ProvTool:
		return "tool"
	case ProvUser:
		return "user"
	default:
		return "unknown"
	}
}

// Fragment 是一条注入上下文。
type Fragment struct {
	Role string // system / user / developer / tool——与 Prov 无关
	Kind string // "<feature>.<name>"，稳定字符串；同 Kind 连续多轮可识别
	// Start/End 是本实例的标记对；都为空则渲染直出不包。
	// 类型级识别不另设常量：Kind 本身就是稳定标识，MatchesText 按渲染标记判。
	Start string
	End   string
	Body  string
	Prov  Provenance
	// NS 是工具命名空间（ProvTool 时）：内建工具空串，MCP 用服务器名（开放集，
	// 不做枚举——gaea 命名空间未稳定，过早枚举每次扩面都要改类型）。
	NS string
}

// fragmentMarkerPrefix 是标记对的统一前缀：可读、稳定、足够不易撞。
const fragmentMarkerPrefix = "gaea-ctx"

// TypeMarkers 返回 Kind 的类型级标记对（识别用，与实例标记解耦）。
func TypeMarkers(kind string) (start, end string) {
	return fmt.Sprintf("<%s kind=%q>", fragmentMarkerPrefix, kind),
		fmt.Sprintf("</%s>", fragmentMarkerPrefix)
}

// Render 渲染片段文本：带标记对则首尾包裹，否则直出。Body 恒为文本
// （结构化内容不进此形状），不存在「图片包标记破坏解析」的分支。
func (f Fragment) Render() string {
	if f.Start == "" && f.End == "" {
		return f.Body
	}
	return f.Start + f.Body + f.End
}

// MatchesText 判定 text 是否含 kind 的类型级标记（识别「这段是不是我这类」）。
// 渲染与识别共用同一标记源（TypeMarkers），同 Kind 连续多轮稳定命中。
func MatchesText(kind, text string) bool {
	if kind == "" {
		return false
	}
	start, end := TypeMarkers(kind)
	return strings.Contains(text, start) && strings.Contains(text, end)
}

// CanAuthorize 报告该片段能否作为授权证据（Guardian 判定入口，T6 验收③）：
// 只有真实用户输入 / 开发者指令 / AGENTS.md 三类可信来源可以支撑 allow。
// 「带 user 角色」不等于「用户说的」——role 与归属是两件事。
func CanAuthorize(f Fragment) bool {
	switch f.Prov {
	case ProvUser, ProvDeveloperInstructions, ProvAgentsMD:
		return true
	default:
		return false
	}
}

// NewToolFragment 是工具侧片段的生产入口：归属恒 ProvTool（生产者声明纪律），
// 标记对自动从 Kind 派生，调用方不手写标记。
func NewToolFragment(role, kind, body, ns string) Fragment {
	start, end := TypeMarkers(kind)
	return Fragment{Role: role, Kind: kind, Start: start, End: end, Body: body, Prov: ProvTool, NS: ns}
}

// NewHarnessFragment 是框架托管片段的生产入口（标记对同样自动派生）。
func NewHarnessFragment(role, kind, body string) Fragment {
	start, end := TypeMarkers(kind)
	return Fragment{Role: role, Kind: kind, Start: start, End: end, Body: body, Prov: ProvHarness}
}
