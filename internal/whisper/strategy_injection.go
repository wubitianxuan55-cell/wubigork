// Package whisper — strategy_injection.go
// 100% 对齐 ackem engine/strategy/injectionPolicy.ts
// 注入槽位分配策略：控制 psycheBlock 中各模块 hint 的注入优先级

package whisper

// ─── 注入标记常量 ─────────────────────────────────────────────

const (
	TemporalHintMarker    = "【时间感知"
	EmergenceHintMarker   = "【情绪涌现"
	CanonTemporalMarker   = "【Canon · 时间锚"
	CanonAnniversaryMarker = "【Canon · 纪念日"
)

// ─── 注入槽位 ─────────────────────────────────────────────────

// InjectionSlot 注入槽位
type InjectionSlot struct {
	Name     string
	Priority int // 1=最高
	Content  string
	Filled   bool
}

// ResolveInjectionSlots 分配注入槽位
// 按优先级填充：special_date > emergence > desire > temporal
func ResolveInjectionSlots(
	specialDate string,
	emergence string,
	desire string,
	temporal string,
) []InjectionSlot {
	slots := []InjectionSlot{
		{Name: "special_date", Priority: 1, Content: specialDate},
		{Name: "emergence", Priority: 2, Content: emergence},
		{Name: "desire", Priority: 3, Content: desire},
		{Name: "temporal", Priority: 4, Content: temporal},
	}

	// 去空
	var filled []InjectionSlot
	for _, s := range slots {
		if s.Content != "" {
			s.Filled = true
			filled = append(filled, s)
		}
	}

	// 最多 3 个槽位
	if len(filled) > 3 {
		// 按 Priority 排序（已按声明顺序，直接截断）
		filled = filled[:3]
	}

	return filled
}

// ShouldApplyResponsiveTemporalInjection 是否应注入响应式时间提示
func ShouldApplyResponsiveTemporalInjection(
	gapHours float64,
	stage RelationshipStage,
	emergenceActive bool,
) bool {
	// 刚回来 → 应该
	if gapHours > 1 && gapHours < 72 && stage != StageStranger {
		return true
	}
	// 已有涌现在进行 → 不抢镜
	if emergenceActive {
		return false
	}
	return false
}

// BuildInjectionBlock 构建注入文本块
func BuildInjectionBlock(slots []InjectionSlot) string {
	if len(slots) == 0 {
		return ""
	}
	var parts []string
	for _, s := range slots {
		if s.Filled && s.Content != "" {
			parts = append(parts, s.Content)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "\n" + joinNonEmpty(parts, "\n")
}

func joinNonEmpty(parts []string, sep string) string {
	result := ""
	for i, p := range parts {
		if p != "" {
			if i > 0 && result != "" {
				result += sep
			}
			result += p
		}
	}
	return result
}
