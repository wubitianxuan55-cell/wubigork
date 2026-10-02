package app

// ── AP1-09 收口用例：「什么级别算阻断」的唯一判据 ──────────────────
//
// 审计原文：该判据此前在 5 处各写一份且大小写口径不一致——
//   create_chapter_context.go planSeverityBlocking（大小写敏感、不 Trim）、
//   novel_plan_handler.go     planBlockingSeverity（ToUpper+TrimSpace）、
//   converge_handler.go       内联 ==（敏感）、
//   novel_book_health.go      两处内联 ==（敏感）。
// 后果：写前硬闸与计划落盘校验对同一条问题可能结论相反（「保存被拒但生成却起来了」）。
//
// 本文件钉三件事：
//   1. 唯一判据 severityBlocking 的四形态（"s1" / " S2 " / "S3" / ""）口径；
//   2. 两道闸（写前硬闸 planPrecheck、计划落盘校验 planBlockingMessage）对同一份
//      问题清单结论一致（反向对照：大小写敏感实现下 "s1" 形态直接分叉）；
//   3. 改坏能红锚点：把 severityBlocking 改回大小写敏感 → 本文件用例 FAIL。

import (
	"testing"

	"github.com/gaea/gaea/internal/novelgate"
	"github.com/gaea/gaea/internal/types"
)

// TestSeverityBlockingForms 唯一判据四形态口径：大小写与首尾空白不敏感。
func TestSeverityBlockingForms(t *testing.T) {
	cases := []struct {
		name string
		sev  string
		want bool
	}{
		{"小写 s1 也阻断", "s1", true},
		{"两侧空白 S2 也阻断", " S2 ", true},
		{"S3 仅提示", "S3", false},
		{"空串不阻断", "", false},
		{"纯空白不阻断", "  ", false},
		{"标准形态 S1", "S1", true},
		{"标准形态 S2", "S2", true},
		{"S4 仅提示", "S4", false},
		{"未知级别不阻断", "S9", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := severityBlocking(c.sev); got != c.want {
				t.Fatalf("severityBlocking(%q) = %v, want %v", c.sev, got, c.want)
			}
		})
	}
}

// TestSeverityBlockingGatesAgree 两道闸对同一份问题清单结论必须一致。
//
// 用真实判据 novelgate.PlanContractIssues 造两组问题（缺计划=S1 阻断 / 齐备=零问题），
// 再对照「写前硬闸」与「落盘校验」两条通道：
//   - 硬闸：severityBlocking（=旧 planSeverityBlocking 的位置）；
//   - 落盘校验：planBlockingMessage（内部同样走唯一判据，非空=拒绝落盘）。
//
// 敏感范围说明（不夸大）：两条通道现在共用同一函数，所以本用例**不会**对
// 「size 敏感 vs ToUpper」这一处改动变红——那种改动的敏感点是
// TestSeverityBlockingForms 的四形态。本用例钉的是另一类回归：有人再次把判据
// 复制成第二份实现（改其中一份口径）时，两条通道会立刻分叉。同向的真实消费端
// 由 TestSeverityBlockingGateIntegration（planPrecheck）与
// TestConvergeCheckSeverityBuckets（convergeCheck）覆盖。
func TestSeverityBlockingGatesAgree(t *testing.T) {
	// s1S2Forms：生产侧判据造出来的真实问题，把严重度改写成各种「等价写法」，
	// 模拟混入小写/空白的来源（数据面回落、外部导入、未来生产者）。
	missing := novelgate.PlanContractIssues(1, nil, nil)
	if len(missing) != 1 || missing[0].Severity != "S1" {
		t.Fatalf("夹具前提：缺计划应为单条 S1，got %+v", missing)
	}
	var s1S2Forms []types.PlanProblem
	for _, sev := range []string{"S1", "s1", " S2 ", "S2"} {
		p := missing[0]
		p.Severity = sev
		s1S2Forms = append(s1S2Forms, p)
	}

	complete := gatePlan(1)
	clear := novelgate.PlanContractIssues(1, &complete, nil)
	for _, p := range clear {
		if severityBlocking(p.Severity) {
			t.Fatalf("夹具前提：齐备计划不应有阻断级问题，got %+v", p)
		}
	}

	for _, tc := range []struct {
		name     string
		problems []types.PlanProblem
		want     bool
	}{
		{"阻断形态逐条：硬闸与落盘校验都拦", s1S2Forms, true},
		{"齐备计划：两条通道都放行", clear, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// 写前硬闸口径（planPrecheck 的 report.Blocking 判据）。
			gate := false
			for _, p := range tc.problems {
				if severityBlocking(p.Severity) {
					gate = true
				}
			}
			// 计划落盘校验口径（planBlockingMessage 非空 = 拒绝落盘）。
			persist := planBlockingMessage(tc.problems) != ""
			if gate != persist {
				t.Fatalf("两闸结论分叉：写前硬闸=%v 落盘校验=%v（问题清单 %+v）",
					gate, persist, tc.problems)
			}
			if gate != tc.want {
				t.Fatalf("结论 = %v, want %v（问题清单 %+v）", gate, tc.want, tc.problems)
			}
		})
	}
}

// TestSeverityBlockingGateIntegration 端到端：真实项目 + 真实判据下，
// planPrecheck 的 Blocking/Allowed 与 severityBlocking 逐条一致（口径统一后
// 硬闸不再有第二条实现可漂移）。
func TestSeverityBlockingGateIntegration(t *testing.T) {
	a, pm := newChapterGateBareApp(t)

	// 缺计划（S1）→ 阻断。
	rep, err := a.planPrecheck(1)
	if err != nil {
		t.Fatalf("预检报错: %v", err)
	}
	if !rep.Blocking || rep.Allowed {
		t.Fatalf("缺计划应阻断: %+v", *rep)
	}
	for _, p := range append(append([]types.PlanProblem{}, rep.PlanProblems...), rep.OutlineIssues...) {
		if !severityBlocking(p.Severity) {
			t.Fatalf("报告的阻断问题严重度 %q 不被唯一判据认可: %+v", p.Severity, p)
		}
	}

	// 齐备计划 + 齐备大纲（S3 提示级字段也补齐）→ 放行。
	gateSeedPlan(t, pm, map[int]types.ChapterPlan{1: gatePlan(1)})
	gateSeedOutline(t, pm, gateOutlineNode(1, []string{"拾起玄铁剑"}, "沉重"))
	rep, err = a.planPrecheck(1)
	if err != nil {
		t.Fatalf("预检报错: %v", err)
	}
	if rep.Blocking || !rep.Allowed {
		t.Fatalf("齐备计划应放行: %+v", *rep)
	}
	if msg := planBlockingMessage(rep.PlanProblems); msg != "" {
		t.Fatalf("齐备计划的落盘校验应放行，却拒绝: %s", msg)
	}
	for _, p := range rep.PlanProblems {
		if severityBlocking(p.Severity) {
			t.Fatalf("齐备计划不应有阻断级问题: %+v", p)
		}
	}
}
