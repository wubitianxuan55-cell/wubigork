// Package skilldistill journal 历史蒸馏的模式挖掘核心（阶段七 7.2-2）：
// 把 journal 证据链的变更卡投影为会话级工具应用流，归一化折叠后做跨会话
// 精确匹配，找出「重复 ≥ MinRepeat 次的任务模式」供上层提议结晶为技能。
//
// 设计纪律（先例 internal/routesuggest、internal/novelgate）：零 IO、零外部
// 依赖——全部数据由调用方组装，便于表驱动测试。不 import evidence（Record
// 本地投影防包环）。
//
// V1 范围裁决（规格 进度计划/gaea-journal-distill-7-2-2-20260915.md §0）：
// 模式 = 折叠后会话级序列的**精确匹配**；滑窗/子序列挖掘不进 V1（误报风险
// 高，真实数据先行——观察池）。
package skilldistill

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// MinRepeat 跨会话重复门槛：模式出现在 ≥3 个不同会话才提议（plan「重复 ≥ N 次」）。
	MinRepeat = 3
	// MaxCandidates 同时至多提议条数（宁少勿扰，先例 routesuggest 单域至多 1 条）。
	MaxCandidates = 3
	// MinFlowSteps 折叠后最少步骤：单步不成流程。
	MinFlowSteps = 2
	// MaxFlowSteps 折叠后最长步骤：超长序列过于特异，不参评。
	MaxFlowSteps = 12
)

// evidence 行上限与单行截断（防视图膨胀）。
const (
	maxEvidenceLines = 5
	evidenceLineMax  = 120
	maxSessions      = 5
)

// Record 是 evidence.ChangeRecord 的本地投影（调用方映射；防包环不 import evidence）。
type Record struct {
	SessionID string
	Tool      string
	Target    string
	At        int64 // unix ms
}

// FlowStep 一次工具应用。
type FlowStep struct {
	Tool   string
	Target string
	At     int64
}

// Flow 一个会话内的工具应用序列（At 升序）。
type Flow struct {
	SessionID string
	Steps     []FlowStep
}

// PatternStep 归一化步骤：Tool + Shape（目标扩展名小写含点；无扩展名 "*"）。
// Target 具体文件名不参与模式（同名不同文件算同形）。
type PatternStep struct {
	Tool  string
	Shape string
}

// Candidate 一个重复模式候选（上层据此提议「结晶为技能」）。
type Candidate struct {
	ID       string   // "jd-"+sha256(模式串接)[:8hex]，确定性、复算幂等
	Pattern  []PatternStep
	Repeat   int      // 出现该模式的不同会话数
	Sessions []string // 证据会话（最近优先，≤5）
	Evidence []string // 人类可读证据行（≤5，行 ≤120 rune）
	FirstAt  int64
	LastAt   int64
}

// MineFlows 把 journal 变更卡按会话聚成流程：SessionID/Tool 为空的记录跳过；
// 组内按 (At, Tool, Target) 升序（确定性）；输出按 SessionID 升序。
func MineFlows(records []Record) []Flow {
	bySession := map[string][]FlowStep{}
	for _, r := range records {
		if r.SessionID == "" || r.Tool == "" {
			continue
		}
		bySession[r.SessionID] = append(bySession[r.SessionID], FlowStep{
			Tool: r.Tool, Target: r.Target, At: r.At,
		})
	}
	ids := make([]string, 0, len(bySession))
	for id := range bySession {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]Flow, 0, len(ids))
	for _, id := range ids {
		steps := bySession[id]
		sort.Slice(steps, func(i, j int) bool {
			if steps[i].At != steps[j].At {
				return steps[i].At < steps[j].At
			}
			if steps[i].Tool != steps[j].Tool {
				return steps[i].Tool < steps[j].Tool
			}
			return steps[i].Target < steps[j].Target
		})
		out = append(out, Flow{SessionID: id, Steps: steps})
	}
	return out
}

// shapeOf 目标形状：扩展名小写含点；无扩展名 "*"。
func shapeOf(target string) string {
	if ext := strings.ToLower(filepath.Ext(target)); ext != "" {
		return ext
	}
	return "*"
}

// collapse 折叠连续同 (Tool, Shape) 步骤（Target 不同也算同形）。
func collapse(steps []FlowStep) []PatternStep {
	out := make([]PatternStep, 0, len(steps))
	for _, s := range steps {
		p := PatternStep{Tool: s.Tool, Shape: shapeOf(s.Target)}
		if n := len(out); n > 0 && out[n-1] == p {
			continue
		}
		out = append(out, p)
	}
	return out
}

