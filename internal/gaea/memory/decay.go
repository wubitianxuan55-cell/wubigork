package memory

import (
	"math"
	"time"
)

// ── 记忆生命周期三态（阶段五 5.3 首刀）──────────────────────────────
//
// 固化（pinned）/ 衰减（decaying）/ 归档（archived）替代「90 天一刀切」：
//
//	固化     用户明示「长期保留」（facts.pinned）：免疫衰减归档、豁免保留期
//	         硬删（CleanupArchived 跳过 pinned=1）、晨报/预载排序加权；
//	衰减     活跃但久未被触达（纯函数评分判定，不是存储列）——衰减是状态
//	         视角，不是定时任务：评分输入（save/touch 事件）自 v4.210 起全部
//	         落 memory_events，状态动作（pin/unpin/archive）同日志留痕；
//	归档     archived=1，保留期内可恢复（Unarchive），超期硬删走既有
//	         CleanupArchived + purge-audit 留痕，固化条豁免。
//
// 全部纯函数：同一输入（记忆 + 时钟值）永远得到同一评分/状态，确定性可测。

// 生命周期状态值（GaeaMemoryLifecycle / 前端徽标共用语义）。
const (
	LifecyclePinned   = "pinned"   // 固化
	LifecycleActive   = "active"   // 活跃（新鲜，衰减评分高）
	LifecycleDecaying = "decaying" // 衰减（活跃但久未触达）
	LifecycleArchived = "archived" // 归档
)

// DecayHalfLifeDays 是衰减评分的半衰期（天）：闲置每过一个半衰期，评分减半。
// 30 天半衰期 → 30 天未用 0.5 分、60 天 0.25 分、90 天 0.125 分。
const DecayHalfLifeDays = 30.0

// DefaultStaleAfterDays 是「衰减」判定的默认闲置阈值（天）：超过即视为
// 衰减中（与评分 0.25 同口径）。与归档保留期（可配置 1–730 天）相互独立。
const DefaultStaleAfterDays = 60

// lastActivityTime 返回记忆最近一次活动时间（触达优先，写入次之）；两者皆
// 零值（从未触达/时间戳缺失的存量行）返回零值时间——按「无衰减证据」处理，
// 诚实不造数：评分满分、不算衰减。
func lastActivityTime(m Memory) time.Time {
	if !m.LastUsedAt.IsZero() {
		return m.LastUsedAt
	}
	return m.UpdatedAt
}

// DecayScore 返回记忆的衰减评分（1.0 新鲜 → 指数衰减，下限 0.01）。
// 固化恒 1.0；无活动时间戳按满分（无衰减证据）。入参限活跃事实
// （Store.List 只返回 archived=0），归档态的展示由生命周期视图单独计数。
func DecayScore(m Memory, now time.Time) float64 {
	if m.Pinned {
		return 1.0
	}
	last := lastActivityTime(m)
	if last.IsZero() || now.Before(last) {
		return 1.0
	}
	days := now.Sub(last).Hours() / 24
	s := math.Exp2(-days / DecayHalfLifeDays)
	if s < 0.01 {
		s = 0.01
	}
	return s
}

// LifecycleOf 返回活跃记忆的生命周期状态（固化/活跃/衰减）。staleAfterDays
// <=0 时取 DefaultStaleAfterDays。
func LifecycleOf(m Memory, now time.Time, staleAfterDays int) string {
	if m.Pinned {
		return LifecyclePinned
	}
	if staleAfterDays <= 0 {
		staleAfterDays = DefaultStaleAfterDays
	}
	last := lastActivityTime(m)
	if last.IsZero() || now.Before(last) {
		return LifecycleActive
	}
	if now.Sub(last) > time.Duration(staleAfterDays)*24*time.Hour {
		return LifecycleDecaying
	}
	return LifecycleActive
}
