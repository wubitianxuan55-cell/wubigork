package schedule

// blocks.go — 斑马复刻新增区块（检视 / 点-箭 / 图面参数 / 视图）的结构校验。
//
// 定位：这些区块是"复刻斑马"的承载位（见 docs/gaea-schedule-zpert8-network-re-20261114.md
// §3/§4/§5/§6）。v1 不让 CPM 消费它们，但**落盘必须 fail-closed**——与既有
// resources/assignments/aoaLayout 同一条纪律：前端 normalize 负责"能打开"，
// Go Validate 负责"不许写坏文件"。
//
// 与前端 blocks.ts 的分工（严格镜像，改一侧必须同步另一侧）：
//   - 前端：逐条容错丢弃，宁可少数据也要让板块打开；
//   - Go：发现非法形状/越界/重复就**拒绝保存**并给出可读原因（agent 回执口径）。

import (
	"fmt"
	"strings"
)

// 区块规模上限（与前端 blocks.ts 的同名常量保持一致）。
const (
	MaxReviews   = 30
	MaxViews     = 20
	MaxAoaDots   = 20000
	MaxAoaArrows = 60000
)

// 枚举白名单（与前端 types.ts 同源）。
var (
	renderModes = map[string]bool{
		"timescaled": true, "timescaledLogic": true, "logic": true, "pdm": true, "ganttNetwork": true,
	}
	arrowKinds  = map[string]bool{"task": true, "dummy": true, "summary": true}
	viewKinds   = map[string]bool{"gantt": true, "pdm": true, "aoa": true, "usage": true, "dcma": true, "monte": true, "evm": true}
	dotStyles   = map[string]bool{"circle": true, "rect": true}
	arrowStyles = map[string]bool{"solid": true, "hollow": true}
	namePosSet  = map[string]bool{"auto": true, "horizontal": true, "vertical": true}
	tickUnitSet = map[string]bool{"yearMonth": true, "monthDay": true, "dayHour": true}
	tickLineSet = map[string]bool{"none": true, "small": true, "large": true}
	densitySet  = map[string]bool{"compact": true, "normal": true, "loose": true}
	sortDirSet  = map[string]bool{"asc": true, "desc": true}
)