// patternKey 模式串接（ID 与分组共用同一口径）：每步 "Tool:Shape"，步间 ">"。
func patternKey(pattern []PatternStep) string {
	parts := make([]string, len(pattern))
	for i, p := range pattern {
		parts[i] = p.Tool + ":" + p.Shape
	}
	return strings.Join(parts, ">")
}

// BuildPatternID 确定性 ID："jd-"+sha256(模式串接)前 8 hex。同输入必同 ID。
func BuildPatternID(pattern []PatternStep) string {
	sum := sha256.Sum256([]byte(patternKey(pattern)))
	return "jd-" + hex.EncodeToString(sum[:])[:8]
}

// ParsePatternID 校验 ID 形状（jd-+8 小写 hex）；合法返回 true（防御残留，
// 先例 routesuggest.ParseSuggestionID）。
func ParsePatternID(id string) bool {
	if len(id) != 11 || !strings.HasPrefix(id, "jd-") {
		return false
	}
	for _, c := range id[3:] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// PatternLine 步骤行渲染（视图层直显）："edit_file .md"。
func PatternLine(p PatternStep) string {
	return p.Tool + " " + p.Shape
}

// Distill 挖掘重复模式：每会话流折叠 → 全序列精确匹配分组 → repeat ≥ MinRepeat
// 且步数 ∈ [MinFlowSteps, MaxFlowSteps] → ignored 中的 ID 静默跳过 → 排序
// （repeat 降序、同数按 ID 升序）→ 截 MaxCandidates。
func Distill(flows []Flow, ignored map[string]struct{}) []Candidate {
	type group struct {
		pattern  []PatternStep
		sessions []Flow // 命中该模式的会话流（每会话至多一流，天然去重）
	}
	groups := map[string]*group{}
	for _, f := range flows {
		pattern := collapse(f.Steps)
		if len(pattern) < MinFlowSteps || len(pattern) > MaxFlowSteps {
			continue
		}
		key := patternKey(pattern)
		g := groups[key]
		if g == nil {
			g = &group{pattern: pattern}
			groups[key] = g
		}
		g.sessions = append(g.sessions, f)
	}
	var out []Candidate
	for _, g := range groups {
		if len(g.sessions) < MinRepeat {
			continue
		}
		id := BuildPatternID(g.pattern)
		if _, skip := ignored[id]; skip {
			continue
		}
		out = append(out, buildCandidate(id, g.pattern, g.sessions))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Repeat != out[j].Repeat {
			return out[i].Repeat > out[j].Repeat
		}
		return out[i].ID < out[j].ID
	})
	if len(out) > MaxCandidates {
		out = out[:MaxCandidates]
	}
	return out
}

// buildCandidate 组装候选：证据会话最近优先（按会话末步 At 降序）截 5；
// FirstAt/LastAt 取全部命中会话的最小/最大步时间。
func buildCandidate(id string, pattern []PatternStep, sessions []Flow) Candidate {
	sorted := make([]Flow, len(sessions))
	copy(sorted, sessions)
	sort.Slice(sorted, func(i, j int) bool {
		return lastAt(sorted[i]) > lastAt(sorted[j])
	})
	c := Candidate{ID: id, Pattern: pattern, Repeat: len(sorted)}
	for _, f := range sorted {
		if c.FirstAt == 0 || firstAt(f) < c.FirstAt {
			c.FirstAt = firstAt(f)
		}
		if lastAt(f) > c.LastAt {
			c.LastAt = lastAt(f)
		}
	}
	n := len(sorted)
	if n > maxSessions {
		n = maxSessions
	}
	for _, f := range sorted[:n] {
		c.Sessions = append(c.Sessions, f.SessionID)
		c.Evidence = append(c.Evidence, evidenceLine(f))
	}
	return c
}

func firstAt(f Flow) int64 {
	if len(f.Steps) == 0 {
		return 0
	}
	return f.Steps[0].At
}

func lastAt(f Flow) int64 {
	if len(f.Steps) == 0 {
		return 0
	}
	return f.Steps[len(f.Steps)-1].At
}

// evidenceLine 证据行："会话 s1（2026-09-15）：edit_file 周报.md → write_file 汇总.xlsx"。
// 整行超 120 rune 截断加 "…"。
func evidenceLine(f Flow) string {
	date := time.UnixMilli(lastAt(f)).Format("2006-01-02")
	steps := make([]string, len(f.Steps))
	for i, s := range f.Steps {
		steps[i] = s.Tool + " " + filepath.Base(s.Target)
	}
	line := fmt.Sprintf("会话 %s（%s）：%s", f.SessionID, date, strings.Join(steps, " → "))
	return truncRunes(line, evidenceLineMax)
}

func truncRunes(s string, max int) string {
	if r := []rune(s); len(r) > max {
		return string(r[:max]) + "…"
	}
	return s
}
