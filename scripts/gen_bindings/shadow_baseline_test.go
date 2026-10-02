// 遮蔽基线闸的自测（批次十三 round 15，审计 X1-13 精化）：
// 归因靠**名字集合差**，不靠「按字典序切片」——切片形态下，新增的名字排序靠前时
// 会把在册条目当成新增报出来（旧实现 `shadow[len(baseline):]`）。这里用纯函数
// 夹具把两种误报都钉住：①新增只报新增的名字；②在册消失不许报成新增。
package main

import (
	"strings"
	"testing"
)

func sp(name, file string, line int) shadowPair {
	return shadowPair{Name: name, Receiver: "core", File: file, Line: line}
}

func names(ps []shadowPair) []string {
	out := make([]string, 0, len(ps))
	for _, p := range ps {
		out = append(out, p.Name)
	}
	return out
}

func sameNames(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ① 实测 == 在册 → exit 0（含「在册位置漂移只提示」的形态）。
func TestReportShadowDriftInSync(t *testing.T) {
	baseline := []shadowPair{sp("Zeta", "internal/app/z.go", 10)}
	r := reportShadowDrift(baseline, baseline)
	if r.code != 0 {
		t.Fatalf("实测==在册应 exit 0，实得 %d：%v", r.code, r.lines)
	}
	joined := strings.Join(r.lines, "\n")
	if strings.Contains(joined, "新增") || strings.Contains(joined, "消失") {
		t.Fatalf("无漂移却报了漂移：%v", r.lines)
	}

	movedOnly := []shadowPair{sp("Zeta", "internal/app/z.go", 99)}
	r = reportShadowDrift(movedOnly, baseline)
	if r.code != 0 {
		t.Fatalf("仅位置漂移不应拦门，实得 %d：%v", r.code, r.lines)
	}
	if !strings.Contains(strings.Join(r.lines, "\n"), "位置漂移") {
		t.Fatalf("位置漂移应如实提示：%v", r.lines)
	}
}

// ② 新增的名字字典序最靠前 → 只报这一个（旧切片实现会指错条目）。
func TestReportShadowDriftOnlyListsAddedName(t *testing.T) {
	baseline := []shadowPair{sp("Zeta", "internal/app/z.go", 10), sp("Zulu", "internal/app/z.go", 20)}
	measured := []shadowPair{sp("Alpha", "internal/app/a.go", 5), sp("Zeta", "internal/app/z.go", 10), sp("Zulu", "internal/app/z.go", 20)}
	added, gone, moved := shadowDiff(measured, baseline)
	if !sameNames(names(added), []string{"Alpha"}) {
		t.Fatalf("added = %v，want [Alpha]", names(added))
	}
	if len(gone) != 0 || len(moved) != 0 {
		t.Fatalf("gone=%v moved=%v，二者都应为空", names(gone), moved)
	}
	r := reportShadowDrift(measured, baseline)
	if r.code != 1 {
		t.Fatalf("新增遮蔽应 exit 1，实得 %d：%v", r.code, r.lines)
	}
	if !strings.Contains(strings.Join(r.lines, "\n"), "Alpha") {
		t.Fatalf("新增项未列出：%v", r.lines)
	}
	for _, line := range r.lines {
		if strings.HasPrefix(line, "  - ") && (strings.Contains(line, "Zeta") || strings.Contains(line, "Zulu")) {
			t.Fatalf("在册条目被误报成明细项：%q", line)
		}
	}
}

// ③ 在册项消失 → 如实报「消失」并提示同步基线，不得误报成新增。
func TestReportShadowDriftGoneIsNotNew(t *testing.T) {
	baseline := []shadowPair{sp("Zeta", "internal/app/z.go", 10), sp("Zulu", "internal/app/z.go", 20)}
	measured := []shadowPair{sp("Zulu", "internal/app/z.go", 20)}
	added, gone, _ := shadowDiff(measured, baseline)
	if len(added) != 0 {
		t.Fatalf("消失不应算新增，added = %v", names(added))
	}
	if !sameNames(names(gone), []string{"Zeta"}) {
		t.Fatalf("gone = %v，want [Zeta]", names(gone))
	}
	r := reportShadowDrift(measured, baseline)
	if r.code != 1 {
		t.Fatalf("在册项消失应 exit 1（提示同步基线），实得 %d：%v", r.code, r.lines)
	}
	joined := strings.Join(r.lines, "\n")
	if !strings.Contains(joined, "消失") || !strings.Contains(joined, "同步") {
		t.Fatalf("应如实报消失并提示同步基线：%v", r.lines)
	}
	// 注意：gone 文案里刻意写了「这不是新增」这句人话，所以断言要看**新增段小标题**
	//（"新增遮蔽 N 处"），不能只搜「新增」二字——首版就是这么误判自己的。
	if strings.Contains(joined, "新增遮蔽") {
		t.Fatalf("消失被误报成新增：%v", r.lines)
	}
	for _, line := range r.lines {
		if strings.HasPrefix(line, "  - ") && !strings.Contains(line, "Zeta") {
			t.Fatalf("消失明细指错条目：%q", line)
		}
	}
}

// ④ 在册清单自身不许有重名（集合语义下重名会被静默吞掉，等于基线虚高）。
func TestShadowBaselineRegistryNoDuplicateNames(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range shadowBaseline {
		if s.Name == "" || s.File == "" {
			t.Fatalf("在册清单有条目缺名字/文件：%+v", s)
		}
		if seen[s.Name] {
			t.Fatalf("在册清单重名：%s（集合语义下重复项会被静默吞掉）", s.Name)
		}
		seen[s.Name] = true
	}
	if len(shadowBaseline) == 0 {
		t.Fatal("在册清单为空——若已把遮蔽归零，请连同本闸一并评估后再删清单")
	}
}