// isISODateShape 只校验 YYYY-MM-DD 的形状（真实日历合法性不在此处判——
// 与既有 StartDate/Deadline 的校验强度一致）。
func isISODateShape(s string) bool {
	if len(s) != 10 || s[4] != '-' || s[7] != '-' {
		return false
	}
	for i, c := range s {
		if i == 4 || i == 7 {
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// validateBlocks 校验四个新增区块；nil 区块一律跳过（缺省=旧文件口径）。
func validateBlocks(p *Project) error {
	if err := validateReviews(p.Reviews); err != nil {
		return err
	}
	if err := validateAoaGraph(p.Aoa); err != nil {
		return err
	}
	if err := validateNetworkStyle(p.NetworkStyle); err != nil {
		return err
	}
	return validateViews(p.Views)
}

func validateReviews(reviews []Review) error {
	if len(reviews) > MaxReviews {
		return fmt.Errorf("检视条数超上限（%d）：%d", MaxReviews, len(reviews))
	}
	seen := make(map[int]bool, len(reviews))
	for _, r := range reviews {
		if r.At < 0 {
			return fmt.Errorf("检视时间非法（须为 >=0 的工作日序号）：%d", r.At)
		}
		if !isISODateShape(r.Date) {
			return fmt.Errorf("检视 %d 的日期口径应为 YYYY-MM-DD：%s", r.At, r.Date)
		}
		if seen[r.At] {
			return fmt.Errorf("检视时间重复：%d", r.At)
		}
		seen[r.At] = true
		for tid, a := range r.Actuals {
			if strings.TrimSpace(tid) == "" {
				return fmt.Errorf("检视 %d 存在空任务 id 的实际数据", r.At)
			}
			if a.Start != nil && *a.Start < 0 {
				return fmt.Errorf("检视 %d 任务 %s 实际开始非法：%d", r.At, tid, *a.Start)
			}
			if a.Finish != nil && *a.Finish < 0 {
				return fmt.Errorf("检视 %d 任务 %s 实际完成非法：%d", r.At, tid, *a.Finish)
			}
			if a.Start != nil && a.Finish != nil && *a.Finish < *a.Start {
				return fmt.Errorf("检视 %d 任务 %s 实际完成早于实际开始：%d<%d", r.At, tid, *a.Finish, *a.Start)
			}
			if a.Percent != nil && (*a.Percent < 0 || *a.Percent > 100) {
				return fmt.Errorf("检视 %d 任务 %s 完成百分比越界（0-100）：%d", r.At, tid, *a.Percent)
			}
		}
	}
	return nil
}

func validateAoaGraph(g *AoaGraph) error {
	if g == nil {
		return nil
	}
	if len(g.Dots) == 0 || len(g.Arrows) == 0 {
		return fmt.Errorf("点-箭图须同时有点和箭（dots=%d, arrows=%d）", len(g.Dots), len(g.Arrows))
	}
	if len(g.Dots) > MaxAoaDots {
		return fmt.Errorf("点数超上限（%d）：%d", MaxAoaDots, len(g.Dots))
	}
	if len(g.Arrows) > MaxAoaArrows {
		return fmt.Errorf("箭数超上限（%d）：%d", MaxAoaArrows, len(g.Arrows))
	}
	dots := make(map[string]bool, len(g.Dots))
	nums := make(map[int]string, len(g.Dots))
	for _, d := range g.Dots {
		if strings.TrimSpace(d.ID) == "" {
			return fmt.Errorf("存在空 id 的点")
		}
		if dots[d.ID] {
			return fmt.Errorf("点 id 重复：%s", d.ID)
		}
		if d.Num < 1 {
			return fmt.Errorf("点 %s 编号非法（须 >=1）：%d", d.ID, d.Num)
		}
		if prev, ok := nums[d.Num]; ok {
			return fmt.Errorf("点编号重复：%d（%s 与 %s）", d.Num, prev, d.ID)
		}
		if d.Layer != nil && (*d.Layer < 0 || *d.Layer > 10000) {
			return fmt.Errorf("点 %s 层号越界（0~10000）：%d", d.ID, *d.Layer)
		}
		dots[d.ID] = true
		nums[d.Num] = d.ID
	}
	begins, ends := 0, 0
	for _, d := range g.Dots {
		if d.IsBegin {
			begins++
		}
		if d.IsEnd {
			ends++
		}
	}
	if begins > 1 {
		return fmt.Errorf("起点事件不唯一（%d 个）", begins)
	}
	if ends > 1 {
		return fmt.Errorf("终点事件不唯一（%d 个）", ends)
	}
	if g.BeginDotID != "" && !dots[g.BeginDotID] {
		return fmt.Errorf("起点事件 id 不在点集中：%s", g.BeginDotID)
	}
	arrowIDs := make(map[string]bool, len(g.Arrows))
	for _, a := range g.Arrows {
		if strings.TrimSpace(a.ID) == "" {
			return fmt.Errorf("存在空 id 的箭")
		}
		if arrowIDs[a.ID] {
			return fmt.Errorf("箭 id 重复：%s", a.ID)
		}
		arrowIDs[a.ID] = true
		if !dots[a.From] {
			return fmt.Errorf("箭 %s 尾事件不存在：%s", a.ID, a.From)
		}
		if !dots[a.To] {
			return fmt.Errorf("箭 %s 头事件不存在：%s", a.ID, a.To)
		}
		if a.From == a.To {
			return fmt.Errorf("箭 %s 起止同一事件（自环）：%s", a.ID, a.From)
		}
		if !arrowKinds[a.Kind] {
			return fmt.Errorf("箭 %s 类型非法（task|dummy|summary）：%s", a.ID, a.Kind)
		}
		if a.Lag < -9999 || a.Lag > 9999 {
			return fmt.Errorf("箭 %s 时距越界（-9999~9999）：%d", a.ID, a.Lag)
		}
	}
	return nil
}

func validateNetworkStyle(s *NetworkStyle) error {
	if s == nil {
		return nil
	}
	type enumCheck struct {
		name  string
		value string
		ok    map[string]bool
	}
	for _, e := range []enumCheck{
		{"mode", s.Mode, renderModes},
		{"dotStyle", s.DotStyle, dotStyles},
		{"arrowStyle", s.ArrowStyle, arrowStyles},
		{"namePos", s.NamePos, namePosSet},
		{"tickUnit", s.TickUnit, tickUnitSet},
		{"tickLines", s.TickLines, tickLineSet},
		{"density", s.Density, densitySet},
	} {
		if e.value != "" && !e.ok[e.value] {
			return fmt.Errorf("图面参数 %s 非法：%s", e.name, e.value)
		}
	}
	for _, r := range []struct {
		name string
		v    *int
		lo   int
		hi   int
	}{
		{"nodeRadius", s.NodeRadius, 4, 200},
		{"headHeight", s.HeadHeight, 2, 60},
		{"layerHeight1", s.LayerHeight1, 20, 2000},
		{"layerHeight2", s.LayerHeight2, 20, 2000},
		{"maxPages", s.MaxPages, 1, 1000},
	} {
		if r.v != nil && (*r.v < r.lo || *r.v > r.hi) {
			return fmt.Errorf("图面参数 %s 越界（%d~%d）：%d", r.name, r.lo, r.hi, *r.v)
		}
	}
	if s.AspectMax != nil && (*s.AspectMax < 0.1 || *s.AspectMax > 20) {
		return fmt.Errorf("图面参数 aspectMax 越界（0.1~20）：%v", *s.AspectMax)
	}
	return nil
}

func validateViews(views []View) error {
	if len(views) > MaxViews {
		return fmt.Errorf("视图条数超上限（%d）：%d", MaxViews, len(views))
	}
	seen := make(map[string]bool, len(views))
	for _, v := range views {
		if strings.TrimSpace(v.ID) == "" {
			return fmt.Errorf("存在空 id 的视图")
		}
		if seen[v.ID] {
			return fmt.Errorf("视图 id 重复：%s", v.ID)
		}
		seen[v.ID] = true
		if strings.TrimSpace(v.Name) == "" {
			return fmt.Errorf("视图 %s 名称为空", v.ID)
		}
		if !viewKinds[v.Kind] {
			return fmt.Errorf("视图 %s 类型非法（gantt|pdm|aoa|usage|dcma|monte|evm）：%s", v.ID, v.Kind)
		}
		if v.Sort != nil {
			if strings.TrimSpace(v.Sort.Field) == "" {
				return fmt.Errorf("视图 %s 排序字段为空", v.ID)
			}
			if !sortDirSet[v.Sort.Dir] {
				return fmt.Errorf("视图 %s 排序方向非法（asc|desc）：%s", v.ID, v.Sort.Dir)
			}
		}
		if v.RenderMode != "" && !renderModes[v.RenderMode] {
			return fmt.Errorf("视图 %s 显示模式非法：%s", v.ID, v.RenderMode)
		}
		if len(v.GroupBy) > 8 {
			return fmt.Errorf("视图 %s 分组层级超上限（8）：%d", v.ID, len(v.GroupBy))
		}
	}
	return nil
}
