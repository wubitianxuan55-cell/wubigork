// Package whisper — memory_taxonomy.go
// 100% 对齐 ackem memory/taxonomy.ts
// L4 记忆分类：6 领域 × 25 子类常量与衰减元数据

package whisper

// ─── 领域常量 ──────────────────────────────────────────────────

const (
	DomainIdentity   = "IDENTITY"
	DomainSocial     = "SOCIAL"
	DomainDailyLife  = "DAILY_LIFE"
	DomainPursuits   = "PURSUITS"
	DomainInnerWorld = "INNER_WORLD"
	DomainTemporal   = "TEMPORAL"
)

// Subcategories 25 子类定义
var Subcategories = map[string][]string{
	DomainIdentity:   {"BASIC_PROFILE", "LIFE_STORY", "VALUES_BELIEFS", "SELF_PERCEPTION"},
	DomainSocial:     {"OUR_BOND", "FAMILY", "FRIENDS", "PARTNER"},
	DomainDailyLife:  {"ROUTINES", "HEALTH", "LIVING_SPACE", "LIFESTYLE"},
	DomainPursuits:   {"CAREER", "LEARNING", "GOALS", "PROJECTS", "PROCEDURES"},
	DomainInnerWorld: {"MOOD", "TASTES", "VULNERABILITIES", "INSIDE_JOKES"},
	DomainTemporal:   {"NOW", "COMMITMENTS", "PLANS", "WORLD"},
}

// CategoryMeta 子类元数据
type CategoryMeta struct {
	DefaultWeight     float64
	DefaultConfidence float64
	DecayLambda       float64
	SelfRelevance     float64
	AutoRetireDays    int // 0 = 不自动退役
}

// CategoryMetaMap 25 子类元数据（对齐 ackem CATEGORY_META）
var CategoryMetaMap = map[string]CategoryMeta{
	"BASIC_PROFILE":   {3, 0.9, 0.001, 1, 0},
	"LIFE_STORY":      {3, 0.9, 0.001, 1, 0},
	"VALUES_BELIEFS":  {2, 0.8, 0.003, 0.95, 0},
	"SELF_PERCEPTION": {2, 0.75, 0.005, 1, 0},
	"OUR_BOND":        {3, 0.9, 0.001, 1, 0},
	"FAMILY":          {2, 0.85, 0.002, 0.9, 0},
	"FRIENDS":         {1.5, 0.75, 0.005, 0.85, 0},
	"PARTNER":         {2, 0.8, 0.003, 0.95, 0},
	"ROUTINES":        {1, 0.7, 0.008, 0.7, 0},
	"HEALTH":          {2, 0.85, 0.002, 0.95, 0},
	"LIVING_SPACE":    {1, 0.75, 0.01, 0.75, 0},
	"LIFESTYLE":       {1, 0.7, 0.01, 0.75, 0},
	"CAREER":          {1.5, 0.8, 0.005, 0.85, 0},
	"LEARNING":        {1.2, 0.75, 0.008, 0.8, 0},
	"GOALS":           {1.5, 0.75, 0.005, 0.85, 0},
	"PROJECTS":        {1.2, 0.75, 0.008, 0.8, 0},
	"PROCEDURES":      {2, 0.85, 0.002, 0.9, 0},
	"MOOD":            {1, 0.65, 0.05, 0.7, 0},
	"TASTES":          {1.2, 0.8, 0.005, 0.85, 0},
	"VULNERABILITIES": {2, 0.7, 0.003, 1, 0},
	"INSIDE_JOKES":    {1.2, 0.8, 0.005, 0.9, 0},
	"NOW":             {0.8, 0.65, 0.1, 0.6, 3},
	"COMMITMENTS":     {2, 0.9, 0, 0.95, 0},
	"PLANS":           {1, 0.75, 0.02, 0.75, 7},
	"WORLD":           {0.8, 0.65, 0.1, 0.55, 7},
}

// IsValidSubcategory 校验子类有效性
func IsValidSubcategory(s string) bool {
	_, ok := CategoryMetaMap[s]
	return ok
}
