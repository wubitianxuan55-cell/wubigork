package memory

// 记忆背压门（codex 蒸馏 T12 护栏二，write guard 同构）：自动蒸馏/做梦的
// 写入不是无条件的——三道前置门全过才跑，跳过时给可读原因（事件记
// skipped: <reason>）。
//
// 务实默认：限流状态拿不到时按「可以跑」处理（对应上游 unwrap_or(true)），
// 不因为限流接口抽风就永久停摆。

// GateInput 是背压门的输入快照：调度方在触发点采集。
type GateInput struct {
	// Ephemeral / Subagent 是会话类型门：临时会话与子代理会话不跑自动写入
	// （生命周期短、无归属价值）。
	Ephemeral bool
	Subagent  bool
	// Enabled 是特性开关（配置面，默认 true=开启——护栏本身是保护性默认）。
	Enabled bool
	// RateLimited 是限流状态：nil=拿不到（按通过处理）；非 nil=false 时跳过。
	RateLimited *bool
}

// GateDecision 是背压门的结论。
type GateDecision struct {
	OK     bool
	Reason string // 拒绝时的可读原因（"skipped: rate_limited" 等），放行为空
}

// GateCheck 按三道门顺序判定：会话类型 → 特性开关 → 限流。
func GateCheck(in GateInput) GateDecision {
	switch {
	case in.Ephemeral:
		return GateDecision{Reason: "skipped: ephemeral"}
	case in.Subagent:
		return GateDecision{Reason: "skipped: subagent"}
	case !in.Enabled:
		return GateDecision{Reason: "skipped: disabled"}
	case in.RateLimited != nil && !*in.RateLimited:
		return GateDecision{Reason: "skipped: rate_limited"}
	default:
		return GateDecision{OK: true}
	}
}
