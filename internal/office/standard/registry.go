package standard

import (
	"strings"
)

// 规范包机制化（v4.6.1 补课：docs/audit-2026-08-30 §C ③「无规范包机制/模板/
// 造价工程表式」）。LintText 从「单个硬编码红头 lint」升级为「注册表 + 可
// 插拔检查器」：每个规范包一个 Checker，GaeaDocumentLint 统一跑全量并聚合。

// Checker 是规范包中的一个检查器：独立命名、独立检查函数、可插拔。
// 每个 Issue 带 Spec 归属，前端按规范包分组展示。
type Checker interface {
	// Name 规范包名（如 "GB/T 9704 红头要素" / "造价工程表式"）。
	Name() string
	// Check 对 head（前若干行）+ body（其余正文）产出不合格项。
	Check(path, head, body string) []Issue
}

// Registry 规范包注册表（有序；先注册先展示）。
type Registry struct {
	checkers []Checker
}

func NewRegistry(checkers ...Checker) *Registry {
	return &Registry{checkers: append([]Checker(nil), checkers...)}
}

// Add 追加检查器（幂等：同名不重复注册）。
func (r *Registry) Add(c Checker) {
	for _, existing := range r.checkers {
		if existing.Name() == c.Name() {
			return
		}
	}
	r.checkers = append(r.checkers, c)
}

// Names 返回已注册规范包名（展示用）。
func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.checkers))
	for _, c := range r.checkers {
		out = append(out, c.Name())
	}
	return out
}

// defaultRegistry 是生产默认规范包：红头要素 + 造价工程表式。
var defaultRegistry = NewRegistry(RedheadChecker{}, CostTableChecker{})

// LintDocument 跑全部已注册检查器，聚合为一份体检报告（v4.6.1 机制化入口；
// GaeaDocumentLint 从 LintText 切换到这里）。每个 Issue 带 Spec 归属，
// Summary 按规范包分别统计。
func LintDocument(path, head, body string) LintReport {
	var issues []Issue
	for _, c := range defaultRegistry.checkers {
		issues = append(issues, c.Check(path, head, body)...)
	}
	missing := 0
	for _, it := range issues {
		if !it.Found {
			missing++
		}
	}
	return LintReport{
		Path:    path,
		Issues:  issues,
		Passed:  missing == 0,
		Summary: registrySummary(issues),
	}
}

// registrySummary 按规范包聚合缺失统计。
func registrySummary(issues []Issue) string {
	if len(issues) == 0 {
		return "无规范包检查项"
	}
	bySpec := map[string][2]int{} // spec → {total, missing}
	var order []string
	for _, it := range issues {
		spec := it.Spec
		if spec == "" {
			spec = "通用规范"
		}
		st := bySpec[spec]
		st[0]++
		if !it.Found {
			st[1]++
		}
		bySpec[spec] = st
		if !containsStr(order, spec) {
			order = append(order, spec)
		}
	}
	parts := make([]string, 0, len(order))
	totalMissing := 0
	for _, spec := range order {
		st := bySpec[spec]
		totalMissing += st[1]
		if st[1] == 0 {
			parts = append(parts, spec+" "+itoa(st[0])+"/"+itoa(st[0])+" 项符合")
		} else {
			parts = append(parts, spec+" 缺失 "+itoa(st[1])+"/"+itoa(st[0])+" 项")
		}
	}
	if totalMissing == 0 {
		return "全部规范包要素齐备：" + strings.Join(parts, "；")
	}
	return "检出 " + itoa(totalMissing) + " 项缺失：" + strings.Join(parts, "；")
}

func containsStr(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
